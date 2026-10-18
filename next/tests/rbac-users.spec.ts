import { test, expect } from '@playwright/test';

test('Frontend - User tanpa izin "users:read" direwrite ke ForbiddenView di URL yang sama', async ({ page }) => {
    const domainHost = 'localhost';
    const targetUrl = `http://${domainHost}:3000`;

    // Suntik kuki dengan permissions kosong []
    await page.context().addCookies([
        {
            name: 'auth_token',
            value: 'MOCK_TOKEN_WITHOUT_PERMISSIONS',
            domain: domainHost,
            path: '/',
        },
        {
            name: 'user_permissions',
            value: JSON.stringify([]), // Kosong, tidak punya "users:read"
            domain: domainHost,
            path: '/',
        }
    ]);

    // Coba paksa buka /users
    await page.goto(`${targetUrl}/users`);

    // middleware.ts memakai NextResponse.rewrite, bukan redirect: URL browser TIDAK
    // berubah (tetap /users), tetapi yang dirender adalah halaman 403.
    await expect(page).toHaveURL(`${targetUrl}/users`);

    // Pastikan teks penolakan 403 terbaca di layar browser
    const forbiddenHeader = page.getByRole('heading', { name: '403 - Forbidden' });
    await expect(forbiddenHeader).toBeVisible();
});

test('Frontend - Error Boundary menangkap error 403 dari API dan menampilkan ForbiddenView', async ({ page }) => {
    const domainHost = 'localhost';

    // Pastikan route membalas dengan status HTTP 403 murni
    // Pola regex agar cocok dengan URL yang membawa query string (?page=...&limit=...).
    await page.route(/\/api\/users/, async (route) => {
        await route.fulfill({
            status: 403,
            contentType: 'application/json',
            body: JSON.stringify({ error: "Forbidden: Akses ditolak sistem granular RBAC" })
        });
    });

    await page.context().addCookies([
        { name: 'auth_token', value: 'VALID_TOKEN_UNTUK_TEST', domain: domainHost, path: '/' },
        { name: 'user_permissions', value: JSON.stringify(["users:read"]), domain: domainHost, path: '/' }
    ]);

    // Buka halaman
    await page.goto('http://localhost:3000/users', { waitUntil: 'networkidle' });

    // Sekarang mestinya heading "403 - Forbidden" dari ForbiddenView langsung muncul di layar
    const forbiddenHeader = page.getByRole('heading', { name: '403 - Forbidden' });
    await expect(forbiddenHeader).toBeVisible({ timeout: 5000 });
});

test('Frontend - Sidebar menyembunyikan grup Users Management jika tidak punya izin terkait', async ({ page }) => {
    const domainHost = 'localhost';
    const targetUrl = `http://${domainHost}:3000`;

    // 1. MENCEGAT PENGALIHAN DARI MIDDLEWARE
    // Jika /dashboard mencoba memanggil endpoint status user atau backend, kita kembalikan status sukses
    // agar middleware mengizinkan browser tetap berada di halaman dashboard.
    await page.route(/\/api\/users/, async (route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) });
    });

    // 2. SUNTIK COOKIE: punya dashboard:read (agar /dashboard lolos gerbang middleware)
    // tetapi tidak punya users:read/roles:read/settings:read.
    await page.context().addCookies([
        { name: 'auth_token', value: 'VALID_MOCK_TOKEN', domain: domainHost, path: '/' },
        { name: 'user_permissions', value: JSON.stringify(['dashboard:read']), domain: domainHost, path: '/' }
    ]);

    // Buka halaman dashboard
    await page.goto(`${targetUrl}/dashboard`, { waitUntil: 'networkidle' });

    // 3. GUNAKAN PENCARIAN BERBASIS TEKS (Jauh lebih aman untuk struktur Shadcn UI Sidebar)
    // Menunggu teks "Dashboard" muncul di area navigasi sidebar
    const dashboardText = page.getByText('Dashboard', { exact: true });
    await expect(dashboardText).toBeVisible({ timeout: 5000 });

    // 4. MEMASTIKAN GRUP "Users Management" BENAR-BENAR HILANG
    // (judul grup di app-sidebar.tsx adalah "Users Management")
    const userManagementText = page.getByText('Users Management');
    await expect(userManagementText).not.toBeVisible();
});
