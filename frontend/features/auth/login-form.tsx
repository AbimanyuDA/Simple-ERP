"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";

type Props = { onLogin?: (email: string, password: string) => Promise<void> };

export function LoginForm({ onLogin }: Props) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(e: React.FormEvent) {
    e.preventDefault(); setError("");
    if (!email) return setError("Email wajib diisi.");
    if (!password) return setError("Kata sandi wajib diisi.");
    setBusy(true);
    try {
      await (onLogin ? onLogin(email, password) : api("/auth/login", { method: "POST", body: JSON.stringify({ email, password }) }));
      if (!onLogin) router.push("/");
    } catch (e) { setError(e instanceof Error ? e.message : "Tidak dapat masuk."); }
    finally { setBusy(false); }
  }
  return <form className="login-form" onSubmit={submit} noValidate>
    <div><label htmlFor="email">Email</label><input id="email" type="email" autoComplete="email" value={email} onChange={e => setEmail(e.target.value)} /></div>
    <div><label htmlFor="password">Kata sandi</label><input id="password" type="password" autoComplete="current-password" value={password} onChange={e => setPassword(e.target.value)} /></div>
    {error && <p className="form-error" role="alert">{error}</p>}
    <Button type="submit" disabled={busy}>{busy ? "Memeriksa…" : "Masuk"}</Button>
  </form>;
}
