"use client";

import { apiGet } from "@/lib/api-client";
import { useEffect, useRef, useState } from "react";
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
import { formatIDR } from "@/lib/money";

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
  /**
   * Pencarian awal dari tautan luar (mis. dari detail penjualan). Dipakai agar
   * pengguna langsung melihat jurnal milik transaksi yang ia buka.
   */
  initialSearch?: string;
  /** Id jurnal yang langsung dibuka (auto-expand) saat halaman dimuat. */
  focusId?: number | null;
  /**
   * Buka otomatis bila hasil pencarian dari tautan hanya menyisakan satu
   * jurnal. Dipakai tautan GRN/PO yang tidak membawa id jurnal.
   */
  autoExpandSingle?: boolean;
}

const STATUS_VARIANT = {
  DRAFT: "neutral",
  POSTED: "success",
  VOID: "danger",
} as const;

export function JournalTable({
  canPost,
  canVoidReverse,
  refreshTrigger,
  onPostClick,
  onVoidClick,
  onReverseClick,
  initialSearch,
  focusId,
  autoExpandSingle,
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
  const loadDetailLines = async (entryId: number) => {
    if (detailLines[entryId]) return;
    setDetailLoading(true);
    try {
      const result = await apiGet<{ lines?: JournalLine[] }>(
        `/api/journals?id=${entryId}`
      );
      const lines = result.data?.lines;
      if (lines) {
        setDetailLines((prev) => ({ ...prev, [entryId]: lines }));
      }
    } catch (err) {
      console.error("Gagal memuat rincian jurnal:", err);
    } finally {
      setDetailLoading(false);
    }
  };

  const toggleRow = async (entry: JournalEntry) => {
    if (expandedId === entry.id) {
      setExpandedId(null);
      return;
    }
    setExpandedId(entry.id);
    await loadDetailLines(entry.id);
  };

  // Terapkan pencarian dari tautan luar sekali saja. Setelah itu kolom
  // pencarian kembali sepenuhnya dikendalikan pengguna.
  const initialSearchApplied = useRef(false);
  useEffect(() => {
    if (initialSearchApplied.current || !initialSearch) return;
    initialSearchApplied.current = true;
    setSearch(initialSearch);
  }, [initialSearch, setSearch]);

  // Buka otomatis jurnal yang dituju tautan begitu entri muncul di daftar
  // (daftar baru terisi setelah pencarian dari tautan selesai dimuat).
  const focusHandled = useRef(false);
  useEffect(() => {
    if (focusHandled.current || !focusId) return;
    if (!journals.some((entry) => entry.id === focusId)) return;
    focusHandled.current = true;
    setExpandedId(focusId);
    void loadDetailLines(focusId);
    // loadDetailLines dan detailLines sengaja tidak masuk dependensi: efek ini
    // hanya perlu berjalan sekali saat entri tujuan sudah tersedia.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [journals, focusId]);

  // Tautan GRN/PO tidak membawa id jurnal, hanya nomor dokumen. Bila hasil
  // pencariannya tunggal dan deskripsinya memang memuat nomor tersebut, buka
  // baris itu otomatis. Pengecekan deskripsi mencegah baris lama ikut terbuka
  // saat daftar hasil pencarian belum selesai dimuat.
  const autoExpandHandled = useRef(false);
  useEffect(() => {
    if (!autoExpandSingle || autoExpandHandled.current) return;
    if (!initialSearch || !initialSearchApplied.current) return;
    if (journals.length !== 1) return;
    const only = journals[0];
    const needle = initialSearch.toLowerCase();
    const haystack = (only.description ?? "").toLowerCase();
    if (!haystack.includes(needle)) return;
    autoExpandHandled.current = true;
    setExpandedId(only.id);
    void loadDetailLines(only.id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoExpandSingle, initialSearch, journals]);

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
          <div className="flex items-center gap-1.5 text-sm bg-slate-100/80 px-2.5 py-1.5 rounded-lg border text-slate-600 font-medium">
            <Filter className="size-3" /> Status Jurnal:
          </div>
          <select
            value={status}
            onChange={(e) => updateFilter("status", e.target.value)}
            className="text-sm h-9 px-3 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer focus:border-blue-500 font-medium text-slate-700"
          >
            <option value="ALL">Semua Status</option>
            <option value="DRAFT">DRAFT</option>
            <option value="POSTED">POSTED</option>
            <option value="VOID">VOID</option>
          </select>
        </div>
      </DataTableToolbar>

      {/* Mobile: expandable card list */}
      <div className="space-y-3 md:hidden">
        {loading && journals.length === 0 ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div
              key={`journal-card-skel-${i}`}
              className="space-y-3 rounded-xl border bg-white p-4 shadow-xs"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1.5">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-3 w-24" />
                </div>
                <Skeleton className="h-5 w-16 rounded-full" />
              </div>
              <Skeleton className="h-3 w-48" />
            </div>
          ))
        ) : journals.length === 0 ? (
          <div className="rounded-xl border bg-white p-8 text-center text-slate-400">
            <div className="flex flex-col items-center gap-1.5">
              <FileText className="size-5" />
              <span className="text-sm font-medium">
                Belum ada entri jurnal yang tersimpan.
              </span>
            </div>
          </div>
        ) : (
          journals.map((entry) => {
            const isOpen = expandedId === entry.id;
            const lines = detailLines[entry.id];

            return (
              <article
                key={entry.id}
                onClick={() => toggleRow(entry)}
                className={`cursor-pointer rounded-xl border bg-white p-4 shadow-xs transition-colors ${isOpen ? "border-blue-200 bg-blue-50/40" : ""
                  }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-start gap-2">
                    <ChevronRight
                      className={`mt-0.5 size-3.5 shrink-0 text-slate-400 transition-transform duration-200 ${isOpen ? "rotate-90" : ""
                        }`}
                    />
                    <div className="min-w-0">
                      <h3 className="truncate font-mono text-sm font-medium text-foreground">
                        {entry.entry_number}
                      </h3>
                      <p className="mt-0.5 text-sm text-slate-500">
                        {new Date(entry.entry_date).toLocaleDateString(
                          "id-ID",
                          {
                            day: "2-digit",
                            month: "short",
                            year: "numeric",
                          }
                        )}{" "}
                        · {entry.period_year}-
                        {String(entry.period_month).padStart(2, "0")}
                      </p>
                    </div>
                  </div>
                  <Badge variant={STATUS_VARIANT[entry.status]} className="shrink-0">
                    {entry.status}
                  </Badge>
                </div>

                <div className="mt-2.5 rounded-lg bg-slate-50 p-2.5">
                  <p className="line-clamp-2 text-sm text-slate-700">
                    {entry.description || "-"}
                  </p>
                  {entry.reversal_of_id && (
                    <p className="mt-0.5 text-sm font-bold text-rose-600">
                      Pembalik dari JV#{entry.reversal_of_id}
                    </p>
                  )}
                  <div className="mt-1.5 flex items-baseline justify-between">
                    <span className="text-sm uppercase text-slate-400">
                      Nominal
                    </span>
                    <span className="font-mono text-sm font-bold text-slate-800">
                      {formatIDR(entry.total_debit)}
                    </span>
                  </div>
                </div>

                <div
                  className={`grid transition-all duration-300 ease-in-out ${isOpen
                    ? "grid-rows-[1fr] opacity-100"
                    : "grid-rows-[0fr] opacity-0"
                    }`}
                >
                  <div className="overflow-hidden">
                    <div className="mt-3 space-y-2 border-t pt-3">
                      {detailLoading && !lines ? (
                        <div className="space-y-2">
                          <Skeleton className="h-10 w-full rounded-lg" />
                          <Skeleton className="h-10 w-full rounded-lg" />
                        </div>
                      ) : lines && lines.length > 0 ? (
                        <ul className="space-y-2">
                          {lines.map((line) => (
                            <li
                              key={line.id}
                              className="rounded-lg border bg-white p-2.5 text-sm"
                            >
                              <div className="flex items-start justify-between gap-2">
                                <div className="min-w-0">
                                  <p className="font-mono font-bold text-slate-800">
                                    {line.account_code}
                                  </p>
                                  <p className="truncate text-sm text-slate-500">
                                    {line.account_name}
                                  </p>
                                </div>
                                <div className="shrink-0 text-right font-mono">
                                  {line.debit > 0 && (
                                    <p className="font-medium text-blue-600">
                                      D {formatIDR(line.debit)}
                                    </p>
                                  )}
                                  {line.credit > 0 && (
                                    <p className="font-medium text-amber-600">
                                      K {formatIDR(line.credit)}
                                    </p>
                                  )}
                                </div>
                              </div>
                              {line.description && (
                                <p className="mt-1 truncate text-sm text-slate-500">
                                  {line.description}
                                </p>
                              )}
                            </li>
                          ))}
                        </ul>
                      ) : (
                        <p className="text-sm text-slate-400">
                          Tidak ada baris rincian pada jurnal ini.
                        </p>
                      )}

                      <div className="flex items-center justify-between rounded-lg bg-slate-100/70 px-2.5 py-2 font-mono text-sm font-bold">
                        <span className="font-sans text-slate-700">TOTAL</span>
                        <span className="text-blue-700">
                          D {formatIDR(entry.total_debit)}
                        </span>
                        <span className="text-amber-700">
                          K {formatIDR(entry.total_credit)}
                        </span>
                      </div>

                      {(entry.source_type ||
                        entry.void_reason ||
                        entry.posted_at) && (
                          <div className="space-y-1 rounded-lg border bg-white px-3 py-2 text-sm text-slate-500">
                            {entry.source_type && (
                              <p>
                                Sumber:{" "}
                                <span className="font-bold text-slate-600">
                                  {entry.source_type}
                                </span>
                              </p>
                            )}
                            {entry.posted_at && (
                              <p>
                                Diposting:{" "}
                                <span className="font-bold text-slate-600">
                                  {new Date(entry.posted_at).toLocaleString(
                                    "id-ID"
                                  )}
                                </span>
                              </p>
                            )}
                            {entry.void_reason && (
                              <p className="flex items-center gap-1 text-rose-600">
                                <AlertTriangle className="size-3" /> Alasan void:{" "}
                                <span className="font-bold">
                                  {entry.void_reason}
                                </span>
                              </p>
                            )}
                          </div>
                        )}
                    </div>
                  </div>
                </div>

                {showActions && (
                  <div className="mt-3 flex flex-wrap justify-end gap-1.5 border-t pt-3">
                    {canPost && entry.status === "DRAFT" && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 cursor-pointer border-emerald-200 text-sm text-emerald-700 hover:bg-emerald-50"
                        onClick={(e) => {
                          e.stopPropagation();
                          onPostClick(entry);
                        }}
                      >
                        <Send className="mr-1 size-3" /> Post
                      </Button>
                    )}
                    {canVoidReverse && entry.status === "DRAFT" && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 cursor-pointer border-rose-200 text-sm text-rose-600 hover:bg-rose-50"
                        onClick={(e) => {
                          e.stopPropagation();
                          onVoidClick(entry);
                        }}
                      >
                        <Ban className="mr-1 size-3" /> Void
                      </Button>
                    )}
                    {canVoidReverse && entry.status === "POSTED" && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 cursor-pointer border-red-200 text-sm text-red-600 hover:bg-red-50"
                        onClick={(e) => {
                          e.stopPropagation();
                          onReverseClick(entry);
                        }}
                      >
                        <RotateCcw className="mr-1 size-3" /> Reverse
                      </Button>
                    )}
                  </div>
                )}
              </article>
            );
          })
        )}
      </div>

      {/* Desktop: table */}
      <div className="hidden border rounded-xl shadow-xs overflow-hidden bg-white relative md:block">
        {loading && journals.length > 0 && (
          <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] z-10 flex items-center justify-center">
            <Skeleton className="h-5 w-40" />
          </div>
        )}

        <Table>
          <TableHeader>
            <TableRow className="bg-slate-50/80 hover:bg-slate-50/80">
              <TableHead>
                No. Jurnal
              </TableHead>
              <TableHead>
                Tanggal
              </TableHead>
              <TableHead>
                Keterangan
              </TableHead>
              <TableHead>
                Periode
              </TableHead>
              <TableHead className="text-right">
                Nominal
              </TableHead>
              <TableHead>
                Status
              </TableHead>
              {showActions && (
                <TableHead className="text-right">
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
                    <span className="text-sm font-medium">
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
                        <span className="font-mono text-sm font-bold text-slate-800">
                          {entry.entry_number}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-slate-600">
                      {new Date(entry.entry_date).toLocaleDateString("id-ID", {
                        day: "2-digit",
                        month: "short",
                        year: "numeric",
                      })}
                    </TableCell>
                    <TableCell className="text-sm text-slate-700 max-w-[260px]">
                      <span className="line-clamp-1">
                        {entry.description || "-"}
                      </span>
                      {entry.reversal_of_id && (
                        <span className="block text-sm font-bold text-rose-600">
                          Pembalik dari JV#{entry.reversal_of_id}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-sm font-mono text-slate-500">
                      {entry.period_year}-
                      {String(entry.period_month).padStart(2, "0")}
                    </TableCell>
                    <TableCell className="text-right">
                      <span className="text-sm font-bold text-slate-800">
                        {formatIDR(entry.total_debit)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge variant={STATUS_VARIANT[entry.status]}>
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
                              className="h-7 text-sm cursor-pointer text-emerald-700 border-emerald-200 hover:bg-emerald-50"
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
                              className="h-7 text-sm cursor-pointer text-rose-600 border-rose-200 hover:bg-rose-50"
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
                              className="h-7 text-sm cursor-pointer text-rose-600 border-rose-200 hover:bg-rose-50"
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

                const detailRow = (
                  <TableRow
                    key={`detail-${entry.id}`}
                    className={`border-b-0 border-t-0 p-0 hover:bg-transparent transition-all duration-300 ease-in-out ${isOpen ? "border-b bg-slate-50/30" : "pointer-events-none"}`}
                  >
                    <TableCell colSpan={columnCount} className="border-t-0 p-0">
                      <div
                        className={`grid transition-all duration-300 ease-in-out ${isOpen ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"}`}
                      >
                        <div className="overflow-hidden">
                          <div className="m-3 ml-8 border rounded-lg overflow-hidden bg-slate-50/40">
                            <Table>
                              <TableHeader>
                                <TableRow className="bg-slate-100/70 hover:bg-slate-100/70">
                                  <TableHead className="pl-4">
                                    Akun
                                  </TableHead>
                                  <TableHead>
                                    Keterangan
                                  </TableHead>
                                  <TableHead className="text-right">
                                    Debit
                                  </TableHead>
                                  <TableHead className="text-right pr-4">
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
                                        <span className="font-mono text-sm font-bold text-slate-800">
                                          {line.account_code}
                                        </span>
                                        <span className="block text-sm text-slate-500">
                                          {line.account_name}
                                        </span>
                                      </TableCell>
                                      <TableCell className="text-sm text-slate-600">
                                        {line.description || "-"}
                                      </TableCell>
                                      <TableCell className="text-right text-sm font-medium text-blue-600">
                                        {line.debit
                                          ? formatIDR(line.debit)
                                          : "-"}
                                      </TableCell>
                                      <TableCell className="text-right text-sm font-medium text-amber-600 pr-4">
                                        {line.credit
                                          ? formatIDR(line.credit)
                                          : "-"}
                                      </TableCell>
                                    </TableRow>
                                  ))
                                ) : (
                                  <TableRow>
                                    <TableCell
                                      colSpan={4}
                                      className="h-14 text-center text-sm text-slate-400"
                                    >
                                      Tidak ada baris rincian pada jurnal ini.
                                    </TableCell>
                                  </TableRow>
                                )}
                                <TableRow className="bg-slate-100/70 font-bold hover:bg-slate-100/70">
                                  <TableCell
                                    colSpan={2}
                                    className="text-sm text-slate-700 pl-4"
                                  >
                                    TOTAL
                                  </TableCell>
                                  <TableCell className="text-right text-sm text-blue-700">
                                    {formatIDR(entry.total_debit)}
                                  </TableCell>
                                  <TableCell className="text-right text-sm text-amber-700 pr-4">
                                    {formatIDR(entry.total_credit)}
                                  </TableCell>
                                </TableRow>
                              </TableBody>
                            </Table>

                            {(entry.source_type ||
                              entry.void_reason ||
                              entry.posted_at) && (
                                <div className="flex flex-wrap items-center gap-3 px-4 py-2 border-t bg-white text-sm text-slate-500">
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
                        </div>
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
