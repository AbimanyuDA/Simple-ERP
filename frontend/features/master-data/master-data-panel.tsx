"use client";

import { FormEvent, useEffect, useState } from "react";
import { api } from "@/lib/api";

type RecordItem = { id: string; code: string; name: string; active: boolean; version: number };
const resources = ["units", "products", "suppliers", "customers", "warehouses"] as const;
type Resource = (typeof resources)[number];

export function MasterDataPanel() {
  const [resource, setResource] = useState<Resource>("units"); const [items, setItems] = useState<RecordItem[]>([]); const [busy, setBusy] = useState(true); const [error, setError] = useState(""); const [code, setCode] = useState(""); const [name, setName] = useState(""); const [revision, setRevision] = useState(0);
  useEffect(() => { api<{ data: RecordItem[] }>(`/master-data/${resource}`).then((value) => setItems(value.data)).catch((cause: Error) => setError(cause.message)).finally(() => setBusy(false)); }, [resource, revision]);
  async function create(event: FormEvent) { event.preventDefault(); setError(""); try { await api(`/master-data/${resource}`, { method: "POST", body: JSON.stringify({ code, name }) }); setCode(""); setName(""); setRevision((value) => value + 1); } catch (cause) { setError(cause instanceof Error ? cause.message : "Data gagal disimpan."); } }
  async function toggle(item: RecordItem) { setError(""); try { await api(`/master-data/${resource}/${item.id}/status`, { method: "PATCH", body: JSON.stringify({ active: !item.active, version: item.version }) }); setRevision((value) => value + 1); } catch (cause) { setError(cause instanceof Error ? cause.message : "Status gagal diubah."); } }
  return <div className="data-workspace"><div className="data-tabs">{resources.map((value) => <button className={value === resource ? "active" : ""} onClick={() => { setError(""); setResource(value); }} key={value}>{value}</button>)}</div><form className="inline-form" onSubmit={create}><input aria-label="Kode" placeholder="Kode" value={code} onChange={(event) => setCode(event.target.value)} required /><input aria-label="Nama" placeholder="Nama" value={name} onChange={(event) => setName(event.target.value)} required /><button className="button">Tambah</button></form>{error && <p className="form-error master-error">{error}</p>}<div className="data-table"><div className="data-head master"><span>Kode</span><span>Nama</span><span>Status</span><span>Aksi</span></div>{busy ? <p>Memuat data…</p> : items.length === 0 ? <p>Belum ada data.</p> : items.map((item) => <div className="data-row master" key={item.id}><b>{item.code}</b><span>{item.name}</span><small className="empty-badge">{item.active ? "Aktif" : "Nonaktif"}</small><span className="row-actions"><button onClick={() => toggle(item)}>{item.active ? "Nonaktifkan" : "Aktifkan"}</button></span></div>)}</div></div>;
}
