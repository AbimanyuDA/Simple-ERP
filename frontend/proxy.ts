import { NextRequest, NextResponse } from "next/server";

const publicRoutes = ["/login", "/activate", "/reset-password"];

export function proxy(request: NextRequest) {
  const pathname = request.nextUrl.pathname;
  // Access token berumur pendek. Refresh token tetap mengizinkan halaman dibuka,
  // lalu API client akan merotasi access token ketika request pertama menerima 401.
  const signedIn = request.cookies.has("__Host-erp_access") || request.cookies.has("__Host-erp_refresh");
  const isPublicRoute = publicRoutes.some((route) => pathname.startsWith(route));
  const isPublicAsset = ["/simple-erp-logo.png", "/simple-erp-mark.png"].includes(pathname);

  if (isPublicAsset) return NextResponse.next();
  if (!signedIn && !isPublicRoute) {
    return NextResponse.redirect(new URL("/login", request.url));
  }
  if (signedIn && pathname.startsWith("/login")) {
    return NextResponse.redirect(new URL("/", request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico|simple-erp-logo.png|simple-erp-mark.png).*)"],
};
