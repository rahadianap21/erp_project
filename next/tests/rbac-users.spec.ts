import { test, expect } from "@playwright/test";

test('Frontend - User tanpa izin "users:read" dialihkan ke halaman 403', async ({
  page,
}) => {
  const domainHost = "localhost";
  const targetUrl = `http://${domainHost}:3000`;

  await page.context().addCookies([
    {
      name: "auth_token",
      value: "MOCK_TOKEN_WITHOUT_PERMISSIONS",
      domain: domainHost,
      path: "/",
    },
    {
      name: "user_permissions",
      value: JSON.stringify([]),
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

  await page.goto(`${targetUrl}/users`);

  // Layout server-side memanggil /api/me dan mengalihkan ke /403 karena
  // permission users:read tidak ada.
  await expect(page).toHaveURL(`${targetUrl}/403`);

  const forbiddenHeader = page.getByRole("heading", {
    name: "403 - Forbidden",
  });
  await expect(forbiddenHeader).toBeVisible();
});

test("Frontend - Error Boundary menangkap error 403 dari API dan menampilkan ForbiddenView", async ({
  page,
}) => {
  const domainHost = "localhost";

  // Semua request client sekarang lewat /api/proxy dengan target ter-encode.
  await page.route("**/api/proxy**", async (route) => {
    const target =
      new URL(route.request().url()).searchParams.get("target") ?? "";
    if (target.startsWith("/api/users")) {
      await route.fulfill({
        status: 403,
        contentType: "application/json",
        body: JSON.stringify({
          error: "Forbidden: Akses ditolak sistem granular RBAC",
        }),
      });
      return;
    }
    await route.fallback();
  });

  await page.context().addCookies([
    {
      name: "auth_token",
      value: "VALID_TOKEN_UNTUK_TEST",
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

  await page.goto("http://localhost:3000/users", { waitUntil: "networkidle" });

  const forbiddenHeader = page.getByRole("heading", {
    name: "403 - Forbidden",
  });
  await expect(forbiddenHeader).toBeVisible({ timeout: 5000 });
});

test("Frontend - Sidebar menyembunyikan grup Users Management jika tidak punya izin terkait", async ({
  page,
}) => {
  const domainHost = "localhost";
  const targetUrl = `http://${domainHost}:3000`;

  await page.context().addCookies([
    {
      name: "auth_token",
      value: "VALID_MOCK_TOKEN",
      domain: domainHost,
      path: "/",
    },
    {
      name: "user_permissions",
      value: JSON.stringify(["dashboard:read"]),
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

  await page.goto(`${targetUrl}/dashboard`, { waitUntil: "networkidle" });

  const dashboardText = page.getByText("Dashboard", { exact: true });
  await expect(dashboardText).toBeVisible({ timeout: 5000 });

  const userManagementText = page.getByText("Users Management");
  await expect(userManagementText).not.toBeVisible();
});
