import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AppShell } from "@/components/app-shell";

let pathname = "/";

vi.mock("next/navigation", () => ({
  usePathname: () => pathname,
}));

describe("application shell", () => {
  beforeEach(() => {
    pathname = "/";
    window.localStorage.clear();
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches: false }));
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("shows the approved navigation and process context", () => {
    render(<AppShell><p>Isi</p></AppShell>);
    expect(screen.getByRole("img", { name: "Simple ERP" })).toBeInTheDocument();
    for (const label of ["Dashboard", "Purchasing", "Sales", "Inventory", "Finance", "Master Data", "Users", "Settings"]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
    expect(screen.getByText("Isi")).toBeInTheDocument();
  });

  it("marks the current module instead of always highlighting Dashboard", () => {
    pathname = "/purchasing";
    render(<AppShell><p>Isi</p></AppShell>);

    expect(screen.getByRole("link", { name: "Purchasing" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Dashboard" })).not.toHaveAttribute("aria-current");
  });

  it("collapses and expands the sidebar from the header control", async () => {
    const user = userEvent.setup();
    render(<AppShell><p>Isi</p></AppShell>);

    await user.click(screen.getByRole("button", { name: "Ciutkan sidebar" }));
    expect(screen.getByRole("button", { name: "Buka sidebar" })).toBeInTheDocument();
    expect(window.localStorage.getItem("simple-erp-sidebar-collapsed")).toBe("true");

    await user.click(screen.getByRole("button", { name: "Buka sidebar" }));
    expect(screen.getByRole("button", { name: "Ciutkan sidebar" })).toBeInTheDocument();
    expect(window.localStorage.getItem("simple-erp-sidebar-collapsed")).toBe("false");
  });

  it("restores the saved collapsed preference", async () => {
    window.localStorage.setItem("simple-erp-sidebar-collapsed", "true");
    render(<AppShell><p>Isi</p></AppShell>);

    expect(await screen.findByRole("button", { name: "Buka sidebar" })).toBeInTheDocument();
  });

  it("starts collapsed on a compact screen without a saved preference", async () => {
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches: true }));
    render(<AppShell><p>Isi</p></AppShell>);

    expect(await screen.findByRole("button", { name: "Buka sidebar" })).toBeInTheDocument();
  });
});
