// Validasi target proxy. Nilai `target` berasal dari query string yang bisa
// dikendalikan klien, jadi jangan pernah menempelkannya ke API_BASE_URL secara
// langsung: string seperti "@evil.example/api" akan mengubah host tujuan dan
// membocorkan header Authorization (JWT httpOnly) ke pihak lain.
export function resolveApiTarget(
  rawTarget: string,
  apiBaseUrl: string
): URL | null {
  if (!rawTarget.startsWith("/api/")) {
    return null;
  }

  let base: URL;
  try {
    base = new URL(apiBaseUrl);
  } catch {
    return null;
  }

  let target: URL;
  try {
    target = new URL(rawTarget, base);
  } catch {
    return null;
  }

  // Tolak userinfo, origin berbeda, dan path yang lolos validasi awal tetapi
  // berubah setelah normalisasi URL (mis. "/api/../admin").
  if (target.username || target.password) return null;
  if (target.origin !== base.origin) return null;

  let decodedPath: string;
  try {
    decodedPath = decodeURIComponent(target.pathname);
  } catch {
    return null;
  }
  if (!decodedPath.startsWith("/api/")) return null;
  if (decodedPath.split("/").some((s) => s === "..")) return null;

  return target;
}
