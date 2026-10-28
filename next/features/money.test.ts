import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  formatIDR,
  formatMoneyInput,
  formatMoneyNumber,
  parseMoney,
  roundMoney,
} from "../lib/money/index.ts";

describe("formatIDR", () => {
  it("menampilkan Rp dan pemisah ribuan", () => {
    assert.equal(formatIDR(1234567), "Rp 1.234.567");
  });

  it("menampilkan desimal otomatis 0-2 digit", () => {
    assert.equal(formatIDR(1234), "Rp 1.234");
    assert.equal(formatIDR(1234.5), "Rp 1.234,5");
    assert.equal(formatIDR(1234.56), "Rp 1.234,56");
    assert.equal(formatIDR(1234.567), "Rp 1.234,57");
  });

  it("tetap menampilkan Rp untuk nol dan nominal negatif", () => {
    assert.equal(formatIDR(0), "Rp 0");
    assert.equal(formatIDR(-1500), "-Rp 1.500");
  });

  it("menganggap nilai tidak valid sebagai nol", () => {
    assert.equal(formatIDR(Number.NaN), "Rp 0");
    assert.equal(formatIDR(Number.POSITIVE_INFINITY), "Rp 0");
  });
});

describe("formatMoneyNumber", () => {
  it("memberi pemisah ribuan tanpa prefiks Rp", () => {
    assert.equal(formatMoneyNumber(1234567), "1.234.567");
    assert.equal(formatMoneyNumber(1234.5), "1.234,5");
  });
});

describe("formatMoneyInput", () => {
  it("memformat angka untuk kolom input", () => {
    assert.equal(formatMoneyInput(1234567), "1.234.567");
    assert.equal(formatMoneyInput(1234.5), "1.234,5");
    assert.equal(formatMoneyInput(0), "0");
  });
});

describe("parseMoney", () => {
  it("membaca input dengan prefiks Rp dan pemisah ribuan", () => {
    assert.equal(parseMoney("Rp 1.234.567"), 1234567);
  });

  it("membaca desimal koma maupun titik", () => {
    assert.equal(parseMoney("1.234,56"), 1234.56);
    assert.equal(parseMoney("1234.56"), 1234.56);
    assert.equal(parseMoney("1.234"), 1234);
  });

  it("mengembalikan nol untuk input kosong atau tidak valid", () => {
    assert.equal(parseMoney(""), 0);
    assert.equal(parseMoney("abc"), 0);
  });

  it("mendukung nominal negatif", () => {
    assert.equal(parseMoney("-Rp 1.500"), -1500);
  });

  it("menerima angka langsung", () => {
    assert.equal(parseMoney(1234.567), 1234.57);
  });
});

describe("roundMoney", () => {
  it("membulatkan ke sen terdekat", () => {
    assert.equal(roundMoney(1.005), 1.01);
    assert.equal(roundMoney(1.004), 1);
    assert.equal(roundMoney(-1.005), -1.01);
    assert.equal(roundMoney(2.675), 2.68);
  });
});
