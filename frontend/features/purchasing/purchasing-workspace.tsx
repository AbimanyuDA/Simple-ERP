"use client";

import { useState } from "react";
import { GoodsReceiptPanel } from "./goods-receipt-panel";
import { PurchaseOrderPanel } from "./purchase-order-panel";
import { PurchaseRequestPanel } from "./purchase-request-panel";

export type PurchasingSection = "purchase-requests" | "purchase-orders" | "goods-receipts";

export function PurchasingWorkspace({ section }: { section: PurchasingSection }) {
  const [notice, setNotice] = useState("");
  return <div className="data-workspace">{notice && <p className="notice">{notice}</p>}{section === "purchase-requests" && <PurchaseRequestPanel onNotice={setNotice} />}{section === "purchase-orders" && <PurchaseOrderPanel onNotice={setNotice} />}{section === "goods-receipts" && <GoodsReceiptPanel onNotice={setNotice} />}</div>;
}
