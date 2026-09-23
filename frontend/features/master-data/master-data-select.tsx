"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Resource = "products" | "suppliers" | "customers" | "warehouses" | "units";
type Option = { id: string; code: string; name: string; active: boolean };

export function MasterDataSelect({ resource, label, value, onChange, required = true }: { resource: Resource; label: string; value: string; onChange: (value: string) => void; required?: boolean }) {
  const [items, setItems] = useState<Option[]>([]);
  const [error, setError] = useState("");
  useEffect(() => { api<{ data: Option[] }>(`/master-data/${resource}`).then((response) => setItems(response.data.filter((item) => item.active))).catch((cause: Error) => setError(cause.message)); }, [resource]);
  return <label>{label}<select value={value} onChange={(event) => onChange(event.target.value)} required={required}><option value="">{error || `Pilih ${label.toLowerCase()}`}</option>{items.map((item) => <option value={item.id} key={item.id}>{item.code} · {item.name}</option>)}</select></label>;
}
