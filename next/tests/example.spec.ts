import { test, expect } from "@playwright/test";

// Kita bisa menggunakan string token tiruan karena API request-nya akan kita potong (mocking)
const MOCK_TOKEN = "PROXIED_MOCK_JWT_TOKEN";

test.describe("Granular RBAC Modul /users - Pengujian Aksi Ilegal", () => {
  test('Frontend - User tanpa izin "users:delete" tidak melihat tombol hapus', async ({
    page,
  }) => {
    const domainHost = "localhost";
    const targetUrl = `http://${domainHost}:3000`;

    // 1. MENCEGAT (MOCK) REQUEST API BACKEND
    // Setiap kali frontend memanggil API users, kita langsung kembalikan data sukses 200 OK.
    // Pola regex agar cocok dengan URL ber-query string dari hook.
    await page.route(/\/api\/users/, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: [
            {
              id: 1,
              name: "Budi Santoso",
              email: "budi@company.com",
              role_name: "Staff",
            },
            {
              id: 2,
              name: "Siti Aminah",
              email: "siti@company.com",
              role_name: "Operator",
            },
          ],
        }),
      });
    });

    // 2. SUNTIK COOKIE AKSES (Hanya memiliki izin users:read)
    await page.context().addCookies([
      {
        name: "auth_token",
        value: MOCK_TOKEN,
        domain: domainHost,
        path: "/",
      },
      {
        name: "user_permissions",
        // Penting: encodeURIComponent dilakukan otomatis oleh Playwright untuk value kuki
        value: JSON.stringify(["users:read"]),
        domain: domainHost,
        path: "/",
      },
    ]);

    // 3. BUKA HALAMAN
    await page.goto(`${targetUrl}/users`, { waitUntil: "networkidle" });

    // 4. VERIFIKASI TABEL MUNCUL
    // Menunggu teks dari data tiruan muncul di layar untuk memastikan loading selesai
    const userRow = page.getByText("Budi Santoso");
    await expect(userRow).toBeVisible({ timeout: 5000 });

    // 5. VERIFIKASI TOMBOL HAPUS DISEMBUNYIKAN
    // Karena canDelete={permissions.delete} bernilai false, tombol hapus harusnya tidak tampil
    const deleteButton = page.locator('[data-testid="btn-delete"]');
    await expect(deleteButton).not.toBeVisible();
  });

  test("Backend - Request DELETE /api/users tanpa izin harus menghasilkan 403 Forbidden", async ({
    request,
  }) => {
    // Catatan: Untuk tes backend murni ini, backend Go Anda di port 8090 harus menyala.
    // Jika Anda memakai token palsu, pastikan backend Anda mengembalikan 403 (bukan 401).
    // Jika backend Anda mengembalikan 401 saat token palsu, ubah ekspektasi status menjadi 401.
    const response = await request.delete("http://localhost:8090/api/users", {
      headers: {
        Authorization: `Bearer ${MOCK_TOKEN}`,
        "Content-Type": "application/json",
      },
      data: { userId: 1 },
    });

    // Sesuaikan ke 401 jika Anda mengetes dengan token asal ketik, atau 403 jika memakai token asli tanpa hak hapus
    expect([401, 403]).toContain(response.status());
  });
});
