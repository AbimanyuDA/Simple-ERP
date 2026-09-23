"use client";

import { useState } from "react";
import { DeliveryPanel } from "./delivery-panel";
import { SalesOrderPanel } from "./sales-order-panel";

export type SalesSection = "sales-orders" | "deliveries";

export function SalesWorkspace({ section }: { section: SalesSection }) {
  const [notice, setNotice] = useState("");
  return <div className="data-workspace">{notice && <p className="notice">{notice}</p>}{section === "sales-orders" ? <SalesOrderPanel onNotice={setNotice} /> : <DeliveryPanel onNotice={setNotice} />}</div>;
}
