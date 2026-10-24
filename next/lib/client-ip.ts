// Hanya percayai X-Forwarded-For/X-Real-IP bila operator menyatakan Next.js
// berada di balik reverse proxy tepercaya (TRUST_PROXY=1). Tanpa flag itu,
// header bisa dipalsukan klien dan membuat rate limit per IP mudah dilewati.
export function trustedForwardedFor(headers: Headers): string | null {
  if (process.env.TRUST_PROXY !== "1") return null;

  const forwarded = headers.get("x-forwarded-for");
  if (forwarded) {
    // Teruskan seluruh rantai apa adanya; Go memilih elemen paling kanan yang
    // bukan proxy tepercaya menggunakan TRUSTED_PROXIES.
    const parts = forwarded
      .split(",")
      .map((part) => part.trim())
      .filter(Boolean);
    return parts.length > 0 ? parts.join(", ") : null;
  }

  return headers.get("x-real-ip")?.trim() || null;
}
