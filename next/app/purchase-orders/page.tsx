"use client";

import {
  ApiError,
  apiDelete,
  apiGet,
  apiPut,
  messageFromError,
} from "@/lib/api-client";
import { createIdempotencyKey } from "@/lib/idempotency";
import { ensureSession, readPermissionList } from "@/lib/permissions";
import React, { useState, useEffect } from "react";
import Cookies from "js-cookie";
import { toast } from "sonner";
import { ShoppingBag } from "lucide-react";
import { PurchaseOrderForm } from "./components/purchase-order-form";
import {
  PurchaseOrderTable,
  PurchaseOrder,
} from "./components/purchase-order-table";
import { GRNFormDialog } from "./components/grn-form-dialog";
import { ForbiddenView } from "@/components/forbidden-view";
import { PageHeader } from "@/components/page-header";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

export default function PurchaseOrdersPage() {
  const [httpStatus, setHttpStatus] = useState<number>(200);
  const [permissions, setPermissions] = useState({
    create: false,
    read: false,
    approve: false,
    cancel: false,
    update: false,
    delete: false,
    grnCreate: false,
    poReceive: false,
  }); // 🌟 TAMBAHKAN UPDATE & DELETE
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  // State Kontrol Dialog Konfirmasi Transaksional Bawah (Gaya Roles CRUD)
  const [poToApprove, setPoToApprove] = useState<PurchaseOrder | null>(null);
  const [poToCancel, setPoToCancel] = useState<PurchaseOrder | null>(null);
  const [poToDelete, setPoToDelete] = useState<PurchaseOrder | null>(null);
  const [editingPO, setEditingPO] = useState<PurchaseOrder | null>(null);
  const [isProcessing, setIsProcessing] = useState<boolean>(false);
  const [poToReceive, setPoToReceive] = useState<PurchaseOrder | null>(null);
  const [poToFinish, setPoToFinish] = useState<PurchaseOrder | null>(null);
  const [finishItems, setFinishItems] = useState<
    {
      product_name: string;
      qty_ordered: number;
      qty_received: number;
      unit_kode: string;
    }[]
  >([]);
  const [finishLoading, setFinishLoading] = useState(false);
  const finishRequestRef = React.useRef<{
    fingerprint: string;
    key: string;
  } | null>(null);

  const authToken = Cookies.get("session_active");

  // 🌟 EVALUASI OTORISASI DEFENSIF MUTLAK (SAMA DENGAN MODUL ROLES)
  const loadPermissions = () => {
    if (!ensureSession()) return;
    const list = readPermissionList() ?? [];
    setPermissions({
      create: list.includes("po:create"),
      read: list.includes("po:read"),
      approve: list.includes("po:approve"),
      cancel: list.includes("po:reject"),
      update: list.includes("po:update"),
      delete: list.includes("po:delete"),
      grnCreate: list.includes("grn:create"),
      poReceive: list.includes("po:receive"),
    });
  };

  // Sisa barang decide apakah perlu, jadi detail PO ditarik
  // saat dialog dibuka supaya angka yang ditampilkan bukan asumsi.
  const openFinishDialog = async (po: PurchaseOrder) => {
    setPoToFinish(po);
    setFinishLoading(true);
    setFinishItems([]);
    try {
      const detail = await apiGet<{ items?: unknown[] }>(
        `/api/purchase-orders?id=${po.id}`
      );
      setFinishItems((detail.data?.items ?? []) as typeof finishItems);
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) setHttpStatus(403);
      toast.error("Gagal Memuat Data", {
        description: messageFromError(error),
      });
      setPoToFinish(null);
    } finally {
      setFinishLoading(false);
    }
  };

  const finishHasRemaining = finishItems.some(
    (i) => i.qty_received < i.qty_ordered
  );

  const handleFinishConfirm = async () => {
    if (!poToFinish) return;

    setIsProcessing(true);
    const targetedNumber = poToFinish.po_number;
    const fingerprint = String(poToFinish.id);
    if (
      !finishRequestRef.current ||
      finishRequestRef.current.fingerprint !== fingerprint
    ) {
      finishRequestRef.current = {
        fingerprint,
        key: createIdempotencyKey(),
      };
    }

    try {
      await apiPut(
        `/api/purchase-orders/finish-receiving?id=${poToFinish.id}&allow_partial=true`,
        undefined,
        {
          redirectOn403: false,
          headers: { "Idempotency-Key": finishRequestRef.current.key },
        }
      );

      finishRequestRef.current = null;
      setPoToFinish(null);
      toast.success("Penerimaan PO Ditutup", {
        description: `Dokumen ${targetedNumber} kini berstatus RECEIVED.`,
      });
      setRefreshTrigger((prev) => prev + 1);
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) setHttpStatus(403);
      toast.error("Operasi Ditolak", { description: messageFromError(error) });
    } finally {
      setIsProcessing(false);
    }
  };

  // 🌟 EKSEKUSI TRANSAKSIONAL: Konfirmasi Pengesahan Dokumen PO (APPROVED)
  const handleApproveConfirm = async () => {
    if (!poToApprove) return;

    setIsProcessing(true);
    const targetedNumber = poToApprove.po_number;

    try {
      await apiPut(
        `/api/purchase-orders/approve?id=${poToApprove.id}`,
        undefined,
        { redirectOn403: false }
      );

      setPoToApprove(null);
      setTimeout(() => {
        toast.success("Dokumen PO Disetujui", {
          description: `Dokumen ${targetedNumber} resmi disalurkan ke rekanan vendor.`,
          duration: 3000,
        });
      }, 100);

      setRefreshTrigger((prev) => prev + 1);
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) setHttpStatus(403);
      toast.error("Operasi Ditolak", {
        description:
          messageFromError(error) || "Terjadi kesalahan jaringan server.",
      });
    } finally {
      setIsProcessing(false);
    }
  };
  // 🌟 EKSEKUSI TRANSAKSIONAL: Konfirmasi Pembatalan Dokumen PO (REJECTED)
  const handleCancelConfirm = async () => {
    if (!poToCancel) return;

    setIsProcessing(true);
    const targetedNumber = poToCancel.po_number;

    try {
      await apiPut(
        `/api/purchase-orders/cancel?id=${poToCancel.id}`,
        undefined,
        { redirectOn403: false }
      );

      setPoToCancel(null);
      setTimeout(() => {
        toast.success("Dokumen PO Dibatalkan", {
          description: `Dokumen ${targetedNumber} telah ditutup dengan status REJECTED.`,
          duration: 3000,
        });
      }, 100);

      setRefreshTrigger((prev) => prev + 1);
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) setHttpStatus(403);
      toast.error("Operasi Gagal", {
        description: messageFromError(error) || "Masalah internal server.",
      });
    } finally {
      setIsProcessing(false);
    }
  };

  // 🌟 EKSEKUSI TRANSAKSIONAL DESTROY: Menghapus Dokumen PO Secara Fisik / Soft Delete dari PostgreSQL
  const handleDeleteConfirm = async () => {
    if (!poToDelete) return;

    setIsProcessing(true);
    const targetedNumber = poToDelete.po_number;

    try {
      await apiDelete(`/api/purchase-orders?id=${poToDelete.id}`, {
        redirectOn403: false,
      });

      setPoToDelete(null);
      setTimeout(() => {
        toast.success("Dokumen PO Berhasil Dihapus", {
          description: `Berkas ${targetedNumber} telah ditarik keluar dari database logistik.`,
          duration: 3000,
        });
      }, 100);

      setRefreshTrigger((prev) => prev + 1);
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) setHttpStatus(403);
      toast.error("Operasi Gagal", {
        description:
          messageFromError(error) || "Terjadi kesalahan koneksi server.",
      });
    } finally {
      setIsProcessing(false);
    }
  };

  // Callback terpusat untuk sukses mutasi form (POST untuk terbit, PUT untuk koreksi)
  const handleFormSuccess = (poNumber: string, wasEdit: boolean) => {
    setRefreshTrigger((prev) => prev + 1);
    if (wasEdit) {
      toast.success("Koreksi PO Berhasil Disimpan", {
        description: `Perubahan pada dokumen ${poNumber} telah tersimpan. Harga satuan dihitung ulang dari master barang.`,
        duration: 3000,
      });
      return;
    }
    toast.success("Purchase Order Diterbitkan", {
      description: `Dokumen ${poNumber} berhasil disimpan nyata ke basis data PostgreSQL.`,
      duration: 3000,
    });
  };

  useEffect(() => {
    loadPermissions();

    // Mengawal ketat fokus jendela tab aplikasi (Idle Handshake Guard)
    window.addEventListener("focus", loadPermissions);
    return () => window.removeEventListener("focus", loadPermissions);
  }, []);

  // Intersepsi Tampilan Boks Larangan Hak Akses (Gaya Roles CRUD)
  if (httpStatus === 403) {
    const handleResetForbidden = () => {
      setHttpStatus(200);
      loadPermissions();
      setRefreshTrigger((prev) => prev + 1);
    };
    return <ForbiddenView reset={handleResetForbidden} />;
  }

  return (
    <main className="flex flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8 w-full">
      {/* Header Judul Section Menawan */}
      <PageHeader
        icon={ShoppingBag}
        title="Manajemen Dokumen Purchase Order (PO)"
        description="Menerbitkan dokumen pesanan pengadaan logistik barang resmi ke vendor / supplier rekanan."
      />

      {/* Form Hanya Dirender Jika Pengguna Punya Akses 'create' */}
      {permissions.create && (
        <PurchaseOrderForm
          token={authToken}
          onSuccess={handleFormSuccess}
          editingPO={editingPO}
          onCancelEdit={() => setEditingPO(null)}
        />
      )}

      {/* Tabel Utama Riwayat Dokumen PO */}
      <PurchaseOrderTable
        canUpdate={permissions.update}
        canDelete={permissions.delete}
        refreshTrigger={refreshTrigger}
        onApproveClick={(po) => setPoToApprove(po)}
        onCancelClick={(po) => setPoToCancel(po)}
        onReceiveClick={(po) => setPoToReceive(po)}
        onFinishReceiveClick={(po) => openFinishDialog(po)}
        onEditClick={(po) => setEditingPO(po)}
        onDeleteClick={(po) => setPoToDelete(po)}
      />
      {/* 🌟 FORM COMPLIANCE 1: ALERT DIALOG PERSETUJUAN DOKUMEN (APPROVE) */}
      <AlertDialog
        open={poToApprove !== null}
        onOpenChange={(open) => {
          if (!open && !isProcessing) setPoToApprove(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Mengesahkan Dokumen Purchase Order?
            </AlertDialogTitle>
            <AlertDialogDescription>
              Tindakan ini akan mengunci HPP barang secara permanen dan merubah
              status dokumen menjadi APPROVED. Silsilah berkas siap disalurkan
              ke rekanan vendor:
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="my-2 p-3 bg-muted rounded-md border text-xs font-mono">
            <p className="font-bold text-blue-600 uppercase">
              {poToApprove?.po_number}
            </p>
            <p className="text-slate-500 text-xs font-sans mt-0.5 font-semibold">
              Vendor: {poToApprove?.supplier_name}
            </p>
          </div>

          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isProcessing}
              onClick={() => setPoToApprove(null)}
              className="text-xs h-8 cursor-pointer"
            >
              Batal
            </Button>
            <Button
              disabled={isProcessing}
              onClick={handleApproveConfirm}
              className="bg-emerald-600 hover:bg-emerald-700 text-white text-xs h-8 cursor-pointer"
            >
              {isProcessing ? "Memproses..." : "Ya, Sahkan Dokumen"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 🌟 FORM COMPLIANCE 2: ALERT DIALOG PEMBATALAN SEPIHAK (CANCEL) */}
      <AlertDialog
        open={poToCancel !== null}
        onOpenChange={(open) => {
          if (!open && !isProcessing) setPoToCancel(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Batalkan Dokumen Pengadaan Ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Tindakan ini tidak dapat dibatalkan. Berkas purchase order berikut
              akan ditutup sepihak dengan status REJECTED:
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="my-2 p-3 bg-muted rounded-md border text-xs font-mono">
            <p className="font-bold text-rose-600 uppercase">
              {poToCancel?.po_number}
            </p>
            <p className="text-slate-500 text-xs font-sans mt-0.5 font-semibold">
              Vendor: {poToCancel?.supplier_name}
            </p>
          </div>

          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isProcessing}
              onClick={() => setPoToCancel(null)}
              className="text-xs h-8 cursor-pointer"
            >
              Kembali
            </Button>
            <Button
              variant="destructive"
              disabled={isProcessing}
              onClick={handleCancelConfirm}
              className="text-xs h-8 cursor-pointer"
            >
              {isProcessing ? "Membatalkan..." : "Ya, Batalkan Dokumen PO"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 🌟 FORM COMPLIANCE 3: ALERT DIALOG PENGHAPUSAN BERKAS PO (DELETE) */}
      <AlertDialog
        open={poToDelete !== null}
        onOpenChange={(open) => {
          if (!open && !isProcessing) setPoToDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Apakah Anda benar-benar yakin?</AlertDialogTitle>
            <AlertDialogDescription>
              Tindakan ini tidak dapat dibatalkan. Berkas purchase order berikut
              akan dimusnahkan secara permanen dari database logistik:
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="my-2 p-3 bg-rose-50 text-rose-700 rounded-md border border-rose-100 text-xs font-mono">
            <p className="font-bold uppercase">{poToDelete?.po_number}</p>
            <p className="text-xs mt-0.5">
              Supplier: {poToDelete?.supplier_name}
            </p>
          </div>

          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isProcessing}
              onClick={() => setPoToDelete(null)}
              className="text-xs h-8 cursor-pointer"
            >
              Batal
            </Button>
            <Button
              variant="destructive"
              disabled={isProcessing}
              onClick={handleDeleteConfirm}
              className="text-xs h-8 cursor-pointer"
            >
              {isProcessing ? "Menghapus..." : "Ya, Musnahkan Berkas PO"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      {/* DIALOG PENCATATAN PENERIMAAN BARANG (GRN) */}
      {permissions.grnCreate && poToReceive && (
        <GRNFormDialog
          open={poToReceive !== null}
          onOpenChange={(o) => !o && setPoToReceive(null)}
          poId={poToReceive.id}
          poNumber={poToReceive.po_number}
          onSuccess={() => {
            setRefreshTrigger((prev) => prev + 1);
          }}
        />
      )}

      {/* KONFIRMASI PENUTUPAN PENERIMAAN PO (APPROVED -> RECEIVED) */}
      <AlertDialog
        open={poToFinish !== null}
        onOpenChange={(open) => {
          if (!open && !isProcessing) setPoToFinish(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Tandai PO Selesai Diterima?</AlertDialogTitle>
            <AlertDialogDescription>
              Stok dari seluruh penerimaan yang tercatat akan ditambahkan ke
              gudang, lalu status dokumen berubah menjadi RECEIVED dan tidak
              bisa Receive Barang lagi.
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="my-2 p-3 bg-slate-50 rounded-md border text-xs font-mono">
            <p className="font-bold text-teal-700 uppercase">
              {poToFinish?.po_number}
            </p>
            <p className="text-slate-500 text-xs font-sans mt-0.5 font-semibold">
              Vendor: {poToFinish?.supplier_name}
            </p>
          </div>

          {finishLoading ? (
            <p className="text-xs text-slate-500 text-center py-3">
              Memuat sisa barang...
            </p>
          ) : finishHasRemaining ? (
            <div className="bg-amber-50 border border-amber-200 text-amber-900 rounded-md p-3 text-xs">
              <p className="font-bold mb-1.5">
                Masih ada barang yang belum diterima:
              </p>
              <ul className="space-y-1 font-mono">
                {finishItems
                  .filter((i) => i.qty_received < i.qty_ordered)
                  .map((i) => (
                    <li
                      key={i.product_name}
                      className="flex justify-between gap-2"
                    >
                      <span className="truncate">{i.product_name}</span>
                      <span className="font-bold shrink-0">
                        {i.qty_received} / {i.qty_ordered} {i.unit_kode}
                      </span>
                    </li>
                  ))}
              </ul>
              <p className="mt-2 text-xs">
                PO akan ditutup dengan sisa. GRN yang terlanjur tercatat tetap
                tersimpan.
              </p>
            </div>
          ) : (
            <div className="bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-md p-3 text-xs">
              Semua barang pada PO ini sudah diterima penuh.
            </div>
          )}

          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isProcessing || finishLoading}
              onClick={() => setPoToFinish(null)}
              className="text-xs h-8 cursor-pointer"
            >
              Kembali
            </Button>
            <Button
              disabled={isProcessing || finishLoading}
              onClick={handleFinishConfirm}
              className="text-xs h-8 cursor-pointer bg-teal-600 hover:bg-teal-700"
            >
              {isProcessing
                ? "Memproses..."
                : finishHasRemaining
                  ? "Ya, Tutup dengan Sisa"
                  : "Ya, Tutup Terima"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  );
}
