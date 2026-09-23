"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { InventoryAdjustmentPanel } from "./inventory-adjustment-panel";

type Balance = { warehouseId: string; productId: string; quantity: number };
type Movement = { id: string; warehouseId: string; productId: string; quantity: number; type: string; sourceType: string; sourceId: string };
export type InventorySection = "stock-balances" | "stock-movements" | "stock-adjustments";

export function InventoryWorkspace({ section }: { section: InventorySection }) {
  const [balances, setBalances] = useState<Balance[]>([]); const [movements, setMovements] = useState<Movement[]>([]); const [notice, setNotice] = useState(""); const [revision, setRevision] = useState(0);
  useEffect(() => { Promise.all([api<{ data: Balance[] }>("/inventory/balances"), api<{ data: Movement[] }>("/inventory/movements")]).then(([balanceData, movementData]) => { setBalances(balanceData.data); setMovements(movementData.data); }).catch((error: Error) => setNotice(error.message)); }, [revision]);
  return <div className="data-workspace">{notice && <p className="notice">{notice}</p>}
    {section === "stock-balances" && <div className="data-table"><div className="data-head"><span>Gudang</span><span>Produk</span><span>Qty</span></div>{balances.length === 0 ? <p>Belum ada saldo stok.</p> : balances.map((item) => <div className="data-row" key={`${item.warehouseId}-${item.productId}`}><small>{item.warehouseId}</small><span>{item.productId}</span><b>{item.quantity}</b></div>)}</div>}
    {section === "stock-movements" && <div className="data-table"><div className="data-head movement"><span>Jenis</span><span>Source</span><span>Produk / Gudang</span><span>Qty</span></div>{movements.length === 0 ? <p>Belum ada movement.</p> : movements.map((item) => <div className="data-row movement" key={item.id}><small>{item.type}</small><span>{item.sourceType} · {item.sourceId}</span><span><small>{item.productId}</small><small>{item.warehouseId}</small></span><b className={item.quantity < 0 ? "negative" : "positive"}>{item.quantity}</b></div>)}</div>}
    {section === "stock-adjustments" && <InventoryAdjustmentPanel onNotice={setNotice} onChanged={() => setRevision((value) => value + 1)} />}
  </div>;
}
