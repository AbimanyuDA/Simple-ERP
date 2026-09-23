import { notFound } from "next/navigation";
import { ModulePage } from "@/components/module-page";
import { SalesWorkspace, type SalesSection } from "@/features/sales/sales-workspace";

const pages: Record<SalesSection, [string, string]> = { "sales-orders": ["Sales Orders", "Kelola pesanan dan validasi ketersediaan stok."], deliveries: ["Deliveries", "Catat pengiriman barang untuk sales order."] };
export default async function Page({ params }: { params: Promise<{ section: string }> }) { const { section } = await params; if (!(section in pages)) notFound(); const key = section as SalesSection; return <ModulePage eyebrow="ORDER-TO-CASH" title={pages[key][0]} description={pages[key][1]}><SalesWorkspace section={key} /></ModulePage>; }
