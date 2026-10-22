"""Build the source-backed BPMN documentation package; never modifies application code."""
from __future__ import annotations

import hashlib
import json
import posixpath
import textwrap
from pathlib import Path
from xml.sax.saxutils import escape
from lxml import etree as E

ROOT = Path(__file__).resolve().parents[3]
OUT = Path(__file__).resolve().parents[1]
NS = {
    "bpmn": "http://www.omg.org/spec/BPMN/20100524/MODEL",
    "bpmndi": "http://www.omg.org/spec/BPMN/20100524/DI",
    "dc": "http://www.omg.org/spec/DD/20100524/DC",
    "di": "http://www.omg.org/spec/DD/20100524/DI",
    "xsi": "http://www.w3.org/2001/XMLSchema-instance",
}
MAKER = "Staff atau Super Admin"
REVIEWER = "Supervisor atau Super Admin"
OWNER = "Super Admin"
USER = "Pengguna (semua role)"
SYSTEM = "Sistem Simple ERP"
DEPLOY = "Pengelola instalasi (CLI)"
PUBLIC = "Penerima token (belum login)"
MODELS = {}
OPERATIONS = []
P2P_L2 = {"l2-pr", "l2-po-gr", "l2-ap", "l2-p2p-correction", "l2-purchase-correction"}
P2P_L3 = {
    "l3-pr-create", "l3-pr-submit", "l3-pr-review", "l3-pr-cancel", "l3-pr-copy",
    "l3-po-convert", "l3-po-issue", "l3-po-cancel",
    "l3-gr-create", "l3-gr-post", "l3-gr-cancel", "l3-gr-reverse",
    "l3-ap-invoice", "l3-ap-match", "l3-ap-resolve", "l3-ap-invoice-void",
    "l3-ap-payment", "l3-ap-payment-post", "l3-ap-payment-void",
}


def elem(parent, tag, text=None, **attrs):
    prefix, name = tag.split(":")
    node = E.SubElement(parent, f"{{{NS[prefix]}}}{name}", **{k: str(v) for k, v in attrs.items()})
    if text is not None:
        node.text = text
    return node


class Model:
    def __init__(self, slug, level, title, sources=(), note=""):
        self.slug, self.level, self.title = slug, level, title
        self.sources, self.note = list(sources), note
        self.nodes, self.edges = [], []
        MODELS[slug] = self

    @property
    def path(self):
        folder = f"level-{self.level}"
        if self.slug in P2P_L2 or self.slug in P2P_L3:
            folder += "-p2p"
        return f"{folder}/{self.slug}.bpmn"

    def n(self, key, label, lane=SYSTEM, kind="serviceTask", child=None, doc=""):
        self.nodes.append(dict(key=key, label=label, lane=lane, kind=kind, child=child, doc=doc))
        return key

    def f(self, source, target, label="", default=False):
        self.edges.append(dict(source=source, target=target, label=label, default=default))

    def chain(self, *keys):
        for a, b in zip(keys, keys[1:]):
            self.f(a, b)


def operation(slug, title, actor, request, guard, effects, source, function,
              routes=(), trace="", note="", failure="Ditolak / gagal; tidak commit"):
    """One atomic request, not a fictitious long-running workflow transaction."""
    m = Model(slug, 3, title, [(source, function)], note)
    m.n("start", "Kebutuhan aksi", actor, "startEvent")
    m.n("input", request, actor, "userTask", doc="Aktor alternatif pada lane, bukan dua persetujuan.")
    m.n("check", guard, doc="Setelah kontrol akses HTTP; lihat l3-kontrol-request. " + note)
    m.n("valid", "Syarat terpenuhi?", kind="exclusiveGateway")
    m.chain("start", "input", "check", "valid")
    previous = "valid"
    for i, effect in enumerate(effects):
        key = f"effect{i}"
        m.n(key, effect)
        m.f(previous, key, "Ya" if i == 0 else "")
        previous = key
    m.n("done", "Aksi selesai; hasil diterima", kind="endEvent")
    m.f(previous, "done")
    m.n("failed", failure, kind="endEvent")
    m.f("valid", "failed", "Tidak", default=True)
    OPERATIONS.append(dict(model=slug, actor=actor, routes=list(routes), trace=trace,
                           source=source, function=function, guard=guard, effects=effects, note=note))
    return m


def menu(slug, level, title, choices, actor="Pemilik kebutuhan (role pada detail)", note=""):
    """An XOR catalog: one invocation handles one selected need, not a parallel launch."""
    m = Model(slug, level, title, note=note)
    m.n("start", "Kebutuhan muncul", actor, "startEvent")
    m.n("choose", "Pilih kebutuhan", actor, "userTask")
    m.chain("start", "choose")
    if len(choices) > 1:
        m.n("branch", "Kebutuhan apa?", actor, "exclusiveGateway")
        m.f("choose", "branch")
    shared_end = slug in {"l2-p2p-correction", "l2-purchase-correction"}
    if shared_end:
        m.n("merge", "Gabung pilihan", actor, "exclusiveGateway")
        m.n("end", "Satu koreksi diproses", actor, "endEvent")
    for i, (label, child) in enumerate(choices):
        key = f"call{i}"
        m.n(key, label, actor, "callActivity", child=child)
        if not shared_end:
            m.n(f"end{i}", "Aksi selesai / hasil diterima", actor, "endEvent")
        m.f("branch" if len(choices) > 1 else "choose", key,
            label if len(choices) > 1 else "",
            default=len(choices) > 1 and not shared_end and i == len(choices) - 1)
        m.f(key, "merge" if shared_end else f"end{i}")
    if shared_end:
        m.f("merge", "end")
    return m


def lifecycle(slug, title, stages, sources, note=""):
    """State-dependent successful lifecycle; failure/rejection explicitly exits the instance."""
    m = Model(slug, 2, title, sources, note)
    m.n("start", "Kebutuhan dokumen", stages[0][1], "startEvent")
    p2p_stops = {
        "l1-p2p": ["PR belum approved", "PO belum diterima penuh", "Pelunasan belum selesai"],
        "l2-pr": ["PR belum dibuat", "PR belum submitted", "PR belum approved"],
        "l2-po-gr": ["PO belum dibuat", "PO belum issued", "GR belum dicatat", "GR belum posted"],
        "l2-ap": ["Invoice belum dicatat", "Match belum selesai", "Payment belum dicatat", "Payment belum posted"],
    }
    previous = "start"
    for i, (label, actor, child, result) in enumerate(stages):
        key = f"stage{i}"
        m.n(key, label, actor, "callActivity", child=child)
        m.f(previous, key)
        if result:
            gate = f"gate{i}"
            m.n(gate, result, kind="exclusiveGateway")
            m.f(key, gate)
            stop = p2p_stops[slug][i] if slug in p2p_stops else "Belum lanjut; periksa hasil aksi"
            m.n(f"exit{i}", stop, kind="endEvent")
            m.f(gate, f"exit{i}", "Tidak", default=True)
            previous = gate
        else:
            previous = key
    m.n("end", "Tahap selesai", kind="endEvent")
    m.f(previous, "end", "Ya" if previous.startswith("gate") else "")
    # Label successful continuation after every state gate.
    for edge in m.edges:
        if edge["source"].startswith("gate") and not edge["default"] and not edge["label"]:
            edge["label"] = "Ya"
    return m


def ident(slug):
    return slug.replace("-", "_")


def namespace(slug):
    return f"https://simple-erp.local/bpm-haqi/{slug}"


def layout(m):
    lanes = list(dict.fromkeys(n["lane"] for n in m.nodes))
    boxes = {}
    is_menu = any(n["key"] == "choose" for n in m.nodes)
    if is_menu:
        calls = [n for n in m.nodes if n["key"].startswith("call")]
        lane_height = max(260, len(calls)*155+80)
        center = 100+lane_height/2
        boxes["start"] = (200, center-19, 38, 38)
        boxes["choose"] = (310, center-45, 178, 90)
        if any(n["key"] == "branch" for n in m.nodes):
            boxes["branch"] = (550, center-27, 54, 54)
        shared_end = any(n["key"] == "merge" for n in m.nodes)
        for i, _ in enumerate(calls):
            boxes[f"call{i}"] = (850, 145+i*155, 178, 90)
            if not shared_end:
                boxes[f"end{i}"] = (1120, 171+i*155, 38, 38)
        if shared_end:
            boxes["merge"] = (1160, center-27, 54, 54)
            boxes["end"] = (1320, center-19, 38, 38)
        paths = []
        for e in m.edges:
            a, b = boxes[e["source"]], boxes[e["target"]]
            x1, y1, x2, y2 = a[0]+a[2], a[1]+a[3]/2, b[0], b[1]+b[3]/2
            if e["target"] == "merge":
                branch = int(e["source"][4:])
                entry = [(1187, center-27), (1172, center-15), (1160, center),
                         (1172, center+15), (1187, center+27)][branch]
                paths.append([(x1,y1),(x1+55,y1),entry])
            else:
                mid = 645 if e["source"] == "branch" else (x1+x2)/2
                paths.append([(x1,y1),(mid,y1),(mid,y2),(x2,y2)])
        return lanes, boxes, paths, 1480 if shared_end else 1320, 100+lane_height
    if m.level == 2 and any(n["key"] == "stage0" for n in m.nodes):
        stages = [n for n in m.nodes if n["key"].startswith("stage")]
        last = len(stages)-1
        system_y = 100+lanes.index(SYSTEM)*300
        width, height = 450+len(stages)*550, 190+len(lanes)*300
        for node in m.nodes:
            key, kind = node["key"], node["kind"]
            w,h = (38,38) if kind.endswith("Event") else ((54,54) if kind.endswith("Gateway") else
                (178,max(90,len(textwrap.wrap(node["label"],24))*16+32)))
            lane_y = 100+lanes.index(node["lane"])*300
            if key == "start":
                x,cy = 85,lane_y+100
            elif key.startswith("stage"):
                x,cy = 200+int(key[5:])*550,lane_y+100
            elif key.startswith("gate"):
                x,cy = 445+int(key[4:])*550,system_y+100
            elif key.startswith("exit"):
                x,cy = 595+int(key[4:])*550,system_y+215
            elif key == "resolve":
                x,cy = 1075,lane_y+100
            elif key == "resolved":
                x,cy = 1280,system_y+100
            elif key == "rejected":
                x,cy = 745+last*550,system_y+215
            elif key == "end":
                x,cy = 690+last*550,system_y+100
            else:
                raise ValueError(f"Unknown lifecycle node {m.slug}:{key}")
            boxes[key] = (x,cy-h/2,w,h)
        paths = []
        for i, edge in enumerate(m.edges):
            a,b = boxes[edge["source"]],boxes[edge["target"]]
            x1,y1,x2,y2 = a[0]+a[2],a[1]+a[3]/2,b[0],b[1]+b[3]/2
            if m.slug in P2P_L2 and edge["default"] and edge["source"].startswith(("gate", "resolved")):
                # Give the fallback its own gateway exit; do not overlap the successful branch.
                x1, y1 = a[0]+a[2]*0.75, a[1]+a[3]*0.75
            if x2 > x1:
                if m.slug == "l2-ap" and edge["source"] == "gate1" and edge["target"] == "stage2":
                    # The POSTED branch must bypass the resolve task between gate and payment.
                    pts=[(x1,y1),(x1+9,y1),(x1+9,300),(x2-20,300),(x2-20,y2),(x2,y2)]
                elif m.slug in {"l2-so", "l2-adjustment"} and edge["target"] == "rejected":
                    # A rejected review exits below the following successful-stage gateway.
                    pts=[(x1,y1),(x1+35,y1),(x1+35,700),(x2-60,700),(x2-60,y2),(x2,y2)]
                else:
                    mid = (x2-60 if edge["target"] == "rejected" else
                           x1+35 if edge["source"].startswith("gate") and edge["target"].startswith("stage") and y2>y1 else
                           (x1+x2)/2)
                    pts=[(x1,y1),(mid,y1),(mid,y2),(x2,y2)]
            else:
                track=height-65+(i%3)*12
                pts=[(x1,y1),(x1+20,y1),(x1+20,track),
                     (x2-20,track),(x2-20,y2),(x2,y2)]
            paths.append(pts)
        return lanes,boxes,paths,width,height
    if any(n["key"] == "positive" for n in m.nodes):
        centers = {
            "start": (200,190), "input": (425,190), "check": (650,600),
            "valid": (875,600), "positive": (1125,510), "negative": (1125,690),
            "effect1": (1375,600), "done": (1625,600), "failed": (1875,600),
        }
        for node in m.nodes:
            kind = node["kind"]
            w,h = (38,38) if kind.endswith("Event") else ((54,54) if kind.endswith("Gateway") else
                (178,max(90,len(textwrap.wrap(node["label"],24))*16+32)))
            x,cy = centers[node["key"]]
            boxes[node["key"]] = (x,cy-h/2,w,h)
        paths = []
        for e in m.edges:
            a,b = boxes[e["source"]],boxes[e["target"]]
            x1,y1,x2,y2 = a[0]+a[2],a[1]+a[3]/2,b[0],b[1]+b[3]/2
            if e["target"] == "failed":
                pts = [(x1,y1),(x1+18,y1),(x1+18,775),(x2-18,775),(x2-18,y2),(x2,y2)]
            else:
                mid = (x1+x2)/2
                pts = [(x1,y1),(mid,y1),(mid,y2),(x2,y2)]
            paths.append(pts)
        return lanes,boxes,paths,2125,800
    # Unique columns avoid vertical connectors passing through unrelated nodes.
    for i, node in enumerate(m.nodes):
        kind = node["kind"]
        w, h = (38, 38) if kind.endswith("Event") else ((54, 54) if kind.endswith("Gateway") else
            (178, max(90, len(textwrap.wrap(node["label"], 24))*16+32)))
        x = 200 + i * 225
        y = 145 + lanes.index(node["lane"]) * 220 + (90 - h) / 2
        boxes[node["key"]] = (x, y, w, h)
    width, height = 260 + len(m.nodes) * 225, 100 + len(lanes) * 220
    paths = []
    for i, edge in enumerate(m.edges):
        a, b = boxes[edge["source"]], boxes[edge["target"]]
        x1, y1 = a[0] + a[2], a[1] + a[3] / 2
        if m.slug == "l1-p2p" and edge["default"] and edge["source"].startswith("gate"):
            x1, y1 = a[0]+a[2]*0.75, a[1]+a[3]*0.75
        x2, y2 = b[0], b[1] + b[3] / 2
        if x2 > x1 and x2 - x1 < 260:
            mid = (x1 + x2) / 2
            pts = [(x1, y1), (mid, y1), (mid, y2), (x2, y2)]
        else:
            lane = next(n["lane"] for n in m.nodes if n["key"] == edge["source"])
            track = 270 + lanes.index(lane) * 220 + (i % 4) * 9
            pts = [(x1, y1), (x1 + 18, y1), (x1 + 18, track),
                   (x2 - 18, track), (x2 - 18, y2), (x2, y2)]
        paths.append(pts)
    return lanes, boxes, paths, width, height


def flow_label_position(m, edge, points, boxes, height):
    """Place decision labels near their own branch, away from task captions."""
    source = boxes[edge["source"]]
    target = boxes[edge["target"]]
    sx, sy = source[0]+source[2], source[1]+source[3]/2
    tx, ty = target[0], target[1]+target[3]/2
    if m.slug == "l2-ap" and edge["source"] == "gate1":
        if edge["target"] == "stage2":
            return 1090, 265
        if edge["target"] == "resolve":
            return 1070, 365
    if m.level == 2 and edge["source"].startswith(("gate", "resolved")):
        if tx <= sx:
            return sx-175, height-(80 if edge["target"].startswith("exit") else 105)
        if edge["target"] == "rejected":
            return sx+30, ty-65
        if edge["target"].startswith(("exit", "rejected")):
            return sx+30, ty-35
        if edge["target"] == "end":
            return sx+30, sy-37
        if abs(ty-sy) > 100:
            return (sx+tx)/2-75, target[1]-38
        return sx+20, sy-43
    if m.slug == "l3-kontrol-request" and edge["target"] == "handler" and len(points) > 4:
        return points[3][0]-170, points[3][1]+10
    branch = edge["source"] == "decision"
    return points[-1][0]-(85 if branch else 170), points[-1][1]-(28 if branch else 55)


def write_model(m):
    children = list(dict.fromkeys(n["child"] for n in m.nodes if n["child"]))
    nsmap = dict(NS, tns=namespace(m.slug))
    nsmap.update({f"c{i}": namespace(child) for i, child in enumerate(children)})
    root = E.Element(f"{{{NS['bpmn']}}}definitions", nsmap=nsmap, id=f"Definitions_{ident(m.slug)}",
                     targetNamespace=namespace(m.slug), exporter="Simple ERP source-backed documentation",
                     exporterVersion="1")
    for child in children:
        cm = MODELS[child]
        elem(root, "bpmn:import", namespace=namespace(child),
             location=posixpath.relpath(cm.path, posixpath.dirname(m.path)), importType=NS["bpmn"])
    process_id = f"Process_{ident(m.slug)}"
    p = elem(root, "bpmn:process", id=process_id, name=m.title, isExecutable="false")
    doc = f"Level {m.level}; source snapshot 2026-10-21; descriptive as-is working tree, not deployed-runtime proof.\n{m.note}"
    doc += "\n" + "\n".join(f"{path} :: {fn}" for path, fn in m.sources)
    elem(p, "bpmn:documentation", doc.strip())
    lanes, boxes, paths, width, height = layout(m)
    ls = elem(p, "bpmn:laneSet", id=f"Lanes_{ident(m.slug)}")
    for i, lane in enumerate(lanes):
        ln = elem(ls, "bpmn:lane", id=f"Lane_{ident(m.slug)}_{i}", name=lane)
        for n in m.nodes:
            if n["lane"] == lane:
                elem(ln, "bpmn:flowNodeRef", f"{ident(m.slug)}_{n['key']}")
    for n in m.nodes:
        attrs = dict(id=f"{ident(m.slug)}_{n['key']}", name=n["label"])
        if n["child"]:
            attrs["calledElement"] = f"c{children.index(n['child'])}:Process_{ident(n['child'])}"
        default = next((i for i, e in enumerate(m.edges) if e["source"] == n["key"] and e["default"]), None)
        if default is not None:
            attrs["default"] = f"{ident(m.slug)}_Flow_{default}"
        if n["kind"].endswith("Gateway"):
            outgoing = sum(e["source"] == n["key"] for e in m.edges)
            incoming = sum(e["target"] == n["key"] for e in m.edges)
            attrs["gatewayDirection"] = "Mixed" if incoming > 1 and outgoing > 1 else ("Diverging" if outgoing > 1 else "Converging")
        nd = elem(p, f"bpmn:{n['kind']}", **attrs)
        if n["doc"]:
            elem(nd, "bpmn:documentation", n["doc"])
        for i, e in enumerate(m.edges):
            if e["target"] == n["key"]:
                elem(nd, "bpmn:incoming", f"{ident(m.slug)}_Flow_{i}")
        for i, e in enumerate(m.edges):
            if e["source"] == n["key"]:
                elem(nd, "bpmn:outgoing", f"{ident(m.slug)}_Flow_{i}")
    for i, e in enumerate(m.edges):
        attrs = dict(id=f"{ident(m.slug)}_Flow_{i}", sourceRef=f"{ident(m.slug)}_{e['source']}",
                     targetRef=f"{ident(m.slug)}_{e['target']}")
        if e["label"]:
            attrs["name"] = e["label"]
        flow = elem(p, "bpmn:sequenceFlow", **attrs)
        source = next(n for n in m.nodes if n["key"] == e["source"])
        if source["kind"].endswith("Gateway") and sum(x["source"] == e["source"] for x in m.edges) > 1 and not e["default"]:
            cond = elem(flow, "bpmn:conditionExpression", e["label"], language="text/plain")
            cond.set(f"{{{NS['xsi']}}}type", "bpmn:tFormalExpression")
    collab_id, pool_id = f"Collab_{ident(m.slug)}", f"Pool_{ident(m.slug)}"
    collab = elem(root, "bpmn:collaboration", id=collab_id)
    elem(collab, "bpmn:participant", id=pool_id, name="Simple ERP", processRef=process_id)
    diagram = elem(root, "bpmndi:BPMNDiagram", id=f"Diagram_{ident(m.slug)}")
    plane = elem(diagram, "bpmndi:BPMNPlane", id=f"Plane_{ident(m.slug)}", bpmnElement=collab_id)
    def shape(ref, x, y, w, h, **attrs):
        s = elem(plane, "bpmndi:BPMNShape", id=ref+"_di", bpmnElement=ref, **attrs)
        elem(s, "dc:Bounds", x=x, y=y, width=w, height=h)
        return s
    shape(pool_id, 20, 100, width-40, height-100, isHorizontal="true")
    for i, _ in enumerate(lanes):
        lane_height = (height-100)/len(lanes)
        shape(f"Lane_{ident(m.slug)}_{i}", 50, 100+i*lane_height, width-70, lane_height, isHorizontal="true")
    for n in m.nodes:
        x, y, w, h = boxes[n["key"]]
        s = shape(f"{ident(m.slug)}_{n['key']}", x, y, w, h)
        if n["kind"].endswith(("Event", "Gateway")):
            label = elem(s, "bpmndi:BPMNLabel")
            elem(label, "dc:Bounds", x=x-55, y=y+h+5, width=w+110, height=45)
    for i, points in enumerate(paths):
        edge = elem(plane, "bpmndi:BPMNEdge", id=f"{ident(m.slug)}_Flow_{i}_di",
                    bpmnElement=f"{ident(m.slug)}_Flow_{i}")
        for x, y in points:
            elem(edge, "di:waypoint", x=x, y=y)
        if m.edges[i]["label"]:
            label = elem(edge, "bpmndi:BPMNLabel")
            lx,ly = flow_label_position(m,m.edges[i],points,boxes,height)
            elem(label, "dc:Bounds", x=lx, y=ly, width=150, height=45)
    file = OUT / m.path
    file.parent.mkdir(parents=True, exist_ok=True)
    xml = E.tostring(root, xml_declaration=True, encoding="UTF-8", pretty_print=True)
    file.write_bytes(xml)
    if m.slug in P2P_L2 or m.slug in P2P_L3:
        # Keep the original general-level copy for existing links and users.
        # Both folders have the same depth, so relative imports remain valid.
        (OUT / f"level-{m.level}" / f"{m.slug}.bpmn").write_bytes(xml)
    write_svg(m, lanes, boxes, paths, width, height)


def write_svg(m, lanes, boxes, paths, width, height):
    def text(x, y, value, size=13, anchor="middle", wrap=25):
        lines = textwrap.wrap(value, width=wrap) or [""]
        return f'<text x="{x}" y="{y}" font-size="{size}" text-anchor="{anchor}">' + "".join(
            f'<tspan x="{x}" dy="{0 if i == 0 else 16}">{escape(line)}</tspan>'
            for i, line in enumerate(lines)) + "</text>"
    svg = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height+35}" viewBox="0 0 {width} {height+35}">',
           '<defs><marker id="arrow" markerWidth="10" markerHeight="10" refX="9" refY="5" orient="auto"><path d="M0,0 L10,5 L0,10 Z" fill="#475569"/></marker></defs>',
           '<rect width="100%" height="100%" fill="white"/>',
           '<g font-family="Arial, sans-serif" fill="#0f172a">',
           text(25, 30, f"L{m.level} | {m.title}", 20, "start", 180),
           text(25, 56, "AS-IS KODE 2026-10-21 | Lane dengan 'atau' berarti pilihan aktor | Detail kegagalan: dokumentasi model", 12, "start", 180)]
    for i, lane in enumerate(lanes):
        lane_height = (height-100)/len(lanes)
        y = 100+i*lane_height
        svg.append(f'<rect x="20" y="{y}" width="{width-40}" height="{lane_height}" fill="{["#f1f5f9","#eff6ff","#f0fdf4","#fff7ed"][i%4]}" stroke="#cbd5e1"/>')
        lane_label_y = y+35 if m.slug in {"l2-pr", "l2-po-gr", "l2-ap"} else y+80
        svg.append(text(100, lane_label_y, lane, 14, wrap=16))
    for i, pts in enumerate(paths):
        svg.append('<polyline points="'+" ".join(f"{x},{y}" for x,y in pts)+'" fill="none" stroke="#475569" stroke-width="1.5" marker-end="url(#arrow)"/>')
        if m.edges[i]["default"]:
            (x1, y1), (x2, y2) = pts[:2]
            if x1 == x2:
                cx, cy = x1, y1 + (18 if y2 > y1 else -18)
                ends = (cx-7, cy-6, cx+7, cy+6)
            else:
                cx, cy = x1 + (18 if x2 > x1 else -18), y1
                ends = (cx-6, cy+7, cx+6, cy-7)
            svg.append('<line class="default-flow-marker" x1="{}" y1="{}" x2="{}" y2="{}" stroke="#475569" stroke-width="1.5"/>'.format(*ends))
        if m.edges[i]["label"]:
            lx,ly = flow_label_position(m,m.edges[i],pts,boxes,height)
            svg.append(text(lx+8,ly+12,m.edges[i]["label"],11,"start",22))
    for n in m.nodes:
        x, y, w, h = boxes[n["key"]]
        kind = n["kind"]
        if kind.endswith("Event"):
            svg.append(f'<circle cx="{x+w/2}" cy="{y+h/2}" r="{w/2}" fill="white" stroke="#334155" stroke-width="{3 if kind=="endEvent" else 1.5}"/>')
            svg.append(text(x+w/2, y+h+20, n["label"], 12, wrap=22))
        elif kind.endswith("Gateway"):
            svg.append(f'<path d="M{x+w/2},{y} L{x+w},{y+h/2} L{x+w/2},{y+h} L{x},{y+h/2} Z" fill="#fef3c7" stroke="#92400e"/>')
            svg.append(text(x+w/2, y+h/2+5, "X", 18))
            svg.append(text(x+w/2, y+h+20, n["label"], 12, wrap=23))
        else:
            svg.append(f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="9" fill="white" stroke="#334155" stroke-width="{3 if kind=="callActivity" else 1.5}"/>')
            marker = "U" if kind == "userTask" else ("CALL" if kind == "callActivity" else "S")
            svg.append(text(x+12, y+15, marker, 9, "start"))
            lines = textwrap.wrap(n["label"], 24)
            svg.append(text(x+w/2, y+32, n["label"], 12, wrap=24))
            if kind == "callActivity":
                mx, my = x+w/2-6, y+h-14
                svg.append(f'<g class="collapsed-call-marker" fill="white" stroke="#334155" stroke-width="1"><rect x="{mx}" y="{my}" width="12" height="12"/><path d="M{mx+3},{my+6} H{mx+9} M{mx+6},{my+3} V{my+9}"/></g>')
    svg.append("</g></svg>")
    preview = OUT / "preview" / f"{m.slug}.svg"
    preview.parent.mkdir(exist_ok=True)
    preview.write_text("\n".join(svg), encoding="utf-8")


def finish():
    for m in MODELS.values():
        write_model(m)
    manifest = {"snapshot": "2026-10-21 working tree", "models": [
        {"id": m.slug, "level": m.level, "title": m.title, "file": m.path,
         "children": list(dict.fromkeys(n["child"] for n in m.nodes if n["child"])),
         "sources": m.sources, "nodes": len(m.nodes), "flows": len(m.edges)}
        for m in MODELS.values()], "operations": OPERATIONS}
    sources = {p.relative_to(ROOT).as_posix() for p in (ROOT/"backend/internal").rglob("*.go") if not p.name.endswith("_test.go")}
    sources.update(p.relative_to(ROOT).as_posix() for p in (ROOT/"backend/migrations").glob("*.up.sql"))
    sources.update(p.relative_to(ROOT).as_posix() for p in (ROOT/"frontend/features").rglob("*.tsx"))
    sources.update(p.relative_to(ROOT).as_posix() for p in (ROOT/"frontend/app").rglob("page.tsx"))
    sources.update(["frontend/components/app-shell.tsx", "frontend/lib/api.ts"])
    sources.update(["backend/cmd/api/main.go", "backend/internal/auth/domain/role.go",
                    "backend/migrations/000010_three_roles.up.sql"])
    manifest["source_sha256"] = {p: hashlib.sha256((ROOT/p).read_bytes()).hexdigest() for p in sorted(sources)}
    (OUT/"manifest.json").write_text(json.dumps(manifest, indent=2, ensure_ascii=False)+"\n", encoding="utf-8")
    index = ["# Katalog BPMN Simple ERP", "", "Buka [README](README.md) dahulu. Struktur khusus P2P ada di [P2P-REVIEW](P2P-REVIEW.md). SVG adalah preview; `.bpmn` adalah model BPMN 2.0 dengan DI.", ""]
    for level in range(4):
        index += [f"## Level {level}", "", "| Diagram | BPMN | Preview |", "| --- | --- | --- |"]
        for m in MODELS.values():
            if m.level == level:
                index.append(f"| {m.title} | [{m.slug}]({m.path}) | [SVG](preview/{m.slug}.svg) |")
        index.append("")
    (OUT/"INDEX.md").write_text("\n".join(index), encoding="utf-8")
    print(f"Built {len(MODELS)} BPMN + SVG models; {len(OPERATIONS)} operation groups.")


if __name__ == "__main__":
    # Catalog is deliberately separate so behavior descriptions are reviewable.
    from catalog import populate
    populate(globals())
    finish()
