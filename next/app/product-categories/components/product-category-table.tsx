"use client";

import { ApiError, apiGet } from "@/lib/api-client";
import {
  Edit2,
  Trash2,
  FolderTree,
  Link2,
  Search,
  ChevronRight,
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

interface CategoryNode {
  cat: ProductCategory;
  depth: number;
  expandable: boolean;
  isExpanded: boolean;
}

interface ProductCategoryTableProps {
  canUpdate: boolean;
  canDelete: boolean;
  refreshTrigger: number;
  onDeleteClick: (cat: ProductCategory) => void;
  onEditClick: (cat: ProductCategory) => void;
  onForbidden?: () => void;
}

export function ProductCategoryTable({
  canUpdate,
  canDelete,
  refreshTrigger,
  onDeleteClick,
  onEditClick,
  onForbidden,
}: ProductCategoryTableProps) {
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
        let result;
        try {
          result = await apiGet<ProductCategory[]>(
            `/api/categories?limit=1000`,
            { redirectOn403: false }
          );
        } catch (err) {
          if (err instanceof ApiError && err.status === 403) {
            onForbiddenRef.current?.();
            return;
          }
          throw err;
        }
        const rows: ProductCategory[] = Array.isArray(result.data)
          ? result.data
          : [];
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
      const parentKey =
        c.parent_id !== null && map.has(c.parent_id) ? c.parent_id : null;
      const bucket = children.get(parentKey) ?? [];
      bucket.push(c);
      children.set(parentKey, bucket);
    });
    children.forEach((bucket) =>
      bucket.sort((x, y) => x.name.localeCompare(y.name))
    );

    return {
      byId: map,
      childrenMap: children,
      roots: children.get(null) ?? [],
    };
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
      .filter(
        (c) =>
          c.name.toLowerCase().includes(query) ||
          c.slug.toLowerCase().includes(query)
      )
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

  // Ratakan hierarki sesuai state expand/search agar satu sumber data dipakai
  // oleh tabel desktop maupun card list mobile.
  const flatNodes = useMemo(() => {
    const walk = (nodes: ProductCategory[], depth: number): CategoryNode[] =>
      nodes.flatMap((cat) => {
        if (!visibleIds.has(cat.id)) return [];

        const kids = (childrenMap.get(cat.id) ?? []).filter((k) =>
          visibleIds.has(k.id)
        );
        const expandable = kids.length > 0;
        const isExpanded = query ? true : expanded.has(cat.id);

        return [
          { cat, depth, expandable, isExpanded },
          ...(expandable && isExpanded ? walk(kids, depth + 1) : []),
        ];
      });

    return walk(roots, 0);
  }, [roots, visibleIds, childrenMap, expanded, query]);

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

      {/* Mobile: hierarchical card list */}
      <div className="space-y-2 md:hidden">
        {loading && categories.length === 0 ? (
          Array.from({ length: 5 }).map((_, idx) => (
            <div
              key={`cat-card-skeleton-${idx}`}
              className="space-y-2 rounded-xl border bg-white p-4 shadow-xs"
            >
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-3 w-28" />
            </div>
          ))
        ) : flatNodes.length === 0 ? (
          <div className="rounded-xl border bg-white p-8 text-center text-muted-foreground">
            <div className="flex flex-col items-center justify-center gap-2">
              <Search className="size-5 text-muted-foreground/50" />
              <p className="text-sm font-medium">
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
                  className="mt-1 h-7 cursor-pointer text-sm"
                >
                  Bersihkan Filter
                </Button>
              )}
            </div>
          </div>
        ) : (
          flatNodes.map(({ cat: c, depth, expandable, isExpanded }) => (
            <article
              key={c.id}
              onClick={expandable ? () => toggleExpand(c.id) : undefined}
              style={{ marginLeft: Math.min(depth, 4) * 12 }}
              className={`rounded-xl border bg-white p-3.5 shadow-xs transition-colors ${expandable ? "cursor-pointer" : ""
                } ${isExpanded ? "border-slate-300 bg-slate-50/60" : ""} ${loading ? "pointer-events-none opacity-50" : ""
                }`}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="flex min-w-0 items-start gap-1.5">
                  {expandable ? (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleExpand(c.id);
                      }}
                      aria-label={
                        isExpanded ? "Tutup sub-kategori" : "Buka sub-kategori"
                      }
                      className="mt-0.5 shrink-0 cursor-pointer rounded p-0.5 text-slate-500 hover:bg-slate-200/70"
                    >
                      <ChevronRight
                        className={`size-3.5 transition-transform duration-200 ${isExpanded ? "rotate-90" : ""
                          }`}
                      />
                    </button>
                  ) : (
                    <span className="mt-0.5 inline-block size-4 shrink-0" />
                  )}
                  <FolderTree className="mt-0.5 size-3.5 shrink-0 text-slate-400" />
                  <h3 className="text-sm font-medium tracking-tight text-foreground">
                    {c.name}
                  </h3>
                </div>
                <Badge
                  variant={c.is_active ? "success" : "neutral"}
                  className="shrink-0"
                >
                  {c.is_active ? "Active" : "OFF"}
                </Badge>
              </div>

              <p className="mt-1.5 flex items-center gap-1 pl-6 font-mono text-sm text-muted-foreground">
                <Link2 className="size-3 text-slate-300" /> {c.slug}
              </p>
              {c.description && (
                <p className="mt-1 line-clamp-2 pl-6 text-sm italic text-slate-500">
                  &quot;{c.description}&quot;
                </p>
              )}

              {showActionColumn && (
                <div className="mt-2.5 flex justify-end gap-1.5 border-t pt-2.5">
                  {canUpdate && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onEditClick(c);
                      }}
                      className="flex cursor-pointer items-center gap-1 rounded-md border border-slate-200 bg-white px-2.5 py-1 text-sm text-slate-600 shadow-3xs hover:bg-slate-50"
                    >
                      <Edit2 className="size-3" /> Edit
                    </button>
                  )}
                  {canDelete && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        onDeleteClick(c);
                      }}
                      className="flex cursor-pointer items-center gap-1 rounded-md border border-slate-200 bg-white px-2.5 py-1 text-sm text-rose-600 shadow-3xs hover:bg-rose-50"
                    >
                      <Trash2 className="size-3" /> Hapus
                    </button>
                  )}
                </div>
              )}
            </article>
          ))
        )}
      </div>

      {/* Desktop: table */}
      <div className="hidden border rounded-xl shadow-xs overflow-hidden bg-white relative md:block">
        {loading && categories.length > 0 && (
          <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] z-10 flex items-center justify-center">
            <Skeleton className="h-5 w-40" />
          </div>
        )}

        <Table>
          <TableHeader className="bg-slate-50/70">
            <TableRow>
              <TableHead className="w-1/3">
                Klasifikasi Kategori
              </TableHead>
              <TableHead className="w-1/4">
                URL Slug
              </TableHead>
              <TableHead className="w-1/3">
                Catatan / Deskripsi
              </TableHead>
              <TableHead className="text-center w-24">
                Status
              </TableHead>
              {showActionColumn && (
                <TableHead className="text-right w-24">
                  Aksi
                </TableHead>
              )}
            </TableRow>
          </TableHeader>

          <TableBody>
            {loading && categories.length === 0 ? (
              Array.from({ length: 5 }).map((_, idx) => (
                <TableRow key={`cat-row-skeleton-${idx}`}>
                  <TableCell colSpan={columnCount} className="py-4">
                    <Skeleton className="h-4 w-full" />
                  </TableCell>
                </TableRow>
              ))
            ) : flatNodes.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={columnCount}
                  className="h-32 text-center text-muted-foreground font-medium text-sm"
                >
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
                        className="mt-1 h-7 text-sm cursor-pointer"
                      >
                        Bersihkan Filter
                      </Button>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              flatNodes.map(({ cat: c, depth, expandable, isExpanded }) => (
                <TableRow
                  key={c.id}
                  onClick={expandable ? () => toggleExpand(c.id) : undefined}
                  className={`transition-colors ${expandable ? "cursor-pointer hover:bg-slate-50/70" : ""} ${expandable && isExpanded ? "bg-slate-50/50" : ""} ${loading ? "opacity-50 pointer-events-none" : ""}`}
                >
                  <TableCell className="text-sm py-3.5">
                    <div
                      className="flex items-center"
                      style={{ paddingLeft: depth * 20 }}
                    >
                      {expandable ? (
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleExpand(c.id);
                          }}
                          title={
                            isExpanded
                              ? "Tutup sub-kategori"
                              : "Buka sub-kategori"
                          }
                          className="mr-1.5 p-0.5 rounded hover:bg-slate-200/70 text-slate-500 cursor-pointer shrink-0"
                        >
                          <ChevronRight
                            className={`size-3.5 transition-transform duration-200 ${isExpanded ? "rotate-90" : ""}`}
                          />
                        </button>
                      ) : (
                        <span className="mr-1.5 inline-block size-4 shrink-0" />
                      )}
                      <FolderTree className="size-3.5 text-slate-400 shrink-0 mr-1.5" />
                      <span className="font-medium text-foreground tracking-tight">
                        {c.name}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-sm py-3.5 font-mono text-slate-500 text-sm">
                    <span className="flex items-center gap-1">
                      <Link2 className="size-3 text-slate-300" /> {c.slug}
                    </span>
                  </TableCell>
                  <TableCell className="text-sm py-3.5 text-slate-500 italic max-w-xs truncate">
                    {c.description ? (
                      `"${c.description}"`
                    ) : (
                      <span className="text-slate-300">-</span>
                    )}
                  </TableCell>
                  <TableCell className="text-center py-3.5">
                    <Badge variant={c.is_active ? "success" : "neutral"}>
                      {c.is_active ? "Active" : "OFF"}
                    </Badge>
                  </TableCell>
                  {showActionColumn && (
                    <TableCell className="text-right py-3.5">
                      <div className="inline-flex gap-1.5 justify-end">
                        {canUpdate && (
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              onEditClick(c);
                            }}
                            className="p-1 border border-slate-200 rounded-md bg-white hover:bg-slate-50 text-slate-600 cursor-pointer shadow-3xs"
                          >
                            <Edit2 className="size-3" />
                          </button>
                        )}
                        {canDelete && (
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              onDeleteClick(c);
                            }}
                            className="p-1 border border-slate-200 rounded-md bg-white hover:bg-rose-50 text-rose-600 cursor-pointer shadow-3xs"
                          >
                            <Trash2 className="size-3" />
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

      <p className="text-sm text-slate-400 text-center">
        {roots.length} kategori utama · klik baris kategori untuk
        melihat/menyembunyikan sub-kategori.
      </p>
    </div>
  );
}
