// Pusat format dan parsing nominal rupiah untuk seluruh aplikasi.
//
// Backend mengirim uang sebagai angka rupiah biasa (bukan string), sehingga
// modul ini sengaja tidak bergantung pada API atau tipe domain. Semua tampilan
// dan input nominal wajib memakai helper di sini agar pemisah ribuan, jumlah
// desimal, dan tanda "Rp" konsisten di seluruh halaman.

const numberFormatter = new Intl.NumberFormat("id-ID", {
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
});

/**
 * Membulatkan nominal ke sen terdekat (setengah menjauhi nol), sama seperti
 * pembulatan `domain.Money` di backend.
 */
export function roundMoney(value: number): number {
  if (!Number.isFinite(value)) return 0;
  const sign = value < 0 ? -1 : 1;
  // toFixed(6) menstabilkan galat floating point seperti 1.005 * 100.
  const cents = Math.round(Number((Math.abs(value) * 100).toFixed(6)));
  return (sign * cents) / 100;
}

/**
 * Mengubah nominal menjadi angka berformat Indonesia tanpa prefiks "Rp".
 * Desimal ditampilkan otomatis 0–2 digit: bulat tanpa desimal, pecahan
 * ditampilkan seperlunya (mis. 1.234,5 atau 1.234,56).
 */
export function formatMoneyNumber(value: number): string {
  return numberFormatter.format(roundMoney(value));
}

/**
 * Format nominal rupiah lengkap dengan prefiks "Rp". Selalu menampilkan "Rp",
 * termasuk untuk nol dan nominal negatif.
 */
export function formatIDR(value: number): string {
  const nominal = roundMoney(value);
  const body = numberFormatter.format(Math.abs(nominal));
  return nominal < 0 ? `-Rp ${body}` : `Rp ${body}`;
}

/**
 * Format nilai untuk kolom input uang: pemisah ribuan titik, desimal koma,
 * tanpa prefiks "Rp" (prefiks ditangani komponen input).
 */
export function formatMoneyInput(value: number): string {
  const nominal = roundMoney(value);
  const abs = Math.abs(nominal);
  const [integerPart, decimalPart] = abs
    .toFixed(2)
    .replace(/0+$/, "")
    .replace(/\.$/, "")
    .split(".");
  const grouped = integerPart.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  const body = decimalPart ? `${grouped},${decimalPart}` : grouped;
  return nominal < 0 ? `-${body}` : body;
}

/**
 * Membaca input nominal dari pengguna. Menerima "Rp", spasi, pemisah ribuan
 * titik, dan desimal koma. Nilai yang tidak valid dianggap nol supaya field
 * uang tidak pernah menghasilkan NaN.
 */
export function parseMoney(input: string | number): number {
  if (typeof input === "number") {
    return roundMoney(input);
  }

  const trimmed = input.trim();
  if (!trimmed) return 0;

  const negative = trimmed.startsWith("-") || /^\(.*\)$/.test(trimmed);
  const cleaned = trimmed.replace(/[^\d,.-]/g, "").replace(/-/g, "");

  let normalized: string;
  if (cleaned.includes(",")) {
    // Format Indonesia: titik adalah pemisah ribuan, koma adalah desimal.
    normalized = cleaned.replace(/\./g, "").replace(/,/g, ".");
  } else {
    const dots = cleaned.split(".").length - 1;
    const lastGroup = cleaned.slice(cleaned.lastIndexOf(".") + 1);
    // Satu titik dengan panjang grup bukan 3 digit dianggap desimal
    // ("1234.56"); selain itu titik diperlakukan sebagai pemisah ribuan
    // ("1.234" atau "1.234.567").
    normalized =
      dots === 1 && lastGroup.length !== 3
        ? cleaned
        : cleaned.replace(/\./g, "");
  }

  const parsed = Number(normalized);
  if (!Number.isFinite(parsed)) return 0;

  const nominal = roundMoney(parsed);
  return negative ? -Math.abs(nominal) : nominal;
}
