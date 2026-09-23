import { notFound } from "next/navigation";
import { ModulePage } from "@/components/module-page";
import { PurchasingWorkspace, type PurchasingSection } from "@/features/purchasing/purchasing-workspace";

const pages: Record<PurchasingSection, [string, string]> = { "purchase-requests": ["Purchase Requests", "Ajukan dan review kebutuhan pembelian."], "purchase-orders": ["Purchase Orders", "Konversi PR yang disetujui menjadi pesanan supplier."], "goods-receipts": ["Goods Receipts", "Catat penerimaan barang dari purchase order."] };
export default async function Page({ params }: { params: Promise<{ section: string }> }) { const { section } = await params; if (!(section in pages)) notFound(); const key = section as PurchasingSection; return <ModulePage eyebrow="PROCURE-TO-PAY" title={pages[key][0]} description={pages[key][1]}><PurchasingWorkspace section={key} /></ModulePage>; }
