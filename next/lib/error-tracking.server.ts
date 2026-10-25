import { API_BASE_URL } from "./config";
import {
  cleanRoute,
  errorMessage,
  errorName,
  errorStack,
  redactSensitiveText,
  shouldIgnoreError,
  truncate,
} from "./error-redact";

export interface ErrorReportPayload {
  name?: string;
  level?: string;
  message?: string;
  stack?: string;
  componentStack?: string;
  route?: string;
  requestId?: string;
  occurredAt?: string;
}

export interface ServerErrorContext {
  path?: string;
  method?: string;
  routePath?: string;
  routeType?: string;
  forwardedFor?: string | null;
}

const MAX_PAYLOAD_BYTES = 64 * 1024;

/**
 * Mengirim laporan yang sudah dibersihkan langsung ke backend. Dipakai oleh
 * route /api/errors (laporan browser) dan instrumentation (error server Next).
 * Tidak pernah meneruskan header, cookie, body, atau query string.
 */
export async function sendErrorReport(
  payload: ErrorReportPayload,
  forwardedFor?: string | null
): Promise<boolean> {
  const message = redactSensitiveText(payload.message ?? "", 2000);
  const stack = redactSensitiveText(payload.stack ?? "", 16000);
  if (!message && !stack) return false;

  const body = JSON.stringify({
    name: truncate(redactSensitiveText(payload.name ?? "Error", 120), 120),
    level: truncate(payload.level ?? "error", 20),
    message,
    stack,
    componentStack: redactSensitiveText(payload.componentStack ?? "", 4000),
    route: cleanRoute(payload.route ?? ""),
    source: "frontend",
    requestId: payload.requestId
      ? truncate(redactSensitiveText(payload.requestId, 64), 64)
      : undefined,
    occurredAt: payload.occurredAt,
  });

  if (new TextEncoder().encode(body).byteLength > MAX_PAYLOAD_BYTES) {
    return false;
  }

  const ingestKey = process.env.ERROR_INGEST_KEY?.trim();
  const url = `${API_BASE_URL.replace(/\/+$/, "")}/api/errors`;

  try {
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(ingestKey ? { "X-Error-Ingest-Key": ingestKey } : {}),
        ...(forwardedFor ? { "X-Forwarded-For": forwardedFor } : {}),
      },
      body,
      cache: "no-store",
      signal: AbortSignal.timeout(5_000),
    });
    return res.ok;
  } catch {
    return false;
  }
}

/** Melaporkan error yang terjadi di server Next.js (SSR/route handler/action). */
export async function reportServerError(
  error: unknown,
  context: ServerErrorContext = {}
): Promise<void> {
  if (shouldIgnoreError(error)) return;

  const route = cleanRoute(context.path ?? context.routePath ?? "");
  // Jangan laporkan kegagalan di jalur ingest itu sendiri agar tidak menjadi
  // loop pelaporan.
  if (route.startsWith("/api/errors")) return;

  const message = errorMessage(error);
  const stack = errorStack(error);
  if (!message && !stack) return;

  await sendErrorReport(
    {
      name: errorName(error),
      level: "server",
      message,
      stack,
      route,
      occurredAt: new Date().toISOString(),
    },
    context.forwardedFor
  );
}
