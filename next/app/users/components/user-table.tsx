"use client";

import React from "react";
import { useDebounceFetch } from "@/hooks/use-debounce";
import { Edit2, Trash2, Search, Filter, Users } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import { DataTablePagination } from "@/components/data-table-pagination";

// 1. Definisikan tipe data entitas
export interface User {
    id: number;
    name: string;
    email: string;
    role_name: string;
}

interface UserTableProps {
    canUpdate: boolean;
    canDelete: boolean;
    refreshTrigger: number;
    onDeleteClick: (user: User) => void;
    onEditClick: (user: User) => void;
    onRoleOptionsChange?: (options: string[]) => void;
    onForbidden?: () => void;
}

// 🌟 Komponen Skeleton Statis Reusable (Aman untuk SSR & Loading State)
export function UserTableSkeleton() {
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
                            <TableHead className="font-semibold text-xs">Nama Karyawan</TableHead>
                            <TableHead className="font-semibold text-xs">Email</TableHead>
                            <TableHead className="font-semibold text-xs">Role</TableHead>
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

export function UserTable({ canUpdate, canDelete, refreshTrigger, onDeleteClick, onEditClick, onRoleOptionsChange, onForbidden }: UserTableProps) {
    // Panggil custom hook universal dengan tipe <User> dan endpoint API Go
    const {
        search,
        setSearch,
        page,
        limit,
        setLimit,
        filters,
        updateFilter,
        data: users,
        total,
        loading,
        handleReset,
        isFiltered,
        roleOptions,
        hasNext,
        hasPrev,
        handleNextPage,
        handlePrevPage,
        httpStatus
    } = useDebounceFetch<User>({
        endpoint: "/api/users",
        delay: 500,
        initialFilters: {
            role: "ALL",
        },
        refreshTrigger: refreshTrigger
    });

    React.useEffect(() => {
        if (roleOptions && roleOptions.length > 0 && onRoleOptionsChange) {
            onRoleOptionsChange(roleOptions);
        }
    }, [roleOptions, onRoleOptionsChange]);

    // Teruskan sinyal 403 dari hook ke halaman induk agar ForbiddenView tampil.
    React.useEffect(() => {
        if (httpStatus === 403) onForbidden?.();
    }, [httpStatus, onForbidden]);

    const role = String(filters.role) || "ALL";
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
                counterLabel={`Total ${total} pengguna ditemukan`}
            >
                {/* Role Filter */}
                <div className="relative min-w-44">
                    <Filter className="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground pointer-events-none" />
                    <select
                        value={role}
                        disabled={loading && users.length === 0}
                        onChange={(e) => updateFilter('role', e.target.value)}
                        suppressHydrationWarning
                        className="h-9 w-full rounded-lg border border-input bg-background pl-8 pr-8 text-xs outline-none cursor-pointer appearance-none disabled:opacity-50"
                    >
                        <option value="ALL">Semua Role</option>
                        {roleOptions && roleOptions.map((roleName: string) => (
                            <option key={roleName} value={roleName} className="capitalize">
                                {roleName.charAt(0).toUpperCase() + roleName.slice(1)}
                            </option>
                        ))}
                    </select>
                </div>
            </DataTableToolbar>

            {/* Table Area */}
            <div className="border rounded-xl shadow-sm overflow-hidden bg-card relative">
                {/* Indikator Loading Tipis Saat Memperbarui Data yang Sudah Ada */}
                {loading && users.length > 0 && (
                    <div className="absolute top-0 left-0 right-0 h-0.5 bg-primary/20 overflow-hidden z-10">
                        <div className="h-full bg-primary animate-pulse w-full" />
                    </div>
                )}

                <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow>
                            <TableHead className="font-semibold text-xs">Nama Karyawan</TableHead>
                            <TableHead className="font-semibold text-xs">Email</TableHead>
                            <TableHead className="font-semibold text-xs">Role</TableHead>
                            {showActionColumn && <TableHead className="text-right font-semibold text-xs w-32">Aksi</TableHead>}
                        </TableRow>
                    </TableHeader>

                    <TableBody>
                        {/* 🌟 1. SKELETON LOADING: Muncul saat pertama kali memuat data */}
                        {loading && users.length === 0 ? (
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
                        ) : users.length === 0 ? (
                            /* 💡 2. EMPTY STATE: Hanya muncul jika selesai loading dan data benar-benar kosong */
                            <TableRow>
                                <TableCell
                                    colSpan={showActionColumn ? 4 : 3}
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
                            users.map((u) => (
                                <TableRow
                                    key={u.id}
                                    className={`hover:bg-muted/30 transition-opacity duration-200 ${loading ? "opacity-50 pointer-events-none" : ""}`}
                                >
                                    <TableCell className="font-medium text-xs py-3">{u.name}</TableCell>
                                    <TableCell className="text-xs text-muted-foreground py-3">{u.email}</TableCell>
                                    <TableCell className="py-3">
                                        <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-semibold border ${u.role_name?.toLowerCase().includes("admin") ? "bg-red-500/10 text-red-500 border-red-500/20" : "bg-blue-500/10 text-blue-500 border-blue-500/20"
                                            }`}>
                                            {u.role_name || "staff"}
                                        </span>
                                    </TableCell>
                                    {showActionColumn && (
                                        <TableCell className="text-right py-3">
                                            <div className="inline-flex gap-2 justify-end">
                                                {canUpdate && (
                                                    <button
                                                        type="button"
                                                        onClick={() => onEditClick(u)}
                                                        title="Edit Pengguna"
                                                        aria-label={`Edit ${u.name}`}
                                                        className="p-1.5 border rounded-lg bg-background text-primary hover:bg-muted transition-colors cursor-pointer"
                                                    >
                                                        <Edit2 className="h-3.5 w-3.5" />
                                                    </button>
                                                )}
                                                {canDelete && (
                                                    <button
                                                        type="button"
                                                        onClick={() => onDeleteClick(u)}
                                                        title="Hapus Pengguna"
                                                        aria-label={`Hapus ${u.name}`}
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
                rowCount={users.length}
                hasNext={hasNext}
                hasPrev={hasPrev}
                onNextPage={handleNextPage}
                onPrevPage={handlePrevPage}
                loading={loading}
            />
        </div>
    );
}
