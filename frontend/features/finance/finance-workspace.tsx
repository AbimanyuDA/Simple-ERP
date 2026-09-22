"use client";

import { FormEvent, useEffect, useState } from "react";
import { api } from "@/lib/api";

type Document = {
  id: string;
  documentNo: string;
  status: string;
  partyId: string;
  sourceId: string;
  currency: string;
  total: number;
  outstanding: number;
  version: number;
};

const today = new Date().toISOString().slice(0, 10);
const money = (value: number) => `Rp${Number(value).toLocaleString("id-ID")}`;

export function FinanceWorkspace() {
  const [tab, setTab] = useState<"ap" | "ar">("ap");
  const [supplierInvoices, setSupplierInvoices] = useState<Document[]>([]);
  const [supplierPayments, setSupplierPayments] = useState<Document[]>([]);
  const [salesInvoices, setSalesInvoices] = useState<Document[]>([]);
  const [customerPayments, setCustomerPayments] = useState<Document[]>([]);
  const [notice, setNotice] = useState("");
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    Promise.all([
      api<{ data: Document[] }>("/supplier-invoices"),
      api<{ data: Document[] }>("/supplier-payments"),
      api<{ data: Document[] }>("/sales-invoices"),
      api<{ data: Document[] }>("/customer-payments"),
    ]).then(([si, sp, ci, cp]) => {
      setSupplierInvoices(si.data);
      setSupplierPayments(sp.data);
      setSalesInvoices(ci.data);
      setCustomerPayments(cp.data);
    }).catch((error: Error) => setNotice(error.message));
  }, [revision]);

  async function command(path: string, body: object = {}) {
    try {
      await api(path, { method: "POST", body: JSON.stringify(body) });
      setNotice("Transaksi finance berhasil diproses.");
      setRevision((value) => value + 1);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Transaksi gagal.");
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
      ? <FinancePanel kind="supplier" invoices={supplierInvoices} payments={supplierPayments} command={command} />
      : <FinancePanel kind="customer" invoices={salesInvoices} payments={customerPayments} command={command} />}
  </div>;
}

function FinancePanel({ kind, invoices, payments, command }: {
  kind: "supplier" | "customer";
  invoices: Document[];
  payments: Document[];
  command: (path: string, body?: object) => Promise<void>;
}) {
  const [source, setSource] = useState("");
  const [party, setParty] = useState("");
  const [externalNumber, setExternalNumber] = useState("");
  const [line, setLine] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [price, setPrice] = useState("0");
  const [invoice, setInvoice] = useState("");
  const [amount, setAmount] = useState("");
  const [voidReason, setVoidReason] = useState("");

  async function createInvoice(event: FormEvent) {
    event.preventDefault();
    if (kind === "supplier") {
      await command("/supplier-invoices", { purchaseOrderId: source, supplierInvoiceNumber: externalNumber, invoiceDate: today, dueDate: today, currency: "IDR", lines: [{ referenceId: line, quantity: Number(quantity), unitPrice: Number(price) }] });
    } else {
      await command(`/sales-orders/${source}/sales-invoice`, { invoiceDate: today, dueDate: today, currency: "IDR" });
    }
  }

  async function createPayment(event: FormEvent) {
    event.preventDefault();
    const prefix = kind === "supplier" ? "/supplier-payments" : "/customer-payments";
    await command(prefix, { partyId: party, paymentDate: today, method: "BANK_TRANSFER", currency: "IDR", amount: Number(amount), allocations: [{ invoiceId: invoice, amount: Number(amount) }] });
  }

  const invoicePrefix = kind === "supplier" ? "/supplier-invoices" : "/sales-invoices";
  const paymentPrefix = kind === "supplier" ? "/supplier-payments" : "/customer-payments";

  return <div className="finance-panel">
    <div className="finance-forms">
      <form onSubmit={createInvoice}>
        <strong>{kind === "supplier" ? "Catat invoice supplier" : "Buat invoice penjualan"}</strong>
        <label>{kind === "supplier" ? "Purchase Order ID" : "Sales Order ID"}<input value={source} onChange={(event) => setSource(event.target.value)} required /></label>
        {kind === "supplier" && <>
          <label>Nomor invoice supplier<input value={externalNumber} onChange={(event) => setExternalNumber(event.target.value)} required /></label>
          <label>PO Line ID<input value={line} onChange={(event) => setLine(event.target.value)} required /></label>
          <div className="form-pair"><label>Qty<input type="number" min="0.0001" step="0.0001" value={quantity} onChange={(event) => setQuantity(event.target.value)} /></label><label>Harga<input type="number" min="0" value={price} onChange={(event) => setPrice(event.target.value)} /></label></div>
        </>}
        <button className="button">Simpan invoice</button>
      </form>
      <form onSubmit={createPayment}>
        <strong>Catat pembayaran parsial</strong>
        <label>{kind === "supplier" ? "Supplier ID" : "Customer ID"}<input value={party} onChange={(event) => setParty(event.target.value)} required /></label>
        <label>Invoice ID<input value={invoice} onChange={(event) => setInvoice(event.target.value)} required /></label>
        <label>Jumlah IDR<input type="number" min="1" value={amount} onChange={(event) => setAmount(event.target.value)} required /></label>
        <button className="button">Buat payment draft</button>
      </form>
    </div>
    <label className="void-reason">Alasan void<input value={voidReason} onChange={(event) => setVoidReason(event.target.value)} placeholder="Wajib sebelum void" /></label>
    <h3>Invoice</h3>
    <div className="data-table">
      <div className="data-head finance"><span>Dokumen</span><span>Status</span><span>Total</span><span>Outstanding</span><span>Aksi</span></div>
      {invoices.map((item) => <div className="data-row finance" key={item.id}>
        <b>{item.documentNo}</b><span className="status-chip">{item.status}</span><span>{money(item.total)}</span><strong>{money(item.outstanding)}</strong>
        <span className="row-actions">
          {kind === "supplier" && item.status === "MATCH_PENDING" && <button onClick={() => command(`${invoicePrefix}/${item.id}/match`)}>Match</button>}
          {item.status === "POSTED" && <button disabled={!voidReason} onClick={() => command(`${invoicePrefix}/${item.id}/void`, { reason: voidReason })}>Void</button>}
        </span>
      </div>)}
    </div>
    <h3>Payment</h3>
    <div className="data-table">
      <div className="data-head finance"><span>Dokumen</span><span>Status</span><span>Jumlah</span><span>Currency</span><span>Aksi</span></div>
      {payments.map((item) => <div className="data-row finance" key={item.id}>
        <b>{item.documentNo}</b><span className="status-chip">{item.status}</span><span>{money(item.total)}</span><span>{item.currency}</span>
        <span className="row-actions">
          {item.status === "DRAFT" && <button onClick={() => command(`${paymentPrefix}/${item.id}/post`, { version: item.version })}>Post</button>}
          {item.status === "POSTED" && <button disabled={!voidReason} onClick={() => command(`${paymentPrefix}/${item.id}/void`, { reason: voidReason })}>Void</button>}
        </span>
      </div>)}
    </div>
  </div>;
}
