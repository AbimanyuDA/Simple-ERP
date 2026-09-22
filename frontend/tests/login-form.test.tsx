import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { LoginForm } from "@/features/auth/login-form";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));

describe("CAP-AUTH-001 login", () => {
  it("validates and submits credentials without browser storage", async () => {
    const login = vi.fn().mockResolvedValue(undefined);
    render(<LoginForm onLogin={login} />);
    await userEvent.click(screen.getByRole("button", { name: /masuk/i }));
    expect(screen.getByText(/email wajib/i)).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText(/email/i), "admin@example.test");
    await userEvent.type(screen.getByLabelText(/kata sandi/i), "Kuat-sekali-2026!");
    await userEvent.click(screen.getByRole("button", { name: /masuk/i }));
    expect(login).toHaveBeenCalledWith("admin@example.test", "Kuat-sekali-2026!");
  });
});
