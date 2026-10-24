import { cache } from "react";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { API_BASE_URL } from "@/lib/config";

export interface SessionData {
  userId: number;
  name: string;
  email: string;
  app: string;
  roleId: number;
  permissions: string[];
}

interface ProfileResponse {
  user_id?: unknown;
  name?: unknown;
  email?: unknown;
  app?: unknown;
  role_id?: unknown;
  permissions?: unknown;
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === "string")
    : [];
}

// Satu-satunya sumber kebenaran untuk sesi halaman. Permission tidak lagi
// dibaca dari cookie yang bisa diubah JavaScript; nilainya datang dari
// /api/me yang memvalidasi JWT dan membaca permission aktual dari database.
export const getSession = cache(async (): Promise<SessionData> => {
  const token = (await cookies()).get("auth_token")?.value;
  if (!token) redirect("/");

  let res: Response | null = null;
  try {
    res = await fetch(`${API_BASE_URL}/api/me`, {
      method: "GET",
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
      signal: AbortSignal.timeout(10_000),
    });
  } catch {
    redirect("/");
  }

  if (!res || res.status === 401) redirect("/");
  if (res.status === 403) redirect("/403");
  if (!res.ok) redirect("/");

  const body = (await res.json().catch(() => null)) as ProfileResponse | null;
  if (!body || typeof body.user_id !== "number") redirect("/");

  return {
    userId: body.user_id,
    name: typeof body.name === "string" ? body.name : "",
    email: typeof body.email === "string" ? body.email : "",
    app: typeof body.app === "string" ? body.app : "erp",
    roleId: typeof body.role_id === "number" ? body.role_id : 0,
    permissions: stringArray(body.permissions),
  };
});

export async function requirePermission(
  permission: string
): Promise<SessionData> {
  const session = await getSession();
  if (!session.permissions.includes(permission)) redirect("/403");
  return session;
}
