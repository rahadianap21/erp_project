"use client";

import { useCallback, useEffect, useState } from "react";
import Cookies from "js-cookie";
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
        const savedPermissions = Cookies.get("user_permissions");
        const currentToken = Cookies.get("session_active");

        if (!savedPermissions || !currentToken) {
            Cookies.remove("session_active");
            Cookies.remove("user_permissions");
            toast.error("Sesi Anda Telah Berakhir", {
                description: "Silakan login ulang untuk melihat laporan keuangan.",
                duration: 4000,
            });
            window.location.href = "/";
            return;
        }

        setToken(currentToken);
        try {
            const parsed: string[] = JSON.parse(decodeURIComponent(savedPermissions));
            if (Array.isArray(parsed)) {
                setCanRead(parsed.includes("accounting:read"));
            }
        } catch (e) {
            console.error("Gagal membaca cookie permissions akuntansi:", e);
        }
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
