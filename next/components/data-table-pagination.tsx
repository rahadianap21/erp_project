"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";

interface DataTablePaginationProps {
  page: number;
  limit: number;
  setLimit: (limit: number) => void;
  total: number;
  rowCount: number;
  hasNext: boolean;
  hasPrev: boolean;
  onNextPage: () => void;
  onPrevPage: () => void;
  loading?: boolean;
}

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

/**
 * Pagination standar untuk semua tabel: selector "Baris per halaman" + info
 * "Menampilkan X data dari Y total record" di kiri, navigasi di kanan. Dipakai
 * seragam oleh seluruh modul.
 */
export function DataTablePagination({
  page,
  limit,
  setLimit,
  total,
  rowCount,
  hasNext,
  hasPrev,
  onNextPage,
  onPrevPage,
  loading = false,
}: DataTablePaginationProps) {
  return (
    <div className="flex flex-col md:flex-row items-center justify-between gap-4 bg-white p-4 rounded-xl border shadow-3xs mt-4">
      <div className="flex flex-wrap items-center gap-4 order-2 md:order-1 w-full md:w-auto justify-between md:justify-start">
        <div className="flex items-center gap-2">
          <label
            htmlFor="rows-per-page"
            className="text-xs text-slate-500 font-medium whitespace-nowrap"
          >
            Baris per halaman:
          </label>
          <select
            id="rows-per-page"
            value={limit}
            disabled={loading}
            onChange={(e) => setLimit(Number(e.target.value))}
            suppressHydrationWarning
            className="h-8 rounded-lg border border-slate-200 bg-white px-2 text-xs text-slate-900 font-medium outline-none cursor-pointer focus:border-blue-500 disabled:opacity-50"
          >
            {PAGE_SIZE_OPTIONS.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </div>

        <p className="text-xs text-slate-500 font-medium">
          Menampilkan{" "}
          <strong className="text-slate-900 font-semibold">{rowCount}</strong>{" "}
          data dari{" "}
          <strong className="text-slate-900 font-semibold">{total}</strong>{" "}
          total record
        </p>
      </div>

      <div className="flex items-center gap-2 order-1 md:order-2 w-full md:w-auto justify-center md:justify-end">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onPrevPage}
          disabled={!hasPrev || loading}
          className="h-8 px-3 text-xs font-medium cursor-pointer flex items-center gap-1"
        >
          <ChevronLeft className="size-3.5" />
          <span>Prev</span>
        </Button>

        <div className="px-3 py-1 bg-blue-600/10 text-blue-600 border border-blue-600/20 rounded-lg text-xs font-semibold">
          {page}
        </div>

        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onNextPage}
          disabled={!hasNext || loading}
          className="h-8 px-3 text-xs font-medium cursor-pointer flex items-center gap-1"
        >
          <span>Next</span>
          <ChevronRight className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}

export default DataTablePagination;
