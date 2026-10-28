"use client";

import Cookies from "js-cookie";

export async function logout() {
  Cookies.remove("session_active", { path: "/" });
  Cookies.remove("user_email", { path: "/" });
  Cookies.remove("user_name", { path: "/" });
  Cookies.remove("user_permissions", { path: "/" });
  Cookies.remove("user_app", { path: "/" });

  try {
    // Cookie auth_token bersifat httpOnly, jadi hanya route handler inilah
    // yang bisa menghapusnya sekaligus meneruskan logout ke backend Go.
    await fetch("/api/auth/logout", { method: "POST" });
  } catch (err) {
    console.error("Gagal logout:", err);
  }

  window.location.replace("/");
}
