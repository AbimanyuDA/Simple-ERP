import { NextRequest, NextResponse } from "next/server";
export function proxy(request: NextRequest){const signedIn=request.cookies.has("__Host-erp_access");const isPublic=["/login","/activate","/reset-password"].some(v=>request.nextUrl.pathname.startsWith(v));if(!signedIn&&!isPublic)return NextResponse.redirect(new URL("/login",request.url));if(signedIn&&request.nextUrl.pathname.startsWith("/login"))return NextResponse.redirect(new URL("/",request.url));return NextResponse.next()}
export const config={matcher:["/((?!api|_next/static|_next/image|favicon.ico).*)"]};
