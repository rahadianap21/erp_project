"use client";

import { apiGet } from "@/lib/api-client";
import { readPermissionList } from "@/lib/permissions";
import { formatIDR } from "@/lib/money";
import Link from "next/link";
import {
  BookOpenText,
  ExternalLink,
  FileText,
  User,
  Calendar,
  DollarSign,
  Filter,
  CheckCircle2,
  XCircle,
  ChevronRight as ArrowRight,
  Loader2,
  ListOrdered,
  Percent,
  Landmark,
  Trash2,
  Edit2,
  PackageCheck,
  CheckCheck,
} from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useDebounceFetch } from "@/hooks/use-api-list";
import { POStatusTimeline } from "./po-status-timeline";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import { DataTablePagination } from "@/components/data-table-pagination";
import React, { useState } from "react";
import Cookies from "js-cookie";

export interface PurchaseOrder {
  id: number;
  po_number: string;
  supplier_id: number;
  supplier_name: string;
  supplier_pic: string;
  supplier_phone: string;
  supplier_email: string;
  po_date: string;
  tax_type: "NON_PPN" | "TAX_EXCLUSIVE" | "TAX_INCLUSIVE";
  dpp_amount: number;
  tax_amount: number;
  total_amount: number;
  // Tipe ini harus sama persis dengan CHECK constraint status di database
  // (migrasi 000005). DRAFT dan SHIPPED pernah ikut ada di sini. Keduanya
  // sudah dibuang: tidak ada alur yang menghasilkan DRAFT, dan SHIPPED
  // ditolak constraint. Daftar yang lebih panjang dari constraint hanya
  // memunculkan opsi filter dan tahap timeline yang tidak pernah berisi.
  status: "PENDING" | "APPROVED" | "RECEIVED" | "REJECTED";
  notes?: string;
  created_at: string;
}

interface DetailItem {
  id: number;
  product_name: string;
  product_sku: string;
  qty_ordered: number;
  qty_received: number;
  unit_kode: string;
  unit_price: number;
  sub_total: number;
}

interface PurchaseOrderTableProps {
  canUpdate: boolean; // 🌟 SUNTIKKAN RBAC PRIVILEGE
  canDelete: boolean; // 🌟 SUNTIKKAN RBAC PRIVILEGE
  refreshTrigger: number;
  onEditClick: (po: PurchaseOrder) => void; // Callback pemicu form edit
  onDeleteClick: (po: PurchaseOrder) => void; // Callback pemicu alert hapus
  onApproveClick?: (po: PurchaseOrder) => void;
  onCancelClick?: (po: PurchaseOrder) => void;
  onReceiveClick?: (po: PurchaseOrder) => void;
  onFinishReceiveClick?: (po: PurchaseOrder) => void;
}

/**
 * Tautan ke jurnal persediaan–hutang milik PO ini. Jurnal dibuat per GRN,
 * jadi pencarian memakai nomor PO yang tercatat di deskripsi jurnal
 * ("Penerimaan barang GRN/... dari PO PO/...").
 */
function POJournalLink({ poNumber }: { poNumber: string }) {
  return (
    <div className="flex flex-col gap-2 rounded-lg border border-blue-200 bg-blue-50 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <p className="flex items-center gap-1.5 text-sm font-medium text-blue-900">
          <BookOpenText className="size-4" /> Jurnal penerimaan barang
        </p>
        <p className="mt-0.5 text-sm text-blue-700">
          Satu jurnal per GRN: debit persediaan/PPN masukan, kredit hutang
          usaha.
        </p>
      </div>
      <Link
        href={`/accounting/journals?search=${encodeURIComponent(
          poNumber
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

export function PurchaseOrderTable({
  canUpdate,
  canDelete,
  refreshTrigger,
  onEditClick,
  onDeleteClick,
  onApproveClick,
  onCancelClick,
  onReceiveClick,
  onFinishReceiveClick,
}: PurchaseOrderTableProps) {
  const token = Cookies.get("session_active") || "";
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
    filters,
    updateFilter,
    data: purchaseOrders,
    total,
    loading,
    handleReset,
    isFiltered,
    hasNext,
    hasPrev,
    handleNextPage,
    handlePrevPage,
  } = useDebounceFetch<PurchaseOrder>({
    endpoint: "/api/purchase-orders",
    delay: 500,
    initialFilters: {
      status: "ALL", // Default filter kaku mengikuti spesifikasi select
    },
    refreshTrigger: refreshTrigger,
    // ❌ Properti mapFilters lama yang memicu error kompilasi TELAH DIHAPUS PERMANEN
  });

  // 🌟 STATE UNTUK KONTROL EXPANDED ROW & LAZY LOADING DETAIL ITEMS
  const [expandedRows, setExpandedRows] = useState<Record<number, boolean>>({});
  const [rowItems, setRowItems] = useState<Record<number, DetailItem[]>>({});
  const [rowLoading, setRowLoading] = useState<Record<number, boolean>>({});

  const toggleRow = async (poId: number) => {
    const isCurrentlyExpanded = !!expandedRows[poId];

    setExpandedRows((prev) => ({
      ...prev,
      [poId]: !isCurrentlyExpanded,
    }));

    // Lazy Fetching: Tarik data item detail dari database Go hanya jika baris belum pernah dimuat
    if (!isCurrentlyExpanded && !rowItems[poId] && token) {
      setRowLoading((prev) => ({ ...prev, [poId]: true }));
      try {
        const result = await apiGet<{ items?: DetailItem[] }>(
          `/api/purchase-orders?id=${poId}`
        );
        if (result.data?.items) {
          setRowItems((prev) => ({ ...prev, [poId]: result.data!.items! }));
        }
      } catch (err) {
        console.error("Gagal memuat sub-item detail PO:", err);
      } finally {
        setRowLoading((prev) => ({ ...prev, [poId]: false }));
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

  const getTaxBadge = (type: PurchaseOrder["tax_type"]) => {
    switch (type) {
      case "TAX_EXCLUSIVE":
        return <Badge variant="warning">EXCL 12%</Badge>;
      case "TAX_INCLUSIVE":
        return <Badge variant="info">INCL 12%</Badge>;
      default:
        return <Badge variant="neutral">NON-PPN</Badge>;
    }
  };

  return (
    <div className="space-y-4">
      {/* Toolbar Kontrol Top Bar */}
      <DataTableToolbar
        search={search}
        setSearch={setSearch}
        placeholder="Cari Nomor PO atau nama supplier..."
        loading={loading}
        isFiltered={isFiltered}
        onReset={handleReset}
        counterIcon={<FileText className="size-3.5 text-slate-400" />}
        counterLabel={`Total ${total} dokumen`}
      >
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1.5 text-sm bg-slate-100/80 px-2.5 py-1.5 rounded-lg border text-slate-600 font-medium">
            <Filter className="size-3" /> Status Dokumen:
          </div>
          <select
            value={String(filters.status ?? "")}
            onChange={(e) => updateFilter("status", e.target.value)}
            className="text-sm h-9 px-3 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer focus:border-blue-500 font-medium text-slate-700"
          >
            <option value="ALL">Semua Dokumen PO</option>
            <option value="PENDING">PENDING</option>
            <option value="APPROVED">APPROVED</option>
            <option value="RECEIVED">RECEIVED</option>
            <option value="REJECTED">REJECTED</option>
          </select>
        </div>
      </DataTableToolbar>
      {/* Mobile: expandable card list */}
      <div className="space-y-3 md:hidden">
        {loading && purchaseOrders.length === 0 ? (
          Array.from({ length: 4 }).map((_, idx) => (
            <div
              key={`po-card-skeleton-${idx}`}
              className="space-y-3 rounded-xl border bg-white p-4 shadow-2xs"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1.5">
                  <Skeleton className="h-4 w-36" />
                  <Skeleton className="h-3 w-28" />
                </div>
                <Skeleton className="h-6 w-24 rounded" />
              </div>
              <Skeleton className="h-3 w-40" />
              <Skeleton className="h-8 w-full rounded-lg" />
            </div>
          ))
        ) : purchaseOrders.length === 0 ? (
          <div className="rounded-xl border bg-white p-8 text-center text-sm text-slate-400">
            Tidak ada riwayat dokumen Purchase Order ditemukan.
          </div>
        ) : (
          purchaseOrders.map((po) => {
            const isExpanded = !!expandedRows[po.id];
            const subItems = rowItems[po.id] || [];
            const isSubLoading = !!rowLoading[po.id];
            const hasReceipts = subItems.some((i) => i.qty_received > 0);

            return (
              <article
                key={po.id}
                onClick={() => toggleRow(po.id)}
                className={`cursor-pointer rounded-xl border bg-white p-4 shadow-2xs transition-colors ${isExpanded ? "border-slate-300 bg-slate-50/60" : ""
                  } ${loading ? "pointer-events-none opacity-50" : ""}`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-start gap-2">
                    <ArrowRight
                      className={`mt-0.5 size-3.5 shrink-0 text-slate-400 transition-transform duration-300 ${isExpanded ? "rotate-90 text-blue-600" : ""
                        }`}
                    />
                    <div className="min-w-0">
                      <h3 className="truncate font-mono text-sm font-medium tracking-tight text-foreground">
                        {po.po_number}
                      </h3>
                      <p className="mt-0.5 truncate text-sm font-bold uppercase text-slate-700">
                        {po.supplier_name}
                      </p>
                      <p className="mt-0.5 flex items-center gap-0.5 truncate text-sm text-slate-500">
                        <User className="size-2.5" /> {po.supplier_pic}
                      </p>
                    </div>
                  </div>
                  <POStatusTimeline status={po.status} />
                </div>

                <dl className="mt-3 grid grid-cols-2 gap-2 rounded-lg bg-slate-50 p-2.5 text-sm">
                  <div>
                    <dt className="flex items-center gap-1 text-slate-400">
                      <Calendar className="size-3" /> Tanggal PO
                    </dt>
                    <dd className="mt-0.5 font-mono font-semibold text-slate-700">
                      {formatDate(po.po_date)}
                    </dd>
                  </div>
                  <div className="text-right">
                    <dt className="flex items-center justify-end gap-1 text-slate-400">
                      <DollarSign className="size-3" /> Total Nilai
                    </dt>
                    <dd className="mt-0.5 font-mono font-bold text-slate-900">
                      {formatIDR(po.total_amount)}
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
                    <div className="mt-3 space-y-3 border-t pt-3">
                      <div className="grid grid-cols-1 gap-2 rounded-lg border bg-white p-2.5 font-mono text-sm font-bold">
                        <div className="flex items-center justify-between rounded-md border border-slate-100 bg-slate-50 p-2">
                          <span className="font-sans text-sm uppercase text-slate-500">
                            Dasar Pajak (DPP)
                          </span>
                          <span className="text-slate-900">
                            {formatIDR(po.dpp_amount || po.total_amount)}
                          </span>
                        </div>
                        <div className="flex items-center justify-between rounded-md border border-slate-100 bg-slate-50 p-2">
                          <span className="flex items-center gap-1 font-sans text-sm uppercase text-slate-500">
                            <Percent className="size-3 text-amber-500" /> PPN
                            (12%)
                          </span>
                          <span className="text-slate-900">
                            {formatIDR(po.tax_amount || 0)}
                          </span>
                        </div>
                        <div className="flex items-center justify-between rounded-md border border-blue-100/60 bg-blue-50/40 p-2">
                          <span className="flex items-center gap-1 font-sans text-sm uppercase text-blue-600">
                            <Landmark className="size-3" /> Kebijakan
                          </span>
                          {getTaxBadge(po.tax_type)}
                        </div>
                      </div>

                      <div className="flex items-center gap-1.5 text-sm font-medium tracking-wide text-muted-foreground">
                        <ListOrdered className="size-3.5 text-blue-500" />
                        Komoditas Item Belanja Terikat
                      </div>

                      {isSubLoading ? (
                        <div className="flex items-center gap-1.5 py-3 text-sm text-slate-400">
                          <Loader2 className="size-3 animate-spin text-blue-600" />
                          <span>Menyisir database logistik Go...</span>
                        </div>
                      ) : subItems.length > 0 ? (
                        <ul className="space-y-2">
                          {subItems.map((item) => (
                            <li
                              key={`m-subitem-po-${item.id}`}
                              className="rounded-lg border bg-white p-2.5 text-sm"
                            >
                              <p className="font-medium tracking-tight text-foreground">
                                {item.product_name}
                              </p>
                              <p className="font-mono text-sm text-slate-400">
                                SKU: {item.product_sku}
                              </p>
                              <div className="mt-1.5 flex flex-wrap items-center justify-between gap-1 font-mono">
                                <span className="font-bold text-slate-700">
                                  {item.qty_ordered} {item.unit_kode}
                                </span>
                                <span className="text-sm font-bold text-emerald-600">
                                  Diterima {item.qty_received}{" "}
                                  {item.unit_kode}
                                </span>
                                <span className="font-bold text-slate-900">
                                  {formatIDR(item.sub_total)}
                                </span>
                              </div>
                              <p className="mt-0.5 text-right font-mono text-sm text-slate-400">
                                @ {formatIDR(item.unit_price)}
                              </p>
                            </li>
                          ))}
                        </ul>
                      ) : (
                        <p className="text-sm italic text-slate-400">
                          Gagal memuat detail item.
                        </p>
                      )}

                      {po.notes && (
                        <div className="rounded-lg border bg-white px-3 py-2 text-sm text-slate-600">
                          <strong className="mb-0.5 block text-sm uppercase tracking-wide text-blue-700">
                            Memo Instruksi Pengiriman:
                          </strong>
                          &quot;{po.notes}&quot;
                        </div>
                      )}

                      {canViewJournal && hasReceipts && (
                        <POJournalLink poNumber={po.po_number} />
                      )}
                    </div>
                  </div>
                </div>

                <div className="mt-3 flex flex-wrap justify-end gap-1.5 border-t pt-3">
                  {canUpdate && po.status === "PENDING" && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onEditClick(po);
                      }}
                      className="flex cursor-pointer items-center gap-1 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-sm font-medium text-slate-700 shadow-3xs hover:bg-slate-50"
                    >
                      <Edit2 className="size-3" /> Edit
                    </button>
                  )}
                  {po.status === "PENDING" && onApproveClick && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onApproveClick(po);
                      }}
                      className="flex cursor-pointer items-center gap-1 rounded-lg border border-emerald-200 bg-white px-2.5 py-1.5 text-sm font-medium text-emerald-600 shadow-3xs hover:bg-emerald-50"
                    >
                      <CheckCircle2 className="size-3" /> Setujui
                    </button>
                  )}
                  {po.status === "PENDING" && onCancelClick && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onCancelClick(po);
                      }}
                      className="flex cursor-pointer items-center gap-1 rounded-lg border border-rose-200 bg-white px-2.5 py-1.5 text-sm font-medium text-rose-600 shadow-3xs hover:bg-rose-50"
                    >
                      <XCircle className="size-3" /> Batalkan
                    </button>
                  )}
                  {po.status === "APPROVED" && onReceiveClick && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onReceiveClick(po);
                      }}
                      className="flex cursor-pointer items-center gap-1 rounded-lg border border-sky-200 bg-white px-2.5 py-1.5 text-sm font-medium text-sky-600 shadow-3xs hover:bg-sky-50"
                    >
                      <PackageCheck className="size-3" /> Terima
                    </button>
                  )}
                  {po.status === "APPROVED" && onFinishReceiveClick && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onFinishReceiveClick(po);
                      }}
                      className="flex cursor-pointer items-center gap-1 rounded-lg border border-teal-200 bg-white px-2.5 py-1.5 text-sm font-medium text-teal-600 shadow-3xs hover:bg-teal-50"
                    >
                      <CheckCheck className="size-3" /> Selesai
                    </button>
                  )}
                  {canDelete && po.status === "PENDING" && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onDeleteClick(po);
                      }}
                      className="flex cursor-pointer items-center gap-1 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-sm font-medium text-rose-600 shadow-3xs hover:bg-rose-50 hover:border-rose-200"
                    >
                      <Trash2 className="size-3" /> Hapus
                    </button>
                  )}
                </div>
              </article>
            );
          })
        )}
      </div>

      {/* Desktop: table */}
      <div className="hidden border border-slate-200 rounded-xl bg-white shadow-2xs overflow-hidden md:block">
        <Table>
          <TableHeader className="bg-slate-50/70 border-b">
            <TableRow>
              <TableHead className="w-10"></TableHead>
              <TableHead className="w-1/4">
                Dokumen Pemesanan / Vendor
              </TableHead>
              <TableHead className="w-1/5">
                <Calendar className="size-3 inline mr-1" /> Tanggal PO
              </TableHead>
              <TableHead className="w-1/4">
                Tahap Dokumen
              </TableHead>
              <TableHead className="text-right w-32">
                <DollarSign className="size-3 inline mr-0.5" /> Total Nilai PO
              </TableHead>
              <TableHead className="text-right w-32">
                Aksi
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && purchaseOrders.length === 0 ? (
              Array.from({ length: 5 }).map((_, idx) => (
                <TableRow key={`po-skel-row-${idx}`} className="border-b">
                  <TableCell className="w-10 py-3.5">
                    <Skeleton className="h-4 w-4 rounded mx-auto" />
                  </TableCell>
                  <TableCell className="py-3.5">
                    <Skeleton className="h-4 w-48 rounded" />
                  </TableCell>
                  <TableCell className="py-3.5">
                    <Skeleton className="h-4 w-28 rounded" />
                  </TableCell>
                  <TableCell className="py-3.5">
                    <Skeleton className="h-4 w-40 rounded" />
                  </TableCell>
                  <TableCell className="py-3.5 text-right">
                    <Skeleton className="h-4 w-24 ml-auto rounded" />
                  </TableCell>
                  <TableCell className="text-right py-3.5">
                    <Skeleton className="h-7 w-20 rounded ml-auto" />
                  </TableCell>
                </TableRow>
              ))
            ) : purchaseOrders.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="h-32 text-center text-muted-foreground font-medium text-sm"
                >
                  Tidak ada riwayat dokumen Purchase Order ditemukan.
                </TableCell>
              </TableRow>
            ) : (
              purchaseOrders.map((po) => {
                const isExpanded = !!expandedRows[po.id];
                const subItems = rowItems[po.id] || [];
                const isSubLoading = !!rowLoading[po.id];
                const hasReceipts = subItems.some((i) => i.qty_received > 0);

                return (
                  <React.Fragment key={po.id}>
                    <TableRow
                      onClick={() => toggleRow(po.id)}
                      className={`cursor-pointer transition-colors border-b-0 hover:bg-muted/20 ${isExpanded ? "bg-slate-50/50 border-b-0" : "border-b"} ${loading ? "opacity-50 pointer-events-none" : ""}`}
                    >
                      {/* Tombol Pemicu Ekspansi Sub-Tabel */}
                      <TableCell className="py-3 text-center w-10">
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleRow(po.id);
                          }}
                          className="p-1 rounded-md hover:bg-slate-200/60 text-slate-400 transition-colors cursor-pointer"
                        >
                          <ArrowRight
                            className={`size-3.5 transition-transform duration-300 ease-in-out ${isExpanded ? "transform rotate-90 text-blue-600 font-bold" : ""}`}
                          />
                        </button>
                      </TableCell>

                      <TableCell className="text-sm py-3 font-medium">
                        <div className="space-y-0.5">
                          <p className="font-medium text-foreground font-mono tracking-tight">
                            {po.po_number}
                          </p>
                          <div className="flex flex-wrap items-center gap-2 text-sm text-slate-500 font-medium">
                            <span className="font-medium text-foreground">
                              {po.supplier_name}
                            </span>
                            <span className="text-slate-300">|</span>
                            <span className="flex items-center gap-0.5">
                              <User className="size-2.5" /> {po.supplier_pic}
                            </span>
                          </div>
                        </div>
                      </TableCell>

                      <TableCell className="text-sm py-3 font-mono text-slate-600 font-semibold">
                        {formatDate(po.po_date)}
                      </TableCell>

                      <TableCell className="text-sm py-3">
                        <POStatusTimeline status={po.status} />
                      </TableCell>

                      <TableCell className="text-sm py-3 text-right font-mono font-bold text-slate-900">
                        {formatIDR(po.total_amount)}
                      </TableCell>

                      <TableCell className="text-right py-3">
                        <div className="inline-flex gap-1 justify-end items-center">
                          {/* TOMBOL EDIT: Muncul jika punya izin DAN status masih PENDING */}
                          {canUpdate && po.status === "PENDING" && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                onEditClick(po);
                              }}
                              className="p-1.5 border border-slate-200 rounded-lg bg-white text-slate-700 hover:bg-slate-50 text-sm shadow-3xs cursor-pointer"
                              title="Ubah Rincian PO"
                            >
                              <Edit2 className="size-3.5" />
                            </button>
                          )}

                          {/* TOMBOL APPROVE & CANCEL BAWAAN */}
                          {po.status === "PENDING" && onApproveClick && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                onApproveClick(po);
                              }}
                              className="p-1.5 border border-emerald-200 rounded-lg bg-white text-emerald-600 hover:bg-emerald-50 text-sm shadow-3xs cursor-pointer"
                              title="Setujui Dokumen PO"
                            >
                              <CheckCircle2 className="size-3.5" />
                            </button>
                          )}

                          {po.status === "PENDING" && onCancelClick && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                onCancelClick(po);
                              }}
                              className="p-1.5 border border-rose-200 rounded-lg bg-white text-rose-600 hover:bg-rose-50 text-sm shadow-3xs cursor-pointer"
                              title="Batalkan Dokumen PO"
                            >
                              <XCircle className="size-3.5" />
                            </button>
                          )}

                          {/* TERIMA BARANG: hanya untuk PO APPROVED yang belum ditutup */}
                          {po.status === "APPROVED" && onReceiveClick && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                onReceiveClick(po);
                              }}
                              className="p-1.5 border border-sky-200 rounded-lg bg-white text-sky-600 hover:bg-sky-50 text-sm shadow-3xs cursor-pointer"
                              title="Catat Penerimaan Barang (GRN)"
                            >
                              <PackageCheck className="size-3.5" />
                            </button>
                          )}

                          {/* SELESAI TERIMA: menutup PO jadi RECEIVED */}
                          {po.status === "APPROVED" && onFinishReceiveClick && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                onFinishReceiveClick(po);
                              }}
                              className="p-1.5 border border-teal-200 rounded-lg bg-white text-teal-600 hover:bg-teal-50 text-sm shadow-3xs cursor-pointer"
                              title="Tandai PO Selesai Diterima"
                            >
                              <CheckCheck className="size-3.5" />
                            </button>
                          )}

                          {/* TOMBOL HAPUS: Muncul jika punya izin DAN status masih PENDING */}
                          {canDelete && po.status === "PENDING" && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                onDeleteClick(po);
                              }}
                              className="p-1.5 border border-slate-200 rounded-lg bg-white text-rose-600 hover:bg-rose-50 hover:border-rose-200 transition-colors shadow-3xs cursor-pointer"
                              title="Hapus Dokumen PO Permanen"
                            >
                              <Trash2 className="size-3.5" />
                            </button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                    {/* 🌟 SUB-ROW EXPANSION AREA: Merender Detail Item Belanja secara In-Line */}
                    {/* 🌟 SUB-ROW EXPANSION AREA: MERENDER TAX SINKRONISASI (DPP & PPN 12%) */}
                    <TableRow
                      className={`border-t-0 p-0 hover:bg-transparent ${isExpanded ? "bg-slate-50/50 border-b" : "pointer-events-none"}`}
                    >
                      <TableCell colSpan={6} className="p-0 border-t-0">
                        <div
                          className={`grid transition-all duration-300 ease-in-out ${isExpanded ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"}`}
                        >
                          <div className="overflow-hidden">
                            {/* 🌟 PERBAIKAN 1: Pastikan padding pembungkus luar rapat & simetris */}
                            <div className="py-4 pl-12 pr-6 space-y-4 bg-slate-50/40 border-t border-slate-100 shadow-inner w-full">
                              {/* Panel Pajak Rangkuman Finansial Atas Sub-Row */}
                              {/* 🌟 PERBAIKAN 2: Ubah max-w-4xl menjadi w-full agar panel pajak mengikuti lebar parent */}
                              <div className="grid grid-cols-1 md:grid-cols-3 gap-3 bg-white p-3 rounded-xl border w-full font-mono text-sm font-bold">
                                <div className="flex justify-between items-center bg-slate-50 p-2 rounded-lg border border-slate-100">
                                  <span className="text-slate-500 font-sans text-sm uppercase">
                                    Dasar Pajak (DPP):
                                  </span>
                                  <span className="text-slate-900">
                                    {formatIDR(
                                      po.dpp_amount || po.total_amount
                                    )}
                                  </span>
                                </div>
                                <div className="flex justify-between items-center bg-slate-50 p-2 rounded-lg border border-slate-100">
                                  <span className="text-slate-500 font-sans text-sm uppercase flex items-center gap-1">
                                    <Percent className="size-3 text-amber-500" />{" "}
                                    PPN (12%):
                                  </span>
                                  <span className="text-slate-900">
                                    {formatIDR(po.tax_amount || 0)}
                                  </span>
                                </div>
                                <div className="flex justify-between items-center bg-blue-50/40 p-2 rounded-lg border border-blue-100/60">
                                  <span className="text-blue-600 font-sans text-sm uppercase flex items-center gap-1">
                                    <Landmark className="size-3" /> Jenis
                                    Kebijakan:
                                  </span>
                                  {getTaxBadge(po.tax_type)}
                                </div>
                              </div>

                              <div className="text-sm font-medium text-muted-foreground flex items-center gap-1.5">
                                <ListOrdered className="size-3.5 text-blue-500" />{" "}
                                Komoditas Item Belanja Terikat
                              </div>

                              {isSubLoading ? (
                                <div className="text-sm text-slate-400 py-3 flex items-center gap-1.5">
                                  <Loader2 className="size-3 animate-spin text-blue-600" />
                                  <span>Menyisir database logistik Go...</span>
                                </div>
                              ) : subItems.length > 0 ? (
                                /* 🌟 PERBAIKAN 3: Ubah max-w-4xl menjadi w-full pada kontainer pembungkus tabel detail item */
                                <div className="border bg-white rounded-lg overflow-hidden shadow-3xs w-full">
                                  <Table>
                                    <TableHeader className="bg-slate-50/60 border-b">
                                      <TableRow className="h-7 text-sm">
                                        <TableHead className="py-1">
                                          Nama Barang
                                        </TableHead>
                                        <TableHead className="py-1 text-center w-28">
                                          Kuantitas Order
                                        </TableHead>
                                        <TableHead className="py-1 text-center w-28">
                                          Diterima (GRN)
                                        </TableHead>
                                        <TableHead className="py-1 text-right w-32">
                                          Harga Satuan
                                        </TableHead>
                                        <TableHead className="py-1 text-right w-36">
                                          Subtotal
                                        </TableHead>
                                      </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                      {subItems.map((item) => (
                                        <TableRow
                                          key={`subitem-po-${item.id}`}
                                          className="h-9 text-sm hover:bg-slate-50/40"
                                        >
                                          <TableCell className="py-1.5 font-medium text-slate-800">
                                            <span className="font-medium tracking-tight block text-foreground">
                                              {item.product_name}
                                            </span>
                                            <span className="text-sm font-mono text-slate-400">
                                              SKU: {item.product_sku}
                                            </span>
                                          </TableCell>
                                          <TableCell className="py-1.5 text-center font-mono font-bold text-slate-700">
                                            {item.qty_ordered} {item.unit_kode}
                                          </TableCell>
                                          <TableCell className="py-1.5 text-center font-mono font-bold text-emerald-600">
                                            {item.qty_received} {item.unit_kode}
                                          </TableCell>
                                          <TableCell className="py-1.5 text-right font-mono text-slate-500">
                                            {formatIDR(item.unit_price)}
                                          </TableCell>
                                          <TableCell className="py-1.5 text-right font-mono font-bold text-slate-900">
                                            {formatIDR(item.sub_total)}
                                          </TableCell>
                                        </TableRow>
                                      ))}
                                    </TableBody>
                                  </Table>
                                </div>
                              ) : (
                                <p className="text-sm text-slate-400 italic">
                                  Gagal memuat detail item.
                                </p>
                              )}

                              {po.notes && (
                                /* 🌟 PERBAIKAN 4: Sesuaikan juga note memo pengiriman menjadi w-full */
                                <div className="text-sm bg-white border px-3 py-2 rounded-lg text-slate-600 w-full">
                                  <strong className="text-blue-700 uppercase text-sm tracking-wide block mb-0.5">
                                    Memo Instruksi Pengiriman:
                                  </strong>
                                  &quot;{po.notes}&quot;
                                </div>
                              )}

                              {canViewJournal && hasReceipts && (
                                <POJournalLink poNumber={po.po_number} />
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

      {/* Pagination Controls Mandiri */}
      <DataTablePagination
        page={page}
        limit={limit}
        setLimit={setLimit}
        total={total}
        rowCount={purchaseOrders.length}
        hasNext={hasNext}
        hasPrev={hasPrev}
        onNextPage={handleNextPage}
        onPrevPage={handlePrevPage}
        loading={loading}
      />
    </div>
  );
}
