"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { useSession } from "@/features/auth/session-context";

type Props = { onLogin?: (email: string, password: string) => Promise<void> };

export function LoginForm({ onLogin }: Props) {
  const router = useRouter();
  const { refreshSession } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (!email) return setError("Email wajib diisi.");
    if (!password) return setError("Kata sandi wajib diisi.");
    setBusy(true);
    try {
      await (onLogin ? onLogin(email, password) : api("/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      }));
      if (!onLogin) {
        if (!await refreshSession()) throw new Error("Sesi belum terbaca. Coba masuk kembali.");
        router.push("/");
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Tidak dapat masuk.");
    } finally {
      setBusy(false);
    }
  }

  return <form className="login-form" onSubmit={submit} noValidate>
    <div><label htmlFor="email">Email</label><input id="email" type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} /></div>
    <div><label htmlFor="password">Kata sandi</label><input id="password" type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} /></div>
    {error && <p className="form-error" role="alert">{error}</p>}
    <Button type="submit" disabled={busy}>{busy ? "Memeriksa..." : "Masuk"}</Button>
  </form>;
}
