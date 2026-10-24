import { API_BASE_URL } from "@/lib/config";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";

// Token sesi kini disimpan di cookie httpOnly, sehingga JavaScript di browser
// tidak bisa menghapusnya. Route handler inilah satu-satunya jalur logout yang
// benar: hapus cookie httpOnly lalu beri tahu backend Go.
export async function POST() {
  const cookieStore = await cookies();
  const token = cookieStore.get("auth_token")?.value;

  if (token) {
    try {
      await fetch(`${API_BASE_URL}/api/logout`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        cache: "no-store",
      });
    } catch {
      // Backend tidak tersedia bukan alasan menahan logout di sisi klien.
    }
  }

  const response = NextResponse.json({ message: "Logout berhasil" });
  response.cookies.delete("auth_token");
  response.cookies.delete("session_active");
  response.cookies.delete("user_permissions");
  response.cookies.delete("user_app");
  response.cookies.delete("user_email");
  response.cookies.delete("user_name");
  return response;
}
