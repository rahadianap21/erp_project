"use client";

import { apiGet, messageFromError } from "@/lib/api-client";
import { readPermissionList } from "@/lib/permissions";
import React, { useState } from "react";
import Link from "next/link";
import Cookies from "js-cookie";
import { toast } from "sonner";
import {
  BookOpenText,
  ChevronRight,
  ExternalLink,
  PackageCheck,
  ListOrdered,
  Loader2,
} from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { useDebounceFetch } from "@/hooks/use-api-list";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import { DataTablePagination } from "@/components/data-table-pagination";

export interface GRNDetail {
  id: number;
  product_id: number;
  sku: string;
  product_name: string;
  purchase_unit_kode: string;
  qty_received: number;
  conversion_rate: number;
}

export interface GoodsReceiptNote {
  id: number;
  grn_number: string;
  purchase_order_id: number;
  po_number?: string;
  received_date: string;
  total_qty: number;
  notes?: string;
  created_at: string;
  items?: GRNDetail[];
}

interface GoodsReceiptNoteTableProps {
  refreshTrigger: number;
  onForbidden?: () => void;
}

/**
 * Tautan ke jurnal otomatis GRN (Debit Persediaan + PPN Masukan, Kredit
 * Hutang Usaha). Jurnal dicari lewat nomor GRN karena nomor itu tersimpan
 * sebagai source_reference sekaligus muncul di deskripsi jurnal.
 */
function GRNJournalLink({ grnNumber }: { grnNumber: string }) {
  return (
    <div className="flex flex-col gap-2 rounded-lg border border-blue-200 bg-blue-50 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <p className="flex items-center gap-1.5 text-sm font-medium text-blue-900">
          <BookOpenText className="size-4" /> Jurnal persediaan &amp; hutang
        </p>
        <p className="mt-0.5 text-sm text-blue-700">
          Debit persediaan/PPN masukan, kredit hutang usaha untuk GRN ini.
        </p>
      </div>
      <Link
        href={`/accounting/journals?search=${encodeURIComponent(
          grnNumber
        )}&autoExpand=1`}
        target="_blank"
        rel="noopener noreferrer"
        onClick={(e) => e.stopPropagation()}
        className="inline-flex shrink-0 cursor-pointer items-center gap-1 rounded-lg border border-blue-300 bg-white px-2.5 py-1.5 text-sm font-medium text-blue-700 hover:bg-blue-100"
      >
        Buka Jurnal <ExternalLink className="size-3.5" />
      </Link>
    </div>
  );
}

/**
 * Tabel riwayat penerimaan barang (GRN).
 *
 * GRN immutable, jadi tabel ini read-only: tidak ada tombol ubah atau hapus.
 * Detail barang ditampilkan sebagai expanded row in-line (pola yang sama dengan
 * modul Purchase Orders) dan baru diambil saat baris dibuka.
 */
export function GoodsReceiptNoteTable({
  refreshTrigger,
  onForbidden,
}: GoodsReceiptNoteTableProps) {
  // State kontrol expanded row + lazy loading detail barang per GRN.
  const [expandedRows, setExpandedRows] = useState<Record<number, boolean>>({});
  const [rowItems, setRowItems] = useState<Record<number, GRNDetail[]>>({});
  const [rowLoading, setRowLoading] = useState<Record<number, boolean>>({});

  const authToken = Cookies.get("session_active");
  const canViewJournal = React.useMemo(
    () => (readPermissionList() ?? []).includes("accounting:read"),
    []
  );

  const {
    search,
    setSearch,
    page,
    limit,
    setLimit,
    data: grns,
    total,
    loading,
    hasNext,
    hasPrev,
    handleNextPage,
    handlePrevPage,
    httpStatus,
  } = useDebounceFetch<GoodsReceiptNote>({
    endpoint: "/api/goods-receipt-notes",
    delay: 500,
    refreshTrigger,
  });

  // Teruskan sinyal 403 dari hook ke halaman induk agar ForbiddenView tampil.
  React.useEffect(() => {
    if (httpStatus === 403) onForbidden?.();
  }, [httpStatus, onForbidden]);

  const toggleRow = async (grn: GoodsReceiptNote) => {
    const id = grn.id;
    const isCurrentlyExpanded = !!expandedRows[id];

    setExpandedRows((prev) => ({ ...prev, [id]: !isCurrentlyExpanded }));

    // Lazy fetch: detail barang hanya ditarik saat baris pertama kali dibuka.
    if (!isCurrentlyExpanded && !rowItems[id] && authToken) {
      setRowLoading((prev) => ({ ...prev, [id]: true }));
      try {
        const result = await apiGet<{ items?: GRNDetail[] }>(
          `/api/goods-receipt-notes?id=${id}`
        );
        setRowItems((prev) => ({ ...prev, [id]: result.data?.items ?? [] }));
      } catch (error) {
        toast.error("Gagal Memuat Detail", {
          description: messageFromError(error),
        });
      } finally {
        setRowLoading((prev) => ({ ...prev, [id]: false }));
      }
    }
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr) return "-";
    return new Date(dateStr).toLocaleDateString("id-ID", {
      day: "numeric",
      month: "short",
      year: "numeric",
    });
  };

  return (
    <div className="space-y-4">
      <DataTableToolbar
        search={search}
        setSearch={setSearch}
        placeholder="Cari nomor GRN atau nomor PO..."
        loading={loading}
        counterIcon={<PackageCheck className="size-3.5 text-slate-400" />}
        counterLabel={`Total ${total} dokumen`}
      />

      {/* Mobile: expandable card list */}
      <div className="space-y-3 md:hidden">
        {loading && grns.length === 0 ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div
              key={`grn-card-skeleton-${i}`}
              className="space-y-3 rounded-xl border bg-white p-4 shadow-3xs"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1.5">
                  <Skeleton className="h-4 w-36" />
                  <Skeleton className="h-3 w-24" />
                </div>
                <Skeleton className="h-5 w-16 rounded-full" />
              </div>
              <Skeleton className="h-3 w-40" />
            </div>
          ))
        ) : grns.length === 0 ? (
          <div className="rounded-xl border bg-white p-8 text-center text-sm text-slate-400">
            Belum ada dokumen penerimaan barang.
          </div>
        ) : (
          grns.map((grn) => {
            const isExpanded = !!expandedRows[grn.id];
            const items = rowItems[grn.id] || [];
            const isRowLoading = !!rowLoading[grn.id];

            return (
              <article
                key={grn.id}
                onClick={() => toggleRow(grn)}
                className={`cursor-pointer rounded-xl border bg-white p-4 shadow-3xs transition-colors ${isExpanded ? "border-slate-300 bg-slate-50/60" : ""
                  } ${loading ? "pointer-events-none opacity-50" : ""}`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-start gap-2">
                    <ChevronRight
                      className={`mt-0.5 size-3.5 shrink-0 text-slate-400 transition-transform duration-300 ${isExpanded ? "rotate-90 " : ""
                        }`}
                    />
                    <div className="min-w-0">
                      <h3 className="truncate font-mono text-sm font-medium ">
                        {grn.grn_number}
                      </h3>
                      <p className="mt-0.5 truncate font-mono text-sm text-slate-500">
                        PO: {grn.po_number || `#${grn.purchase_order_id}`}
                      </p>
                    </div>
                  </div>
                  <span className="shrink-0 rounded-lg border border-emerald-200 bg-emerald-50 px-2 py-0.5 font-mono text-sm font-bold ">
                    {grn.total_qty} qty
                  </span>
                </div>

                <dl className="mt-3 grid grid-cols-2 gap-2 rounded-lg bg-slate-50 p-2.5 text-sm">
                  <div>
                    <dt className="text-slate-400">Tanggal Terima</dt>
                    <dd className="mt-0.5 font-medium text-slate-700">
                      {formatDate(grn.received_date)}
                    </dd>
                  </div>
                  <div className="min-w-0">
                    <dt className="text-slate-400">Catatan</dt>
                    <dd className="mt-0.5 truncate text-slate-700">
                      {grn.notes || "-"}
                    </dd>
                  </div>
                </dl>

                <div
                  className={`grid transition-all duration-300 ease-in-out ${isExpanded
                    ? "grid-rows-[1fr] opacity-100"
                    : "grid-rows-[0fr] opacity-0"
                    }`}
                >
                  <div className="overflow-hidden">
                    <div className="mt-3 space-y-2 border-t pt-3">
                      <div className="flex items-center gap-1.5 text-sm font-medium tracking-wide text-muted-foreground">
                        <ListOrdered className="size-3.5 text-emerald-500" />
                        Rincian Barang Diterima
                      </div>

                      {isRowLoading ? (
                        <div className="flex items-center gap-1.5 py-3 text-sm text-slate-400">
                          <Loader2 className="size-3 animate-spin " />
                          <span>Memuat rincian penerimaan...</span>
                        </div>
                      ) : items.length > 0 ? (
                        <ul className="space-y-2">
                          {items.map((item) => (
                            <li
                              key={`m-subitem-grn-${item.id}`}
                              className="rounded-lg border bg-white p-2.5 text-sm"
                            >
                              <p className="font-medium tracking-tight text-foreground">
                                {item.product_name}
                              </p>
                              <p className="font-mono text-sm text-slate-400">
                                SKU: {item.sku}
                              </p>
                              <div className="mt-1.5 flex items-center justify-between font-mono">
                                <span className="font-bold ">
                                  {item.qty_received}{" "}
                                  {item.purchase_unit_kode}
                                </span>
                                <span className="text-sm text-slate-500">
                                  Konversi:{" "}
                                  {item.conversion_rate > 1
                                    ? `1 : ${item.conversion_rate}`
                                    : "1 : 1"}
                                </span>
                              </div>
                            </li>
                          ))}
                        </ul>
                      ) : (
                        <p className="text-sm italic text-slate-400">
                          Tidak ada rincian barang pada dokumen ini.
                        </p>
                      )}

                      {grn.notes && (
                        <div className="rounded-lg border bg-white px-3 py-2 text-sm text-slate-600">
                          <strong className="mb-0.5 block text-sm uppercase tracking-wide ">
                            Catatan Penerimaan:
                          </strong>
                          &quot;{grn.notes}&quot;
                        </div>
                      )}

                      {canViewJournal && (
                        <GRNJournalLink grnNumber={grn.grn_number} />
                      )}
                    </div>
                  </div>
                </div>
              </article>
            );
          })
        )}
      </div>

      {/* Desktop: table */}
      <div className="hidden bg-white rounded-xl border shadow-3xs overflow-hidden md:block">
        <Table>
          <TableHeader className="bg-slate-50/70 border-b">
            <TableRow>
              <TableHead className="w-10"></TableHead>
              <TableHead>
                Nomor GRN
              </TableHead>
              <TableHead>
                Nomor PO
              </TableHead>
              <TableHead className="text-center">
                Tanggal Terima
              </TableHead>
              <TableHead className="text-center w-28">
                Total Qty
              </TableHead>
              <TableHead>
                Catatan
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={`grn-skeleton-${i}`} className="border-b">
                  <TableCell className="w-10 py-3.5">
                    <Skeleton className="h-4 w-4 rounded mx-auto" />
                  </TableCell>
                  <TableCell className="py-3.5">
                    <Skeleton className="h-4 w-40 rounded" />
                  </TableCell>
                  <TableCell className="py-3.5">
                    <Skeleton className="h-4 w-32 rounded" />
                  </TableCell>
                  <TableCell className="py-3.5">
                    <Skeleton className="h-4 w-28 rounded mx-auto" />
                  </TableCell>
                  <TableCell className="py-3.5">
                    <Skeleton className="h-4 w-16 rounded mx-auto" />
                  </TableCell>
                  <TableCell className="py-3.5">
                    <Skeleton className="h-4 w-24 rounded" />
                  </TableCell>
                </TableRow>
              ))
            ) : grns.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="h-32 text-center text-sm text-slate-400"
                >
                  Belum ada dokumen penerimaan barang.
                </TableCell>
              </TableRow>
            ) : (
              grns.map((grn) => {
                const isExpanded = !!expandedRows[grn.id];
                const items = rowItems[grn.id] || [];
                const isRowLoading = !!rowLoading[grn.id];

                return (
                  <React.Fragment key={grn.id}>
                    <TableRow
                      onClick={() => toggleRow(grn)}
                      className={`cursor-pointer transition-colors border-b-0 hover:bg-slate-50/70 ${isExpanded ? "bg-slate-50/50 border-b-0" : "border-b"
                        } ${loading ? "opacity-50 pointer-events-none" : ""}`}
                    >
                      <TableCell className="py-3 text-center w-10">
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleRow(grn);
                          }}
                          className="p-1 rounded hover:bg-slate-100 transition-colors cursor-pointer"
                          title={
                            isExpanded
                              ? "Tutup rincian"
                              : "Lihat rincian barang"
                          }
                        >
                          <ChevronRight
                            className={`size-3.5 transition-transform duration-300 ease-in-out ${isExpanded
                              ? "transform rotate-90  font-bold"
                              : ""
                              }`}
                          />
                        </button>
                      </TableCell>
                      <TableCell className="py-3 font-mono font-bold  text-sm">
                        {grn.grn_number}
                      </TableCell>
                      <TableCell className="py-3 font-mono text-sm text-slate-600">
                        {grn.po_number || `#${grn.purchase_order_id}`}
                      </TableCell>
                      <TableCell className="py-3 text-center text-sm text-slate-600">
                        {formatDate(grn.received_date)}
                      </TableCell>
                      <TableCell className="py-3 text-center font-mono font-bold text-sm text-slate-900">
                        {grn.total_qty}
                      </TableCell>
                      <TableCell className="py-3 text-sm text-slate-500 max-w-[16rem] truncate">
                        {grn.notes || "-"}
                      </TableCell>
                    </TableRow>

                    {/* Sub-row expansion in-line: rincian barang diterima */}
                    <TableRow
                      className={`border-t-0 p-0 hover:bg-transparent ${isExpanded
                        ? "bg-slate-50/50 border-b"
                        : "pointer-events-none"
                        }`}
                    >
                      <TableCell colSpan={6} className="p-0 border-t-0">
                        <div
                          className={`grid transition-all duration-300 ease-in-out ${isExpanded
                            ? "grid-rows-[1fr] opacity-100"
                            : "grid-rows-[0fr] opacity-0"
                            }`}
                        >
                          <div className="overflow-hidden">
                            <div className="py-4 pl-12 pr-6 space-y-4 bg-slate-50/40 border-t border-slate-100 shadow-inner w-full">
                              <div className="text-sm font-medium text-muted-foreground flex items-center gap-1.5">
                                <ListOrdered className="size-3.5 text-emerald-500" />
                                Rincian Barang Diterima
                              </div>

                              {isRowLoading ? (
                                <div className="text-sm text-slate-400 py-3 flex items-center gap-1.5">
                                  <Loader2 className="size-3 animate-spin " />
                                  <span>Memuat rincian penerimaan...</span>
                                </div>
                              ) : items.length > 0 ? (
                                <div className="border bg-white rounded-lg overflow-hidden shadow-3xs w-full">
                                  <Table>
                                    <TableHeader className="bg-slate-50/60 border-b">
                                      <TableRow className="h-7 text-sm">
                                        <TableHead className="py-1">
                                          Nama Barang
                                        </TableHead>
                                        <TableHead className="py-1 text-center w-32">
                                          Diterima
                                        </TableHead>
                                        <TableHead className="py-1 text-center w-32">
                                          Konversi
                                        </TableHead>
                                      </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                      {items.map((item) => (
                                        <TableRow
                                          key={`subitem-grn-${item.id}`}
                                          className="h-9 text-sm hover:bg-slate-50/40"
                                        >
                                          <TableCell className="py-1.5 font-medium text-slate-800">
                                            <span className="font-medium tracking-tight block text-foreground">
                                              {item.product_name}
                                            </span>
                                            <span className="text-sm font-mono text-slate-400">
                                              SKU: {item.sku}
                                            </span>
                                          </TableCell>
                                          <TableCell className="py-1.5 text-center font-mono font-bold ">
                                            {item.qty_received}{" "}
                                            {item.purchase_unit_kode}
                                          </TableCell>
                                          <TableCell className="py-1.5 text-center font-mono text-sm text-slate-500">
                                            {item.conversion_rate > 1
                                              ? `1 : ${item.conversion_rate}`
                                              : "1 : 1"}
                                          </TableCell>
                                        </TableRow>
                                      ))}
                                    </TableBody>
                                  </Table>
                                </div>
                              ) : (
                                <p className="text-sm text-slate-400 italic">
                                  Tidak ada rincian barang pada dokumen ini.
                                </p>
                              )}

                              {grn.notes && (
                                <div className="text-sm bg-white border px-3 py-2 rounded-lg text-slate-600 w-full">
                                  <strong className=" uppercase text-sm tracking-wide block mb-0.5">
                                    Catatan Penerimaan:
                                  </strong>
                                  &quot;{grn.notes}&quot;
                                </div>
                              )}

                              {canViewJournal && (
                                <GRNJournalLink grnNumber={grn.grn_number} />
                              )}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  </React.Fragment>
                );
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
        rowCount={grns.length}
        hasNext={hasNext}
        hasPrev={hasPrev}
        onNextPage={handleNextPage}
        onPrevPage={handlePrevPage}
        loading={loading}
      />
    </div>
  );
}

export default GoodsReceiptNoteTable;
