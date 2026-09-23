import { notFound } from "next/navigation";
import { ModulePage } from "@/components/module-page";
import { InventoryWorkspace, type InventorySection } from "@/features/inventory/inventory-workspace";

const pages: Record<InventorySection, [string, string]> = { "stock-balances": ["Stock Balances", "Pantau saldo stok per gudang dan produk."], "stock-movements": ["Stock Movements", "Telusuri ledger perubahan stok dari setiap transaksi."], "stock-adjustments": ["Stock Adjustments", "Sesuaikan hasil perhitungan fisik melalui workflow persetujuan."] };
export default async function Page({ params }: { params: Promise<{ section: string }> }) { const { section } = await params; if (!(section in pages)) notFound(); const key = section as InventorySection; return <ModulePage eyebrow="INVENTORY" title={pages[key][0]} description={pages[key][1]}><InventoryWorkspace section={key} /></ModulePage>; }
