// features/session.test.ts
import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  DEPRECATED_SESSION_KEYS,
  SESSION_MINUTES,
  SESSION_REFRESH_KEYS,
  buildSessionRefresh,
  sessionCookieOptions,
  sessionExpiry,
} from "./session.ts";

describe("buildSessionRefresh", () => {
  it("menyertakan user_permissions", () => {
    // Ini regresi untuk bug sesi mati tepat 15 menit: permission hilang
    // karena tidak ikut di-refresh.
    const refresh = buildSessionRefresh(() => "nilai");

    assert.ok(
      "user_permissions" in refresh,
      "user_permissions wajib ada, kalau tidak cookie ini hilang 15 menit setelah login",
    );
    assert.equal(refresh.user_permissions, "nilai");
  });

  it("menyertakan seluruh cookie sesi yang dibutuhkan", () => {
    const refresh = buildSessionRefresh(() => "x");

    assert.deepEqual(Object.keys(refresh).sort(), [
      "session_active",
      "user_app",
      "user_email",
      "user_name",
      "user_permissions",
    ]);
  });

  it("tidak menulis ulang key yang tidak ada di cookie", () => {
    const store: Record<string, string> = { user_email: "a@b.c" };
    const refresh = buildSessionRefresh((key) => store[key]);

    // user_email ada, sisanya tidak -> hanya user_email yang dikembalikan.
    assert.deepEqual(refresh, { user_email: "a@b.c" });
    assert.equal("user_permissions" in refresh, false);
  });

  it("melewati nilai kosong", () => {
    const refresh = buildSessionRefresh((key) =>
      key === "user_email" ? "a@b.c" : "",
    );

    assert.deepEqual(refresh, { user_email: "a@b.c" });
  });

  it("melewati nilai undefined", () => {
    const refresh = buildSessionRefresh((key) =>
      key === "user_permissions" ? undefined : "x",
    );

    assert.equal("user_permissions" in refresh, false);
    assert.equal(refresh.user_email, "x");
  });

  it("menulis ulang nilai kosong hanya jika cookie memang kosong di store", () => {
    // Jadi string kosong yang benar-benar ada tetap dilewati, agar tidak
    // mengunci user ke state "permission kosong".
    const store: Record<string, string> = {
      user_email: "a@b.c",
      user_permissions: "",
    };
    const refresh = buildSessionRefresh((key) => store[key]);

    assert.deepEqual(refresh, { user_email: "a@b.c" });
  });

  it("menolak key yang sudah deprecated", () => {
    const refresh = buildSessionRefresh(() => "x");

    for (const key of DEPRECATED_SESSION_KEYS) {
      assert.equal(
        key in refresh,
        false,
        `${key} sudah tidak dipakai dan harus dibuang`,
      );
    }
  });

  it("mengembalikan objek kosong saat tidak ada cookie sama sekali", () => {
    assert.deepEqual(
      buildSessionRefresh(() => undefined),
      {},
    );
  });

  it("mengembalikan objek baru setiap panggilan", () => {
    const first = buildSessionRefresh(() => "x");
    const second = buildSessionRefresh(() => "x");

    assert.notEqual(
      first,
      second,
      "fungsi harus murni, tidak boleh mengembalikan objek bersama",
    );
    assert.deepEqual(first, second);
  });

  it("bekerja saat reader dioper tanpa bind, seperti Cookies.get", () => {
    // session-extender meneruskan `Cookies.get` secara unbound. js-cookie
    // tidak memakai `this`, tapi guard `arguments.length && !name` di
    // dalamnya berarti reader akan mengembalikan undefined kalau kita
    // mengirim key kosong. Test ini mengunci kedua kontrak itu.
    const jar: Record<string, string> = {
      user_email: "a@b.c",
      user_name: "Budi",
      user_permissions: '["po:read"]',
    };

    const jsCookieLike = function (name: string): string | undefined {
      if (name === undefined || name === "") return undefined;
      return jar[name];
    };

    const refresh = buildSessionRefresh(jsCookieLike);

    assert.deepEqual(refresh, {
      user_email: "a@b.c",
      user_name: "Budi",
      user_permissions: '["po:read"]',
    });

    // Tidak ada key kosong yang pernah dikirim ke reader.
    const requested: string[] = [];
    buildSessionRefresh((key) => {
      requested.push(key);
      return "x";
    });
    assert.equal(
      requested.every((key) => key.length > 0),
      true,
    );
  });
});

describe("SESSION_REFRESH_KEYS", () => {
  it("tidak memuat key deprecated", () => {
    for (const key of DEPRECATED_SESSION_KEYS) {
      assert.equal(
        (SESSION_REFRESH_KEYS as readonly string[]).includes(key),
        false,
        `${key} tidak boleh ada di daftar refresh`,
      );
    }
  });
});

describe("sessionExpiry", () => {
  it("menambah 15 menit", () => {
    const now = new Date("2026-10-03T08:00:00.000Z");
    const expiry = sessionExpiry(now);

    assert.equal(expiry.toISOString(), "2026-10-03T08:15:00.000Z");
  });

  it("memakai 15 menit", () => {
    assert.equal(SESSION_MINUTES, 15);
  });
});

describe("sessionCookieOptions", () => {
  it("memakai path, sameSite, dan secure yang sama dengan login", () => {
    const expires = new Date("2026-10-03T08:15:00.000Z");
    const options = sessionCookieOptions(expires);

    assert.equal(options.expires, expires);
    assert.equal(options.sameSite, "lax");
    assert.equal(options.secure, process.env.NODE_ENV === 'production');
    assert.equal(
      options.path,
      "/",
      "path harus eksplisit agar konsisten dengan register-form",
    );
  });
});
