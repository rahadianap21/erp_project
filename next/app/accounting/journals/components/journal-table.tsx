"use client";

import { apiGet } from "@/lib/api-client";
import { useState } from "react";
import {
  Send,
  Ban,
  RotateCcw,
  FileText,
  ChevronRight,
  AlertTriangle,
  Filter,
} from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useDebounceFetch } from "@/hooks/use-api-list";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import { DataTablePagination } from "@/components/data-table-pagination";

const formatRupiah = (num: number) =>
  new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(num);

export interface JournalEntry {
  id: number;
  entry_number: string;
  entry_date: string;
  description?: string;
  source_type?: string;
  status: "DRAFT" | "POSTED" | "VOID";
  total_debit: number;
  total_credit: number;
  period_year: number;
  period_month: number;
  posted_at?: string;
  void_reason?: string;
  reversal_of_id?: number | null;
  reversed_by_id?: number | null;
  lines?: JournalLine[];
}

export interface JournalLine {
  id: number;
  line_no: number;
  account_id: number;
  account_code: string;
  account_name: string;
  description?: string;
  debit: number;
  credit: number;
}

interface JournalTableProps {
  canPost: boolean;
  canVoidReverse: boolean;
  refreshTrigger: number;
  onPostClick: (entry: JournalEntry) => void;
  onVoidClick: (entry: JournalEntry) => void;
  onReverseClick: (entry: JournalEntry) => void;
}

const STATUS_BADGE: Record<JournalEntry["status"], string> = {
  DRAFT: "bg-slate-100 text-slate-600 border-slate-200",
  POSTED: "bg-emerald-50 text-emerald-700 border-emerald-200",
  VOID: "bg-rose-50 text-rose-600 border-rose-200",
};

export function JournalTable({
  canPost,
  canVoidReverse,
  refreshTrigger,
  onPostClick,
  onVoidClick,
  onReverseClick,
}: JournalTableProps) {
  const {
    search,
    setSearch,
    page,
    limit,
    setLimit,
    data: journals,
    total,
    loading,
    handleReset,
    isFiltered,
    hasNext,
    hasPrev,
    handleNextPage,
    handlePrevPage,
    updateFilter,
    filters,
  } = useDebounceFetch<JournalEntry>({
    endpoint: "/api/journals",
    delay: 500,
    initialFilters: { status: "ALL" },
    refreshTrigger: refreshTrigger,
  });

  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [detailLines, setDetailLines] = useState<Record<number, JournalLine[]>>(
    {}
  );
  const [detailLoading, setDetailLoading] = useState<boolean>(false);

  const status = (filters.status as string) || "ALL";

  const showActions = canPost || canVoidReverse;

  // Baris dari endpoint list tidak menyertakan `lines` (hanya header). Saat
  // baris dibuka, tarik detail lengkap berisi akun + keterangan per baris
  // lalu cache agar tidak diambil ulang saat ditutup-buka lagi.
  const toggleRow = async (entry: JournalEntry) => {
    if (expandedId === entry.id) {
      setExpandedId(null);
      return;
    }
    setExpandedId(entry.id);
    if (detailLines[entry.id]) return;

    setDetailLoading(true);
    try {
      const result = await apiGet<{ lines?: JournalLine[] }>(
        `/api/journals?id=${entry.id}`
      );
      const lines = result.data?.lines;
      if (lines) {
        setDetailLines((prev) => ({ ...prev, [entry.id]: lines }));
      }
    } catch (err) {
      console.error("Gagal memuat rincian jurnal:", err);
    } finally {
      setDetailLoading(false);
    }
  };

  const columnCount = showActions ? 7 : 6;

  return (
    <div className="space-y-4">
      <DataTableToolbar
        search={search}
        setSearch={setSearch}
        placeholder="Cari nomor jurnal atau deskripsi..."
        loading={loading}
        isFiltered={isFiltered}
        onReset={handleReset}
        counterIcon={<FileText className="size-3.5 text-slate-400" />}
        counterLabel={`Total ${total} jurnal terverifikasi`}
      >
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1.5 text-xs bg-slate-100/80 px-2.5 py-1.5 rounded-lg border text-slate-600 font-medium">
            <Filter className="size-3" /> Status Jurnal:
          </div>
          <select
            value={status}
            onChange={(e) => updateFilter("status", e.target.value)}
            className="text-xs h-9 px-3 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer focus:border-blue-500 font-medium text-slate-700"
          >
            <option value="ALL">Semua Status</option>
            <option value="DRAFT">DRAFT</option>
            <option value="POSTED">POSTED</option>
            <option value="VOID">VOID</option>
          </select>
        </div>
      </DataTableToolbar>

      <div className="border rounded-xl shadow-xs overflow-hidden bg-white relative">
        {loading && journals.length > 0 && (
          <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] z-10 flex items-center justify-center">
            <Skeleton className="h-5 w-40" />
          </div>
        )}

        <Table>
          <TableHeader>
            <TableRow className="bg-slate-50/80 hover:bg-slate-50/80">
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                No. Jurnal
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Tanggal
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Keterangan
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Periode
              </TableHead>
              <TableHead className="text-right text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Nominal
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Status
              </TableHead>
              {showActions && (
                <TableHead className="text-right text-[11px] font-bold uppercase tracking-wider text-slate-500">
                  Aksi
                </TableHead>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && journals.length === 0 ? (
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Skeleton className="h-4 w-32" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-52" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-16" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-20 ml-auto" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-16" />
                  </TableCell>
                  {showActions && (
                    <TableCell>
                      <Skeleton className="h-4 w-24 ml-auto" />
                    </TableCell>
                  )}
                </TableRow>
              ))
            ) : journals.length === 0 ? (
              <TableRow>
                <TableCell colSpan={columnCount} className="h-24 text-center">
                  <div className="flex flex-col items-center gap-1.5 text-slate-400">
                    <FileText className="size-5" />
                    <span className="text-xs font-medium">
                      Belum ada entri jurnal yang tersimpan.
                    </span>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              journals.flatMap((entry) => {
                const isOpen = expandedId === entry.id;
                const lines = detailLines[entry.id];

                const mainRow = (
                  <TableRow
                    key={entry.id}
                    className={`cursor-pointer transition-colors ${isOpen ? "bg-blue-50/40 hover:bg-blue-50/50" : "hover:bg-slate-50/70"}`}
                    onClick={() => toggleRow(entry)}
                  >
                    <TableCell>
                      <div className="flex items-center gap-1.5">
                        <ChevronRight
                          className={`size-3.5 shrink-0 text-slate-400 transition-transform duration-200 ${isOpen ? "rotate-90" : ""}`}
                        />
                        <span className="font-mono text-xs font-bold text-slate-800">
                          {entry.entry_number}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-slate-600">
                      {new Date(entry.entry_date).toLocaleDateString("id-ID", {
                        day: "2-digit",
                        month: "short",
                        year: "numeric",
                      })}
                    </TableCell>
                    <TableCell className="text-xs text-slate-700 max-w-[260px]">
                      <span className="line-clamp-1">
                        {entry.description || "-"}
                      </span>
                      {entry.reversal_of_id && (
                        <span className="block text-[10px] font-bold text-violet-600">
                          Pembalik dari JV#{entry.reversal_of_id}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-[11px] font-mono text-slate-500">
                      {entry.period_year}-
                      {String(entry.period_month).padStart(2, "0")}
                    </TableCell>
                    <TableCell className="text-right">
                      <span className="text-xs font-bold text-slate-800">
                        {formatRupiah(entry.total_debit)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="outline"
                        className={`text-[10px] font-bold border ${STATUS_BADGE[entry.status]}`}
                      >
                        {entry.status}
                      </Badge>
                    </TableCell>
                    {showActions && (
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1.5">
                          {canPost && entry.status === "DRAFT" && (
                            <Button
                              variant="outline"
                              size="sm"
                              className="h-7 text-[11px] cursor-pointer text-emerald-700 border-emerald-200 hover:bg-emerald-50"
                              onClick={(e) => {
                                e.stopPropagation();
                                onPostClick(entry);
                              }}
                            >
                              <Send className="size-3 mr-1" /> Post
                            </Button>
                          )}
                          {canVoidReverse && entry.status === "DRAFT" && (
                            <Button
                              variant="outline"
                              size="sm"
                              className="h-7 text-[11px] cursor-pointer text-rose-600 border-rose-200 hover:bg-rose-50"
                              onClick={(e) => {
                                e.stopPropagation();
                                onVoidClick(entry);
                              }}
                            >
                              <Ban className="size-3 mr-1" /> Void
                            </Button>
                          )}
                          {canVoidReverse && entry.status === "POSTED" && (
                            <Button
                              variant="outline"
                              size="sm"
                              className="h-7 text-[11px] cursor-pointer text-violet-600 border-violet-200 hover:bg-violet-50"
                              onClick={(e) => {
                                e.stopPropagation();
                                onReverseClick(entry);
                              }}
                            >
                              <RotateCcw className="size-3 mr-1" /> Reverse
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    )}
                  </TableRow>
                );

                if (!isOpen) return [mainRow];

                const detailRow = (
                  <TableRow
                    key={`detail-${entry.id}`}
                    className="hover:bg-transparent"
                  >
                    <TableCell colSpan={columnCount} className="p-0">
                      <div className="m-3 ml-8 border rounded-lg overflow-hidden bg-slate-50/40">
                        <Table>
                          <TableHeader>
                            <TableRow className="bg-slate-100/70 hover:bg-slate-100/70">
                              <TableHead className="text-[10px] font-bold uppercase text-slate-500 pl-4">
                                Akun
                              </TableHead>
                              <TableHead className="text-[10px] font-bold uppercase text-slate-500">
                                Keterangan
                              </TableHead>
                              <TableHead className="text-right text-[10px] font-bold uppercase text-slate-500">
                                Debit
                              </TableHead>
                              <TableHead className="text-right text-[10px] font-bold uppercase text-slate-500 pr-4">
                                Kredit
                              </TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {detailLoading && !lines ? (
                              Array.from({ length: 2 }).map((_, i) => (
                                <TableRow key={i}>
                                  <TableCell className="pl-4">
                                    <Skeleton className="h-4 w-40" />
                                  </TableCell>
                                  <TableCell>
                                    <Skeleton className="h-4 w-48" />
                                  </TableCell>
                                  <TableCell>
                                    <Skeleton className="h-4 w-20 ml-auto" />
                                  </TableCell>
                                  <TableCell className="pr-4">
                                    <Skeleton className="h-4 w-20 ml-auto" />
                                  </TableCell>
                                </TableRow>
                              ))
                            ) : lines && lines.length > 0 ? (
                              lines.map((line) => (
                                <TableRow
                                  key={line.id}
                                  className="border-b last:border-b-0"
                                >
                                  <TableCell className="pl-4">
                                    <span className="font-mono text-[11px] font-bold text-slate-800">
                                      {line.account_code}
                                    </span>
                                    <span className="block text-[10px] text-slate-500">
                                      {line.account_name}
                                    </span>
                                  </TableCell>
                                  <TableCell className="text-[11px] text-slate-600">
                                    {line.description || "-"}
                                  </TableCell>
                                  <TableCell className="text-right text-[11px] font-medium text-blue-600">
                                    {line.debit
                                      ? formatRupiah(line.debit)
                                      : "-"}
                                  </TableCell>
                                  <TableCell className="text-right text-[11px] font-medium text-amber-600 pr-4">
                                    {line.credit
                                      ? formatRupiah(line.credit)
                                      : "-"}
                                  </TableCell>
                                </TableRow>
                              ))
                            ) : (
                              <TableRow>
                                <TableCell
                                  colSpan={4}
                                  className="h-14 text-center text-[11px] text-slate-400"
                                >
                                  Tidak ada baris rincian pada jurnal ini.
                                </TableCell>
                              </TableRow>
                            )}
                            <TableRow className="bg-slate-100/70 font-bold hover:bg-slate-100/70">
                              <TableCell
                                colSpan={2}
                                className="text-[11px] text-slate-700 pl-4"
                              >
                                TOTAL
                              </TableCell>
                              <TableCell className="text-right text-[11px] text-blue-700">
                                {formatRupiah(entry.total_debit)}
                              </TableCell>
                              <TableCell className="text-right text-[11px] text-amber-700 pr-4">
                                {formatRupiah(entry.total_credit)}
                              </TableCell>
                            </TableRow>
                          </TableBody>
                        </Table>

                        {(entry.source_type ||
                          entry.void_reason ||
                          entry.posted_at) && (
                          <div className="flex flex-wrap items-center gap-3 px-4 py-2 border-t bg-white text-[10px] text-slate-500">
                            {entry.source_type && (
                              <span>
                                Sumber:{" "}
                                <span className="font-bold text-slate-600">
                                  {entry.source_type}
                                </span>
                              </span>
                            )}
                            {entry.posted_at && (
                              <span>
                                Diposting:{" "}
                                <span className="font-bold text-slate-600">
                                  {new Date(entry.posted_at).toLocaleString(
                                    "id-ID"
                                  )}
                                </span>
                              </span>
                            )}
                            {entry.void_reason && (
                              <span className="flex items-center gap-1 text-rose-600">
                                <AlertTriangle className="size-3" /> Alasan
                                void:{" "}
                                <span className="font-bold">
                                  {entry.void_reason}
                                </span>
                              </span>
                            )}
                          </div>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );

                return [mainRow, detailRow];
              })
            )}
          </TableBody>
        </Table>
      </div>

      <DataTablePagination
        page={page}
        limit={limit}
        setLimit={setLimit}
        total={total}
        rowCount={journals.length}
        hasNext={hasNext}
        hasPrev={hasPrev}
        onNextPage={handleNextPage}
        onPrevPage={handlePrevPage}
        loading={loading}
      />
    </div>
  );
}
