import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import LoginPage from "@/app/login/page";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));

describe("login branding", () => {
  it("presents the Simple ERP logo before authentication", () => {
    render(<LoginPage />);
    expect(screen.getByRole("img", { name: "Simple ERP" })).toBeInTheDocument();
  });
});
