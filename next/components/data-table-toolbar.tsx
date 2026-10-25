"use client";

import type { ReactNode } from "react";
import { Loader2, Search, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface DataTableToolbarProps {
  search: string;
  setSearch: (value: string) => void;
  placeholder?: string;
  loading?: boolean;
  counterLabel?: string;
  counterIcon?: ReactNode;
  isFiltered?: boolean;
  onReset?: () => void;
  children?: ReactNode;
}

/**
 * Toolbar standar untuk semua tabel daftar: search di kiri, aksi filter/reset
 * di tengah, dan counter total di kanan. Dipakai seragam oleh seluruh modul
 * agar posisi search & filter konsisten.
 */
export function DataTableToolbar({
  search,
  setSearch,
  placeholder = "Cari...",
  loading = false,
  counterLabel,
  counterIcon,
  isFiltered = false,
  onReset,
  children,
}: DataTableToolbarProps) {
  return (
    <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3 bg-white p-4 rounded-xl border shadow-3xs relative">
      <div className="flex flex-1 flex-col md:flex-row items-stretch md:items-center gap-3">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-slate-400 pointer-events-none" />
          <Input
            type="text"
            placeholder={placeholder}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 pr-8 h-9 text-xs"
            suppressHydrationWarning
          />
          {loading && (
            <Loader2 className="absolute right-8 top-1/2 -translate-y-1/2 size-3.5 text-blue-600 animate-spin" />
          )}
          {search && (
            <button
              type="button"
              onClick={() => setSearch("")}
              title="Hapus pencarian"
              className="absolute right-0 top-1/2 grid size-12 -translate-y-1/2 place-items-center text-slate-400 hover:text-slate-600 cursor-pointer md:right-2.5 md:size-4"
            >
              <X className="size-3.5" />
            </button>
          )}
        </div>

        {children}

        {isFiltered && onReset && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onReset}
            className="h-9 text-xs gap-1.5 text-slate-500 cursor-pointer"
          >
            <X className="size-3.5" /> Reset Filter
          </Button>
        )}
      </div>

      {counterLabel && (
        <div className="flex items-center gap-1.5 text-xs text-slate-500 font-medium">
          {loading ? (
            <Loader2 className="size-3.5 animate-spin text-blue-600" />
          ) : (
            counterIcon
          )}
          <span>{counterLabel}</span>
        </div>
      )}
    </div>
  );
}

export default DataTableToolbar;
