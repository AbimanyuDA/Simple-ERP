"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { BarChart3, Boxes, ChevronDown, CircleDollarSign, LayoutDashboard, PackageCheck, PanelLeftClose, PanelLeftOpen, Settings, ShoppingBag, Truck, Users } from "lucide-react";

const sidebarPreferenceKey = "simple-erp-sidebar-collapsed";
const navigation = [
  { label: "Dashboard", href: "/", icon: LayoutDashboard },
  { label: "Purchasing", href: "/purchasing/purchase-requests", icon: ShoppingBag, items: [["Purchase Requests", "/purchasing/purchase-requests"], ["Purchase Orders", "/purchasing/purchase-orders"], ["Goods Receipts", "/purchasing/goods-receipts"]] },
  { label: "Sales", href: "/sales/sales-orders", icon: Truck, items: [["Sales Orders", "/sales/sales-orders"], ["Deliveries", "/sales/deliveries"]] },
  { label: "Inventory", href: "/inventory/stock-balances", icon: Boxes, items: [["Stock Balances", "/inventory/stock-balances"], ["Stock Movements", "/inventory/stock-movements"], ["Stock Adjustments", "/inventory/stock-adjustments"]] },
  { label: "Finance", href: "/finance", icon: CircleDollarSign },
  { label: "Master Data", href: "/master-data", icon: PackageCheck },
  { label: "Users", href: "/users", icon: Users },
  { label: "Settings", href: "/settings", icon: Settings },
] as const;

function isCurrentRoute(pathname: string, href: string) { if (href === "/") return pathname === href; const root = `/${href.split("/")[1]}`; return pathname === root || pathname === href || pathname.startsWith(`${root}/`); }

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname(); const [collapsed, setCollapsed] = useState(false); const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  useEffect(() => { const saved = window.localStorage.getItem(sidebarPreferenceKey); const compact = window.matchMedia("(max-width: 900px)").matches; const initial = saved === null ? compact : saved === "true"; queueMicrotask(() => setCollapsed(initial)); }, []);
  function toggleSidebar() { setCollapsed((current) => { const next = !current; window.localStorage.setItem(sidebarPreferenceKey, String(next)); return next; }); }
  return <div className={`shell${collapsed ? " shell--collapsed" : ""}`}>
    <header className="app-header"><div className="header-brand"><Link className="brand" href="/" aria-label="Simple ERP — Dashboard"><Image className={`brand-logo${collapsed ? " brand-logo--mark" : ""}`} src={collapsed ? "/simple-erp-mark.png" : "/simple-erp-logo.png"} alt="Simple ERP" width={collapsed ? 48 : 200} height={collapsed ? 48 : 67} priority /></Link><button type="button" className="sidebar-toggle" aria-controls="app-sidebar" aria-expanded={!collapsed} aria-label={collapsed ? "Buka sidebar" : "Ciutkan sidebar"} onClick={toggleSidebar}>{collapsed ? <PanelLeftOpen size={17} /> : <PanelLeftClose size={17} />}</button></div><div className="header-context"><div><span className="eyebrow">OPERATIONS</span><strong>Simple ERP</strong></div><div className="avatar">ERP</div></div></header>
    <aside className="sidebar" id="app-sidebar"><nav aria-label="Navigasi utama">{navigation.map((item) => { const active = isCurrentRoute(pathname, item.href); const hasItems = "items" in item; const open = hasItems && (expanded[item.label] ?? active); const Icon = item.icon; return <div className={`nav-group${active ? " active" : ""}`} key={item.label}><div className="nav-group-row"><Link href={item.href} aria-label={item.label} aria-current={active ? "page" : undefined} title={collapsed ? item.label : undefined}><Icon size={18} /><span>{item.label}</span></Link>{hasItems && <button type="button" className="nav-expander" aria-label={`${open ? "Tutup" : "Buka"} submenu ${item.label}`} aria-expanded={open} onClick={() => setExpanded((current) => ({ ...current, [item.label]: !open }))}><ChevronDown size={15} /></button>}</div>{hasItems && open && <div className="subnav">{item.items.map(([label, href]) => <Link href={href} key={href} aria-current={pathname === href ? "page" : undefined}>{label}</Link>)}</div>}</div>; })}</nav><div className="sidebar-foot"><BarChart3 size={17}/><span>Operational workspace</span></div></aside>
    <main className="main">{children}</main>
  </div>;
}
