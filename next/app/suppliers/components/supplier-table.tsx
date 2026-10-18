"use client";

import { Edit2, Trash2, Search, Users, CreditCard } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useDebounceFetch } from "@/hooks/use-api-list";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import { DataTablePagination } from "@/components/data-table-pagination";
import React from "react";
import { Badge } from "@/components/ui/badge";

export interface Supplier {
    id: number;
    name: string;
    pic_name: string;
    phone: string;
    email: string;
    address: string;
    village_id: string;
    village_name: string;
    postal_code: string;
    district_name: string;
    regency_name: string;
    province_name: string;
    bank_id?: number | null;
    bank_kode?: string;
    bank_name?: string;
    account_number?: string;
    account_name?: string;
    is_active: boolean;
}

interface SupplierTableProps {
    canUpdate: boolean;
    canDelete: boolean;
    refreshTrigger: number;
    onDeleteClick: (supplier: Supplier) => void;
    onEditClick: (supplier: Supplier) => void;
    onRoleOptionsChange?: (options: string[]) => void;
    onForbidden?: () => void;
}

export function SupplierTableSkeleton() {
    return (
        <div className="space-y-4">
            {/* Filter & Search Bar Skeleton */}
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-card p-4 rounded-xl border shadow-sm">
                <div className="flex flex-1 flex-col sm:flex-row items-stretch sm:items-center gap-3">
                    <Skeleton className="h-9 flex-1 max-w-md rounded-md" />
                    <Skeleton className="h-9 w-44 rounded-md" />
                </div>
                <Skeleton className="h-5 w-36" />
            </div>

            {/* Table Skeleton */}
            <div className="border rounded-xl shadow-sm overflow-hidden bg-card">
                <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow>
                            <TableHead className="font-semibold text-xs">Nama Vendor</TableHead>
                            <TableHead className="font-semibold text-xs">PIC Vendor</TableHead>
                            <TableHead className="font-semibold text-xs">No. Telepon</TableHead>
                            <TableHead className="font-semibold text-xs">Email</TableHead>
                            <TableHead className="font-semibold text-xs">Alamat</TableHead>
                            <TableHead className="font-semibold text-xs">Status</TableHead>
                            <TableHead className="text-right font-semibold text-xs w-32">Aksi</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {Array.from({ length: 5 }).map((_, idx) => (
                            <TableRow key={`user-skel-${idx}`} className="hover:bg-transparent">
                                <TableCell className="py-3.5"><Skeleton className="h-4 w-32" /></TableCell>
                                <TableCell className="py-3.5"><Skeleton className="h-4 w-44" /></TableCell>
                                <TableCell className="py-3.5"><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                                <TableCell className="text-right py-3.5">
                                    <div className="inline-flex gap-2 justify-end">
                                        <Skeleton className="h-7 w-7 rounded-md" />
                                        <Skeleton className="h-7 w-7 rounded-md" />
                                    </div>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>

            {/* Pagination Skeleton */}
            <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-card p-4 rounded-xl border shadow-2xs">
                <Skeleton className="h-5 w-48" />
                <div className="flex items-center gap-2">
                    <Skeleton className="h-8 w-24 rounded-md" />
                    <Skeleton className="h-8 w-8 rounded-md" />
                    <Skeleton className="h-8 w-24 rounded-md" />
                </div>
            </div>
        </div>
    );
}

export function SupplierTable({ canUpdate, canDelete, refreshTrigger, onDeleteClick, onEditClick, onRoleOptionsChange, onForbidden }: SupplierTableProps) {
    // Panggil custom hook universal dengan tipe <User> dan endpoint API Go
    const {
        search,
        setSearch,
        page,
        limit,
        setLimit,
        data: suppliers,
        total,
        loading,
        handleReset,
        isFiltered,
        hasNext,
        hasPrev,
        handleNextPage,
        handlePrevPage,
        httpStatus
    } = useDebounceFetch<Supplier>({
        endpoint: "/api/suppliers",
        delay: 500,
        refreshTrigger: refreshTrigger
    });

    // Teruskan sinyal 403 dari hook ke halaman induk agar ForbiddenView tampil.
    React.useEffect(() => {
        if (httpStatus === 403) onForbidden?.();
    }, [httpStatus, onForbidden]);

    const showActionColumn = canUpdate || canDelete;

    return (
        <div className="space-y-4">
            {/* Filter & Search Bar */}
            <DataTableToolbar
                search={search}
                setSearch={setSearch}
                placeholder="Cari berdasarkan nama atau email..."
                loading={loading}
                isFiltered={isFiltered}
                onReset={handleReset}
                counterIcon={<Users className="size-3.5 text-slate-400" />}
                counterLabel={`Total ${total} vendor ditemukan`}
            />

            {/* Table Area */}
            <div className="border rounded-xl shadow-sm overflow-hidden bg-card relative">
                {/* Indikator Loading Tipis Saat Memperbarui Data yang Sudah Ada */}
                {loading && suppliers.length > 0 && (
                    <div className="absolute top-0 left-0 right-0 h-0.5 bg-primary/20 overflow-hidden z-10">
                        <div className="h-full bg-primary animate-pulse w-full" />
                    </div>
                )}

                <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow>
                            <TableHead className="font-semibold text-xs px-4">Nama Vendor</TableHead>
                            <TableHead className="font-semibold text-xs">PIC Vendor</TableHead>
                            <TableHead className="font-semibold text-xs">No. Telepon</TableHead>
                            <TableHead className="font-semibold text-xs">Email</TableHead>
                            <TableHead className="font-semibold text-xs">Alamat</TableHead>
                            <TableHead className="font-semibold text-xs">Rekening Bank</TableHead>
                            <TableHead className="font-semibold text-xs">Status</TableHead>
                            {showActionColumn && <TableHead className="text-right font-semibold text-xs w-32 px-4">Aksi</TableHead>}
                        </TableRow>
                    </TableHeader>

                    <TableBody>
                        {/* 🌟 1. SKELETON LOADING: Muncul saat pertama kali memuat data */}
                        {loading && suppliers.length === 0 ? (
                            Array.from({ length: 5 }).map((_, idx) => (
                                <TableRow key={`user-row-skeleton-${idx}`} className="hover:bg-transparent">
                                    <TableCell className="py-3.5">
                                        <Skeleton className="h-4 w-32" />
                                    </TableCell>
                                    <TableCell className="py-3.5">
                                        <Skeleton className="h-4 w-44" />
                                    </TableCell>
                                    <TableCell className="py-3.5">
                                        <Skeleton className="h-5 w-16 rounded-full" />
                                    </TableCell>
                                    {showActionColumn && (
                                        <TableCell className="text-right py-3.5">
                                            <div className="inline-flex gap-2 justify-end">
                                                <Skeleton className="h-7 w-7 rounded-md" />
                                                <Skeleton className="h-7 w-7 rounded-md" />
                                            </div>
                                        </TableCell>
                                    )}
                                </TableRow>
                            ))
                        ) : suppliers.length === 0 ? (
                            /* 💡 2. EMPTY STATE: Hanya muncul jika selesai loading dan data benar-benar kosong */
                            <TableRow>
                                <TableCell
                                    colSpan={showActionColumn ? 8 : 7}
                                    className="h-32 text-center text-muted-foreground font-medium text-xs"
                                >
                                    <div className="flex flex-col items-center justify-center gap-2 py-4">
                                        <Search className="size-5 text-muted-foreground/50" />
                                        <p>
                                            {isFiltered
                                                ? "Data tidak ditemukan. Coba sesuaikan kata kunci atau filter pencarian Anda."
                                                : "Tidak ada data pengguna yang terindeks di sistem."}
                                        </p>
                                        {isFiltered && (
                                            <Button
                                                type="button"
                                                variant="outline"
                                                size="sm"
                                                onClick={handleReset}
                                                className="mt-1 h-7 text-[11px] cursor-pointer"
                                            >
                                                Bersihkan Filter
                                            </Button>
                                        )}
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : (
                            /* 💡 3. DATA ROWS: Tampilkan data dengan efek opacity saat sedang memperbarui */
                            suppliers.map((s) => (
                                <TableRow
                                    key={s.id}
                                    className={`hover:bg-muted/30 transition-opacity duration-200 ${loading ? "opacity-50 pointer-events-none" : ""}`}
                                >
                                    <TableCell className="font-medium text-xs py-3 px-4">{s.name}</TableCell>
                                    <TableCell className="font-medium text-xs py-3">{s.pic_name}</TableCell>
                                    <TableCell className="font-medium text-xs py-3">{s.phone}</TableCell>
                                    <TableCell className="font-medium text-xs py-3">{s.email}</TableCell>
                                    <TableCell className="text-xs py-3.5">
                                        <div className="space-y-0.5 text-slate-600 max-w-xs">
                                            <p className="line-clamp-1 italic text-[11px] text-slate-400">{s.address}</p>
                                            <p className="text-[10px] text-slate-500 font-medium uppercase tracking-tight">{s.district_name}, {s.regency_name}</p>
                                        </div>
                                    </TableCell>
                                    {/* 🌟 KOLOM BARU: DETAIL REKENING BANK VENDOR */}
                                    <TableCell className="text-xs py-3.5">
                                        {s.account_number ? (
                                            <div className="space-y-1 max-w-[200px]">
                                                <p className="font-bold text-slate-800 flex items-center gap-1 text-[11px] tracking-tight uppercase">
                                                    <CreditCard className="size-3 text-blue-500" /> {s.bank_name || "BANK"}
                                                </p>
                                                <p className="text-[11px] font-mono font-bold text-slate-700 bg-slate-100/80 border border-slate-200/60 rounded px-1.5 py-0.5 inline-block tracking-wide">
                                                    {s.account_number}
                                                </p>
                                                {s.account_name && (
                                                    <p className="text-[10px] text-slate-500 font-semibold truncate pl-1 uppercase tracking-tight" title={s.account_name}>
                                                        <span className="text-slate-400 font-medium">A/N:</span> {s.account_name}
                                                    </p>
                                                )}
                                            </div>
                                        ) : (
                                            <span className="text-[10px] italic text-slate-400 pl-1">Belum diatur</span>
                                        )}
                                    </TableCell>

                                    <TableCell className="font-medium text-xs py-3">
                                        {s.is_active === true ? (
                                            <Badge variant="outline" className="border-green-50 bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-[10px] px-2">
                                                Active
                                            </Badge>
                                        ) : (
                                            <Badge variant="destructive">
                                                Inactive
                                            </Badge>
                                        )}
                                    </TableCell>
                                    {showActionColumn && (
                                        <TableCell className="text-right py-3 px-4">
                                            <div className="inline-flex gap-2 justify-end">
                                                {canUpdate && (
                                                    <button
                                                        type="button"
                                                        onClick={() => onEditClick(s)}
                                                        title="Edit Supplier"
                                                        aria-label={`Edit ${s.name}`}
                                                        className="p-1.5 border rounded-lg bg-background text-primary hover:bg-muted transition-colors cursor-pointer"
                                                    >
                                                        <Edit2 className="h-3.5 w-3.5" />
                                                    </button>
                                                )}
                                                {canDelete && (
                                                    <button
                                                        type="button"
                                                        onClick={() => onDeleteClick(s)}
                                                        title="Hapus Supplier"
                                                        aria-label={`Hapus ${s.name}`}
                                                        className="p-1.5 border rounded-lg bg-background text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                                                    >
                                                        <Trash2 className="h-3.5 w-3.5" />
                                                    </button>
                                                )}
                                            </div>
                                        </TableCell>
                                    )}
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            {/* Pagination Controls */}
            <DataTablePagination
                page={page}
                limit={limit}
                setLimit={setLimit}
                total={total}
                rowCount={suppliers.length}
                hasNext={hasNext}
                hasPrev={hasPrev}
                onNextPage={handleNextPage}
                onPrevPage={handlePrevPage}
                loading={loading}
            />
        </div>
    );
}