import { test, expect } from "@playwright/test";

const MOCK_TOKEN = "PROXIED_MOCK_JWT_TOKEN";

test.describe("Granular RBAC Modul /users - Pengujian Aksi Ilegal", () => {
  test('Frontend - User tanpa izin "users:delete" tidak melihat tombol hapus', async ({
    page,
  }) => {
    const domainHost = "localhost";
    const targetUrl = `http://${domainHost}:3000`;

    await page.route("**/api/proxy**", async (route) => {
      const target =
        new URL(route.request().url()).searchParams.get("target") ?? "";
      if (target.startsWith("/api/users")) {
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
            total: 2,
          }),
        });
        return;
      }
      await route.fallback();
    });

    await page.context().addCookies([
      {
        name: "auth_token",
        value: MOCK_TOKEN,
        domain: domainHost,
        path: "/",
      },
      {
        name: "user_permissions",
        value: JSON.stringify(["users:read"]),
        domain: domainHost,
        path: "/",
      },
      {
        name: "session_active",
        value: "1",
        domain: domainHost,
        path: "/",
      },
    ]);

    await page.goto(`${targetUrl}/users`, { waitUntil: "networkidle" });

    const userRow = page.getByText("Budi Santoso");
    await expect(userRow).toBeVisible({ timeout: 5000 });

    const deleteButton = page.locator('[data-testid="btn-delete"]');
    await expect(deleteButton).not.toBeVisible();
  });

  test("Backend - Request DELETE /api/users tanpa izin harus menghasilkan 403 Forbidden", async ({
    request,
  }) => {
    const target = encodeURIComponent("/api/users");
    const response = await request.delete(
      `http://localhost:3000/api/proxy?target=${target}`,
      {
        headers: {
          Cookie: `auth_token=${MOCK_TOKEN}`,
          "Content-Type": "application/json",
        },
        data: { userId: 1 },
      }
    );

    expect(response.status()).toBe(403);
  });
});
