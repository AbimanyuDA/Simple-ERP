import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FinanceWorkspace } from "@/features/finance/finance-workspace";

vi.mock("@/lib/api", () => ({
  api: vi.fn((path: string) => Promise.resolve({
    data: path === "/supplier-invoices"
      ? [{ id: "i1", documentNo: "SINV-001", status: "PARTIALLY_PAID", partyId: "s1", sourceId: "po1", currency: "IDR", total: 1000, outstanding: 600, version: 3 }]
      : [],
  })),
}));

describe("finance workspace", () => {
  it("shows AP outstanding and posted immutability guidance", async () => {
    render(<FinanceWorkspace />);
    expect(await screen.findByText("SINV-001")).toBeInTheDocument();
    expect(screen.getByText("Rp600")).toBeInTheDocument();
    expect(screen.getByText(/Posted records tidak diedit/)).toBeInTheDocument();
  });
});
