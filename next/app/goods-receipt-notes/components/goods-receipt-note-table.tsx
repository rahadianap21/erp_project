"use client";

import React, { useState } from "react";
import Cookies from "js-cookie";
import { toast } from "sonner";
import { ChevronRight, PackageCheck, ListOrdered, Loader2 } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { useDebounceFetch } from "@/hooks/use-debounce";
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
 * Tabel riwayat penerimaan barang (GRN).
 *
 * GRN immutable, jadi tabel ini read-only: tidak ada tombol ubah atau hapus.
 * Detail barang ditampilkan sebagai expanded row in-line (pola yang sama dengan
 * modul Purchase Orders) dan baru diambil saat baris dibuka.
 */
export function GoodsReceiptNoteTable({ refreshTrigger, onForbidden }: GoodsReceiptNoteTableProps) {
    // State kontrol expanded row + lazy loading detail barang per GRN.
    const [expandedRows, setExpandedRows] = useState<Record<number, boolean>>({});
    const [rowItems, setRowItems] = useState<Record<number, GRNDetail[]>>({});
    const [rowLoading, setRowLoading] = useState<Record<number, boolean>>({});

    const authToken = Cookies.get("session_active");

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
                const res = await fetch(`/api/goods-receipt-notes?id=${id}`, {
                    headers: { Authorization: `Bearer ${authToken}` },
                });
                const result = await res.json();
                if (!res.ok) throw new Error(result.message || "Gagal memuat detail GRN.");
                setRowItems((prev) => ({ ...prev, [id]: result.data?.items ?? [] }));
            } catch (error: any) {
                toast.error("Gagal Memuat Detail", { description: error.message });
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

            <div className="bg-white rounded-xl border shadow-3xs overflow-hidden">
                <Table>
                    <TableHeader className="bg-slate-50/70 border-b">
                        <TableRow>
                            <TableHead className="w-10"></TableHead>
                            <TableHead className="font-bold text-xs text-slate-700">Nomor GRN</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700">Nomor PO</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-center">
                                Tanggal Terima
                            </TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-center w-28">
                                Total Qty
                            </TableHead>
                            <TableHead className="font-bold text-xs text-slate-700">Catatan</TableHead>
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
                                <TableCell colSpan={6} className="h-32 text-center text-xs text-slate-400">
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
                                                    onClick={(e) => { e.stopPropagation(); toggleRow(grn); }}
                                                    className="p-1 rounded hover:bg-slate-100 transition-colors cursor-pointer"
                                                    title={isExpanded ? "Tutup rincian" : "Lihat rincian barang"}
                                                >
                                                    <ChevronRight
                                                        className={`size-3.5 transition-transform duration-300 ease-in-out ${isExpanded
                                                            ? "transform rotate-90 text-emerald-600 font-bold"
                                                            : ""
                                                            }`}
                                                    />
                                                </button>
                                            </TableCell>
                                            <TableCell className="py-3 font-mono font-bold text-emerald-700 text-xs">
                                                {grn.grn_number}
                                            </TableCell>
                                            <TableCell className="py-3 font-mono text-xs text-slate-600">
                                                {grn.po_number || `#${grn.purchase_order_id}`}
                                            </TableCell>
                                            <TableCell className="py-3 text-center text-xs text-slate-600">
                                                {formatDate(grn.received_date)}
                                            </TableCell>
                                            <TableCell className="py-3 text-center font-mono font-bold text-xs text-slate-900">
                                                {grn.total_qty}
                                            </TableCell>
                                            <TableCell className="py-3 text-xs text-slate-500 max-w-[16rem] truncate">
                                                {grn.notes || "-"}
                                            </TableCell>
                                        </TableRow>

                                        {/* Sub-row expansion in-line: rincian barang diterima */}
                                        <TableRow
                                            className={`border-t-0 p-0 hover:bg-transparent ${isExpanded ? "bg-slate-50/50 border-b" : "pointer-events-none"
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
                                                            <div className="text-[10px] font-bold text-slate-500 flex items-center gap-1.5 uppercase tracking-wider">
                                                                <ListOrdered className="size-3.5 text-emerald-500" />
                                                                Rincian Barang Diterima
                                                            </div>

                                                            {isRowLoading ? (
                                                                <div className="text-xs text-slate-400 py-3 flex items-center gap-1.5">
                                                                    <Loader2 className="size-3 animate-spin text-emerald-600" />
                                                                    <span>Memuat rincian penerimaan...</span>
                                                                </div>
                                                            ) : items.length > 0 ? (
                                                                <div className="border bg-white rounded-lg overflow-hidden shadow-3xs w-full">
                                                                    <Table>
                                                                        <TableHeader className="bg-slate-50/60 border-b">
                                                                            <TableRow className="h-7 text-[10px]">
                                                                                <TableHead className="font-bold py-1 text-slate-600">
                                                                                    Nama Barang
                                                                                </TableHead>
                                                                                <TableHead className="font-bold py-1 text-center w-32">
                                                                                    Diterima
                                                                                </TableHead>
                                                                                <TableHead className="font-bold py-1 text-center w-32">
                                                                                    Konversi
                                                                                </TableHead>
                                                                            </TableRow>
                                                                        </TableHeader>
                                                                        <TableBody>
                                                                            {items.map((item) => (
                                                                                <TableRow
                                                                                    key={`subitem-grn-${item.id}`}
                                                                                    className="h-9 text-[11px] hover:bg-slate-50/40"
                                                                                >
                                                                                    <TableCell className="py-1.5 font-medium text-slate-800">
                                                                                        <span className="font-bold uppercase tracking-tight block">
                                                                                            {item.product_name}
                                                                                        </span>
                                                                                        <span className="text-[9px] font-mono text-slate-400">
                                                                                            SKU: {item.sku}
                                                                                        </span>
                                                                                    </TableCell>
                                                                                    <TableCell className="py-1.5 text-center font-mono font-bold text-emerald-600">
                                                                                        {item.qty_received}{" "}
                                                                                        {item.purchase_unit_kode}
                                                                                    </TableCell>
                                                                                    <TableCell className="py-1.5 text-center font-mono text-[10px] text-slate-500">
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
                                                                <p className="text-xs text-slate-400 italic">
                                                                    Tidak ada rincian barang pada dokumen ini.
                                                                </p>
                                                            )}

                                                            {grn.notes && (
                                                                <div className="text-[10px] bg-white border px-3 py-2 rounded-lg text-slate-600 w-full">
                                                                    <strong className="text-emerald-700 uppercase text-[9px] tracking-wide block mb-0.5">
                                                                        Catatan Penerimaan:
                                                                    </strong>
                                                                    &quot;{grn.notes}&quot;
                                                                </div>
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
