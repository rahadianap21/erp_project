"use client";

import { ApiError, apiGet, apiPost, messageFromError } from "@/lib/api-client";
import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  ChevronDown,
  ChevronUp,
  Loader2,
  FileText,
  CalendarDays,
  Plus,
  Trash2,
  Equal,
  ClipboardSignature,
} from "lucide-react";
import { toast } from "sonner";
import { ComboboxSelect } from "@/components/combobox-select";
import { ChartOfAccount } from "../../chart-of-accounts/components/chart-of-account-table";

interface JournalLineInput {
  account_id: number;
  description: string;
  debit: number;
  credit: number;
}

interface JournalFormProps {
  token: string | undefined;
  onSuccess: (entryNumber: string) => void;
  onPermissionDenied?: () => void;
}

export function JournalForm({
  token,
  onSuccess,
  onPermissionDenied,
}: JournalFormProps) {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [accountOptions, setAccountOptions] = useState<ChartOfAccount[]>([]);

  const [entryDate, setEntryDate] = useState<string>(
    new Date().toISOString().slice(0, 10)
  );
  const [description, setDescription] = useState<string>("");
  const [lines, setLines] = useState<JournalLineInput[]>([
    { account_id: 0, description: "", debit: 0, credit: 0 },
    { account_id: 0, description: "", debit: 0, credit: 0 },
  ]);

  const fetchAccountOptions = async () => {
    try {
      const result = await apiGet<ChartOfAccount[]>(
        `/api/chart-of-accounts?limit=500`
      );
      if (result.data) {
        setAccountOptions(
          result.data.filter((a: ChartOfAccount) => !a.is_header && a.is_active)
        );
      }
    } catch (err) {
      console.error("Gagal memuat daftar akun jurnal:", err);
    }
  };

  useEffect(() => {
    if (token) fetchAccountOptions();
  }, [token]);

  const totals = useMemo(() => {
    const debit = lines.reduce((sum, l) => sum + (Number(l.debit) || 0), 0);
    const credit = lines.reduce((sum, l) => sum + (Number(l.credit) || 0), 0);
    return { debit, credit, balanced: Math.abs(debit - credit) < 0.5 };
  }, [lines]);

  const updateLine = (index: number, patch: Partial<JournalLineInput>) => {
    setLines((prev) =>
      prev.map((l, i) => (i === index ? { ...l, ...patch } : l))
    );
  };

  const addLine = () => {
    setLines((prev) => [
      ...prev,
      { account_id: 0, description: "", debit: 0, credit: 0 },
    ]);
  };

  const removeLine = (index: number) => {
    setLines((prev) =>
      prev.length > 2 ? prev.filter((_, i) => i !== index) : prev
    );
  };

  const tx = (value: string) => Number(value.replace(/[^0-9.]/g, "")) || 0;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!totals.balanced) {
      toast.error("Jurnal Tidak Seimbang", {
        description:
          "Jumlah debit harus sama dengan jumlah kredit sebelum diposting.",
      });
      return;
    }
    const validLines = lines.filter(
      (l) => l.account_id > 0 && (l.debit > 0 || l.credit > 0)
    );
    if (validLines.length < 2) {
      toast.error("Baris Jurnal Tidak Lengkap", {
        description:
          "Minimal dua baris akun dengan minimal satu debit dan satu kredit.",
      });
      return;
    }

    setIsLoading(true);
    try {
      const errorData = await apiPost<{ entry_number?: string }>(
        `/api/journals`,
        { entry_date: entryDate, description, lines: validLines },
        { redirectOn403: false }
      );

      setIsExpanded(false);
      setDescription("");
      setLines([
        { account_id: 0, description: "", debit: 0, credit: 0 },
        { account_id: 0, description: "", debit: 0, credit: 0 },
      ]);
      onSuccess(errorData.data?.entry_number || "");
    } catch (error) {
      if (error instanceof ApiError && error.status === 403)
        onPermissionDenied?.();
      toast.error("Gagal Menyimpan Jurnal", {
        description: messageFromError(error),
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-2xs border border-slate-200 overflow-hidden transition-all duration-300">
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full px-6 py-3.5 flex items-center justify-between bg-slate-50/80 hover:bg-slate-100/60 transition-colors text-left cursor-pointer"
      >
        <span className="text-xs font-bold uppercase tracking-wider text-slate-700 flex items-center gap-2">
          <ClipboardSignature className="size-4 text-blue-600" />
          Buat Jurnal Umum Manual (Double-Entry)
        </span>
        {isExpanded ? (
          <ChevronUp className="h-4 w-4" />
        ) : (
          <ChevronDown className="h-4 w-4" />
        )}
      </button>

      <div
        className={`transition-all duration-300 ease-in-out ${isExpanded ? "max-h-none p-6 border-t border-slate-100 overflow-y-auto" : "max-h-0 opacity-0 pointer-events-none"}`}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <CalendarDays className="size-3" /> Tanggal Entri
              </label>
              <input
                type="date"
                required
                value={entryDate}
                onChange={(e) => setEntryDate(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900"
              />
            </div>
            <div className="md:col-span-2 space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <FileText className="size-3" /> Keterangan Jurnal
              </label>
              <input
                type="text"
                required
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900"
                placeholder="Contoh: Penyesuaian akhir bulan"
              />
            </div>
          </div>

          <div className="border rounded-xl overflow-x-auto">
            <div className="min-w-[720px] bg-slate-100/80 px-4 py-2 grid grid-cols-12 gap-2 text-[10px] font-bold uppercase tracking-wider text-slate-500">
              <span className="col-span-4">Akun</span>
              <span className="col-span-3">Keterangan</span>
              <span className="col-span-2 text-right">Debit (Rp)</span>
              <span className="col-span-2 text-right">Kredit (Rp)</span>
              <span className="col-span-1"></span>
            </div>

            {lines.map((line, index) => (
              <div
                key={index}
                className="min-w-[720px] border-t border-slate-100 px-4 py-2 grid grid-cols-12 gap-2 items-start"
              >
                <div className="col-span-4 space-y-1">
                  <ComboboxSelect
                    key={`jrnl-acct-combo-${index}-${line.account_id || "empty"}`}
                    options={accountOptions.map((acc) => ({
                      value: String(acc.id),
                      label: `${acc.code} · ${acc.name}`,
                    }))}
                    value={line.account_id ? String(line.account_id) : ""}
                    placeholder="-- Cari Akun --"
                    emptyText="Akun tidak ditemukan..."
                    onChange={(val) =>
                      updateLine(index, { account_id: Number(val) || 0 })
                    }
                  />
                </div>
                <div className="col-span-3">
                  <input
                    type="text"
                    value={line.description}
                    onChange={(e) =>
                      updateLine(index, { description: e.target.value })
                    }
                    className="w-full text-[11px] px-2 py-1.5 bg-slate-50/50 border border-slate-200 rounded-md text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                    placeholder="Opsional"
                  />
                </div>
                <div className="col-span-2">
                  <input
                    type="text"
                    inputMode="decimal"
                    value={line.debit || ""}
                    onChange={(e) =>
                      updateLine(index, {
                        debit: tx(e.target.value),
                        credit: 0,
                      })
                    }
                    className="w-full text-[11px] px-2 py-1.5 bg-blue-50/40 border border-slate-200 rounded-md text-right text-blue-700 font-semibold focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                    placeholder="0"
                  />
                </div>
                <div className="col-span-2">
                  <input
                    type="text"
                    inputMode="decimal"
                    value={line.credit || ""}
                    onChange={(e) =>
                      updateLine(index, {
                        credit: tx(e.target.value),
                        debit: 0,
                      })
                    }
                    className="w-full text-[11px] px-2 py-1.5 bg-amber-50/40 border border-slate-200 rounded-md text-right text-amber-700 font-semibold focus:outline-none focus:ring-2 focus:ring-amber-500/20"
                    placeholder="0"
                  />
                </div>
                <div className="col-span-1 text-right">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 text-rose-400 hover:text-rose-600 hover:bg-rose-50 cursor-pointer"
                    onClick={() => removeLine(index)}
                    disabled={lines.length <= 2}
                  >
                    <Trash2 className="size-3" />
                  </Button>
                </div>
              </div>
            ))}

            <div className="px-4 py-2 bg-slate-50/60 border-t border-slate-100 flex items-center justify-between">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addLine}
                className="text-[11px] h-7 cursor-pointer"
              >
                <Plus className="size-3 mr-1" /> Tambah Baris
              </Button>
              <div
                className={`flex items-center gap-3 text-[11px] font-bold ${totals.balanced ? "text-emerald-600" : "text-rose-600"}`}
              >
                <Equal className="size-3.5" />
                <span>
                  Debit: {new Intl.NumberFormat("id-ID").format(totals.debit)}
                </span>
                <span>
                  Kredit: {new Intl.NumberFormat("id-ID").format(totals.credit)}
                </span>
                <span
                  className={`px-2 py-0.5 rounded-full border ${totals.balanced ? "bg-emerald-50 border-emerald-200" : "bg-rose-50 border-rose-200"}`}
                >
                  {totals.balanced ? "SEIMBANG" : "BELUM SEIMBANG"}
                </span>
              </div>
            </div>
          </div>

          <div className="flex justify-end space-x-3 pt-3 border-t border-slate-100">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setIsExpanded(false)}
              className="text-xs h-8 cursor-pointer"
            >
              Batal
            </Button>
            <Button
              type="submit"
              size="sm"
              disabled={isLoading}
              className="bg-blue-600 hover:bg-blue-700 text-white text-xs h-8 shadow-xs cursor-pointer"
            >
              {isLoading ? (
                <span className="flex items-center gap-1.5">
                  <Loader2 className="size-3 animate-spin" /> Memproses...
                </span>
              ) : (
                "Simpan Jurnal (DRAFT)"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
