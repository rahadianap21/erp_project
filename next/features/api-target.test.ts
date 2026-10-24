import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { resolveApiTarget } from "../lib/api-target.ts";

const BASE = "http://localhost:8090";

describe("resolveApiTarget", () => {
  it("menerima path API relatif", () => {
    const target = resolveApiTarget("/api/users?page=1", BASE);

    assert.ok(target);
    assert.equal(target.origin, BASE);
    assert.equal(target.pathname, "/api/users");
    assert.equal(target.search, "?page=1");
  });

  it("menolak target yang mengubah host", () => {
    assert.equal(resolveApiTarget("@evil.example/api", BASE), null);
    assert.equal(resolveApiTarget("//evil.example/api", BASE), null);
    assert.equal(resolveApiTarget("http://evil.example/api/users", BASE), null);
  });

  it("menolak path di luar /api", () => {
    assert.equal(resolveApiTarget("/login", BASE), null);
    assert.equal(resolveApiTarget("/", BASE), null);
    assert.equal(resolveApiTarget("/api/../admin", BASE), null);
  });

  it("menolak traversal yang di-encode", () => {
    assert.equal(resolveApiTarget("/api/%2e%2e/admin", BASE), null);
  });

  it("menolak base URL yang tidak valid", () => {
    assert.equal(resolveApiTarget("/api/users", "bukan-url"), null);
  });
});
