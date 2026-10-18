"use client";

import React, { useState } from "react";
import { useDebounceFetch } from "@/hooks/use-debounce";
import { Edit2, Trash2, Search, Users, KeyRound, ChevronRight, Folder } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import { DataTablePagination } from "@/components/data-table-pagination";

// 1. Definisikan tipe data entitas
export interface Role {
    id: number;
    name: string;
    permissions: string[];
}

interface RoleTableProps {
    canUpdate: boolean;
    canDelete: boolean;
    refreshTrigger: number;
    onDeleteClick: (role: Role) => void;
    onEditClick: (role: Role) => void;
    onForbidden?: () => void;
}

// 🌟 Komponen Skeleton Statis Reusable (Aman untuk SSR & Loading State)
export function RoleTableSkeleton() {
    return (
        <div className="space-y-4">
            {/* Filter & Search Bar Skeleton */}
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-card p-4 rounded-xl border shadow-sm">
                <div className="flex flex-1 flex-col sm:flex-row items-stretch sm:items-center gap-3">
                    <Skeleton className="h-9 flex-1 max-w-md rounded-md" />
                </div>
                <Skeleton className="h-5 w-36" />
            </div>

            {/* Table Skeleton */}
            <div className="border rounded-xl shadow-sm overflow-hidden bg-card">
                <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow>
                            <TableHead className="w-10"></TableHead>
                            <TableHead className="font-semibold text-xs">Nama Peran</TableHead>
                            <TableHead className="text-right font-semibold text-xs w-32">Aksi</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {Array.from({ length: 5 }).map((_, idx) => (
                            <TableRow key={`role-skel-${idx}`} className="hover:bg-transparent">
                                <TableCell className="py-3.5 text-center w-10">
                                    <Skeleton className="h-4 w-4 mx-auto rounded" />
                                </TableCell>
                                <TableCell className="py-3.5">
                                    <Skeleton className="h-4 w-40" />
                                </TableCell>
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

export function RoleTable({ canUpdate, canDelete, refreshTrigger, onDeleteClick, onEditClick, onForbidden }: RoleTableProps) {
    // Panggil custom hook universal dengan tipe <Role> dan endpoint API Go
    const {
        search,
        setSearch,
        page,
        limit,
        setLimit,
        data: roles,
        total,
        loading,
        handleReset,
        isFiltered,
        hasNext,
        hasPrev,
        handleNextPage,
        handlePrevPage,
        httpStatus
    } = useDebounceFetch<Role>({
        endpoint: "/api/roles",
        delay: 500,
        initialFilters: {},
        refreshTrigger: refreshTrigger
    });

    // Teruskan sinyal 403 dari hook ke halaman induk agar ForbiddenView tampil.
    React.useEffect(() => {
        if (httpStatus === 403) onForbidden?.();
    }, [httpStatus, onForbidden]);

    const showActionColumn = canUpdate || canDelete;

    const [expandedRows, setExpandedRows] = useState<Record<number, boolean>>({});
    const toggleRow = (roleId: number) => {
        setExpandedRows((prev) => ({
            ...prev,
            [roleId]: !prev[roleId]
        }));
    };

    const getPermissionBadgeColor = (permName: string) => {
        const lowerPerm = permName.toLowerCase();

        if (lowerPerm.includes("read") || lowerPerm.includes("view")) {
            return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20";
        }
        if (lowerPerm.includes("create") || lowerPerm.includes("write") || lowerPerm.includes("add")) {
            return "bg-blue-500/10 text-blue-600 border-blue-500/20";
        }
        if (lowerPerm.includes("update") || lowerPerm.includes("edit") || lowerPerm.includes("modify")) {
            return "bg-amber-500/10 text-amber-600 border-amber-500/20";
        }
        if (lowerPerm.includes("delete") || lowerPerm.includes("remove") || lowerPerm.includes("destroy")) {
            return "bg-rose-500/10 text-rose-600 border-rose-500/20";
        }

        return "bg-slate-500/10 text-slate-600 border-slate-500/20";
    };

    const groupPermissionsByPrefix = (permissions: string[]) => {
        if (!permissions) return {};

        return permissions.reduce((acc, perm) => {
            const parts = perm.split(":");
            const prefix = parts[0] || "Lainnya";

            if (!acc[prefix]) {
                acc[prefix] = [];
            }
            acc[prefix].push(perm);
            return acc;
        }, {} as Record<string, string[]>);
    };

    return (
        <div className="space-y-4">
            {/* Filter & Search Bar */}
            <DataTableToolbar
                search={search}
                setSearch={setSearch}
                placeholder="Cari berdasarkan nama peran..."
                loading={loading}
                isFiltered={isFiltered}
                onReset={handleReset}
                counterIcon={<Users className="size-3.5 text-slate-400" />}
                counterLabel={`Total ${total} peran ditemukan`}
            />

            {/* Table Area */}
            <div className="border rounded-xl shadow-sm overflow-hidden bg-card relative">
                {/* Indikator Loading Tipis Saat Memperbarui Data yang Sudah Ada */}
                {loading && roles.length > 0 && (
                    <div className="absolute top-0 left-0 right-0 h-0.5 bg-primary/20 overflow-hidden z-10">
                        <div className="h-full bg-primary animate-pulse w-full" />
                    </div>
                )}

                <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow>
                            <TableHead className="w-10"></TableHead>
                            <TableHead className="font-semibold text-xs">Nama Peran</TableHead>
                            {showActionColumn && <TableHead className="text-right font-semibold text-xs w-32">Aksi</TableHead>}
                        </TableRow>
                    </TableHeader>

                    <TableBody>
                        {/* 🌟 1. SKELETON LOADING: Muncul saat pertama kali memuat data */}
                        {loading && roles.length === 0 ? (
                            Array.from({ length: 5 }).map((_, idx) => (
                                <TableRow key={`role-row-skeleton-${idx}`} className="hover:bg-transparent">
                                    <TableCell className="py-3.5 text-center w-10">
                                        <Skeleton className="h-4 w-4 mx-auto rounded" />
                                    </TableCell>
                                    <TableCell className="py-3.5">
                                        <Skeleton className="h-4 w-40" />
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
                        ) : roles.length === 0 ? (
                            /* 💡 2. EMPTY STATE: Hanya muncul jika selesai loading dan data benar-benar kosong */
                            <TableRow>
                                <TableCell
                                    colSpan={showActionColumn ? 3 : 2}
                                    className="h-32 text-center text-muted-foreground font-medium text-xs"
                                >
                                    <div className="flex flex-col items-center justify-center gap-2 py-4">
                                        <Search className="size-5 text-muted-foreground/50" />
                                        <p>
                                            {isFiltered
                                                ? "Data tidak ditemukan. Coba sesuaikan kata kunci pencarian Anda."
                                                : "Tidak ada data peran yang terindeks di sistem."}
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
                            roles.map((r) => {
                                const isExpanded = !!expandedRows[r.id];
                                const groupedPermissions = groupPermissionsByPrefix(r.permissions);
                                const prefixes = Object.keys(groupedPermissions);
                                return (
                                    <React.Fragment key={r.id}>
                                        <TableRow onClick={() => toggleRow(r.id)} className={`cursor-pointer hover:bg-muted/30 transition-colors ${isExpanded ? "bg-muted/10 border-b-0" : ""} ${loading ? "opacity-50 pointer-events-none" : ""}`}>
                                            <TableCell className="py-3 text-center">
                                                <button
                                                    type="button"
                                                    onClick={(e) => { e.stopPropagation(); toggleRow(r.id); }}
                                                    title={isExpanded ? "Tutup detail hak akses" : "Lihat detail hak akses"}
                                                    aria-label={isExpanded ? `Tutup detail hak akses untuk peran ${r.name}` : `Lihat detail hak akses untuk peran ${r.name}`}
                                                    className="p-1 rounded-md hover:bg-muted text-muted-foreground transition-colors cursor-pointer"
                                                >
                                                    <ChevronRight className={`size-3.5 transition-transform duration-300 ease-in-out ${isExpanded ? "transform rotate-90 text-blue-600" : ""}`} />
                                                </button>
                                            </TableCell>
                                            <TableCell className="font-medium text-xs py-3 text-foreground capitalize">{r.name}</TableCell>
                                            {showActionColumn && (
                                                <TableCell className="text-right py-3">
                                                    <div className="inline-flex gap-2">
                                                        {canUpdate && (
                                                            <button
                                                                type="button"
                                                                onClick={(e) => { e.stopPropagation(); onEditClick(r); }}
                                                                title="Edit Peran"
                                                                aria-label={`Edit peran ${r.name}`}
                                                                className="p-1.5 border rounded-lg bg-background text-primary hover:bg-muted transition-colors cursor-pointer"
                                                            >
                                                                <Edit2 className="h-3.5 w-3.5" />
                                                            </button>
                                                        )}
                                                        {canDelete && (
                                                            <button
                                                                type="button"
                                                                onClick={(e) => { e.stopPropagation(); onDeleteClick(r); }}
                                                                title="Hapus Peran"
                                                                aria-label={`Hapus peran ${r.name}`}
                                                                className="p-1.5 border rounded-lg bg-background text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                                                            >
                                                                <Trash2 className="h-3.5 w-3.5" />
                                                            </button>
                                                        )}
                                                    </div>
                                                </TableCell>
                                            )}
                                        </TableRow>
                                        <TableRow className={`border-t-0 p-0 hover:bg-transparent transition-all duration-300 ease-in-out ${isExpanded ? "bg-muted/10 shadow-inner" : "pointer-events-none"}`}>
                                            <TableCell colSpan={showActionColumn ? 3 : 2} className="p-0 border-t-0">
                                                <div className={`grid transition-all duration-300 ease-in-out ${isExpanded ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"}`}>
                                                    <div className="overflow-hidden">
                                                        <div className="py-4 pl-12 pr-6 space-y-2">
                                                            <div className="text-[11px] font-semibold text-foreground/80 flex items-center gap-1.5 uppercase tracking-wider">
                                                                <KeyRound className="size-3 text-blue-500" /> Permissions
                                                            </div>
                                                            {prefixes.length > 0 ? (
                                                                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 pt-1">
                                                                    {prefixes.map((prefix) => (
                                                                        <div
                                                                            key={`group-${prefix}-${r.id}`}
                                                                            className="bg-card border p-3 rounded-lg shadow-2xs flex flex-col gap-2 transition-all duration-300 ease-in-out"
                                                                        >
                                                                            <div className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide flex items-center gap-1.5 border-b pb-1.5 border-border">
                                                                                <Folder className="size-3 text-muted-foreground/70" />
                                                                                <span>Modul {prefix}</span>
                                                                            </div>

                                                                            <div className="flex flex-wrap gap-1.5">
                                                                                {groupedPermissions[prefix].map((perm) => (
                                                                                    <span
                                                                                        key={perm}
                                                                                        className={`inline-flex items-center px-2 py-0.5 rounded-md border font-mono text-[10px] transition-all duration-200 ease-in-out ${getPermissionBadgeColor(perm)}`}
                                                                                    >
                                                                                        {perm}
                                                                                    </span>
                                                                                ))}
                                                                            </div>
                                                                        </div>
                                                                    ))}
                                                                </div>
                                                            ) : (
                                                                <p className="text-xs text-muted-foreground italic pl-0.5 pt-0.5">
                                                                    Belum ada hak akses permission yang disematkan untuk peran ini.
                                                                </p>
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

            {/* Pagination Controls */}
            <DataTablePagination
                page={page}
                limit={limit}
                setLimit={setLimit}
                total={total}
                rowCount={roles.length}
                hasNext={hasNext}
                hasPrev={hasPrev}
                onNextPage={handleNextPage}
                onPrevPage={handlePrevPage}
                loading={loading}
            />
        </div>
    );
}
