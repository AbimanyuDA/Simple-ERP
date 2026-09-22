import { AppShell } from "@/components/app-shell";

export default function Dashboard() {
  return <AppShell><section className="content">
    <div className="page-title"><div><span className="eyebrow">22 SEPTEMBER 2026</span><h1>Selamat pagi, Admin.</h1><p>Ringkasan aktivitas usaha yang perlu perhatian hari ini.</p></div><button className="button">Buat transaksi</button></div>
    <div className="process-ribbon"><span>ALUR HARI INI</span><div><b>Permintaan</b><i/> <b>Persetujuan</b><i/> <b>Pemenuhan</b><i/> <b>Pembayaran</b></div></div>
    <div className="metrics"><article><span>Penjualan bulan ini</span><strong>Rp 0</strong><small>Belum ada transaksi</small></article><article><span>Pesanan aktif</span><strong>0</strong><small>Semua terkendali</small></article><article><span>Stok perlu perhatian</span><strong>0</strong><small>Gudang Utama</small></article><article><span>Piutang berjalan</span><strong>Rp 0</strong><small>Belum ada tagihan</small></article></div>
    <div className="panel"><div><h2>Aktivitas operasional</h2><p>Transaksi terbaru akan tampil di sini dan selalu terhubung ke proses BPMN.</p></div><span className="empty-badge">Siap digunakan</span></div>
  </section></AppShell>;
}
