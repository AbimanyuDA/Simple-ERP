"use client";

import { FormEvent, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { WorkflowDocument } from "@/features/operations/types";
import { MasterDataSelect } from "@/features/master-data/master-data-select";

export function PurchaseRequestPanel({ onNotice }: { onNotice: (message: string) => void }) {
  const [items, setItems] = useState<WorkflowDocument[]>([]);
  const [productId, setProductId] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [price, setPrice] = useState("0");
  const [revision, setRevision] = useState(0);
  useEffect(() => { api<{ data: WorkflowDocument[] }>("/purchase-requests").then((value) => setItems(value.data)).catch((error: Error) => onNotice(error.message)); }, [revision, onNotice]);
  async function create(event: FormEvent) {
    event.preventDefault();
    try {
      await api("/purchase-requests", { method: "POST", body: JSON.stringify({ lines: [{ productId, quantity: Number(quantity), unitPrice: Number(price) }] }) });
      onNotice("Purchase request draft berhasil dibuat."); setRevision((value) => value + 1);
    } catch (error) { onNotice(error instanceof Error ? error.message : "Purchase request gagal dibuat."); }
  }
  async function action(document: WorkflowDocument, name: string) {
    try {
      const reason = name === "reject" || name === "cancel" ? "Diproses dari demo Mini ERP" : "";
      await api(`/purchase-requests/${document.id}/${name}`, { method: "POST", body: JSON.stringify({ version: document.version, reason }) });
      onNotice(`Aksi ${name} berhasil untuk ${document.documentNo}.`); setRevision((value) => value + 1);
    } catch (error) { onNotice(error instanceof Error ? error.message : "Aksi purchase request gagal."); }
  }
  return <section>
    <form className="operation-form" onSubmit={create}>
      <MasterDataSelect resource="products" label="Product" value={productId} onChange={setProductId} />
      <label>Jumlah<input type="number" min="0.0001" step="0.0001" value={quantity} onChange={(event) => setQuantity(event.target.value)} required /></label>
      <label>Estimasi harga<input type="number" min="0" value={price} onChange={(event) => setPrice(event.target.value)} required /></label>
      <button className="button">Buat PR</button>
    </form>
    <div className="data-table"><div className="data-head operation"><span>Dokumen / ID</span><span>Status</span><span>Aksi sesuai state</span></div>
      {items.length === 0 ? <p>Belum ada purchase request.</p> : items.map((document) => <div className="data-row operation" key={document.id}>
        <span><b>{document.documentNo}</b><small className="record-id">{document.id}</small></span><span className="status-chip">{document.status}</span>
        <span className="row-actions">
          {document.status === "DRAFT" && <><button onClick={() => action(document, "submit")}>Submit</button><button onClick={() => action(document, "copy")}>Copy</button><button onClick={() => action(document, "cancel")}>Cancel</button></>}
          {document.status === "SUBMITTED" && <><button onClick={() => action(document, "approve")}>Approve</button><button onClick={() => action(document, "reject")}>Reject</button></>}
          {document.status === "APPROVED" && <small>Konversi di tab Purchase Orders</small>}
        </span>
      </div>)}
    </div>
  </section>;
}
