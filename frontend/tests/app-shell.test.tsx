import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AppShell } from "@/components/app-shell";

describe("application shell", () => {
  it("shows the approved navigation and process context", () => {
    render(<AppShell><p>Isi</p></AppShell>);
    expect(screen.getByRole("img", { name: "Simple ERP" })).toBeInTheDocument();
    for (const label of ["Dashboard", "Purchasing", "Sales", "Inventory", "Finance", "Master Data", "Users", "Settings"]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
    expect(screen.getByText("Isi")).toBeInTheDocument();
  });
});
