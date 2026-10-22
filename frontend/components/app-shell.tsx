"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { BarChart3, Boxes, ChevronDown, CircleDollarSign, LayoutDashboard, LogOut, PackageCheck, PanelLeftClose, PanelLeftOpen, Settings, ShoppingBag, Truck, Users } from "lucide-react";
import { useSession } from "@/features/auth/session-context";
import { api } from "@/lib/api";

const sidebarPreferenceKey = "simple-erp-sidebar-collapsed";
const roleLabels = { STAFF: "Staff", SUPERVISOR: "Supervisor", SUPER_ADMIN: "Super Admin" } as const;
const navigation = [
  { label: "Dashboard", href: "/", icon: LayoutDashboard },
  { label: "Purchasing", href: "/purchasing/purchase-requests", icon: ShoppingBag, items: [["Purchase Requests", "/purchasing/purchase-requests"], ["Purchase Orders", "/purchasing/purchase-orders"], ["Goods Receipts", "/purchasing/goods-receipts"]] },
  { label: "Sales", href: "/sales/sales-orders", icon: Truck, items: [["Sales Orders", "/sales/sales-orders"], ["Deliveries", "/sales/deliveries"]] },
  { label: "Inventory", href: "/inventory/stock-balances", icon: Boxes, items: [["Stock Balances", "/inventory/stock-balances"], ["Stock Movements", "/inventory/stock-movements"], ["Stock Adjustments", "/inventory/stock-adjustments"]] },
  { label: "Finance", href: "/finance", icon: CircleDollarSign },
  { label: "Master Data", href: "/master-data", icon: PackageCheck },
  { label: "Users", href: "/users", icon: Users, superAdminOnly: true },
  { label: "Settings", href: "/settings", icon: Settings, superAdminOnly: true },
] as const;

function isCurrentRoute(pathname: string, href: string) { if (href === "/") return pathname === href; const root = `/${href.split("/")[1]}`; return pathname === root || pathname === href || pathname.startsWith(`${root}/`); }

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname(); const router = useRouter(); const [collapsed, setCollapsed] = useState(false); const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [loggingOut, setLoggingOut] = useState(false); const [logoutError, setLogoutError] = useState("");
  const { session, clearSession } = useSession();
  const accountName = session?.displayName?.trim() || session?.email || "Memuat akun";
  const initials = accountName.split(/\s+/).map((part) => part[0]).slice(0, 2).join("").toUpperCase();
  useEffect(() => { const saved = window.localStorage.getItem(sidebarPreferenceKey); const compact = window.matchMedia("(max-width: 900px)").matches; const initial = saved === null ? compact : saved === "true"; queueMicrotask(() => setCollapsed(initial)); }, []);
  function toggleSidebar() { setCollapsed((current) => { const next = !current; window.localStorage.setItem(sidebarPreferenceKey, String(next)); return next; }); }
  async function logout() {
    setLoggingOut(true); setLogoutError("");
    try { await api("/auth/logout", { method: "POST" }); clearSession(); router.replace("/login"); }
    catch { setLogoutError("Gagal keluar. Coba lagi."); setLoggingOut(false); }
  }
  return <div className={`shell${collapsed ? " shell--collapsed" : ""}`}>
    <header className="app-header"><div className="header-brand"><Link className="brand" href="/" aria-label="Simple ERP — Dashboard"><Image className={`brand-logo${collapsed ? " brand-logo--mark" : ""}`} src={collapsed ? "/simple-erp-mark.png" : "/simple-erp-logo.png"} alt="Simple ERP" width={collapsed ? 48 : 200} height={collapsed ? 48 : 67} priority /></Link><button type="button" className="sidebar-toggle" aria-controls="app-sidebar" aria-expanded={!collapsed} aria-label={collapsed ? "Buka sidebar" : "Ciutkan sidebar"} onClick={toggleSidebar}>{collapsed ? <PanelLeftOpen size={17} /> : <PanelLeftClose size={17} />}</button></div><div className="header-context"><div className="header-title"><span className="eyebrow">WORKSPACE</span><strong>Simple ERP</strong></div><div className="header-account" aria-label="Akun yang sedang masuk"><span className="avatar" aria-hidden="true">{initials || "?"}</span><div className="header-account-details"><strong>{accountName}</strong><span>{session?.role ? roleLabels[session.role] : "Memuat role"} · {session?.email || ""}</span></div></div>{logoutError && <span role="alert">{logoutError}</span>}<button type="button" className="logout-button" onClick={logout} disabled={loggingOut} aria-label="Keluar"><LogOut size={16} />Keluar</button></div></header>
    <aside className="sidebar" id="app-sidebar"><nav aria-label="Navigasi utama">{navigation.filter((item) => (!("reviewerOnly" in item) || !item.reviewerOnly || session?.role === "SUPERVISOR" || session?.role === "SUPER_ADMIN") && (!("superAdminOnly" in item) || !item.superAdminOnly || session?.role === "SUPER_ADMIN")).map((item) => { const active = isCurrentRoute(pathname, item.href); const hasItems = "items" in item; const open = hasItems && (expanded[item.label] ?? active); const Icon = item.icon; return <div className={`nav-group${active ? " active" : ""}`} key={item.label}><div className="nav-group-row"><Link href={item.href} aria-label={item.label} aria-current={active ? "page" : undefined} title={collapsed ? item.label : undefined}><Icon size={18} /><span>{item.label}</span></Link>{hasItems && <button type="button" className="nav-expander" aria-label={`${open ? "Tutup" : "Buka"} submenu ${item.label}`} aria-expanded={open} onClick={() => setExpanded((current) => ({ ...current, [item.label]: !open }))}><ChevronDown size={15} /></button>}</div>{hasItems && open && <div className="subnav">{item.items.map(([label, href]) => <Link href={href} key={href} aria-current={pathname === href ? "page" : undefined}>{label}</Link>)}</div>}</div>; })}</nav><div className="sidebar-foot"><BarChart3 size={17}/><span>Operational workspace</span></div></aside>
    <main className="main">{children}</main>
  </div>;
}
