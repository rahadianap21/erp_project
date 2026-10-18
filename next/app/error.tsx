"use client";

import { useEffect } from "react";
import { ForbiddenView } from "@/components/forbidden-view";
import { Button } from "@/components/ui/button";

export default function ErrorBoundary({
  error,
  reset,
}: {
  error: Error & { digest?: string; status?: number };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("ErrorBoundary menangkap error:", error);
  }, [error]);

  // JAUH LEBIH AMAN: Langsung cek status HTTP 403 atau jika teks mengandung kata kunci otorisasi
  const isForbidden =
    error.status === 403 ||
    error.message?.includes("403") ||
    error.message?.toLowerCase().includes("forbidden") ||
    error.message?.toLowerCase().includes("unauthorized") ||
    error.message?.toLowerCase().includes("izin");

  if (isForbidden) {
    return <ForbiddenView reset={reset} />;
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4 text-center">
      <h2 className="text-2xl font-bold mb-4">Terjadi Kesalahan Sistem</h2>
      <p className="text-muted-foreground mb-6 text-sm max-w-sm">
        {error.message || "Gagal memuat data. Silakan coba beberapa saat lagi."}
      </p>
      <Button onClick={() => reset()}>Coba Muat Ulang</Button>
    </div>
  );
}
