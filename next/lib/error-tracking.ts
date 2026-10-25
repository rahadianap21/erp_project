import {
  MAX_MESSAGE_CHARS,
  MAX_ROUTE_CHARS,
  MAX_STACK_CHARS,
  cleanRoute,
  errorMessage,
  errorName,
  errorStack,
  redactSensitiveText,
  shouldIgnoreError,
  truncate,
} from "./error-redact";

export interface ErrorReport {
  name: string;
  level: string;
  message: string;
  stack: string;
  componentStack: string;
  route: string;
  source: "frontend";
  requestId?: string;
  occurredAt: string;
}

export interface ReportContext {
  level?: string;
  route?: string;
  componentStack?: string;
}

const DEDUPE_WINDOW_MS = 30_000;
const MAX_DEDUPE_ENTRIES = 100;
const recentlyReported = new Map<string, number>();

function makeRequestId(): string | undefined {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return undefined;
}

/**
 * Membersihkan error menjadi payload minimal. Tidak ada request body, query
 * string, cookie, header, maupun objek error mentah yang ikut dikirim.
 */
export function buildErrorReport(
  error: unknown,
  context: ReportContext = {}
): ErrorReport | null {
  const message = redactSensitiveText(errorMessage(error), MAX_MESSAGE_CHARS);
  const stack = redactSensitiveText(errorStack(error), MAX_STACK_CHARS);
  if (!message && !stack) return null;

  const route =
    context.route ??
    (typeof window === "undefined" ? "" : window.location.pathname);

  return {
    name: truncate(redactSensitiveText(errorName(error), 120), 120),
    level: (context.level ?? "error").toLowerCase(),
    message,
    stack,
    componentStack: redactSensitiveText(context.componentStack ?? "", 4000),
    route: truncate(cleanRoute(route), MAX_ROUTE_CHARS),
    source: "frontend",
    requestId: makeRequestId(),
    occurredAt: new Date().toISOString(),
  };
}

function dedupeKey(report: ErrorReport): string {
  return [
    report.name,
    report.message,
    report.stack.slice(0, 200),
    report.route,
  ].join("|");
}

function isDuplicate(key: string): boolean {
  const now = Date.now();
  const last = recentlyReported.get(key);
  if (last !== undefined && now - last < DEDUPE_WINDOW_MS) return true;

  recentlyReported.delete(key);
  recentlyReported.set(key, now);

  if (recentlyReported.size > MAX_DEDUPE_ENTRIES) {
    const oldest = recentlyReported.keys().next().value;
    if (oldest !== undefined) recentlyReported.delete(oldest);
  }
  return false;
}

/**
 * Melaporkan error dari browser ke route handler /api/errors. Fire-and-forget:
 * kegagalan pelaporan tidak boleh mengganggu aplikasi atau memicu loop error.
 */
export function reportClientError(
  error: unknown,
  context: ReportContext = {}
): void {
  if (typeof window === "undefined" || shouldIgnoreError(error)) return;

  const report = buildErrorReport(error, context);
  if (!report) return;

  if (isDuplicate(dedupeKey(report))) return;

  void fetch("/api/errors", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(report),
    keepalive: true,
  }).catch(() => {
    // Sengaja diam: pelaporan error tidak boleh membuat error baru.
  });
}

/** Memasang listener global; mengembalikan fungsi pembersih untuk useEffect. */
export function installGlobalErrorHandlers(): () => void {
  if (typeof window === "undefined") return () => {};

  const onError = (event: ErrorEvent) => {
    if (event.error) {
      reportClientError(event.error);
      return;
    }
    if (event.message) {
      reportClientError(event.message);
    }
  };

  const onUnhandledRejection = (event: PromiseRejectionEvent) => {
    reportClientError(event.reason);
  };

  window.addEventListener("error", onError);
  window.addEventListener("unhandledrejection", onUnhandledRejection);
  return () => {
    window.removeEventListener("error", onError);
    window.removeEventListener("unhandledrejection", onUnhandledRejection);
  };
}
