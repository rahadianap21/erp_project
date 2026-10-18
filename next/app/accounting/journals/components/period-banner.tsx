"use client";

import { apiGet, apiPost, apiPut } from "@/lib/api-client";
import { useEffect, useState } from "react";
import { CalendarRange, Lock, LockOpen, RefreshCw, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ComboboxSelect } from "@/components/combobox-select";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { toast } from "sonner";

export interface AccountingPeriod {
  id: number;
  period_year: number;
  period_month: number;
  status: "OPEN" | "CLOSED";
  closed_at?: string;
}

interface PeriodBannerProps {
  revision: number;
  canManagePeriod: boolean;
}

const monthNames = [
  "Januari",
  "Februari",
  "Maret",
  "April",
  "Mei",
  "Juni",
  "Juli",
  "Agustus",
  "September",
  "Oktober",
  "November",
  "Desember",
];

export function PeriodBanner({ revision, canManagePeriod }: PeriodBannerProps) {
  const [periods, setPeriods] = useState<AccountingPeriod[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [processing, setProcessing] = useState<string | null>(null);
  const [openYear, setOpenYear] = useState<number>(new Date().getFullYear());
  const [openMonth, setOpenMonth] = useState<number>(new Date().getMonth() + 1);
  const [target, setTarget] = useState<AccountingPeriod | null>(null);

  const fetchPeriods = async () => {
    setLoading(true);
    try {
      const result = await apiGet<AccountingPeriod[]>(
        `/api/accounting-periods`
      );
      if (Array.isArray(result.data)) setPeriods(result.data);
    } catch (err) {
      console.error("Gagal memuat periode fiskal:", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPeriods();
  }, [revision]);

  const openPeriod = async () => {
    setProcessing("open");
    try {
      await apiPost(`/api/accounting-periods`, {
        year: openYear,
        month: openMonth,
      });
      toast.success("Periode Fiskal Dibuka", {
        description: `${monthNames[openMonth - 1]} ${openYear} kini aktif.`,
      });
      fetchPeriods();
    } catch (error: any) {
      toast.error("Operasi Gagal", { description: error.message });
    } finally {
      setProcessing(null);
    }
  };

  const togglePeriod = async (period: AccountingPeriod) => {
    const isClose = period.status === "OPEN";
    setProcessing(`toggle-${period.id}`);
    try {
      await apiPut(
        `/api/accounting-periods/${isClose ? "close" : "reopen"}?year=${period.period_year}&month=${period.period_month}`,
        undefined
      );
      toast.success(
        isClose ? "Periode Fiskal Ditutup" : "Periode Fiskal Dibuka Kembali",
        {
          description: isClose
            ? `Jurnal penutup diterbitkan: pendapatan & beban dinolkan ke Laba Ditahan. ${monthNames[period.period_month - 1]} ${period.period_year} terkunci dari posting baru.`
            : `Jurnal penutup dibalik. ${monthNames[period.period_month - 1]} ${period.period_year} kini dapat diposting kembali.`,
        }
      );
      fetchPeriods();
    } catch (error: any) {
      toast.error("Operasi Gagal", { description: error.message });
    } finally {
      setProcessing(null);
      setTarget(null);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-2xs border border-slate-200 p-4">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-slate-700">
          <CalendarRange className="size-4 text-indigo-600" />
          Periode Fiskal
          {loading && (
            <RefreshCw className="size-3 animate-spin text-slate-400" />
          )}
        </div>

        {canManagePeriod && (
          <div className="flex items-center gap-2">
            <ComboboxSelect
              options={[
                new Date().getFullYear() - 1,
                new Date().getFullYear(),
                new Date().getFullYear() + 1,
              ].map((y) => ({ value: String(y), label: String(y) }))}
              value={String(openYear)}
              placeholder="Tahun..."
              emptyText="Tahun tidak ditemukan..."
              onChange={(val) =>
                setOpenYear(Number(val) || new Date().getFullYear())
              }
              className="w-28 cursor-pointer"
              popoverClassName="!w-28 cursor-pointer"
            />
            <ComboboxSelect
              options={monthNames.map((m, i) => ({
                value: String(i + 1),
                label: m,
              }))}
              value={String(openMonth)}
              placeholder="Bulan..."
              emptyText="Bulan tidak ditemukan..."
              onChange={(val) =>
                setOpenMonth(Number(val) || new Date().getMonth() + 1)
              }
              className="w-48 cursor-pointer"
              popoverClassName="!w-48 cursor-pointer"
            />
            <Button
              size="sm"
              className="h-8 text-[11px] cursor-pointer bg-indigo-600 hover:bg-indigo-700 text-white"
              onClick={openPeriod}
              disabled={processing === "open"}
            >
              {processing === "open" ? (
                <RefreshCw className="size-3 animate-spin" />
              ) : (
                <Plus className="size-3" />
              )}{" "}
              Buka Periode
            </Button>
          </div>
        )}
      </div>

      <div className="mt-3 flex flex-wrap gap-2">
        {periods.length === 0 && !loading && (
          <span className="text-[11px] text-slate-400">
            Belum ada periode fiskal tercatat.
          </span>
        )}
        {periods.map((period) => {
          const label = `${monthNames[period.period_month - 1]} ${period.period_year}`;
          const isOpen = period.status === "OPEN";
          return (
            <div
              key={period.id}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg border text-[11px] font-semibold ${
                isOpen
                  ? "bg-emerald-50 border-emerald-200 text-emerald-700"
                  : "bg-slate-50 border-slate-200 text-slate-500"
              }`}
            >
              {isOpen ? (
                <LockOpen className="size-3.5" />
              ) : (
                <Lock className="size-3.5" />
              )}
              <span>{label}</span>
              <span
                className={`px-1.5 py-0.5 rounded-full text-[9px] font-bold ${isOpen ? "bg-emerald-100 text-emerald-700" : "bg-slate-200 text-slate-600"}`}
              >
                {isOpen ? "OPEN" : "CLOSED"}
              </span>
              {canManagePeriod && (
                <Button
                  variant="ghost"
                  size="sm"
                  className={`h-6 text-[10px] px-2 cursor-pointer ${isOpen ? "text-rose-600 hover:bg-rose-50" : "text-slate-600 hover:bg-slate-100"}`}
                  onClick={() => setTarget(period)}
                  disabled={processing === `toggle-${period.id}`}
                >
                  {isOpen ? "Tutup" : "Buka Ulang"}
                </Button>
              )}
            </div>
          );
        })}
      </div>

      <AlertDialog
        open={target !== null}
        onOpenChange={(open) => {
          if (!open && !processing) setTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {target?.status === "OPEN"
                ? "Tutup Buku Periode Ini?"
                : "Buka Ulang Periode Ini?"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {target?.status === "OPEN" ? (
                <>
                  Sistem akan menerbitkan{" "}
                  <span className="font-semibold">jurnal penutup</span>: seluruh
                  saldo akun pendapatan dan beban dinolkan, dan laba/rugi
                  periode ini dipindahkan ke akun Laba Ditahan. Setelah itu
                  periode{" "}
                  {target &&
                    `${monthNames[target.period_month - 1]} ${target.period_year}`}{" "}
                  dikunci dari posting baru.
                </>
              ) : (
                <>
                  Jurnal penutup periode{" "}
                  {target &&
                    `${monthNames[target.period_month - 1]} ${target.period_year}`}{" "}
                  akan <span className="font-semibold">dibalik</span> sehingga
                  saldo pendapatan dan beban kembali seperti sebelum tutup buku,
                  lalu periode dibuka untuk posting baru.
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <Button
              variant="outline"
              size="sm"
              className="cursor-pointer"
              onClick={() => setTarget(null)}
              disabled={processing !== null}
            >
              Batal
            </Button>
            <Button
              size="sm"
              className={`cursor-pointer text-white ${target?.status === "OPEN" ? "bg-rose-600 hover:bg-rose-700" : "bg-indigo-600 hover:bg-indigo-700"}`}
              onClick={() => target && togglePeriod(target)}
              disabled={processing !== null}
            >
              {processing ? (
                <RefreshCw className="size-3 animate-spin" />
              ) : null}
              {target?.status === "OPEN" ? "Ya, Tutup Buku" : "Ya, Buka Ulang"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
