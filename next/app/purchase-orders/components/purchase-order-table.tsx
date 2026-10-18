"use client";

import { apiGet } from "@/lib/api-client";
import { FileText, Phone, User, Calendar, DollarSign, Filter, CheckCircle2, XCircle, ChevronRight as ArrowRight, Loader2, Package, ListOrdered, Percent, Landmark, Trash2, Edit2, PackageCheck, CheckCheck } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { useDebounceFetch } from "@/hooks/use-api-list";
import { POStatusTimeline } from "./po-status-timeline";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import { DataTablePagination } from "@/components/data-table-pagination";
import React, { useState, useEffect } from "react";
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
    canUpdate: boolean;    // 🌟 SUNTIKKAN RBAC PRIVILEGE
    canDelete: boolean;    // 🌟 SUNTIKKAN RBAC PRIVILEGE
    refreshTrigger: number;
    onEditClick: (po: PurchaseOrder) => void;   // Callback pemicu form edit
    onDeleteClick: (po: PurchaseOrder) => void; // Callback pemicu alert hapus
    onApproveClick?: (po: PurchaseOrder) => void;
    onCancelClick?: (po: PurchaseOrder) => void;
    onReceiveClick?: (po: PurchaseOrder) => void;
    onFinishReceiveClick?: (po: PurchaseOrder) => void;
}

export function PurchaseOrderTable({ canUpdate,
    canDelete,
    refreshTrigger,
    onEditClick,
    onDeleteClick,
    onApproveClick,
    onCancelClick,
    onReceiveClick,
    onFinishReceiveClick }: PurchaseOrderTableProps) {
    const token = Cookies.get("session_active") || "";

    const {
        search, setSearch, page, limit, setLimit, filters, updateFilter, data: purchaseOrders, total, loading,
        handleReset, isFiltered, hasNext, hasPrev, handleNextPage, handlePrevPage
    } = useDebounceFetch<PurchaseOrder>({
        endpoint: "/api/purchase-orders",
        delay: 500,
        initialFilters: {
            status: "ALL" // Default filter kaku mengikuti spesifikasi select
        },
        refreshTrigger: refreshTrigger
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
            [poId]: !isCurrentlyExpanded
        }));

        // Lazy Fetching: Tarik data item detail dari database Go hanya jika baris belum pernah dimuat
        if (!isCurrentlyExpanded && !rowItems[poId] && token) {
            setRowLoading((prev) => ({ ...prev, [poId]: true }));
            try {
                const result = await apiGet<{ items?: DetailItem[] }>(`/api/purchase-orders?id=${poId}`);
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

    const formatIDR = (num: number) => {
        return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(num);
    };

    const formatDate = (dateStr: string) => {
        if (!dateStr) return "-";
        return new Date(dateStr).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
    };

    const getTaxBadge = (type: PurchaseOrder["tax_type"]) => {
        switch (type) {
            case "TAX_EXCLUSIVE":
                return <span className="px-1.5 py-0.5 text-[9px] font-bold bg-amber-50 text-amber-700 border border-amber-200 rounded">EXCL 12%</span>;
            case "TAX_INCLUSIVE":
                return <span className="px-1.5 py-0.5 text-[9px] font-bold bg-blue-50 text-blue-700 border border-blue-200 rounded">INCL 12%</span>;
            default:
                return <span className="px-1.5 py-0.5 text-[9px] font-bold bg-slate-100 text-slate-500 border border-slate-200 rounded">NON-PPN</span>;
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
                    <div className="flex items-center gap-1.5 text-xs bg-slate-100/80 px-2.5 py-1.5 rounded-lg border text-slate-600 font-medium">
                        <Filter className="size-3" /> Status Dokumen:
                    </div>
                    <select
                        value={filters.status}
                        onChange={(e) => updateFilter("status", e.target.value)}
                        className="text-xs h-9 px-3 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer focus:border-blue-500 font-medium text-slate-700"
                    >
                        <option value="ALL">Semua Dokumen PO</option>
                        <option value="PENDING">PENDING</option>
                        <option value="APPROVED">APPROVED</option>
                        <option value="RECEIVED">RECEIVED</option>
                        <option value="REJECTED">REJECTED</option>
                    </select>
                </div>
            </DataTableToolbar>
            {/* Layout Grid Tabel Transaksi Utama */}
            <div className="border border-slate-200 rounded-xl bg-white shadow-2xs overflow-hidden">
                <Table>
                    <TableHeader className="bg-slate-50/70 border-b">
                        <TableRow>
                            <TableHead className="w-10"></TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/4">Dokumen Pemesanan / Vendor</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/5"><Calendar className="size-3 inline mr-1" /> Tanggal PO</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/4">Tahap Dokumen</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-right w-32"><DollarSign className="size-3 inline mr-0.5" /> Total Nilai PO</TableHead>
                            <TableHead className="text-right font-bold text-xs text-slate-700 w-32">Aksi</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {loading && purchaseOrders.length === 0 ? (
                            Array.from({ length: 5 }).map((_, idx) => (
                                <TableRow key={`po-skel-row-${idx}`} className="border-b">
                                    <TableCell className="w-10 py-3.5"><Skeleton className="h-4 w-4 rounded mx-auto" /></TableCell>
                                    <TableCell className="py-3.5"><Skeleton className="h-4 w-48 rounded" /></TableCell>
                                    <TableCell className="py-3.5"><Skeleton className="h-4 w-28 rounded" /></TableCell>
                                    <TableCell className="py-3.5"><Skeleton className="h-4 w-40 rounded" /></TableCell>
                                    <TableCell className="py-3.5 text-right"><Skeleton className="h-4 w-24 ml-auto rounded" /></TableCell>
                                    <TableCell className="text-right py-3.5"><Skeleton className="h-7 w-20 rounded ml-auto" /></TableCell>
                                </TableRow>
                            ))
                        ) : purchaseOrders.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={6} className="h-32 text-center text-muted-foreground font-medium text-xs">
                                    Tidak ada riwayat dokumen Purchase Order ditemukan.
                                </TableCell>
                            </TableRow>
                        ) : (
                            purchaseOrders.map((po) => {
                                const isExpanded = !!expandedRows[po.id];
                                const subItems = rowItems[po.id] || [];
                                const isSubLoading = !!rowLoading[po.id];

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
                                                    onClick={(e) => { e.stopPropagation(); toggleRow(po.id); }}
                                                    className="p-1 rounded-md hover:bg-slate-200/60 text-slate-400 transition-colors cursor-pointer"
                                                >
                                                    <ArrowRight className={`size-3.5 transition-transform duration-300 ease-in-out ${isExpanded ? "transform rotate-90 text-blue-600 font-bold" : ""}`} />
                                                </button>
                                            </TableCell>

                                            <TableCell className="text-xs py-3 font-medium">
                                                <div className="space-y-0.5">
                                                    <p className="font-bold text-slate-900 font-mono tracking-tight uppercase">{po.po_number}</p>
                                                    <div className="flex flex-wrap items-center gap-2 text-[10px] text-slate-500 font-medium">
                                                        <span className="font-bold text-slate-800 uppercase">{po.supplier_name}</span>
                                                        <span className="text-slate-300">|</span>
                                                        <span className="flex items-center gap-0.5"><User className="size-2.5" /> {po.supplier_pic}</span>
                                                    </div>
                                                </div>
                                            </TableCell>

                                            <TableCell className="text-xs py-3 font-mono text-slate-600 font-semibold">
                                                {formatDate(po.po_date)}
                                            </TableCell>

                                            <TableCell className="text-xs py-3">
                                                <POStatusTimeline status={po.status} />
                                            </TableCell>

                                            <TableCell className="text-xs py-3 text-right font-mono font-bold text-slate-900">
                                                {formatIDR(po.total_amount)}
                                            </TableCell>

                                            <TableCell className="text-right py-3">
                                                <div className="inline-flex gap-1 justify-end items-center">

                                                    {/* TOMBOL EDIT: Muncul jika punya izin DAN status masih PENDING */}
                                                    {canUpdate && po.status === "PENDING" && (
                                                        <button
                                                            type="button"
                                                            onClick={(e) => { e.stopPropagation(); onEditClick(po); }}
                                                            className="p-1.5 border border-slate-200 rounded-lg bg-white text-slate-700 hover:bg-slate-50 text-[10px] shadow-3xs cursor-pointer"
                                                            title="Ubah Rincian PO"
                                                        >
                                                            <Edit2 className="size-3.5" />
                                                        </button>
                                                    )}

                                                    {/* TOMBOL APPROVE & CANCEL BAWAAN */}
                                                    {po.status === "PENDING" && onApproveClick && (
                                                        <button type="button" onClick={(e) => { e.stopPropagation(); onApproveClick(po); }} className="p-1.5 border border-emerald-200 rounded-lg bg-white text-emerald-600 hover:bg-emerald-50 text-[10px] shadow-3xs cursor-pointer" title="Setujui Dokumen PO"><CheckCircle2 className="size-3.5" /></button>
                                                    )}

                                                    {po.status === "PENDING" && onCancelClick && (
                                                        <button type="button" onClick={(e) => { e.stopPropagation(); onCancelClick(po); }} className="p-1.5 border border-rose-200 rounded-lg bg-white text-rose-600 hover:bg-rose-50 text-[10px] shadow-3xs cursor-pointer" title="Batalkan Dokumen PO"><XCircle className="size-3.5" /></button>
                                                    )}

                                                    {/* TERIMA BARANG: hanya untuk PO APPROVED yang belum ditutup */}
                                                    {po.status === "APPROVED" && onReceiveClick && (
                                                        <button
                                                            type="button"
                                                            onClick={(e) => { e.stopPropagation(); onReceiveClick(po); }}
                                                            className="p-1.5 border border-sky-200 rounded-lg bg-white text-sky-600 hover:bg-sky-50 text-[10px] shadow-3xs cursor-pointer"
                                                            title="Catat Penerimaan Barang (GRN)"
                                                        >
                                                            <PackageCheck className="size-3.5" />
                                                        </button>
                                                    )}

                                                    {/* SELESAI TERIMA: menutup PO jadi RECEIVED */}
                                                    {po.status === "APPROVED" && onFinishReceiveClick && (
                                                        <button
                                                            type="button"
                                                            onClick={(e) => { e.stopPropagation(); onFinishReceiveClick(po); }}
                                                            className="p-1.5 border border-teal-200 rounded-lg bg-white text-teal-600 hover:bg-teal-50 text-[10px] shadow-3xs cursor-pointer"
                                                            title="Tandai PO Selesai Diterima"
                                                        >
                                                            <CheckCheck className="size-3.5" />
                                                        </button>
                                                    )}

                                                    {/* TOMBOL HAPUS: Muncul jika punya izin DAN status masih PENDING */}
                                                    {canDelete && po.status === "PENDING" && (
                                                        <button
                                                            type="button"
                                                            onClick={(e) => { e.stopPropagation(); onDeleteClick(po); }}
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
                                        <TableRow className={`border-t-0 p-0 hover:bg-transparent ${isExpanded ? "bg-slate-50/50 border-b" : "pointer-events-none"}`}>
                                            <TableCell colSpan={6} className="p-0 border-t-0">
                                                <div className={`grid transition-all duration-300 ease-in-out ${isExpanded ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"}`}>
                                                    <div className="overflow-hidden">
                                                        {/* 🌟 PERBAIKAN 1: Pastikan padding pembungkus luar rapat & simetris */}
                                                        <div className="py-4 pl-12 pr-6 space-y-4 bg-slate-50/40 border-t border-slate-100 shadow-inner w-full">

                                                            {/* Panel Pajak Rangkuman Finansial Atas Sub-Row */}
                                                            {/* 🌟 PERBAIKAN 2: Ubah max-w-4xl menjadi w-full agar panel pajak mengikuti lebar parent */}
                                                            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 bg-white p-3 rounded-xl border w-full font-mono text-[11px] font-bold">
                                                                <div className="flex justify-between items-center bg-slate-50 p-2 rounded-lg border border-slate-100">
                                                                    <span className="text-slate-500 font-sans text-[10px] uppercase">Dasar Pajak (DPP):</span>
                                                                    <span className="text-slate-900">{formatIDR(po.dpp_amount || po.total_amount)}</span>
                                                                </div>
                                                                <div className="flex justify-between items-center bg-slate-50 p-2 rounded-lg border border-slate-100">
                                                                    <span className="text-slate-500 font-sans text-[10px] uppercase flex items-center gap-1"><Percent className="size-3 text-amber-500" /> PPN (12%):</span>
                                                                    <span className="text-slate-900">{formatIDR(po.tax_amount || 0)}</span>
                                                                </div>
                                                                <div className="flex justify-between items-center bg-blue-50/40 p-2 rounded-lg border border-blue-100/60">
                                                                    <span className="text-blue-600 font-sans text-[10px] uppercase flex items-center gap-1"><Landmark className="size-3" /> Jenis Kebijakan:</span>
                                                                    {getTaxBadge(po.tax_type)}
                                                                </div>
                                                            </div>

                                                            <div className="text-[10px] font-bold text-slate-500 flex items-center gap-1.5 uppercase tracking-wider">
                                                                <ListOrdered className="size-3.5 text-blue-500" /> Komoditas Item Belanja Terikat
                                                            </div>

                                                            {isSubLoading ? (
                                                                <div className="text-xs text-slate-400 py-3 flex items-center gap-1.5">
                                                                    <Loader2 className="size-3 animate-spin text-blue-600" />
                                                                    <span>Menyisir database logistik Go...</span>
                                                                </div>
                                                            ) : subItems.length > 0 ? (
                                                                /* 🌟 PERBAIKAN 3: Ubah max-w-4xl menjadi w-full pada kontainer pembungkus tabel detail item */
                                                                <div className="border bg-white rounded-lg overflow-hidden shadow-3xs w-full">
                                                                    <Table>
                                                                        <TableHeader className="bg-slate-50/60 border-b">
                                                                            <TableRow className="h-7 text-[10px]">
                                                                                <TableHead className="font-bold py-1 text-slate-600">Nama Barang</TableHead>
                                                                                <TableHead className="font-bold py-1 text-center w-28">Kuantitas Order</TableHead>
                                                                                <TableHead className="font-bold py-1 text-center w-28">Diterima (GRN)</TableHead>
                                                                                <TableHead className="font-bold py-1 text-right w-32">Harga Satuan</TableHead>
                                                                                <TableHead className="font-bold py-1 text-right w-36">Subtotal</TableHead>
                                                                            </TableRow>
                                                                        </TableHeader>
                                                                        <TableBody>
                                                                            {subItems.map((item) => (
                                                                                <TableRow key={`subitem-po-${item.id}`} className="h-9 text-[11px] hover:bg-slate-50/40">
                                                                                    <TableCell className="py-1.5 font-medium text-slate-800">
                                                                                        <span className="font-bold uppercase tracking-tight block">{item.product_name}</span>
                                                                                        <span className="text-[9px] font-mono text-slate-400">SKU: {item.product_sku}</span>
                                                                                    </TableCell>
                                                                                    <TableCell className="py-1.5 text-center font-mono font-bold text-slate-700">{item.qty_ordered} {item.unit_kode}</TableCell>
                                                                                    <TableCell className="py-1.5 text-center font-mono font-bold text-emerald-600">{item.qty_received} {item.unit_kode}</TableCell>
                                                                                    <TableCell className="py-1.5 text-right font-mono text-slate-500">{formatIDR(item.unit_price)}</TableCell>
                                                                                    <TableCell className="py-1.5 text-right font-mono font-bold text-slate-900">{formatIDR(item.sub_total)}</TableCell>
                                                                                </TableRow>
                                                                            ))}
                                                                        </TableBody>
                                                                    </Table>
                                                                </div>
                                                            ) : (
                                                                <p className="text-xs text-slate-400 italic">Gagal memuat detail item.</p>
                                                            )}

                                                            {po.notes && (
                                                                /* 🌟 PERBAIKAN 4: Sesuaikan juga note memo pengiriman menjadi w-full */
                                                                <div className="text-[10px] bg-white border px-3 py-2 rounded-lg text-slate-600 w-full">
                                                                    <strong className="text-blue-700 uppercase text-[9px] tracking-wide block mb-0.5">Memo Instruksi Pengiriman:</strong>
                                                                    "{po.notes}"
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
