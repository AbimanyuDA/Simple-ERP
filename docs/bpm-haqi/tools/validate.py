"""Validate BPMN XML, graph, DI, cross-file calls, source drift and API coverage."""
from __future__ import annotations
import argparse
import hashlib
import json
import re
import urllib.request
from urllib.parse import unquote
from collections import Counter
from pathlib import Path
from lxml import etree as E

OUT = Path(__file__).resolve().parents[1]
ROOT = OUT.parents[1]
NS = {"b": "http://www.omg.org/spec/BPMN/20100524/MODEL",
      "d": "http://www.omg.org/spec/BPMN/20100524/DI",
      "dc": "http://www.omg.org/spec/DD/20100524/DC",
      "di": "http://www.omg.org/spec/DD/20100524/DI"}
SCHEMAS = ("BPMN20.xsd", "Semantic.xsd", "BPMNDI.xsd", "DC.xsd", "DI.xsd")


def expected_routes():
    result = {}
    for file in (ROOT/"backend/internal").glob("*/adapter/http/*.go"):
        if file.name.endswith("_test.go"):
            continue
        prefix = "/users" if file.as_posix().endswith("auth/adapter/http/users.go") else ""
        if file.as_posix().endswith("masterdata/adapter/http/handler.go"):
            prefix = "/master-data"
        text = file.read_text(encoding="utf-8")
        for match in re.finditer(r'\br\.(Get|Post|Patch|Put|Delete)\("([^"]+)"', text):
            method, route = match.groups()
            route = (prefix+route).rstrip("/")
            source = f"{file.relative_to(ROOT).as_posix()}:{text[:match.start()].count(chr(10))+1}"
            if "{action}" in route:
                actions = {
                    "/purchase-requests/{id}/{action}": ["submit", "approve", "reject", "cancel", "copy", "purchase-order"],
                    "/sales-orders/{id}/{action}": ["validate-stock", "submit", "approve", "reject", "confirm", "cancel", "copy"],
                    "/inventory-adjustments/{id}/{action}": ["submit", "approve", "reject", "cancel", "post", "reverse"],
                }[route]
                for action in actions:
                    result[f"{method.upper()} {route.replace('{action}', action)}"] = source
            else:
                result[f"{method.upper()} {route}"] = source
    main = "backend/cmd/api/main.go"
    for route in ["POST /auth/refresh", "POST /auth/logout", "GET /auth/session"]:
        result[route] = main
    for method in ["GET", "PATCH"]:
        result[f"{method} /settings/organization"] = "backend/internal/masterdata/adapter/http/settings.go"
    result["GET /healthz"] = "backend/internal/platform/httpserver/router.go"
    return result


def main():
    args = argparse.ArgumentParser()
    args.add_argument("--fetch-schema", action="store_true", help="Download official OMG schemas into this documentation folder.")
    args.add_argument("--require-schema", action="store_true")
    flags = args.parse_args()
    schema_dir = OUT/"tools/schema"
    if flags.fetch_schema:
        schema_dir.mkdir(exist_ok=True)
        for name in SCHEMAS:
            url = "https://www.omg.org/spec/BPMN/20100501/"+name
            with urllib.request.urlopen(url, timeout=30) as response:
                (schema_dir/name).write_bytes(response.read())
    schema = E.XMLSchema(E.parse(str(schema_dir/"BPMN20.xsd"))) if (schema_dir/"BPMN20.xsd").exists() else None
    errors = []
    if flags.require_schema and schema is None:
        errors.append("Official OMG XSD schema is required but missing.")
    manifest = json.loads((OUT/"manifest.json").read_text(encoding="utf-8"))
    count = Counter()
    all_models = {x["id"]: x for x in manifest["models"]}
    actual_branches = {}
    for model in manifest["models"]:
        path = OUT/model["file"]
        if path.parent.name.endswith("-p2p"):
            mirror = OUT/f"level-{model['level']}"/path.name
            if not mirror.exists() or mirror.read_bytes() != path.read_bytes():
                errors.append(f"Missing or stale general-level P2P copy: {mirror.relative_to(OUT)}")
        tree = E.parse(str(path))
        if schema is not None and not schema.validate(tree):
            errors.extend(f"{path.name}: {err.message}" for err in schema.error_log)
        ids = tree.xpath("//@id")
        if len(ids) != len(set(ids)):
            errors.append(f"{path.name}: duplicate IDs")
        process = tree.find("b:process", NS)
        nodes = {n.get("id"): n for n in process if E.QName(n).localname in (
            "startEvent", "endEvent", "userTask", "serviceTask", "callActivity", "exclusiveGateway")}
        flows = process.findall("b:sequenceFlow", NS)
        preview = E.parse(str(OUT/"preview"/(model["id"]+".svg")))
        default_flows = [n.get("default") for n in nodes.values() if n.get("default")]
        default_markers = preview.xpath("//*[local-name()='line' and @class='default-flow-marker']")
        if len(default_markers) != len(default_flows):
            errors.append(f"{path.name}: SVG default-flow markers do not match BPMN defaults")
        call_markers = preview.xpath("//*[local-name()='g' and @class='collapsed-call-marker']")
        if len(call_markers) != sum(E.QName(n).localname == "callActivity" for n in nodes.values()):
            errors.append(f"{path.name}: SVG collapsed-call markers do not match BPMN call activities")
        prefix = model["id"].replace("-", "_")+"_"
        actual_branches[model["id"]] = {
            (f.get("sourceRef").removeprefix(prefix), f.get("name", ""),
             f.get("targetRef").removeprefix(prefix))
            for f in flows
        }
        incoming = {key: [] for key in nodes}
        outgoing = {key: [] for key in nodes}
        for flow in flows:
            a, b = flow.get("sourceRef"), flow.get("targetRef")
            if a not in nodes or b not in nodes:
                errors.append(f"{path.name}: dangling flow {flow.get('id')}")
                continue
            outgoing[a].append(b); incoming[b].append(a)
            if not tree.xpath("//d:BPMNEdge[@bpmnElement=$id]", namespaces=NS, id=flow.get("id")):
                errors.append(f"{path.name}: missing edge DI")
            if E.QName(nodes[a]).localname == "exclusiveGateway" and E.QName(nodes[b]).localname == "exclusiveGateway":
                errors.append(f"{path.name}: adjacent gateways {a} -> {b}")
        starts = [key for key,n in nodes.items() if E.QName(n).localname == "startEvent"]
        ends = [key for key,n in nodes.items() if E.QName(n).localname == "endEvent"]
        if model["id"] == "l1-p2p" or path.parent.name.endswith("-p2p"):
            end_names = [nodes[key].get("name", "") for key in ends]
            if len(end_names) != len(set(end_names)):
                errors.append(f"{path.name}: repeated P2P end-event meaning")
        if model["id"] in {"l2-p2p-correction", "l2-purchase-correction"}:
            merge = next((key for key in nodes if key.endswith("_merge")), None)
            if len(ends) != 1 or merge is None or len(incoming[merge]) != 5 or len(outgoing[merge]) != 1:
                errors.append(f"{path.name}: correction choices must converge to one end")
        def reachable(seed, links):
            seen, stack = set(), list(seed)
            while stack:
                key = stack.pop()
                if key not in seen:
                    seen.add(key); stack.extend(links[key])
            return seen
        if not starts or not ends or reachable(starts, outgoing) != set(nodes) or reachable(ends, incoming) != set(nodes):
            errors.append(f"{path.name}: unreachable node or no path to end")
        lane_refs = tree.xpath("//b:lane/b:flowNodeRef/text()", namespaces=NS)
        if Counter(lane_refs) != Counter(nodes.keys()):
            errors.append(f"{path.name}: every node must belong to exactly one lane")
        for key, node in nodes.items():
            kind = E.QName(node).localname
            if kind == "startEvent" and incoming[key] or kind == "endEvent" and outgoing[key]:
                errors.append(f"{path.name}: invalid event direction {key}")
            if kind == "exclusiveGateway":
                outgoing_flows = [f for f in flows if f.get("sourceRef") == key]
                if len(outgoing_flows) > 1:
                    default_id = node.get("default")
                    if default_id and sum(f.get("id") == default_id for f in outgoing_flows) != 1:
                        errors.append(f"{path.name}: gateway default does not reference an outgoing flow {key}")
                    for flow in outgoing_flows:
                        if flow.get("id") != default_id and (not flow.get("name") or flow.find("b:conditionExpression", NS) is None):
                            errors.append(f"{path.name}: decision branch needs a named condition {flow.get('id')}")
                elif len(incoming[key]) < 2 or len(outgoing_flows) != 1 or node.get("default"):
                    errors.append(f"{path.name}: invalid converging gateway {key}")
            expected_in = {f.get("id") for f in flows if f.get("targetRef") == key}
            expected_out = {f.get("id") for f in flows if f.get("sourceRef") == key}
            if set(node.xpath("b:incoming/text()", namespaces=NS)) != expected_in or set(node.xpath("b:outgoing/text()", namespaces=NS)) != expected_out:
                errors.append(f"{path.name}: incoming/outgoing mismatch")
            if not tree.xpath("//d:BPMNShape[@bpmnElement=$id]", namespaces=NS, id=key):
                errors.append(f"{path.name}: missing shape DI {key}")
            if kind == "callActivity":
                prefix, target = node.get("calledElement").split(":")
                uri = node.nsmap[prefix]
                imports = tree.xpath("//b:import[@namespace=$uri]", namespaces=NS, uri=uri)
                if len(imports) != 1:
                    errors.append(f"{path.name}: unresolved call namespace")
                    continue
                childpath = (path.parent/imports[0].get("location")).resolve()
                child = E.parse(str(childpath))
                if not child.xpath("//b:process[@id=$target]", namespaces=NS, target=target):
                    errors.append(f"{path.name}: unresolved call {target}")
        # Rectangular task/event bounds must not intersect other node bounds.
        boxes = []
        for shape in tree.xpath("//d:BPMNShape", namespaces=NS):
            if shape.get("bpmnElement") not in nodes:
                continue
            box = shape.find("dc:Bounds", NS)
            boxes.append((shape.get("bpmnElement"), *(float(box.get(k)) for k in ("x","y","width","height"))))
        for i, a in enumerate(boxes):
            for b in boxes[i+1:]:
                if a[1] < b[1]+b[3] and b[1] < a[1]+a[3] and a[2] < b[2]+b[4] and b[2] < a[2]+a[4]:
                    errors.append(f"{path.name}: overlapping nodes {a[0]} and {b[0]}")
        for flow in flows:
            edge = tree.xpath("//d:BPMNEdge[@bpmnElement=$id]", namespaces=NS, id=flow.get("id"))[0]
            points = [(float(p.get("x")), float(p.get("y"))) for p in edge.findall("di:waypoint", NS)]
            for node_id, x, y, w, h in boxes:
                if node_id in {flow.get("sourceRef"), flow.get("targetRef")}:
                    continue
                for (ax, ay), (bx, by) in zip(points, points[1:]):
                    crosses = (abs(ay-by) < 0.01 and y < ay < y+h and
                               max(min(ax,bx),x) < min(max(ax,bx),x+w)) or (abs(ax-bx) < 0.01 and x < ax < x+w and
                               max(min(ay,by),y) < min(max(ay,by),y+h))
                    if crosses:
                        errors.append(f"{path.name}: edge {flow.get('id')} crosses node {node_id}")
                        break
        count[model["level"]] += 1
    # These branches encode real state transitions. XML validity alone cannot
    # detect an invoice exception flowing from the POSTED outcome.
    expected_branches = {
        "l1-p2p": [("mode", "Proses baru", "stage0"),
                    ("mode", "Koreksi", "correction"),
                    ("gate0", "Ya", "stage1"),
                    ("gate1", "Ya", "stage2"),
                    ("gate2", "Ya", "end")],
        "l2-po-gr": [("gate3", "PO RECEIVED", "end"),
                     ("gate3", "PO PARTIALLY_RECEIVED", "stage2"),
                     ("gate3", "GR tidak POSTED", "exit3")],
        "l2-delivery": [("gate1", "SO DELIVERED", "end"),
                        ("gate1", "SO PARTIALLY_DELIVERED", "stage0")],
        "l2-ap": [("gate1", "POSTED", "stage2"),
                   ("gate1", "Mismatch", "resolve"),
                   ("gate1", "Request match gagal", "exit1"),
                   ("resolved", "MATCH_PENDING", "stage1"),
                   ("gate3", "PO CLOSED", "end"),
                   ("gate3", "PO RECEIVED", "stage2")],
        "l2-ar": [("gate3", "SO COMPLETED", "end"),
                   ("gate3", "SO DELIVERED", "stage2")],
        "l2-pr": [("gate2", "PR REJECTED", "rejected"),
                   ("gate2", "APPROVED", "end")],
        "l2-so": [("gate2", "SO REJECTED", "rejected"),
                   ("gate2", "APPROVED", "stage3")],
        "l2-adjustment": [("gate2", "Adjustment REJECTED", "rejected"),
                           ("gate2", "APPROVED", "stage3")],
        "l3-pr-review": [("valid", "Approve", "positive"),
                          ("valid", "Reject", "negative")],
        "l3-so-review": [("valid", "Approve", "positive"),
                          ("valid", "Reject", "negative")],
        "l3-adj-review": [("valid", "Approve", "positive"),
                           ("valid", "Reject", "negative")],
        "l3-ap-match": [("valid", "Cocok", "positive"),
                         ("valid", "Mismatch", "negative")],
    }
    for model_id, required in expected_branches.items():
        missing = set(required)-actual_branches[model_id]
        for source, label, target in sorted(missing):
            errors.append(f"{model_id}: missing business branch {source} --{label}--> {target}")
    expected_p2p_children = {
        "l1-p2p": {"l2-pr", "l2-po-gr", "l2-ap", "l2-p2p-correction"},
        "l2-p2p-correction": {"l2-purchase-correction", "l3-ap-resolve", "l3-ap-match",
                              "l3-ap-payment-void", "l3-ap-invoice-void"},
    }
    for model_id, children in expected_p2p_children.items():
        if set(all_models[model_id]["children"]) != children:
            errors.append(f"{model_id}: incorrect P2P decomposition children")
    expected = expected_routes()
    covered = {}
    for op in manifest["operations"]:
        source_text = (ROOT/op["source"]).read_text(encoding="utf-8")
        for function in op["function"].split(" / "):
            if not re.search(r"\bfunc\s+(?:\([^)]*\)\s*)?"+re.escape(function)+r"\s*\(", source_text):
                errors.append(f"Missing source function: {op['source']} :: {function}")
        for route in op["routes"]:
            if not route.startswith("CLI"):
                covered[route] = op["model"]
    for route in sorted(set(expected)-set(covered)):
        errors.append(f"Uncovered API: {route}")
    for route in sorted(set(covered)-set(expected)):
        errors.append(f"Undeclared API in model: {route}")
    for source, digest in manifest["source_sha256"].items():
        if hashlib.sha256((ROOT/source).read_bytes()).hexdigest() != digest:
            errors.append(f"Source changed after snapshot: {source}")
    # Every non-role model must be reachable through L0 decomposition.
    seen, stack = set(), ["l0-simple-erp"]
    while stack:
        key = stack.pop()
        if key not in seen:
            seen.add(key); stack.extend(all_models[key]["children"])
    for key in all_models:
        if not key.startswith("role-") and key not in seen:
            errors.append(f"Orphan decomposition model: {key}")
    link_count = 0
    for doc in OUT.rglob("*.md"):
        content = doc.read_text(encoding="utf-8")
        for target in re.findall(r"\]\(([^)]+)\)", content):
            if target.startswith(("http:", "https:", "#")):
                continue
            linked = (doc.parent/unquote(target.split("#")[0])).resolve()
            if not linked.exists():
                errors.append(f"Broken local link: {doc.relative_to(OUT)} -> {target}")
            link_count += 1
    report = {"passed": not errors, "bpmn_count": sum(count.values()), "by_level": dict(sorted(count.items())),
              "endpoint_count": len(expected), "covered_endpoint_count": len(set(expected)&set(covered)),
              "local_links_checked": link_count,
              "official_omg_xsd": schema is not None,
              "checks": ["XML/XSD", "IDs", "flow references", "lane ownership", "start/end reachability",
                         "DI completeness", "node overlaps", "edge crossing unrelated nodes", "cross-file calledElement", "hierarchy",
                         "API coverage", "source SHA256", "source function names",
                         "business state branches", "local documentation links"],
              "errors": errors,
              "limits": "Static documentation validation only; no live database or browser business workflow test."}
    (OUT/"validation.json").write_text(json.dumps(report, indent=2)+"\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    raise SystemExit(bool(errors))


if __name__ == "__main__":
    main()
