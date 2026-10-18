// features/session.ts
//
// Logika sesi dipisah dari komponen supaya bisa diuji tanpa browser.
// Fungsi-fungsi di file ini murni: tidak menyentuh `document`, `window`,
// atau `js-cookie`, sehingga aman dipanggil dari `node --test`.

/**
 * Cookie yang wajib ikut diperpanjang agar sesi tidak mati di tengah usage.
 *
 * `user_permissions` WAJIB ada di daftar ini. Cookie ini ditulis saat login
 * dengan umur 15 menit, dan enam halaman memakainya sebagai syarat login.
 * Kalau tidak ikut di-refresh, permission akan hilang 15 menit setelah login
 * walaupun user masih aktif mengetik — user lalu dipaksa logout.
 */
export const SESSION_REFRESH_KEYS = [
    'user_email',
    'user_name',
    'user_permissions',
    // Menandai aplikasi pengguna ('erp' atau 'pos'). Dipakai proxy untuk
    // mengunci kasir ke area /pos, jadi wajib ikut diperpanjang.
    'user_app',
    // Penanda sesi yang bisa dibaca JavaScript; pengganti `auth_token` yang
    // sekarang httpOnly. Dipakai halaman dan proxy sebagai gerbang login.
    'session_active',
] as const;

export type SessionKey = (typeof SESSION_REFRESH_KEYS)[number];

/**
 * Cookie yang dulu ikut di-refresh tapi tidak pernah ditulis di mana pun:
 * - `user_role` dibaca di session-extender, tapi tidak pernah di-`Cookies.set`.
 * - `role_id` ditulis di session-extender dari `user_role` (jadi selalu
 *   `undefined`), sementara penulisan aslinya di login-form dikomentari.
 * Keduanya tidak pernah dibaca halaman mana pun, jadi dibuang di sini.
 */
export const DEPRECATED_SESSION_KEYS = ['user_role', 'role_id'] as const;

/** Sumber nilai cookie, mis. dibungkus `Cookies.get`. */
export type SessionCookieReader = (key: string) => string | undefined;

/**
 * Mengembalikan daftar key -> nilai yang perlu ditulis ulang.
 *
 * Key yang tidak ada di cookie atau bernilai kosong string TIDAK ikut
 * dikembalikan, supaya tidak menimpa cookie yang sudah ada dengan `undefined`.
 */
export function buildSessionRefresh(read: SessionCookieReader): Record<string, string> {
    const refresh: Record<string, string> = {};

    for (const key of SESSION_REFRESH_KEYS) {
        const value = read(key);
        if (value !== undefined && value !== '') {
            refresh[key] = value;
        }
    }

    return refresh;
}

/** Opsi penulisan cookie, harus sama dengan yang dipakai saat login. */
export function sessionCookieOptions(expires: Date) {
    return {
        expires,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'lax' as const,
        path: '/',
    };
}

/** Umur sesi cookie, dalam menit. */
export const SESSION_MINUTES = 15;

export function sessionExpiry(now: Date = new Date()): Date {
    return new Date(now.getTime() + SESSION_MINUTES * 60 * 1000);
}