import { NextRequest } from "next/server";
import { describe, expect, it } from "vitest";
import { proxy } from "@/proxy";

describe("authentication proxy", () => {
  it("allows public branding assets before authentication", () => {
    const response = proxy(
      new NextRequest("http://localhost:3000/simple-erp-logo.png"),
    );

    expect(response.status).toBe(200);
    expect(response.headers.get("x-middleware-next")).toBe("1");
  });

  it("still protects arbitrary routes containing a file extension", () => {
    const response = proxy(
      new NextRequest("http://localhost:3000/reports/confidential.csv"),
    );

    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe("http://localhost:3000/login");
  });
});
