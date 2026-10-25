import { NextResponse } from "next/server";

import { trustedForwardedFor } from "@/lib/client-ip";
import { sendErrorReport } from "@/lib/error-tracking.server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const MAX_PAYLOAD_BYTES = 64 * 1024;

function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

export async function POST(request: Request) {
  // Tolak laporan lintas-origin dari browser. Ini bukan pengganti rate limit,
  // tetapi mencegah situs lain membanjiri endpoint ini lewat pengunjungnya.
  const origin = request.headers.get("origin");
  if (origin && origin !== new URL(request.url).origin) {
    return NextResponse.json(
      { error: "Origin tidak diizinkan" },
      { status: 403 }
    );
  }

  const contentLength = Number(request.headers.get("content-length") ?? "0");
  if (Number.isFinite(contentLength) && contentLength > MAX_PAYLOAD_BYTES) {
    return NextResponse.json(
      { error: "Payload terlalu besar" },
      { status: 413 }
    );
  }

  const text = await request.text();
  if (new TextEncoder().encode(text).byteLength > MAX_PAYLOAD_BYTES) {
    return NextResponse.json(
      { error: "Payload terlalu besar" },
      { status: 413 }
    );
  }

  let raw: unknown;
  try {
    raw = JSON.parse(text);
  } catch {
    return NextResponse.json({ error: "Payload tidak valid" }, { status: 400 });
  }

  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return NextResponse.json({ error: "Payload tidak valid" }, { status: 400 });
  }

  const payload = raw as Record<string, unknown>;
  const message = asString(payload.message);
  const stack = asString(payload.stack);
  if (!message && !stack) {
    return NextResponse.json(
      { error: "Pesan atau stack wajib diisi" },
      { status: 400 }
    );
  }

  // Fire-and-forget: kegagalan backend tidak boleh memicu retry loop di browser.
  await sendErrorReport(
    {
      name: asString(payload.name),
      level: asString(payload.level),
      message,
      stack,
      componentStack: asString(payload.componentStack),
      route: asString(payload.route),
      requestId: asString(payload.requestId),
      occurredAt: asString(payload.occurredAt),
    },
    trustedForwardedFor(request.headers)
  );

  return NextResponse.json({ status: "accepted" }, { status: 202 });
}
