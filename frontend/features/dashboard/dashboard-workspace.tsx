"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { canCreate, useSession } from "@/features/auth/session-context";

type Summary = { pendingApprovals:number;activePurchases:number;activeSales:number;zeroStockItems:number;payables:number;receivables:number;overduePayables:number;overdueReceivables:number };
type Report = { documentNo:string;party:string;status:string;amount:number;outstanding:number;dueDate:string;daysOverdue:number };
type Inventory = { warehouse:string;sku:string;product:string;quantity:number };
type Audit = { id:number;occurredAt:string;actor:string;action:string;traceId:string;entityType:string;entityId:string;previousState?:string;newState?:string;reason?:string;requestId:string };
type View = "overview"|"purchases"|"sales"|"inventory"|"payables"|"receivables"|"audit";

const currency = (value:number) => `Rp${Number(value).toLocaleString("id-ID")}`;
const viewLabels: Record<View,string> = { overview:"Perhatian", purchases:"Pembelian", sales:"Penjualan", inventory:"Persediaan", payables:"Utang", receivables:"Piutang", audit:"Audit" };

export function DashboardWorkspace() {
  const { session } = useSession();
  const canViewReports = session?.role === "SUPERVISOR" || session?.role === "SUPER_ADMIN";
  const [summary, setSummary] = useState<Summary | null>(null);
  const [view, setView] = useState<View>("overview");
  const [rows, setRows] = useState<(Report | Inventory | Audit)[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");

  useEffect(() => {
    api<Summary>("/dashboard/summary").then(setSummary).catch((reason: Error) => setError(reason.message));
  }, []);

  async function open(next: View) {
    if (next !== "overview" && !canViewReports) return;
    setView(next);
    setError("");
    setSearch("");
    if (next === "overview") return;
    setLoading(true);
    try {
      const path = next === "audit" ? "/audit-logs" : `/reports/${next}`;
      const result = await api<{ data: (Report | Inventory | Audit)[] }>(path);
      setRows(result.data);
    } catch (reason) {
      setRows([]);
      setError(reason instanceof Error ? reason.message : "Laporan gagal dimuat.");
    } finally {
      setLoading(false);
    }
  }

  const visibleRows = view === "audit" && search.trim()
    ? rows.filter((item) => Object.values(item).some((value) => String(value ?? "").toLocaleLowerCase("id-ID").includes(search.trim().toLocaleLowerCase("id-ID"))))
    : rows;

  return <>
    {canCreate(session?.role) && <div className="dashboard-actions">
      <Link className="button" href="/purchasing">Buat purchase request</Link>
      <Link className="quiet-link" href="/sales">Buat sales order</Link>
    </div>}
    <div className="metrics">
      <article><span>Perlu persetujuan</span><strong>{summary?.pendingApprovals ?? "-"}</strong><small>{summary ? `${summary.pendingApprovals} perlu review` : "Memuat"}</small></article>
      <article><span>Transaksi aktif</span><strong>{summary ? summary.activePurchases + summary.activeSales : "-"}</strong><small>{summary ? `${summary.activePurchases} purchase / ${summary.activeSales} sales` : "Memuat"}</small></article>
      <article><span>Stok perlu perhatian</span><strong>{summary?.zeroStockItems ?? "-"}</strong><small>{summary ? `${summary.zeroStockItems} item stok nol` : "Memuat"}</small></article>
      <article><span>Piutang berjalan</span><strong>{summary ? currency(summary.receivables) : "-"}</strong><small>{summary?.overdueReceivables ? `${currency(summary.overdueReceivables)} lewat jatuh tempo` : "Tidak ada overdue"}</small></article>
    </div>
    <section className="report-panel">
      {canViewReports && <div className="data-tabs report-tabs">{(["overview","purchases","sales","inventory","payables","receivables","audit"] as View[]).map((item) => <button key={item} className={view === item ? "active" : ""} onClick={() => open(item)}>{viewLabels[item]}</button>)}</div>}
      {error && <p className="form-error report-error">{error}</p>}
      {view === "audit" && !loading && <label className="audit-search">Cari audit<input aria-label="Cari audit" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Actor, action, entity, trace ID, atau request ID" /></label>}
      {view === "overview"
        ? <div className={`attention-grid${canViewReports ? "" : " attention-grid--compact"}`}>
            <article><span>Utang berjalan</span><b>{summary ? currency(summary.payables) : "-"}</b><small>{summary?.overduePayables ? `${currency(summary.overduePayables)} overdue` : "Jadwal aman"}</small></article>
            <article><span>Piutang overdue</span><b>{summary ? currency(summary.overdueReceivables) : "-"}</b><small>Prioritaskan penagihan yang lewat jatuh tempo</small></article>
            {canViewReports && <article><span>Audit trail</span><b>Audit</b><small>Telusuri aktivitas melalui tab Audit.</small></article>}
          </div>
        : loading ? <p className="loading-state">Memuat laporan...</p> : <ReportTable view={view} rows={visibleRows} />}
    </section>
  </>;
}

function ReportTable({ view, rows }: { view: View; rows: (Report | Inventory | Audit)[] }) {
  if (rows.length === 0) return <p className="loading-state">Belum ada data untuk laporan ini.</p>;
  if (view === "inventory") return <div className="data-table"><div className="data-head report"><span>Gudang</span><span>SKU</span><span>Produk</span><span>Qty</span></div>{(rows as Inventory[]).map((row) => <div className="data-row report" key={`${row.warehouse}-${row.sku}`}><span>{row.warehouse}</span><b>{row.sku}</b><span>{row.product}</span><strong>{row.quantity}</strong></div>)}</div>;
  if (view === "audit") return <div className="data-table"><div className="data-head audit"><span>Waktu / Actor</span><span>Action</span><span>Entity</span><span>State / Trace</span></div>{(rows as Audit[]).map((row) => <div className="data-row audit" key={row.id}><span><small>{new Date(row.occurredAt).toLocaleString("id-ID")}</small>{row.actor}</span><b>{row.action}</b><span>{row.entityType}<small>{row.entityId}</small></span><span>{row.previousState || "-"} to {row.newState || "-"}<small>{row.traceId}</small></span></div>)}</div>;
  return <div className="data-table"><div className="data-head report"><span>Dokumen</span><span>Pihak</span><span>Status</span><span>Nilai</span></div>{(rows as Report[]).map((row) => <div className="data-row report" key={row.documentNo}><b>{row.documentNo}</b><span>{row.party}</span><span className="status-chip">{row.status}</span><strong>{currency(row.outstanding || row.amount)}</strong></div>)}</div>;
}
