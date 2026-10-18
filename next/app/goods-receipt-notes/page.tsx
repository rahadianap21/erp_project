"use client";

import { ensureSession, readPermissionList } from "@/lib/permissions";
import React, { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { PackageCheck } from "lucide-react";
import { ForbiddenView } from "@/components/forbidden-view";

const GoodsReceiptNoteTable = dynamic(
  () =>
    import("./components/goods-receipt-note-table").then(
      (mod) => mod.GoodsReceiptNoteTable
    ),
  {
    ssr: false,
    loading: () => (
      <div className="w-full min-h-[30vh] flex items-center justify-center text-slate-500 text-xs font-medium">
        <span className="animate-pulse">
          Menyiapkan riwayat penerimaan barang (GRN)...
        </span>
      </div>
    ),
  }
);

/**
 * Halaman riwayat penerimaan barang.
 *
 * GRN immutable, jadi halaman ini read-only: tidak ada tombol ubah atau hapus.
 * Satu PO bisa punya beberapa GRN karena barang bisa diterima bertahap.
 */
export default function GoodsReceiptNotesPage() {
  const [httpStatus, setHttpStatus] = useState<number>(200);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const loadPermissions = () => {
    if (!ensureSession()) return;
    const list = readPermissionList() ?? [];
    // Diperbarui lewat efek, jadi render pertama server dan client
    // sama-sama memakai status awal 200.
    setHttpStatus(list.includes("grn:read") ? 200 : 403);
  };

  useEffect(() => {
    loadPermissions();
  }, []);

  const handleResetForbidden = () => {
    setRefreshTrigger((prev) => prev + 1);
    loadPermissions();
  };

  // Hanya httpStatus yang boleh Stateless-render di server dan client. Guard
  // berbasis hasRead/typeof window akan membuat server merender null sementara
  // client merender ForbiddenView pada render hydration pertama, dan itu
  // selalu menghasilkan hydration mismatch.
  if (httpStatus === 403) {
    return <ForbiddenView reset={handleResetForbidden} />;
  }

  return (
    <main className="flex flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8 w-full">
      <div className="flex items-center gap-2 border-b pb-4">
        <div className="p-2 bg-emerald-50 text-emerald-600 rounded-xl border border-emerald-100">
          <PackageCheck className="size-5" />
        </div>
        <div>
          <h1 className="text-sm font-black uppercase tracking-wider text-slate-900">
            Riwayat Penerimaan Barang (GRN)
          </h1>
          <p className="text-xs text-slate-500">
            Dokumen batch penerimaan barang dari PO yang sudah disetujui.
            Dokumen ini tidak bisa diubah atau dihapus.
          </p>
        </div>
      </div>

      <GoodsReceiptNoteTable
        refreshTrigger={refreshTrigger}
        onForbidden={() => setHttpStatus(403)}
      />
    </main>
  );
}
