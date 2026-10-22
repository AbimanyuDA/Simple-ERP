"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { Role } from "@/features/auth/session-context";

type User = { id: string; email: string; displayName: string; role: Role; status: string; version: number };

export function UsersPanel() {
  const [users, setUsers] = useState<User[]>([]);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [role, setRole] = useState<Role>("STAFF");
  const [message, setMessage] = useState("");
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    api<{ data: User[] }>("/users").then((value) => setUsers(value.data)).catch((error: Error) => setMessage(error.message));
  }, [revision]);
  async function invite(event: React.FormEvent) {
    event.preventDefault();
    try {
      const value = await api<{ invitationToken: string }>("/user-invitations", { method: "POST", body: JSON.stringify({ email, displayName: name, role }) });
      setMessage(`Token undangan (tampil sekali): ${value.invitationToken}`);
      setEmail(""); setName(""); setRevision((current) => current + 1);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Gagal mengundang.");
    }
  }
  return <div className="data-workspace">
    <form className="inline-form" onSubmit={invite}>
      <input aria-label="Nama pengguna" placeholder="Nama" value={name} onChange={(event) => setName(event.target.value)} required />
      <input aria-label="Email pengguna" type="email" placeholder="email@usaha.id" value={email} onChange={(event) => setEmail(event.target.value)} required />
      <select aria-label="Role" value={role} onChange={(event) => setRole(event.target.value as Role)}>
        <option value="STAFF">Staff</option><option value="SUPERVISOR">Supervisor</option><option value="SUPER_ADMIN">Super Admin</option>
      </select>
      <button className="button">Undang</button>
    </form>
    {message && <p className="notice">{message}</p>}
    <div className="data-table"><div className="data-head users"><span>Pengguna</span><span>Role</span><span>Status</span></div>
      {users.map((user) => <div className="data-row users" key={user.id}><span><b>{user.displayName}</b><small>{user.email}</small></span><span>{user.role.replaceAll("_", " ")}</span><small className="empty-badge">{user.status}</small></div>)}
    </div>
  </div>;
}
