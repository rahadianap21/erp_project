"use client";

import { ApiError, apiPut, messageFromError } from "@/lib/api-client";
import { ensureSession, readPermissionList } from "@/lib/permissions";
import React, { useState, useEffect } from "react";
import Cookies from "js-cookie";
import { toast } from "sonner";
import { BookOpenText } from "lucide-react";
import { JournalForm } from "./components/journal-form";
import { JournalTable, JournalEntry } from "./components/journal-table";
import { PeriodBanner } from "./components/period-banner";
import { ForbiddenView } from "@/components/forbidden-view";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

export default function JournalsPage() {
  const [httpStatus, setHttpStatus] = useState<number>(200);
  const [permissions, setPermissions] = useState({
    create: false,
    read: false,
    post: false,
    void: false,
    closePeriod: false,
  });
  const [refreshTrigger, setRefreshTrigger] = useState(0);
  const [periodRevision, setPeriodRevision] = useState(0);

  const [entryToPost, setEntryToPost] = useState<JournalEntry | null>(null);
  const [entryToVoid, setEntryToVoid] = useState<JournalEntry | null>(null);
  const [entryToReverse, setEntryToReverse] = useState<JournalEntry | null>(
    null
  );
  const [isProcessing, setIsProcessing] = useState<boolean>(false);

  const authToken = Cookies.get("session_active");

  const loadPermissions = () => {
    if (!ensureSession()) return;
    const list = readPermissionList() ?? [];
    setPermissions({
      create: list.includes("accounting:create"),
      read: list.includes("accounting:read"),
      post: list.includes("accounting:post"),
      void: list.includes("accounting:post"),
      closePeriod: list.includes("accounting:close-period"),
    });
  };

  const executeTransition = async (
    entry: JournalEntry,
    action: "post" | "void" | "reverse",
    successTitle: string,
    successDesc: (entryNumber: string) => string
  ) => {
    setIsProcessing(true);

    try {
      await apiPut(`/api/journals/${action}?id=${entry.id}`, undefined, {
        redirectOn403: false,
      });

      toast.success(successTitle, {
        description: successDesc(entry.entry_number),
        duration: 3500,
      });
      setRefreshTrigger((prev) => prev + 1);
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) setHttpStatus(403);
      toast.error("Operasi Gagal", { description: messageFromError(error) });
    } finally {
      setIsProcessing(false);
      setEntryToPost(null);
      setEntryToVoid(null);
      setEntryToReverse(null);
    }
  };

  const handleFormSuccess = (entryNumber: string) => {
    setRefreshTrigger((prev) => prev + 1);
    setPeriodRevision((prev) => prev + 1);
    toast.success("Jurnal Tersimpan", {
      description: `Dokumen ${entryNumber} berstatus DRAFT sudah tercatat di buku besar.`,
      duration: 3500,
    });
  };

  useEffect(() => {
    loadPermissions();
    window.addEventListener("focus", loadPermissions);
    return () => window.removeEventListener("focus", loadPermissions);
  }, []);

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
      <div className="flex items-center gap-2 border-b pb-4">
        <div className="p-2 bg-blue-50 text-blue-600 rounded-xl border border-blue-100">
          <BookOpenText className="size-5" />
        </div>
        <div>
          <h1 className="text-sm font-black uppercase tracking-wider text-slate-900">
            Jurnal Umum (General Journal)
          </h1>
          <p className="text-xs text-slate-500">
            Pencatatan transaksi double-entry: debit = kredit, terkunci oleh
            periode fiskal.
          </p>
        </div>
      </div>

      <PeriodBanner
        revision={periodRevision}
        canManagePeriod={permissions.closePeriod}
      />

      {permissions.create && (
        <JournalForm
          token={authToken}
          onSuccess={handleFormSuccess}
          onPermissionDenied={() => setHttpStatus(403)}
        />
      )}

      <JournalTable
        canPost={permissions.post}
        canVoidReverse={permissions.void}
        refreshTrigger={refreshTrigger}
        onPostClick={(entry) => setEntryToPost(entry)}
        onVoidClick={(entry) => setEntryToVoid(entry)}
        onReverseClick={(entry) => setEntryToReverse(entry)}
      />

      {/* DIALOG: Posting Jurnal DRAFT -> POSTED */}
      <AlertDialog
        open={entryToPost !== null}
        onOpenChange={(open) => {
          if (!open && !isProcessing) setEntryToPost(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Posting Jurnal Ini Ke Buku Besar?
            </AlertDialogTitle>
            <AlertDialogDescription>
              Jurnal akan terkunci permanen ke periode fiskal aktif dan menjadi
              dasar laporan keuangan.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="my-2 p-3 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-md font-mono text-xs">
            <p className="font-bold uppercase">{entryToPost?.entry_number}</p>
            <p className="text-[11px] mt-0.5">{entryToPost?.description}</p>
          </div>
          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isProcessing}
              onClick={() => setEntryToPost(null)}
              className="text-xs h-8 cursor-pointer"
            >
              Batal
            </Button>
            <Button
              disabled={isProcessing}
              onClick={() =>
                entryToPost &&
                executeTransition(
                  entryToPost,
                  "post",
                  "Jurnal Berhasil Diposting",
                  (n) => `Dokumen ${n} kini POSTED di buku besar.`
                )
              }
              className="bg-emerald-600 hover:bg-emerald-700 text-white text-xs h-8 cursor-pointer"
            >
              {isProcessing ? "Memproses..." : "Ya, Posting"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* DIALOG: Void Jurnal DRAFT */}
      <AlertDialog
        open={entryToVoid !== null}
        onOpenChange={(open) => {
          if (!open && !isProcessing) setEntryToVoid(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Batalkan (Void) Jurnal Draft Ini?
            </AlertDialogTitle>
            <AlertDialogDescription>
              Dokumen akan ditandai VOID dan tidak bisa diposting lagi.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="my-2 p-3 bg-rose-50 border border-rose-200 text-rose-700 rounded-md font-mono text-xs">
            <p className="font-bold uppercase">{entryToVoid?.entry_number}</p>
            <p className="text-[11px] mt-0.5">{entryToVoid?.description}</p>
          </div>
          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isProcessing}
              onClick={() => setEntryToVoid(null)}
              className="text-xs h-8 cursor-pointer"
            >
              Batal
            </Button>
            <Button
              variant="destructive"
              disabled={isProcessing}
              onClick={() =>
                entryToVoid &&
                executeTransition(
                  entryToVoid,
                  "void",
                  "Jurnal Dibatalkan",
                  (n) => `Dokumen ${n} berstatus VOID.`
                )
              }
              className="text-xs h-8 cursor-pointer"
            >
              {isProcessing ? "Memproses..." : "Ya, Vibuat Void"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* DIALOG: Reverse Jurnal POSTED */}
      <AlertDialog
        open={entryToReverse !== null}
        onOpenChange={(open) => {
          if (!open && !isProcessing) setEntryToReverse(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Terbitkan Jurnal Pembalik?</AlertDialogTitle>
            <AlertDialogDescription>
              Sistem akan membuat jurnal baru dengan posisi debit/kredit
              terbalik lalu memostingnya.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="my-2 p-3 bg-violet-50 border border-violet-200 text-violet-800 rounded-md font-mono text-xs">
            <p className="font-bold uppercase">
              {entryToReverse?.entry_number}
            </p>
            <p className="text-[11px] mt-0.5">{entryToReverse?.description}</p>
          </div>
          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isProcessing}
              onClick={() => setEntryToReverse(null)}
              className="text-xs h-8 cursor-pointer"
            >
              Batal
            </Button>
            <Button
              disabled={isProcessing}
              onClick={() =>
                entryToReverse &&
                executeTransition(
                  entryToReverse,
                  "reverse",
                  "Jurnal Pembalik Diterbitkan",
                  (n) => `${n} dibatalkan lewat jurnal pembalik baru.`
                )
              }
              className="bg-violet-600 hover:bg-violet-700 text-white text-xs h-8 cursor-pointer"
            >
              {isProcessing ? "Memproses..." : "Ya, Reverse"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  );
}
