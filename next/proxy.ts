import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Pemetaan prefiks halaman -> permission yang dibutuhkan untuk membacanya.
// Prefiks dicocokkan sebagai segmen penuh: '/pos' cocok untuk '/pos' dan
// '/pos/apa-pun', tapi tidak untuk '/positions'.
const PERMISSION_GATES: { prefix: string; permission: string }[] = [
    { prefix: '/users', permission: 'users:read' },
    { prefix: '/roles', permission: 'roles:read' },
    { prefix: '/suppliers', permission: 'suppliers:read' },
    { prefix: '/products', permission: 'products:read' },
    { prefix: '/product-categories', permission: 'categories:read' },
    { prefix: '/purchase-orders', permission: 'po:read' },
    { prefix: '/goods-receipt-notes', permission: 'grn:read' },
    { prefix: '/sales', permission: 'sales:read' },
    { prefix: '/payment-methods', permission: 'payment-methods:read' },
    { prefix: '/accounting', permission: 'accounting:read' },
    { prefix: '/dashboard', permission: 'dashboard:read' },
    { prefix: '/members', permission: 'members:read' },
    { prefix: '/pos', permission: 'pos:read' },
    { prefix: '/settings', permission: 'settings:read' },
];

function parsePermissions(raw?: string): string[] | null {
    if (!raw) return null;
    try {
        const parsed = JSON.parse(decodeURIComponent(raw));
        return Array.isArray(parsed) ? parsed : null;
    } catch {
        return null;
    }
}

function matchesPrefix(pathname: string, prefix: string): boolean {
    return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

export function proxy(request: NextRequest) {
    const { pathname } = request.nextUrl;
    // auth_token (httpOnly) membuktikan sesi asli; session_active (bisa dibaca
    // JavaScript) menandai bahwa klien belum logout. Keduanya harus ada agar
    // logout sisi klien langsung menutup akses ke halaman.
    const token = request.cookies.get('auth_token')?.value;
    const sessionActive = request.cookies.get('session_active')?.value;
    const isLoggedIn = Boolean(token && sessionActive);
    const app = request.cookies.get('user_app')?.value ?? 'erp';

    // Belum login: hanya halaman login (root) dan /403 yang boleh diakses.
    if (!isLoggedIn) {
        if (pathname === '/' || pathname.startsWith('/403')) {
            return NextResponse.next();
        }
        return NextResponse.redirect(new URL('/', request.url));
    }

    // Sudah login. /403 selalu boleh dirender agar pengguna tanpa izin
    // melihat halaman "Dilarang", bukan dilempar keluar.
    if (pathname.startsWith('/403')) {
        return NextResponse.next();
    }

    // Membuka halaman login saat sudah login -> arahkan ke landing sesuai app.
    if (pathname === '/') {
        return NextResponse.redirect(new URL(app === 'pos' ? '/pos' : '/dashboard', request.url));
    }

    // Pengguna aplikasi POS (kasir) dikunci ke area /pos saja.
    if (app === 'pos' && !matchesPrefix(pathname, '/pos')) {
        return NextResponse.redirect(new URL('/pos', request.url));
    }

    // Gerbang permission per halaman.
    for (const gate of PERMISSION_GATES) {
        if (matchesPrefix(pathname, gate.prefix)) {
            const permissions = parsePermissions(request.cookies.get('user_permissions')?.value);
            if (!permissions || !permissions.includes(gate.permission)) {
                return NextResponse.rewrite(new URL('/403', request.url));
            }
            break;
        }
    }

    return NextResponse.next();
}

export const config = {
    // Jalankan pada semua halaman; lewati API, aset Next, dan berkas ber-ekstensi.
    matcher: ['/((?!api|_next|favicon.ico|.*\\..*).*)'],
};
