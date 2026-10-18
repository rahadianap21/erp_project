"use client";

import Cookies from "js-cookie";
import { toast } from "sonner";

// Pusat pembacaan izin di sisi client. proxy.ts tetap gerbang utama (menolak
// request tanpa izin sebelum halaman render); helper ini dipakai komponen
// client untuk menampilkan/menyembunyikan kontrol aksi (tombol create/edit/
// delete) dan menangani sesi yang kedaluwarsa secara seragam.

const PERMISSIONS_COOKIE = "user_permissions";
const SESSION_COOKIE = "session_active";

/** Baca daftar izin dari kuki `user_permissions`. null bila hilang/tidak valid. */
export function readPermissionList(): string[] | null {
  const raw = Cookies.get(PERMISSIONS_COOKIE);
  if (!raw) return null;
  try {
    const parsed: unknown = JSON.parse(decodeURIComponent(raw));
    return Array.isArray(parsed) ? (parsed as string[]) : null;
  } catch {
    return null;
  }
}

/** true bila daftar izin memuat `permission` persis (mis. "users:create"). */
export function hasPermission(permission: string): boolean {
  return readPermissionList()?.includes(permission) ?? false;
}

export interface PermissionFlags {
  create: boolean;
  update: boolean;
  delete: boolean;
}

/**
 * readPermissionFlags mengembalikan trio create/update/delete untuk satu
 * prefiks fitur (mis. "suppliers" -> suppliers:create|update|delete).
 */
export function readPermissionFlags(prefix: string): PermissionFlags {
  const list = readPermissionList() ?? [];
  return {
    create: list.includes(`${prefix}:create`),
    update: list.includes(`${prefix}:update`),
    delete: list.includes(`${prefix}:delete`),
  };
}

/**
 * ensureSession adalah guard sesi komponen client: kalau kuki otorisasi
 * hilang (mis. kena kedaluwarsa saat idle), bersihkan sesi, tampilkan toast,
 * dan tendang ke halaman login. false berarti caller harus berhenti.
 */
export function ensureSession(): boolean {
  if (Cookies.get(PERMISSIONS_COOKIE) && Cookies.get(SESSION_COOKIE)) {
    return true;
  }
  Cookies.remove(SESSION_COOKIE, { path: "/" });
  Cookies.remove(PERMISSIONS_COOKIE, { path: "/" });
  toast.error("Sesi Anda Telah Berakhir", {
    description: "Silakan lakukan login ulang untuk mengamankan sesi Anda.",
    duration: 4000,
  });
  window.location.replace("/");
  return false;
}
