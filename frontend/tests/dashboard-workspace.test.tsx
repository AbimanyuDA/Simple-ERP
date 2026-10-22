import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DashboardWorkspace } from "@/features/dashboard/dashboard-workspace";

const session = vi.hoisted(() => ({ role: "SUPERVISOR" as "SUPERVISOR" | "STAFF" }));
vi.mock("@/features/auth/session-context", () => ({
  useSession: () => ({ session, loading: false }),
  canCreate: (role: string) => role === "STAFF" || role === "SUPER_ADMIN",
}));

vi.mock("@/lib/api", () => ({ api: vi.fn((path: string) => Promise.resolve(path === "/dashboard/summary" ? {
  pendingApprovals: 2, activePurchases: 3, activeSales: 4, zeroStockItems: 1,
  payables: 500000, receivables: 750000, overduePayables: 0, overdueReceivables: 250000,
} : path === "/audit-logs" ? { data: [{
  id: 1, occurredAt: "2026-09-23T01:00:00Z", actor: "admin@example.test", action: "CREATE",
  traceId: "PUR-PR-001", entityType: "PURCHASE_REQUEST", entityId: "pr-1", requestId: "request-1",
}] } : { data: [] })) }));

afterEach(() => { cleanup(); session.role = "SUPERVISOR"; });

describe("dashboard workspace", () => {
  it("shows actionable reconciled metrics", async () => {
    render(<DashboardWorkspace />);
    expect(await screen.findByText("Rp750.000")).toBeInTheDocument();
    expect(screen.getByText("2 perlu review")).toBeInTheDocument();
    expect(screen.getByText("1 item stok nol")).toBeInTheDocument();
  });

  it("filters audit evidence by stable trace data", async () => {
    const view = render(<DashboardWorkspace />);
    fireEvent.click(view.getByRole("button", { name: "Audit" }));
    const search = await view.findByLabelText("Cari audit");
    fireEvent.change(search, { target: { value: "PUR-PR-001" } });
    expect(await screen.findByText("PUR-PR-001")).toBeInTheDocument();
    fireEvent.change(search, { target: { value: "tidak-ada" } });
    expect(await screen.findByText("Belum ada data untuk laporan ini.")).toBeInTheDocument();
  });

  it("keeps unauthorized reports out of the staff view", async () => {
    session.role = "STAFF";
    const view = render(<DashboardWorkspace />);
    expect(await screen.findByText("Rp750.000")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Audit" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Buat purchase request" })).toBeInTheDocument();
    expect(view.container.querySelector(".attention-grid--compact")?.children).toHaveLength(2);
  });
});
