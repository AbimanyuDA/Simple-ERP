"use client";

import { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";

export type Role = "STAFF" | "SUPERVISOR" | "SUPER_ADMIN";
type Session = { userId: string; email: string; displayName: string; role: Role };
export const canCreate = (role?: Role) => role === "STAFF" || role === "SUPER_ADMIN";
export const canApprove = (role?: Role) => role === "SUPERVISOR" || role === "SUPER_ADMIN";
type SessionContextValue = { session: Session | null; loading: boolean; refreshSession: () => Promise<Session | null>; clearSession: () => void };
const SessionContext = createContext<SessionContextValue>({ session: null, loading: true, refreshSession: async () => null, clearSession: () => {} });

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [loading, setLoading] = useState(true);
  const requestId = useRef(0);
  const refreshSession = useCallback(async () => {
    const currentRequest = ++requestId.current;
    try {
      const current = await api<Session>("/auth/session");
      if (currentRequest === requestId.current) setSession(current);
      return current;
    } catch {
      if (currentRequest === requestId.current) setSession(null);
      return null;
    } finally {
      if (currentRequest === requestId.current) setLoading(false);
    }
  }, []);
  useEffect(() => {
    const initialRequest = ++requestId.current;
    api<Session>("/auth/session")
      .then((current) => { if (initialRequest === requestId.current) setSession(current); })
      .catch(() => { if (initialRequest === requestId.current) setSession(null); })
      .finally(() => { if (initialRequest === requestId.current) setLoading(false); });
  }, []);
  return <SessionContext.Provider value={{ session, loading, refreshSession, clearSession: () => setSession(null) }}>{children}</SessionContext.Provider>;
}

export function useSession() { return useContext(SessionContext); }
