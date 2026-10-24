/**
 * Membuat kunci idempotensi untuk satu percobaan mutasi.
 *
 * Kunci harus dipertahankan selama percobaan yang sama masih bisa di-retry,
 * lalu dibuat ulang setelah sukses atau ketika payload berubah. Backend
 * menolak key yang sama bila request hash-nya berbeda.
 */
export function createIdempotencyKey(): string {
  const cryptoApi = globalThis.crypto;
  if (cryptoApi && typeof cryptoApi.randomUUID === "function") {
    return cryptoApi.randomUUID();
  }
  if (cryptoApi && typeof cryptoApi.getRandomValues === "function") {
    const bytes = new Uint8Array(16);
    cryptoApi.getRandomValues(bytes);
    return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}${Math.random()
    .toString(36)
    .slice(2)}`;
}
