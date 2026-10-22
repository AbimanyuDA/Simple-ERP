"use client";

import { FormEvent, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { canApprove, canCreate, useSession } from "@/features/auth/session-context";
import type { WorkflowDocument, WorkflowDocumentDetail } from "@/features/operations/types";

type Document = {
  id: string;
  documentNo: string;
  status: string;
  creatorId?: string;
  partyId: string;
  sourceId: string;
  currency: string;
  total: number;
  outstanding: number;
  version: number;
};

type InvoiceLineDraft = {
  referenceId: string;
  productId: string;
  quantity: string;
  unitPrice: string;
};

const today = new Date().toISOString().slice(0, 10);
const money = (value: number) => `Rp${Number(value).toLocaleString("id-ID")}`;

export function FinanceWorkspace() {
  const { session } = useSession();
  const role = session?.role;
  const [tab, setTab] = useState<"ap" | "ar">("ap");
  const [supplierInvoices, setSupplierInvoices] = useState<Document[]>([]);
  const [supplierPayments, setSupplierPayments] = useState<Document[]>([]);
  const [salesInvoices, setSalesInvoices] = useState<Document[]>([]);
  const [customerPayments, setCustomerPayments] = useState<Document[]>([]);
  const [purchaseOrders, setPurchaseOrders] = useState<WorkflowDocument[]>([]);
  const [salesOrders, setSalesOrders] = useState<WorkflowDocument[]>([]);
  const [notice, setNotice] = useState("");
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    if (!role) return;
    Promise.all([
      api<{ data: Document[] }>("/supplier-invoices"),
      api<{ data: Document[] }>("/supplier-payments"),
      api<{ data: Document[] }>("/sales-invoices"),
      api<{ data: Document[] }>("/customer-payments"),
      api<{ data: WorkflowDocument[] }>("/purchase-orders"),
      api<{ data: WorkflowDocument[] }>("/sales-orders"),
    ]).then(([si, sp, ci, cp, po, so]) => {
      setSupplierInvoices(si.data);
      setSupplierPayments(sp.data);
      setSalesInvoices(ci.data);
      setCustomerPayments(cp.data);
      setPurchaseOrders(po.data.filter((item) => item.status === "RECEIVED"));
      setSalesOrders(so.data.filter((item) => item.status === "DELIVERED"));
    }).catch((error: Error) => setNotice(error.message));
  }, [revision, role]);

  async function command(path: string, body: object = {}) {
    try {
      await api(path, { method: "POST", body: JSON.stringify(body) });
      setNotice("Transaksi finance berhasil diproses.");
      setRevision((value) => value + 1);
      return true;
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Transaksi gagal.");
      return false;
    }
  }

  return <div className="data-workspace finance-workspace">
    <div className="data-tabs">
      <button className={tab === "ap" ? "active" : ""} onClick={() => setTab("ap")}>Payables · Supplier</button>
      <button className={tab === "ar" ? "active" : ""} onClick={() => setTab("ar")}>Receivables · Customer</button>
    </div>
    <p className="finance-guidance">Posted records tidak diedit. Koreksi dilakukan dengan void beralasan agar jejak transaksi tetap utuh.</p>
    {notice && <p className="notice">{notice}</p>}
    {tab === "ap"
      ? <FinancePanel key="supplier" kind="supplier" invoices={supplierInvoices} payments={supplierPayments} sources={purchaseOrders} command={command} allowCreate={canCreate(role)} allowApprove={canApprove(role)} allowSelfApprove={role === "SUPER_ADMIN"} actorId={session?.userId} />
      : <FinancePanel key="customer" kind="customer" invoices={salesInvoices} payments={customerPayments} sources={salesOrders} command={command} allowCreate={canCreate(role)} allowApprove={canApprove(role)} allowSelfApprove={role === "SUPER_ADMIN"} actorId={session?.userId} />}
  </div>;
}

function FinancePanel({ kind, invoices, payments, sources, command, allowCreate, allowApprove, allowSelfApprove, actorId }: {
  kind: "supplier" | "customer";
  invoices: Document[];
  payments: Document[];
  sources: WorkflowDocument[];
  command: (path: string, body?: object) => Promise<boolean>;
  allowCreate: boolean;
  allowApprove: boolean;
  allowSelfApprove: boolean;
  actorId?: string;
}) {
  const [source, setSource] = useState("");
  const [party, setParty] = useState("");
  const [externalNumber, setExternalNumber] = useState("");
  const [invoiceLines, setInvoiceLines] = useState<InvoiceLineDraft[]>([]);
  const [invoice, setInvoice] = useState("");
  const [amount, setAmount] = useState("");
  const [voidReason, setVoidReason] = useState("");
  const [sourceError, setSourceError] = useState("");
  const [resolvingInvoice, setResolvingInvoice] = useState<Document | null>(null);

  async function selectSource(id: string) {
    setSource(id);
    setInvoiceLines([]);
    setSourceError("");
    if (!id || kind !== "supplier") return;
    try {
      const detail = await api<WorkflowDocumentDetail>(`/purchase-orders/${id}`);
      setInvoiceLines(detail.lines.map((item) => ({
        referenceId: item.id,
        productId: item.productId,
        quantity: String(item.processedQuantity || item.quantity),
        unitPrice: String(item.unitPrice),
      })));
    } catch (error) {
      setSourceError(error instanceof Error ? error.message : "Detail purchase order gagal dibaca.");
    }
  }

  function updateInvoiceLine(id: string, field: "quantity" | "unitPrice", value: string) {
    setInvoiceLines((items) => items.map((item) => item.referenceId === id ? { ...item, [field]: value } : item));
  }

  async function beginResolve(item: Document) {
    setResolvingInvoice(item);
    await selectSource(item.sourceId);
  }

  async function createInvoice(event: FormEvent) {
    event.preventDefault();
    if (kind === "supplier") {
      const lines = invoiceLines.map((item) => ({ referenceId: item.referenceId, quantity: Number(item.quantity), unitPrice: Number(item.unitPrice) }));
      if (resolvingInvoice) {
        const resolved = await command(`/supplier-invoices/${resolvingInvoice.id}/resolve-match`, { lines });
        if (resolved) {
          setResolvingInvoice(null);
          setSource("");
          setInvoiceLines([]);
        }
        return;
      }
      const replacement = invoices.find((item) => item.sourceId === source && item.status === "VOID");
      await command("/supplier-invoices", { purchaseOrderId: source, supplierInvoiceNumber: externalNumber, invoiceDate: today, dueDate: today, currency: "IDR", replacesInvoiceId: replacement?.id || "", lines });
    } else {
      const replacement = invoices.find((item) => item.sourceId === source && item.status === "VOID");
      await command(`/sales-orders/${source}/sales-invoice`, { invoiceDate: today, dueDate: today, currency: "IDR", replacesInvoiceId: replacement?.id || "" });
    }
  }

  async function createPayment(event: FormEvent) {
    event.preventDefault();
    const prefix = kind === "supplier" ? "/supplier-payments" : "/customer-payments";
    const created = await command(prefix, { partyId: party, paymentDate: today, method: "BANK_TRANSFER", currency: "IDR", amount: Number(amount), allocations: [{ invoiceId: invoice, amount: Number(amount) }] });
    if (created) {
      setParty("");
      setInvoice("");
      setAmount("");
    }
  }

  const invoicePrefix = kind === "supplier" ? "/supplier-invoices" : "/sales-invoices";
  const paymentPrefix = kind === "supplier" ? "/supplier-payments" : "/customer-payments";
  const availableSources = resolvingInvoice
    ? sources.filter((item) => item.id === resolvingInvoice.sourceId)
    : sources.filter((item) => !invoices.some((candidate) => candidate.sourceId === item.id && candidate.status !== "VOID"));
  const replacementInvoice = resolvingInvoice ? undefined : invoices.find((item) => item.sourceId === source && item.status === "VOID");
  const payableInvoices = invoices.filter((item) => (item.status === "POSTED" || item.status === "PARTIALLY_PAID") && item.outstanding > 0);

  return <div className="finance-panel">
    <div className="finance-forms">
      {allowCreate && <form onSubmit={createInvoice}>
        <strong>{resolvingInvoice ? `Koreksi ${resolvingInvoice.documentNo}` : kind === "supplier" ? "Catat invoice supplier" : "Buat invoice penjualan"}</strong>
        <label>{kind === "supplier" ? "Purchase Order" : "Sales Order"}<select value={source} onChange={(event) => { void selectSource(event.target.value); }} required disabled={Boolean(resolvingInvoice)}><option value="">Pilih dokumen</option>{availableSources.map((item) => <option key={item.id} value={item.id}>{item.documentNo} · {item.status}</option>)}</select></label>
        {sourceError && <small>{sourceError}</small>}
        {replacementInvoice && <small>Replacement untuk invoice VOID {replacementInvoice.documentNo}.</small>}
        {kind === "supplier" && <>
          {!resolvingInvoice && <label>Nomor invoice supplier<input value={externalNumber} onChange={(event) => setExternalNumber(event.target.value)} required /></label>}
          {invoiceLines.map((item) => <div className="invoice-line-editor" key={item.referenceId}>
            <small>{item.productId}</small>
            <div className="form-pair"><label>Qty<input type="number" min="0.0001" step="0.0001" value={item.quantity} onChange={(event) => updateInvoiceLine(item.referenceId, "quantity", event.target.value)} required /></label><label>Harga<input type="number" min="0" step="0.01" value={item.unitPrice} onChange={(event) => updateInvoiceLine(item.referenceId, "unitPrice", event.target.value)} required /></label></div>
          </div>)}
        </>}
        <span className="row-actions">
          <button className="button" disabled={kind === "supplier" && invoiceLines.length === 0}>{resolvingInvoice ? "Simpan koreksi" : "Simpan invoice"}</button>
          {resolvingInvoice && <button type="button" onClick={() => { setResolvingInvoice(null); setSource(""); setInvoiceLines([]); }}>Batal</button>}
        </span>
      </form>}
      {allowCreate && <form onSubmit={createPayment}>
        <strong>Catat pembayaran parsial</strong>
        <label>Invoice<select value={invoice} onChange={(event) => {
          const selected = payableInvoices.find((item) => item.id === event.target.value);
          setInvoice(event.target.value);
          setParty(selected?.partyId || "");
          setAmount(selected ? String(selected.outstanding) : "");
        }} required><option value="">Pilih invoice</option>{payableInvoices.map((item) => <option key={item.id} value={item.id}>{item.documentNo} · outstanding {money(item.outstanding)}</option>)}</select></label>
        <label>{kind === "supplier" ? "Supplier ID" : "Customer ID"}<input value={party} readOnly required /></label>
        <label>Jumlah IDR<input type="number" min="0.01" step="0.01" max={payableInvoices.find((item) => item.id === invoice)?.outstanding} value={amount} onChange={(event) => setAmount(event.target.value)} required /></label>
        <button className="button">Buat payment draft</button>
      </form>}
    </div>
    {allowApprove && <label className="void-reason">Alasan void<input value={voidReason} onChange={(event) => setVoidReason(event.target.value)} placeholder="Wajib sebelum void" /></label>}
    <h3>Invoice</h3>
    <div className="data-table">
      <div className="data-head finance"><span>Dokumen</span><span>Status</span><span>Total</span><span>Outstanding</span><span>Aksi</span></div>
      {invoices.map((item) => <div className="data-row finance" key={item.id}>
        <b>{item.documentNo}</b><span className="status-chip">{item.status}</span><span>{money(item.total)}</span><strong>{money(item.outstanding)}</strong>
        <span className="row-actions">
          {kind === "supplier" && allowApprove && item.status === "MATCH_PENDING" && (item.creatorId !== actorId || allowSelfApprove) && <button onClick={() => command(`${invoicePrefix}/${item.id}/match`)}>Match</button>}
          {kind === "supplier" && allowCreate && item.status === "MATCH_EXCEPTION" && <button onClick={() => { void beginResolve(item); }}>Resolve</button>}
          {kind === "customer" && allowApprove && item.status === "DRAFT" && (item.creatorId !== actorId || allowSelfApprove) && <button onClick={() => command(`${invoicePrefix}/${item.id}/post`, { version: item.version })}>Post</button>}
          {allowApprove && kind === "customer" && item.status === "DRAFT" && <button disabled={!voidReason} onClick={() => command(`${invoicePrefix}/${item.id}/void`, { reason: voidReason })}>Void</button>}
          {allowApprove && item.status === "POSTED" && <button disabled={!voidReason} onClick={() => command(`${invoicePrefix}/${item.id}/void`, { reason: voidReason })}>Void</button>}
        </span>
      </div>)}
    </div>
    <h3>Payment</h3>
    <div className="data-table">
      <div className="data-head finance"><span>Dokumen</span><span>Status</span><span>Jumlah</span><span>Currency</span><span>Aksi</span></div>
      {payments.map((item) => <div className="data-row finance" key={item.id}>
        <b>{item.documentNo}</b><span className="status-chip">{item.status}</span><span>{money(item.total)}</span><span>{item.currency}</span>
        <span className="row-actions">
          {allowApprove && item.status === "DRAFT" && (item.creatorId !== actorId || allowSelfApprove) && <button onClick={() => command(`${paymentPrefix}/${item.id}/post`, { version: item.version })}>Post</button>}
          {allowApprove && item.status === "POSTED" && <button disabled={!voidReason} onClick={() => command(`${paymentPrefix}/${item.id}/void`, { reason: voidReason })}>Void</button>}
        </span>
      </div>)}
    </div>
  </div>;
}
