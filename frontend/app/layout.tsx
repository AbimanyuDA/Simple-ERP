import type { Metadata } from "next";
import { Plus_Jakarta_Sans, JetBrains_Mono } from "next/font/google";
import "./globals.css";

const sans = Plus_Jakarta_Sans({ subsets: ["latin"], variable: "--font-sans" });
const mono = JetBrains_Mono({ subsets: ["latin"], variable: "--font-mono" });
export const metadata: Metadata = { title: "Arus ERP", description: "Mini ERP operasional untuk UMKM" };
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <html lang="id"><body className={`${sans.variable} ${mono.variable}`}>{children}</body></html>;
}
