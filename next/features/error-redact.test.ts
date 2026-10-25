import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  cleanRoute,
  errorMessage,
  redactSensitiveText,
  shouldIgnoreError,
} from "../lib/error-redact.ts";

describe("redactSensitiveText", () => {
  it("menghapus password, token, email, dan telepon", () => {
    const output = redactSensitiveText(
      `password=rahasia123 token=abc123 email=budi@example.com phone=081234567890`
    );

    for (const forbidden of [
      "rahasia123",
      "abc123",
      "budi@example.com",
      "081234567890",
    ]) {
      assert.equal(output.includes(forbidden), false, `${forbidden} masih ada`);
    }
  });

  it("menghapus bearer token", () => {
    const output = redactSensitiveText("Authorization: Bearer abc.def.ghi");
    assert.equal(output.includes("abc.def.ghi"), false);
    assert.match(output, /\[redacted\]/);
  });

  it("tidak mengubah pesan biasa", () => {
    const output = redactSensitiveText("gagal memuat produk");
    assert.equal(output, "gagal memuat produk");
  });
});

describe("cleanRoute", () => {
  it("membuang query dan fragment", () => {
    assert.equal(
      cleanRoute("/members/123?email=budi@example.com#detail"),
      "/members/123"
    );
  });
});

describe("errorMessage", () => {
  it("tidak men-stringify objek sembarang", () => {
    assert.equal(errorMessage({ customer: "Budi", phone: "0812" }), "");
  });
});

describe("shouldIgnoreError", () => {
  it("mengabaikan error 4xx", () => {
    assert.equal(
      shouldIgnoreError({ status: 403, message: "forbidden" }),
      true
    );
  });

  it("mengabaikan redirect internal Next.js", () => {
    assert.equal(shouldIgnoreError({ digest: "NEXT_REDIRECT;push;/" }), true);
  });

  it("melacak error server biasa", () => {
    assert.equal(shouldIgnoreError(new Error("koneksi database gagal")), false);
  });
});
