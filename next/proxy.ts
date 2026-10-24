import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

function matchesPrefix(pathname: string, prefix: string): boolean {
  return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  // auth_token (httpOnly) adalah satu-satunya sinyal login yang tidak bisa
  // dipalsukan JavaScript. Permission tidak lagi diperiksa di sini karena
  // cookie user_permissions bisa diubah klien; layout server-side (lib/dal)
  // yang memverifikasi permission ke /api/me.
  const token = request.cookies.get("auth_token")?.value;
  const isLoggedIn = Boolean(token);
  const app = request.cookies.get("user_app")?.value ?? "erp";

  // Belum login: hanya halaman login (root) dan /403 yang boleh diakses.
  if (!isLoggedIn) {
    if (pathname === "/" || pathname.startsWith("/403")) {
      return NextResponse.next();
    }
    return NextResponse.redirect(new URL("/", request.url));
  }

  // Sudah login. /403 selalu boleh dirender agar pengguna tanpa izin
  // melihat halaman "Dilarang", bukan dilempar keluar.
  if (pathname.startsWith("/403")) {
    return NextResponse.next();
  }

  // Pengguna aplikasi POS (kasir) dikunci ke area /pos saja. Ini hanya
  // kenyamanan navigasi; pembatas keamanan tetap permission di server.
  if (app === "pos" && !matchesPrefix(pathname, "/pos")) {
    return NextResponse.redirect(new URL("/pos", request.url));
  }

  return NextResponse.next();
}

export const config = {
  // Jalankan pada semua halaman; lewati API, aset Next, dan berkas ber-ekstensi.
  matcher: ["/((?!api|_next|favicon.ico|.*\\..*).*)"],
};
