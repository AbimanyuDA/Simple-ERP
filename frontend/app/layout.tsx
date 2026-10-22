import type { Metadata } from "next";
import "./globals.css";
import "./operations.css";
import "./navigation.css";
import "./apple-theme.css";
import { SessionProvider } from "@/features/auth/session-context";

export const metadata: Metadata = {
  title: "Simple ERP",
  description: "Mini ERP operasional untuk UMKM",
  icons: { icon: "/simple-erp-mark.png", apple: "/simple-erp-mark.png" },
};
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <html lang="id"><body><SessionProvider>{children}</SessionProvider></body></html>;
}
