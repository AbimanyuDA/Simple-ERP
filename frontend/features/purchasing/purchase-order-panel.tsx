"use client";

import { FormEvent, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { CommandResult, WorkflowDocument } from "@/features/operations/types";
import { MasterDataSelect } from "@/features/master-data/master-data-select";
import { canApprove, canCreate, useSession } from "@/features/auth/session-context";

export function PurchaseOrderPanel({ onNotice }: { onNotice: (message: string) => void }) {
  const { session } = useSession();
  const [requestId, setRequestId] = useState(""); const [supplierId, setSupplierId] = useState(""); const [warehouseId, setWarehouseId] = useState("");
  const [orderId, setOrderId] = useState(""); const [version, setVersion] = useState("1"); const [reason, setReason] = useState(""); const [result, setResult] = useState<CommandResult>(null);
  const [items, setItems] = useState<WorkflowDocument[]>([]); const [revision, setRevision] = useState(0);
  useEffect(() => { api<{ data: WorkflowDocument[] }>("/purchase-orders").then((value) => setItems(value.data)).catch((error: Error) => onNotice(error.message)); }, [revision, onNotice]);
  async function run(path: string, body: object, success: string) {
    try { const document = await api<WorkflowDocument>(path, { method: "POST", body: JSON.stringify(body) }); setResult(document); if (document.id) setOrderId(document.id); if (document.version) setVersion(String(document.version)); setRevision((value) => value + 1); onNotice(success); }
    catch (error) { onNotice(error instanceof Error ? error.message : "Aksi purchase order gagal."); }
  }
  function convert(event: FormEvent) { event.preventDefault(); return run(`/purchase-requests/${requestId}/purchase-order`, { supplierId, warehouseId }, "Purchase order berhasil dibuat."); }
  return <section className="operation-grid">
    {canCreate(session?.role) && <form className="operation-card" onSubmit={convert}><strong>Konversi PR menjadi PO</strong><label>Approved PR ID<input value={requestId} onChange={(event) => setRequestId(event.target.value)} required /></label><MasterDataSelect resource="suppliers" label="Supplier" value={supplierId} onChange={setSupplierId} /><MasterDataSelect resource="warehouses" label="Warehouse" value={warehouseId} onChange={setWarehouseId} /><button className="button">Buat PO</button></form>}
    {canApprove(session?.role) && <form className="operation-card" onSubmit={(event) => { event.preventDefault(); void run(`/purchase-orders/${orderId}/issue`, { version: Number(version) }, "Purchase order berhasil diterbitkan."); }}><strong>Review Purchase Order</strong><label>Purchase Order ID<input value={orderId} onChange={(event) => setOrderId(event.target.value)} required /></label><label>Version<input type="number" min="1" value={version} onChange={(event) => setVersion(event.target.value)} required /></label><label>Alasan pembatalan<input value={reason} onChange={(event) => setReason(event.target.value)} /></label><span className="row-actions"><button className="button">Issue</button><button type="button" disabled={!reason} onClick={() => run(`/purchase-orders/${orderId}/cancel`, { reason }, "Purchase order dibatalkan.")}>Cancel</button></span></form>}
    <aside className="operation-result"><strong>Hasil terakhir</strong>{result ? <><b>{result.documentNo}</b><small>{result.id}</small><span className="status-chip">{result.status}</span><small>Version {result.version}</small></> : <p>Pilih PO dari daftar untuk memprosesnya.</p>}</aside>
    <div className="data-table operation-list"><div className="data-head operation"><span>Dokumen / ID</span><span>Status</span><span>Aksi</span></div>{items.length === 0 ? <p>Belum ada purchase order.</p> : items.map((item) => <div className="data-row operation" key={item.id}><span><b>{item.documentNo}</b><small className="record-id">{item.id}</small></span><span className="status-chip">{item.status}</span><span className="row-actions"><button onClick={() => { setOrderId(item.id); setVersion(String(item.version)); }}>Pilih</button></span></div>)}</div>
  </section>;
}
