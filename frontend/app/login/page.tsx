import Image from "next/image";
import { LoginForm } from "@/features/auth/login-form";

export default function LoginPage() {
  return <main className="login-page">
    <section className="login-story">
      <span className="eyebrow">SIMPLE ERP / OPERASIONAL UMKM</span>
      <h1>Operasional yang<br />lebih tertata.</h1>
      <p>Pembelian, stok, penjualan, dan pembayaran dalam satu workspace yang terhubung.</p>
      <div className="story-line"><span>PEMBELIAN</span><i /><span>STOK</span><i /><span>PENJUALAN</span></div>
    </section>
    <section className="login-card">
      <div>
        <Image className="login-brand-logo" src="/simple-erp-logo.png" alt="Simple ERP" width={217} height={72} priority />
        <h2>Masuk ke workspace</h2>
        <p>Gunakan akun yang diberikan Administrator.</p>
      </div>
      <LoginForm />
      <small>Aktivitas operasional dicatat dalam audit trail.</small>
    </section>
  </main>;
}
