// API client tunggal untuk halaman client.
//
// Semua request lewat /api/proxy (route handler server) sehingga token asli
// (auth_token, httpOnly) tidak pernah disentuh JavaScript browser. Route
// handler mengirim Authorization: Bearer <auth_token> ke backend dan
// membalas ulang body + status apa adanya.

export interface ApiEnvelope<T = unknown> {
  status: number
  data: T | null
  message: string | null
  ok: boolean
  /** Body JSON utuh — dipakai caller yang butuh field lain (mis. `total`). */
  body: unknown
}

export class ApiError extends Error {
  readonly status: number
  readonly body: unknown

  constructor(status: number, message: string, body?: unknown) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.body = body
  }
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  /** Body JSON; otomatis di-stringify. Jangan isi `body` mentah di sini. */
  json?: unknown
  /** Set false bila caller ingin menangani 403 sendiri (tanpa redirect). */
  redirectOn403?: boolean
}

async function parseBody(res: Response): Promise<unknown> {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

function messageFromBody(body: unknown, fallback: string): string {
  if (body && typeof body === "object" && "message" in body) {
    const m = (body as { message?: unknown }).message
    if (typeof m === "string" && m) return m
  }
  return fallback
}

function dataFromBody<T>(body: unknown): T | null {
  if (body && typeof body === "object" && "data" in body) {
    return (body as { data: T }).data
  }
  return (body as T) ?? null
}

/** Jalur inti semua method HTTP. */
export async function request<T = unknown>(
  path: string,
  options: RequestOptions = {},
): Promise<ApiEnvelope<T>> {
  const { json, redirectOn403 = true, headers, ...init } = options
  const res = await fetch(`/api/proxy?target=${encodeURIComponent(path)}`, {
    method: init.method ?? "GET",
    credentials: "include",
    cache: "no-store",
    ...init,
    headers: {
      ...(json !== undefined ? { "Content-Type": "application/json" } : {}),
      ...(headers ?? {}),
    },
    ...(json !== undefined ? { body: JSON.stringify(json) } : {}),
  })

  const body = await parseBody(res)

  // 403 dari API berarti izin tidak cukup untuk aksi/halaman ini.
  if (res.status === 403 && redirectOn403 && typeof window !== "undefined") {
    window.location.href = "/403"
  }

  if (!res.ok) {
    throw new ApiError(
      res.status,
      messageFromBody(body, `Permintaan gagal (HTTP ${res.status})`),
      body,
    )
  }

  return {
    status: res.status,
    ok: res.ok,
    data: dataFromBody<T>(body),
    message: messageFromBody(body, ""),
    body,
  }
}

export function apiGet<T = unknown>(path: string, options?: RequestOptions) {
  return request<T>(path, { ...options, method: "GET" })
}

export function apiPost<T = unknown>(
  path: string,
  json: unknown,
  options?: RequestOptions,
) {
  return request<T>(path, { ...options, method: "POST", json })
}

export function apiPut<T = unknown>(
  path: string,
  json: unknown,
  options?: RequestOptions,
) {
  return request<T>(path, { ...options, method: "PUT", json })
}

export function apiDelete<T = unknown>(path: string, options?: RequestOptions) {
  return request<T>(path, { ...options, method: "DELETE" })
}

/**
 * fetchLaporan adalah wrapper ringan untuk endpoint laporan akuntansi:
 * tidak melempar error, mengembalikan {status, data?, message?} agar view
 * bisa membedakan 403 (izin) dari kegagalan lain tanpa redirect otomatis.
 */
export async function fetchLaporan<T>(
  path: string,
): Promise<{ status: number; data?: T; message?: string }> {
  try {
    const result = await request<T>(path, { redirectOn403: false })
    return { status: result.status, data: result.data ?? undefined, message: result.message || undefined }
  } catch (err) {
    if (err instanceof ApiError) {
      return { status: err.status, message: err.message }
    }
    return { status: 0, message: "Tidak dapat menghubungi server." }
  }
}
