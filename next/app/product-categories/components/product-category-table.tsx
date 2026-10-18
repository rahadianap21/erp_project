"use client";

import { Edit2, Trash2, FolderTree, Link2, Search, ChevronRight } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import React, { useEffect, useMemo, useState } from "react";
import Cookies from "js-cookie";

interface ProductCategory {
    id: number;
    name: string;
    slug: string;
    description: string;
    parent_id: number | null;
    parent_name: string;
    is_active: boolean;
}

interface ProductCategoryTableProps {
    canUpdate: boolean;
    canDelete: boolean;
    refreshTrigger: number;
    onDeleteClick: (cat: ProductCategory) => void;
    onEditClick: (cat: ProductCategory) => void;
    onForbidden?: () => void;
}

export function ProductCategoryTable({ canUpdate, canDelete, refreshTrigger, onDeleteClick, onEditClick, onForbidden }: ProductCategoryTableProps) {
    const [categories, setCategories] = useState<ProductCategory[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [search, setSearch] = useState<string>("");
    const [expanded, setExpanded] = useState<Set<number>>(new Set());

    const authToken = Cookies.get("session_active");

    // Simpan callback di ref agar perubahan identitas fungsi dari halaman induk
    // tidak memicu ulang fetch (dependensi efek tetap [authToken, refreshTrigger]).
    const onForbiddenRef = React.useRef(onForbidden);
    React.useEffect(() => {
        onForbiddenRef.current = onForbidden;
    });

    useEffect(() => {
        const fetchAll = async () => {
            setLoading(true);
            try {
                const res = await fetch(`/api/categories?limit=1000`, {
                    headers: { "Authorization": `Bearer ${authToken}` },
                    cache: "no-store",
                });
                if (res.status === 403) {
                    onForbiddenRef.current?.();
                    return;
                }
                const result = await res.json();
                const rows: ProductCategory[] = Array.isArray(result.data) ? result.data : [];
                setCategories(rows);
            } catch (err) {
                console.error("Gagal memuat kategori produk:", err);
            } finally {
                setLoading(false);
            }
        };
        fetchAll();
    }, [authToken, refreshTrigger]);

    // Peta relasi: anak per induk + lookup cepat per id. Induk yang tidak ada
    // di daftar diperlakukan sebagai root agar tidak hilang.
    const { byId, childrenMap, roots } = useMemo(() => {
        const map = new Map<number, ProductCategory>();
        categories.forEach((c) => map.set(c.id, c));

        const children = new Map<number | null, ProductCategory[]>();
        categories.forEach((c) => {
            const parentKey = c.parent_id !== null && map.has(c.parent_id) ? c.parent_id : null;
            const bucket = children.get(parentKey) ?? [];
            bucket.push(c);
            children.set(parentKey, bucket);
        });
        children.forEach((bucket) => bucket.sort((x, y) => x.name.localeCompare(y.name)));

        return { byId: map, childrenMap: children, roots: children.get(null) ?? [] };
    }, [categories]);

    const query = search.trim().toLowerCase();

    // Saat mencari: tampilkan kategori yang cocok beserta seluruh leluhurnya agar
    // konteks hierarki tetap terlihat; anak yang tidak cocok disembunyikan.
    const visibleIds = useMemo(() => {
        const visible = new Set<number>();
        if (!query) {
            categories.forEach((c) => visible.add(c.id));
            return visible;
        }
        categories
            .filter((c) => c.name.toLowerCase().includes(query) || c.slug.toLowerCase().includes(query))
            .forEach((match) => {
                visible.add(match.id);
                let parent = match.parent_id;
                while (parent !== null && byId.has(parent)) {
                    visible.add(parent);
                    parent = byId.get(parent)!.parent_id;
                }
            });
        return visible;
    }, [categories, byId, query]);

    const toggleExpand = (id: number) => {
        setExpanded((prev) => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id);
            else next.add(id);
            return next;
        });
    };

    const showActionColumn = canUpdate || canDelete;
    const columnCount = showActionColumn ? 5 : 4;

    const renderRows = (nodes: ProductCategory[], depth: number): React.ReactNode[] => {
        return nodes.flatMap((c) => {
            if (!visibleIds.has(c.id)) return [];

            const kids = (childrenMap.get(c.id) ?? []).filter((k) => visibleIds.has(k.id));
            const expandable = kids.length > 0;
            const isExpanded = query ? true : expanded.has(c.id);

            const row = (
                <TableRow
                    key={c.id}
                    onClick={expandable ? () => toggleExpand(c.id) : undefined}
                    className={`transition-colors ${expandable ? "cursor-pointer hover:bg-slate-50/70" : ""} ${expandable && isExpanded ? "bg-slate-50/50" : ""} ${loading ? "opacity-50 pointer-events-none" : ""}`}
                >
                    <TableCell className="text-xs py-3.5">
                        <div className="flex items-center" style={{ paddingLeft: depth * 20 }}>
                            {expandable ? (
                                <button
                                    type="button"
                                    onClick={(e) => { e.stopPropagation(); toggleExpand(c.id); }}
                                    title={isExpanded ? "Tutup sub-kategori" : "Buka sub-kategori"}
                                    className="mr-1.5 p-0.5 rounded hover:bg-slate-200/70 text-slate-500 cursor-pointer shrink-0"
                                >
                                    <ChevronRight className={`size-3.5 transition-transform duration-200 ${isExpanded ? "rotate-90" : ""}`} />
                                </button>
                            ) : (
                                <span className="mr-1.5 inline-block size-4 shrink-0" />
                            )}
                            <FolderTree className="size-3.5 text-slate-400 shrink-0 mr-1.5" />
                            <span className="font-bold text-slate-900 uppercase tracking-tight">{c.name}</span>
                        </div>
                    </TableCell>
                    <TableCell className="text-xs py-3.5 font-mono text-slate-500 text-[11px]">
                        <span className="flex items-center gap-1"><Link2 className="size-3 text-slate-300" /> {c.slug}</span>
                    </TableCell>
                    <TableCell className="text-xs py-3.5 text-slate-500 italic max-w-xs truncate">
                        {c.description ? `"${c.description}"` : <span className="text-slate-300">-</span>}
                    </TableCell>
                    <TableCell className="text-center py-3.5">
                        <span className={`inline-flex px-2 py-0.5 rounded-full text-[10px] font-bold border ${c.is_active ? "bg-emerald-50 text-emerald-600 border-emerald-200" : "bg-slate-50 text-slate-400 border-slate-200"}`}>
                            {c.is_active ? "AKTIF" : "OFF"}
                        </span>
                    </TableCell>
                    {showActionColumn && (
                        <TableCell className="text-right py-3.5">
                            <div className="inline-flex gap-1.5 justify-end">
                                {canUpdate && <button type="button" onClick={(e) => { e.stopPropagation(); onEditClick(c); }} className="p-1 border border-slate-200 rounded-md bg-white hover:bg-slate-50 text-slate-600 cursor-pointer shadow-3xs"><Edit2 className="size-3" /></button>}
                                {canDelete && <button type="button" onClick={(e) => { e.stopPropagation(); onDeleteClick(c); }} className="p-1 border border-slate-200 rounded-md bg-white hover:bg-rose-50 text-rose-600 cursor-pointer shadow-3xs"><Trash2 className="size-3" /></button>}
                            </div>
                        </TableCell>
                    )}
                </TableRow>
            );

            const childRows = expandable && isExpanded ? renderRows(kids, depth + 1) : [];
            return [row, ...childRows];
        });
    };

    const bodyRows = renderRows(roots, 0);
    const visibleCount = visibleIds.size;

    return (
        <div className="space-y-4">
            {/* Filter Search Bar */}
            <DataTableToolbar
                search={search}
                setSearch={setSearch}
                placeholder="Cari berdasarkan nama atau slug kategori..."
                loading={loading}
                isFiltered={search !== ""}
                onReset={() => setSearch("")}
                counterIcon={<FolderTree className="size-3.5 text-slate-400" />}
                counterLabel={`Total ${visibleCount} kategori terdaftar`}
            />

            {/* Table Area */}
            <div className="border rounded-xl shadow-xs overflow-hidden bg-white relative">
                {loading && categories.length > 0 && (
                    <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] z-10 flex items-center justify-center">
                        <Skeleton className="h-5 w-40" />
                    </div>
                )}

                <Table>
                    <TableHeader className="bg-slate-50/70">
                        <TableRow>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/3">Klasifikasi Kategori</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/4">URL Slug</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/3">Catatan / Deskripsi</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-center w-24">Status</TableHead>
                            {showActionColumn && <TableHead className="text-right font-bold text-xs w-24">Aksi</TableHead>}
                        </TableRow>
                    </TableHeader>

                    <TableBody>
                        {loading && categories.length === 0 ? (
                            Array.from({ length: 5 }).map((_, idx) => (
                                <TableRow key={`cat-row-skeleton-${idx}`}>
                                    <TableCell colSpan={columnCount} className="py-4"><Skeleton className="h-4 w-full" /></TableCell>
                                </TableRow>
                            ))
                        ) : bodyRows.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={columnCount} className="h-32 text-center text-muted-foreground font-medium text-xs">
                                    <div className="flex flex-col items-center justify-center gap-2 py-4">
                                        <Search className="size-5 text-muted-foreground/50" />
                                        <p>
                                            {search !== ""
                                                ? "Data tidak ditemukan. Coba sesuaikan kata kunci pencarian Anda."
                                                : "Tidak ada data kategori yang terindeks di sistem."}
                                        </p>
                                        {search !== "" && (
                                            <Button
                                                type="button"
                                                variant="outline"
                                                size="sm"
                                                onClick={() => setSearch("")}
                                                className="mt-1 h-7 text-[11px] cursor-pointer"
                                            >
                                                Bersihkan Filter
                                            </Button>
                                        )}
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : (
                            bodyRows
                        )}
                    </TableBody>
                </Table>
            </div>

            <p className="text-[11px] text-slate-400 text-center">
                {roots.length} kategori utama · klik baris kategori untuk melihat/menyembunyikan sub-kategori.
            </p>
        </div>
    );
}
