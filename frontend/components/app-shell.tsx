import Image from "next/image";
import Link from "next/link";
import { BarChart3, Boxes, CircleDollarSign, LayoutDashboard, PackageCheck, Settings, ShoppingBag, Truck, Users } from "lucide-react";

const navigation = [
  ["Dashboard", "/", LayoutDashboard], ["Purchasing", "/purchasing", ShoppingBag],
  ["Sales", "/sales", Truck], ["Inventory", "/inventory", Boxes],
  ["Finance", "/finance", CircleDollarSign], ["Master Data", "/master-data", PackageCheck],
  ["Users", "/users", Users], ["Settings", "/settings", Settings],
] as const;

export function AppShell({ children }: { children: React.ReactNode }) {
  return <div className="shell">
    <aside className="sidebar">
      <Link className="brand" href="/" aria-label="Simple ERP — Dashboard">
        <Image className="brand-logo" src="/simple-erp-logo.png" alt="Simple ERP" width={181} height={60} priority />
      </Link>
      <nav>{navigation.map(([label, href, Icon]) => <Link href={href} key={label}><Icon size={18}/><span>{label}</span></Link>)}</nav>
      <div className="sidebar-foot"><BarChart3 size={17}/><span>Semua proses terlacak</span></div>
    </aside>
    <main className="main"><header><div><span className="eyebrow">WORKSPACE</span><strong>CV Usaha Bersama</strong></div><div className="avatar">AD</div></header>{children}</main>
  </div>;
}
