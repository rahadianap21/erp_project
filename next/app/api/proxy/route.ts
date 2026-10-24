import { API_BASE_URL } from "@/lib/config";
import { resolveApiTarget } from "@/lib/api-target";
import { trustedForwardedFor } from "@/lib/client-ip";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";

const METHODS_WITH_BODY = new Set(["POST", "PUT", "PATCH"]);

async function forward(request: Request, method: string) {
  const rawTarget = new URL(request.url).searchParams.get("target") ?? "";
  const target = resolveApiTarget(rawTarget, API_BASE_URL);
  if (!target) {
    return NextResponse.json(
      { error: "Target API tidak valid" },
      { status: 400 }
    );
  }

  const token = (await cookies()).get("auth_token")?.value;
  if (!token) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const forwardedFor = trustedForwardedFor(request.headers);
  const idempotencyKey = request.headers.get("Idempotency-Key");
  const hasBody = METHODS_WITH_BODY.has(method);
  let res: Response;
  try {
    res = await fetch(target, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        ...(forwardedFor ? { "X-Forwarded-For": forwardedFor } : {}),
        ...(idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}),
        ...(hasBody ? { "Content-Type": "application/json" } : {}),
      },
      body: hasBody ? await request.text() : undefined,
      cache: "no-store",
      redirect: "manual",
      signal: AbortSignal.timeout(15_000),
    });
  } catch {
    return NextResponse.json(
      { error: "Tidak dapat menghubungi server API" },
      { status: 502 }
    );
  }

  // Jangan meneruskan redirect: respons 3xx dari API bisa mengarahkan request
  // (beserta header Authorization) ke host lain.
  if (res.status >= 300 && res.status < 400) {
    return NextResponse.json(
      { error: "API mengembalikan redirect yang tidak diizinkan" },
      { status: 502 }
    );
  }

  if (res.status === 204) {
    return new NextResponse(null, { status: 204 });
  }

  const body = await res.text();
  const contentType = res.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      return NextResponse.json(JSON.parse(body), { status: res.status });
    } catch {
      // fall through ke respons teks di bawah
    }
  }

  return new NextResponse(body, {
    status: res.status,
    headers: contentType ? { "Content-Type": contentType } : undefined,
  });
}

export async function GET(request: Request) {
  return forward(request, "GET");
}

export async function POST(request: Request) {
  return forward(request, "POST");
}

export async function PUT(request: Request) {
  return forward(request, "PUT");
}

export async function PATCH(request: Request) {
  return forward(request, "PATCH");
}

export async function DELETE(request: Request) {
  return forward(request, "DELETE");
}
