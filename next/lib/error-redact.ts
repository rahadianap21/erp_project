// Helper pembersihan data untuk pelaporan error. File ini murni (tanpa akses
// browser/Node) supaya bisa dipakai di client, route handler, dan instrumentation.
//
// Pelapor error tidak pernah mengirim request body, query string, cookie, atau
// header. Fungsi di sini adalah lapisan tambahan untuk membersihkan pesan/stack
// yang bisa saja mengandung email, telepon, token, atau data pelanggan lain.

const AUTH_HEADER =
  /\bauthorization\s*[:=]\s*(?:bearer\s+)?[A-Za-z0-9._~+/=-]+/gi;
const BEARER = /\bbearer\s+[A-Za-z0-9._~+/=-]+/gi;
const SENSITIVE_KV =
  /("?(?:password|passwd|pwd|secret|token|authorization|api[_-]?key|apikey|access[_-]?token|refresh[_-]?token|cookie|email|e-mail|phone|telepon|no[_-]?hp|alamat|address|nama|name|username|full[_-]?name|nik|npwp|no[_-]?rekening|rekening|account[_-]?number|card[_-]?number|customer|member)"?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;}&]+)/gi;
const EMAIL = /\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi;
const PHONE = /(\+?62|0)8[0-9]{7,12}/g;
const LONG_DIGITS = /\b[0-9]{13,19}\b/g;

export const MAX_MESSAGE_CHARS = 2000;
export const MAX_STACK_CHARS = 16000;
export const MAX_ROUTE_CHARS = 512;

export function truncate(value: string, max: number): string {
  if (max <= 0) return "";
  const chars = Array.from(value);
  if (chars.length <= max) return value;
  return chars.slice(0, max).join("");
}

export function redactSensitiveText(
  value: string,
  max = MAX_MESSAGE_CHARS
): string {
  if (!value) return "";
  const redacted = value
    .replace(AUTH_HEADER, "authorization: [redacted]")
    .replace(BEARER, "Bearer [redacted]")
    .replace(SENSITIVE_KV, "$1[redacted]")
    .replace(EMAIL, "[email]")
    .replace(PHONE, "[phone]")
    .replace(LONG_DIGITS, "[number]");
  return truncate(redacted.trim(), max);
}

export function cleanRoute(value: string): string {
  if (!value) return "";
  const pathOnly = value.split(/[?#]/, 1)[0] ?? "";
  return truncate(
    redactSensitiveText(pathOnly, MAX_ROUTE_CHARS),
    MAX_ROUTE_CHARS
  );
}

/** Ambil pesan hanya dari Error/string; objek sembarang tidak di-stringify. */
export function errorMessage(error: unknown): string {
  if (typeof error === "string") return error;
  if (error instanceof Error) return error.message;
  if (
    error &&
    typeof error === "object" &&
    "message" in error &&
    typeof (error as { message?: unknown }).message === "string"
  ) {
    return (error as { message: string }).message;
  }
  return "";
}

export function errorName(error: unknown): string {
  if (error instanceof Error && error.name) return error.name;
  return "Error";
}

export function errorStack(error: unknown): string {
  return error instanceof Error && typeof error.stack === "string"
    ? error.stack
    : "";
}

export function errorDigest(error: unknown): string {
  if (
    error &&
    typeof error === "object" &&
    "digest" in error &&
    typeof (error as { digest?: unknown }).digest === "string"
  ) {
    return (error as { digest: string }).digest;
  }
  return "";
}

/** Error 4xx adalah penolakan yang diharapkan, bukan kegagalan sistem. */
export function isExpectedClientError(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  const status = (error as { status?: unknown }).status;
  return typeof status === "number" && status >= 400 && status < 500;
}

/** Error internal Next.js (redirect/not-found) bukan kegagalan aplikasi. */
export function isNextInternalError(error: unknown): boolean {
  const digest = errorDigest(error);
  const message = errorMessage(error);
  return (
    digest.startsWith("NEXT_REDIRECT") ||
    digest.startsWith("NEXT_NOT_FOUND") ||
    digest.startsWith("NEXT_HTTP_ERROR_FALLBACK") ||
    message.includes("NEXT_REDIRECT") ||
    message.includes("NEXT_NOT_FOUND")
  );
}

export function shouldIgnoreError(error: unknown): boolean {
  if (isExpectedClientError(error) || isNextInternalError(error)) return true;

  const text = `${errorName(error)} ${errorMessage(error)} ${errorStack(error)}`;
  return (
    /AbortError/i.test(text) ||
    /ResizeObserver loop/i.test(text) ||
    /chrome-extension:\/\//i.test(text) ||
    /moz-extension:\/\//i.test(text)
  );
}
