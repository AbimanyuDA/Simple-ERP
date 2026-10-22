import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AppShell } from "@/components/app-shell";

let pathname = "/";
const session = vi.hoisted(() => ({ role: "SUPER_ADMIN" as "SUPER_ADMIN" | "SUPERVISOR" | "STAFF" }));
const actions = vi.hoisted(() => ({ replace: vi.fn(), clearSession: vi.fn(), logout: vi.fn() }));

vi.mock("next/navigation", () => ({
  usePathname: () => pathname,
  useRouter: () => ({ replace: actions.replace }),
}));
vi.mock("@/features/auth/session-context", () => ({
  useSession: () => ({ session: { userId: "test-user", email: "reviewer@example.test", displayName: "Rina Reviewer", role: session.role }, loading: false, clearSession: actions.clearSession }),
}));
vi.mock("@/lib/api", () => ({ api: actions.logout }));

describe("application shell", () => {
  beforeEach(() => {
    pathname = "/";
    session.role = "SUPER_ADMIN";
    actions.replace.mockReset(); actions.clearSession.mockReset(); actions.logout.mockReset();
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
    expect(screen.getByText("Rina Reviewer")).toBeInTheDocument();
    expect(screen.getByText("Super Admin · reviewer@example.test")).toBeInTheDocument();
  });

  it("marks the current module instead of always highlighting Dashboard", () => {
    pathname = "/purchasing";
    render(<AppShell><p>Isi</p></AppShell>);

    expect(screen.getByRole("link", { name: "Purchasing" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Dashboard" })).not.toHaveAttribute("aria-current");
  });

  it("hides privileged navigation for staff", () => {
    session.role = "STAFF";
    render(<AppShell><p>Isi</p></AppShell>);
    expect(screen.getByText("Finance")).toBeInTheDocument();
    expect(screen.queryByText("Users")).not.toBeInTheDocument();
    expect(screen.queryByText("Settings")).not.toBeInTheDocument();
    expect(screen.getByText("Staff · reviewer@example.test")).toBeInTheDocument();
  });

  it("shows finance but not user administration for supervisor", () => {
    session.role = "SUPERVISOR";
    render(<AppShell><p>Isi</p></AppShell>);
    expect(screen.getByText("Finance")).toBeInTheDocument();
    expect(screen.queryByText("Users")).not.toBeInTheDocument();
    expect(screen.queryByText("Settings")).not.toBeInTheDocument();
  });

  it("collapses and expands the sidebar from the header control", async () => {
    const user = userEvent.setup();
    render(<AppShell><p>Isi</p></AppShell>);

    await user.click(screen.getByRole("button", { name: "Ciutkan sidebar" }));
    expect(screen.getByRole("button", { name: "Buka sidebar" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Simple ERP" })).toHaveAttribute(
      "src",
      expect.stringContaining("simple-erp-mark.png"),
    );
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

  it("revokes the session before navigating to login", async () => {
    actions.logout.mockResolvedValue({});
    render(<AppShell><p>Isi</p></AppShell>);
    await userEvent.click(screen.getByRole("button", { name: "Keluar" }));
    expect(actions.logout).toHaveBeenCalledWith("/auth/logout", { method: "POST" });
    expect(actions.clearSession).toHaveBeenCalledOnce();
    expect(actions.replace).toHaveBeenCalledWith("/login");
  });

  it("keeps the session when logout fails", async () => {
    actions.logout.mockRejectedValue(new Error("network"));
    render(<AppShell><p>Isi</p></AppShell>);
    await userEvent.click(screen.getByRole("button", { name: "Keluar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Gagal keluar");
    expect(actions.clearSession).not.toHaveBeenCalled();
    expect(actions.replace).not.toHaveBeenCalled();
  });
});
