"use client";

import { useCallback, useEffect, useState } from "react";
import Cookies from "js-cookie";
import { ensureSession, hasPermission } from "@/lib/permissions";
import { toast } from "sonner";

// useReportAccess memusatkan pemeriksaan sesi dan izin accounting:read untuk
// seluruh halaman laporan keuangan. Pola ini meniru halaman akuntansi lain
// (mis. chart-of-accounts) supaya perilaku sesi kadaluarsa seragam.
export function useReportAccess() {
  const [ready, setReady] = useState<boolean>(false);
  const [canRead, setCanRead] = useState<boolean>(false);
  const [token, setToken] = useState<string | undefined>(undefined);
  const [httpStatus, setHttpStatus] = useState<number>(200);

  const loadPermissions = useCallback(() => {
    if (!ensureSession()) return;

    setToken(Cookies.get("session_active"));
    setCanRead(hasPermission("accounting:read"));
    setReady(true);
  }, []);

  useEffect(() => {
    loadPermissions();
    window.addEventListener("focus", loadPermissions);
    return () => window.removeEventListener("focus", loadPermissions);
  }, [loadPermissions]);

  const reset = useCallback(() => {
    setHttpStatus(200);
    loadPermissions();
  }, [loadPermissions]);

  return { ready, canRead, token, httpStatus, setHttpStatus, reset };
}
