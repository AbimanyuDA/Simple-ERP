import { AppShell } from "@/components/app-shell";
import { DashboardWorkspace } from "@/features/dashboard/dashboard-workspace";

export default function Dashboard() {
  return <AppShell><section className="content">
    <div className="page-title"><div><span className="eyebrow">WORKSPACE HARI INI</span><h1>Kendali usaha, tanpa kebisingan.</h1><p>Ringkasan aktivitas yang benar-benar memerlukan perhatian.</p></div></div>
    <div className="process-ribbon"><span>ALUR HARI INI</span><div><b>Permintaan</b><i/> <b>Persetujuan</b><i/> <b>Pemenuhan</b><i/> <b>Pembayaran</b></div></div>
    <DashboardWorkspace />
  </section></AppShell>;
}
