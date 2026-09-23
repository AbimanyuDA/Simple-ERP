"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import {
  BarChart3,
  Boxes,
  CircleDollarSign,
  LayoutDashboard,
  PackageCheck,
  PanelLeftClose,
  PanelLeftOpen,
  Settings,
  ShoppingBag,
  Truck,
  Users,
} from "lucide-react";

const sidebarPreferenceKey = "simple-erp-sidebar-collapsed";

const navigation = [
  ["Dashboard", "/", LayoutDashboard],
  ["Purchasing", "/purchasing", ShoppingBag],
  ["Sales", "/sales", Truck],
  ["Inventory", "/inventory", Boxes],
  ["Finance", "/finance", CircleDollarSign],
  ["Master Data", "/master-data", PackageCheck],
  ["Users", "/users", Users],
  ["Settings", "/settings", Settings],
] as const;

function isCurrentRoute(pathname: string, href: string) {
  return href === "/" ? pathname === href : pathname === href || pathname.startsWith(`${href}/`);
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    const savedPreference = window.localStorage.getItem(sidebarPreferenceKey);
    const compactScreen = window.matchMedia("(max-width: 900px)").matches;
    const initialPreference = savedPreference === null ? compactScreen : savedPreference === "true";
    queueMicrotask(() => setCollapsed(initialPreference));
  }, []);

  function toggleSidebar() {
    setCollapsed((current) => {
      const next = !current;
      window.localStorage.setItem(sidebarPreferenceKey, String(next));
      return next;
    });
  }

  return (
    <div className={`shell${collapsed ? " shell--collapsed" : ""}`}>
      <header className="app-header">
        <div className="header-brand">
          <Link className="brand" href="/" aria-label="Simple ERP — Dashboard">
            <Image
              className={`brand-logo${collapsed ? " brand-logo--mark" : ""}`}
              src={collapsed ? "/simple-erp-mark.png" : "/simple-erp-logo.png"}
              alt="Simple ERP"
              width={collapsed ? 48 : 200}
              height={collapsed ? 48 : 67}
              priority
            />
          </Link>
          <button
            type="button"
            className="sidebar-toggle"
            aria-controls="app-sidebar"
            aria-expanded={!collapsed}
            aria-label={collapsed ? "Buka sidebar" : "Ciutkan sidebar"}
            onClick={toggleSidebar}
          >
            {collapsed ? <PanelLeftOpen size={17} /> : <PanelLeftClose size={17} />}
          </button>
        </div>
        <div className="header-context">
          <div><span className="eyebrow">WORKSPACE</span><strong>CV Usaha Bersama</strong></div>
          <div className="avatar">AD</div>
        </div>
      </header>
      <aside className="sidebar" id="app-sidebar">
        <nav aria-label="Navigasi utama">
          {navigation.map(([label, href, Icon]) => {
            const active = isCurrentRoute(pathname, href);
            return (
              <Link
                href={href}
                key={label}
                aria-label={label}
                aria-current={active ? "page" : undefined}
                title={collapsed ? label : undefined}
              >
                <Icon size={18} />
                <span>{label}</span>
              </Link>
            );
          })}
        </nav>
        <div className="sidebar-foot"><BarChart3 size={17}/><span>Semua proses terlacak</span></div>
      </aside>
      <main className="main">{children}</main>
    </div>
  );
}
