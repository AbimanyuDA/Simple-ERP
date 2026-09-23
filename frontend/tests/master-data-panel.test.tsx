import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MasterDataPanel } from "@/features/master-data/master-data-panel";

vi.mock("@/lib/api", () => ({ api: vi.fn().mockResolvedValue({ data: [{ id: "1", code: "PCS", name: "Pieces", active: true, version: 1 }] }) }));
describe("CAP-MDM-001 master data", () => { it("shows server records", async () => { render(<MasterDataPanel />); expect(await screen.findByText("Pieces")).toBeInTheDocument(); expect(screen.getByText("PCS")).toBeInTheDocument(); }); });
