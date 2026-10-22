import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FinanceWorkspace } from "@/features/finance/finance-workspace";

const session = vi.hoisted(() => ({ role: "SUPERVISOR" as "STAFF" | "SUPERVISOR" | "SUPER_ADMIN", userId: "reviewer" }));
vi.mock("@/features/auth/session-context", () => ({
  useSession: () => ({ session, loading: false }),
  canCreate: (role: string) => role === "STAFF" || role === "SUPER_ADMIN",
  canApprove: (role: string) => role === "SUPERVISOR" || role === "SUPER_ADMIN",
}));

vi.mock("@/lib/api", () => ({
  api: vi.fn((path: string) => Promise.resolve({
    data: path === "/supplier-invoices"
      ? [{ id: "i1", documentNo: "SINV-001", status: "PARTIALLY_PAID", partyId: "s1", sourceId: "po1", currency: "IDR", total: 1000, outstanding: 600, version: 3 },
        { id: "i2", documentNo: "SINV-002", status: "MATCH_PENDING", creatorId: "maker", partyId: "s1", sourceId: "po2", currency: "IDR", total: 1000, outstanding: 1000, version: 1 }]
      : path === "/supplier-payments"
        ? [{ id: "p1", documentNo: "SPAY-001", status: "DRAFT", creatorId: "maker", partyId: "s1", sourceId: "", currency: "IDR", total: 100, outstanding: 0, version: 1 }]
      : path === "/sales-invoices"
        ? [{ id: "i3", documentNo: "CINV-001", status: "DRAFT", creatorId: "maker", partyId: "c1", sourceId: "so1", currency: "IDR", total: 1000, outstanding: 1000, version: 1 }]
      : path === "/customer-payments"
        ? [{ id: "p2", documentNo: "CPAY-001", status: "DRAFT", creatorId: "maker", partyId: "c1", sourceId: "", currency: "IDR", total: 100, outstanding: 0, version: 1 }]
      : [],
  })),
}));

describe("finance workspace", () => {
  beforeEach(() => { session.role = "SUPERVISOR"; session.userId = "reviewer"; });
  afterEach(() => { cleanup(); });
  it("shows AP outstanding and posted immutability guidance", async () => {
    render(<FinanceWorkspace />);
    expect(await screen.findByText("SINV-001")).toBeInTheDocument();
    expect(screen.getByText("Rp600")).toBeInTheDocument();
    expect(screen.getByText(/Posted records tidak diedit/)).toBeInTheDocument();
  });

  it("lets staff draft AP and AR but not match, post, or void", async () => {
    session.role = "STAFF";
    render(<FinanceWorkspace />);
    expect(await screen.findByText("SINV-002")).toBeInTheDocument();
    expect(screen.getByText("Catat invoice supplier")).toBeInTheDocument();
    expect(screen.getByText("Catat pembayaran parsial")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Receivables · Customer" }));
    expect(await screen.findByText("CINV-001")).toBeInTheDocument();
    expect(screen.getByText("Buat invoice penjualan")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Post" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Payables · Supplier" }));
    expect(screen.queryByRole("button", { name: "Match" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Post" })).not.toBeInTheDocument();
  });

  it("lets supervisor match and post AP but not create its drafts", async () => {
    render(<FinanceWorkspace />);
    expect(await screen.findByText("SINV-002")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Match" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Post" })).toBeInTheDocument();
    expect(screen.queryByText("Catat invoice supplier")).not.toBeInTheDocument();
    expect(screen.queryByText("Catat pembayaran parsial")).not.toBeInTheDocument();
  });

  it("hides self-approval from supervisor but allows the super admin bypass", async () => {
    session.userId = "maker";
    const supervisor = render(<FinanceWorkspace />);
    expect(await screen.findByText("SINV-002")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Match" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Post" })).not.toBeInTheDocument();
    supervisor.unmount();

    session.role = "SUPER_ADMIN";
    render(<FinanceWorkspace />);
    expect(await screen.findByText("SINV-002")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Match" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Post" })).toBeInTheDocument();
  });

  it("shows AR post actions to a different supervisor, not its creator", async () => {
    render(<FinanceWorkspace />);
    fireEvent.click(screen.getByRole("button", { name: "Receivables · Customer" }));
    expect(await screen.findByText("CINV-001")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Post" })).toHaveLength(2);
    expect(screen.queryByText("Buat invoice penjualan")).not.toBeInTheDocument();
  });
});
