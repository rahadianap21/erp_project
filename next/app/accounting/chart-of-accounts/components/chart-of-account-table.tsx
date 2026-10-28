"use client";

import { apiGet } from "@/lib/api-client";
import {
  Edit2,
  Trash2,
  BookOpenCheck,
  Shield,
  FolderOpen,
  FileText,
  CornerUpLeft,
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
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import React, { useEffect, useMemo, useRef, useState } from "react";

export interface ChartOfAccount {
  id: number;
  code: string;
  name: string;
  account_type: "ASSET" | "LIABILITY" | "EQUITY" | "REVENUE" | "EXPENSE";
  normal_balance: "DEBIT" | "CREDIT";
  parent_id: number | null;
  parent_code?: string;
  parent_name?: string;
  is_header: boolean;
  is_active: boolean;
}

interface AccountNode {
  acc: ChartOfAccount;
  depth: number;
  expandable: boolean;
  isExpanded: boolean;
  /** Sedang beranimasi menutup: baris masih tampil sampai animasi selesai. */
  exiting: boolean;
}

const EXPAND_ANIMATION_MS = 200;

interface ChartOfAccountTableProps {
  canUpdate: boolean;
  canDelete: boolean;
  refreshTrigger: number;
  onDeleteClick: (acc: ChartOfAccount) => void;
  onEditClick: (acc: ChartOfAccount) => void;
}

const TYPE_VARIANT = {
  ASSET: "info",
  LIABILITY: "warning",
  EQUITY: "violet",
  REVENUE: "success",
  EXPENSE: "danger",
} as const;

export function ChartOfAccountTable({
  canUpdate,
  canDelete,
  refreshTrigger,
  onDeleteClick,
  onEditClick,
}: ChartOfAccountTableProps) {
  const [accounts, setAccounts] = useState<ChartOfAccount[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [search, setSearch] = useState<string>("");
  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [closing, setClosing] = useState<Set<number>>(new Set());
  const closeTimers = useRef<Map<number, ReturnType<typeof setTimeout>>>(
    new Map()
  );

  // Bersihkan timer animasi tutup saat komponen dilepas.
  useEffect(() => {
    const timers = closeTimers.current;
    return () => {
      timers.forEach((timer) => clearTimeout(timer));
      timers.clear();
    };
  }, []);

  useEffect(() => {
    const fetchAll = async () => {
      setLoading(true);
      try {
        const result = await apiGet<ChartOfAccount[]>(
          `/api/chart-of-accounts?limit=1000`
        );
        const rows: ChartOfAccount[] = Array.isArray(result.data)
          ? result.data
          : [];
        setAccounts(rows);

        // Semua akun induk (yang punya anak) dibuka otomatis di awal.
        // const ids = new Set<number>();
        // const parentIds = new Set(rows.map((r) => r.parent_id).filter((x): x is number => x !== null));
        // rows.forEach((r) => { if (parentIds.has(r.id)) ids.add(r.id); });
        // setExpanded(ids);
      } catch (err) {
        console.error("Gagal memuat bagan akun:", err);
      } finally {
        setLoading(false);
      }
    };
    fetchAll();
  }, [refreshTrigger]);

  // Peta relasi: anak per induk + lookup cepat per id. Induk yang tidak ada
  // di daftar (mis. sudah nonaktif) diperlakukan sebagai root agar tidak hilang.
  const { byId, childrenMap, roots } = useMemo(() => {
    const map = new Map<number, ChartOfAccount>();
    accounts.forEach((a) => map.set(a.id, a));

    const children = new Map<number | null, ChartOfAccount[]>();
    accounts.forEach((a) => {
      const parentKey =
        a.parent_id !== null && map.has(a.parent_id) ? a.parent_id : null;
      const bucket = children.get(parentKey) ?? [];
      bucket.push(a);
      children.set(parentKey, bucket);
    });
    children.forEach((bucket) =>
      bucket.sort((x, y) => x.code.localeCompare(y.code))
    );

    return {
      byId: map,
      childrenMap: children,
      roots: children.get(null) ?? [],
    };
  }, [accounts]);

  const query = search.trim().toLowerCase();

  // Saat mencari: tampilkan akun yang cocok beserta seluruh leluhurnya agar
  // konteks hierarki tetap terlihat; anak yang tidak cocok disembunyikan.
  const visibleIds = useMemo(() => {
    const visible = new Set<number>();
    if (!query) {
      accounts.forEach((a) => visible.add(a.id));
      return visible;
    }
    accounts
      .filter(
        (a) =>
          a.code.toLowerCase().includes(query) ||
          a.name.toLowerCase().includes(query)
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
  }, [accounts, byId, query]);

  // Tutup dilakukan dengan menunda pelepasan anak akun selama durasi animasi,
  // sehingga baris anak sempat memudar sebelum dihapus dari pohon.
  const toggleExpand = (id: number) => {
    const pendingClose = closeTimers.current.get(id);
    if (pendingClose) {
      clearTimeout(pendingClose);
      closeTimers.current.delete(id);
      setClosing((prev) => {
        if (!prev.has(id)) return prev;
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      setExpanded((prev) => (prev.has(id) ? prev : new Set(prev).add(id)));
      return;
    }

    if (expanded.has(id)) {
      setClosing((prev) => new Set(prev).add(id));
      const timer = setTimeout(() => {
        closeTimers.current.delete(id);
        setExpanded((prev) => {
          if (!prev.has(id)) return prev;
          const next = new Set(prev);
          next.delete(id);
          return next;
        });
        setClosing((prev) => {
          if (!prev.has(id)) return prev;
          const next = new Set(prev);
          next.delete(id);
          return next;
        });
      }, EXPAND_ANIMATION_MS);
      closeTimers.current.set(id, timer);
      return;
    }

    setExpanded((prev) => new Set(prev).add(id));
  };

  // Ratakan hierarki agar tabel desktop dan card mobile memakai sumber data
  // yang sama (termasuk state expand dan pencarian).
  const flatNodes = useMemo(() => {
    const walk = (
      nodes: ChartOfAccount[],
      depth: number,
      parentExiting = false
    ): AccountNode[] =>
      nodes.flatMap((acc) => {
        if (!visibleIds.has(acc.id)) return [];

        const kids = (childrenMap.get(acc.id) ?? []).filter((c) =>
          visibleIds.has(c.id)
        );
        const expandable = kids.length > 0;
        const isOpen = query ? true : expanded.has(acc.id);
        // Saat menutup, status visual langsung berubah (chevron & highlight),
        // tetapi anak tetap dirender sampai animasi selesai.
        const isExpanded = query
          ? true
          : expanded.has(acc.id) && !closing.has(acc.id);
        const exiting = query ? false : parentExiting || closing.has(acc.id);

        return [
          { acc, depth, expandable, isExpanded, exiting },
          ...(expandable && isOpen
            ? walk(kids, depth + 1, query ? false : exiting)
            : []),
        ];
      });

    return walk(roots, 0);
  }, [roots, visibleIds, childrenMap, expanded, closing, query]);

  const showActionColumn = canUpdate || canDelete;
  const columnCount = showActionColumn ? 7 : 6;
  const visibleCount = visibleIds.size;

  return (
    <div className="space-y-4">
      <DataTableToolbar
        search={search}
        setSearch={setSearch}
        placeholder="Cari kode atau nama akun..."
        loading={loading}
        isFiltered={search !== ""}
        onReset={() => setSearch("")}
        counterIcon={<BookOpenCheck className="size-3.5 text-slate-400" />}
        counterLabel={`Total ${visibleCount} akun aktif terdaftar`}
      />

      {/* Mobile: hierarchical card list */}
      <div className="space-y-2 md:hidden">
        {loading && accounts.length === 0 ? (
          Array.from({ length: 5 }).map((_, i) => (
            <div
              key={`coa-card-skel-${i}`}
              className="space-y-2 rounded-xl border bg-white p-4 shadow-xs"
            >
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-40" />
            </div>
          ))
        ) : flatNodes.length === 0 ? (
          <div className="rounded-xl border bg-white p-8 text-center text-slate-400">
            <div className="flex flex-col items-center gap-1.5">
              <Shield className="size-5" />
              <span className="text-xs font-medium">
                Belum ada bagan akun yang cocok.
              </span>
            </div>
          </div>
        ) : (
          flatNodes.map(({ acc, depth, expandable, isExpanded, exiting }) => (
            <article
              key={acc.id}
              onClick={expandable ? () => toggleExpand(acc.id) : undefined}
              style={{ marginLeft: Math.min(depth, 4) * 12 }}
              className={`rounded-xl border bg-white p-3.5 shadow-xs transition-colors ${
                expandable ? "cursor-pointer" : ""
              } ${isExpanded ? "border-slate-300 bg-slate-50/60" : ""} ${
                exiting
                  ? "pointer-events-none animate-out fade-out slide-out-to-top-1 duration-200 fill-mode-forwards"
                  : depth > 0 && !query
                    ? "animate-in fade-in slide-in-from-top-1 duration-200"
                    : ""
              }`}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="flex min-w-0 items-start gap-1.5">
                  {expandable ? (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleExpand(acc.id);
                      }}
                      aria-label={
                        isExpanded ? "Tutup anak akun" : "Buka anak akun"
                      }
                      className="mt-0.5 shrink-0 cursor-pointer rounded p-0.5 text-slate-500 hover:bg-slate-200/70"
                    >
                      <ChevronRight
                        className={`size-3.5 transition-transform duration-200 ${
                          isExpanded ? "rotate-90" : ""
                        }`}
                      />
                    </button>
                  ) : (
                    <span className="mt-0.5 inline-block size-4 shrink-0" />
                  )}
                  {acc.is_header ? (
                    <FolderOpen className="mt-0.5 size-3.5 shrink-0 text-blue-500" />
                  ) : (
                    <FileText className="mt-0.5 size-3.5 shrink-0 text-slate-400" />
                  )}
                  <div className="min-w-0">
                    <p className="font-mono text-xs font-bold text-slate-800">
                      {acc.code}
                    </p>
                    <h3 className="truncate text-sm font-medium text-foreground">
                      {acc.name}
                    </h3>
                  </div>
                </div>
                <Badge
                  variant={acc.is_active ? "success" : "danger"}
                  className="shrink-0"
                >
                  {acc.is_active ? "AKTIF" : "NONAKTIF"}
                </Badge>
              </div>

              <div className="mt-2 flex flex-wrap items-center gap-1.5 pl-6">
                <Badge variant={TYPE_VARIANT[acc.account_type]}>
                  {acc.account_type}
                </Badge>
                <span
                  className={`text-xs font-bold ${
                    acc.normal_balance === "DEBIT"
                      ? "text-blue-600"
                      : "text-amber-600"
                  }`}
                >
                  Saldo {acc.normal_balance === "DEBIT" ? "Debit" : "Kredit"}
                </span>
                {acc.parent_id ? (
                  <span className="flex items-center gap-1 text-xs text-slate-500">
                    <CornerUpLeft className="size-3" />
                    <span className="font-mono">{acc.parent_code || ""}</span>
                  </span>
                ) : (
                  <span className="text-xs text-slate-400">Root</span>
                )}
              </div>

              {showActionColumn && (
                <div className="mt-2.5 flex justify-end gap-1.5 border-t pt-2.5">
                  {canUpdate && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 cursor-pointer text-xs"
                      onClick={(e) => {
                        e.stopPropagation();
                        onEditClick(acc);
                      }}
                    >
                      <Edit2 className="size-3.5" /> Edit
                    </Button>
                  )}
                  {canDelete && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 cursor-pointer text-xs text-rose-500 hover:bg-rose-50 hover:text-rose-600"
                      onClick={(e) => {
                        e.stopPropagation();
                        onDeleteClick(acc);
                      }}
                    >
                      <Trash2 className="size-3.5" /> Nonaktifkan
                    </Button>
                  )}
                </div>
              )}
            </article>
          ))
        )}
      </div>

      {/* Desktop: table */}
      <div className="hidden border rounded-xl shadow-xs overflow-hidden bg-white relative md:block">
        {loading && accounts.length > 0 && (
          <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] z-10 flex items-center justify-center">
            <Skeleton className="h-5 w-40" />
          </div>
        )}

        <Table>
          <TableHeader>
            <TableRow className="bg-slate-50/80 hover:bg-slate-50/80">
              <TableHead>
                Kode
              </TableHead>
              <TableHead>
                Nama Akun
              </TableHead>
              <TableHead>
                Tipe
              </TableHead>
              <TableHead>
                Saldo Normal
              </TableHead>
              <TableHead>
                Induk
              </TableHead>
              <TableHead>
                Status
              </TableHead>
              {showActionColumn && (
                <TableHead className="text-right">
                  Aksi
                </TableHead>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && accounts.length === 0 ? (
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-40" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-16" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-16" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-16" />
                  </TableCell>
                  {showActionColumn && (
                    <TableCell>
                      <Skeleton className="h-4 w-16 ml-auto" />
                    </TableCell>
                  )}
                </TableRow>
              ))
            ) : flatNodes.length === 0 ? (
              <TableRow>
                <TableCell colSpan={columnCount} className="h-24 text-center">
                  <div className="flex flex-col items-center gap-1.5 text-slate-400">
                    <Shield className="size-5" />
                    <span className="text-xs font-medium">
                      Belum ada bagan akun yang cocok.
                    </span>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              flatNodes.map(({ acc, depth, expandable, isExpanded, exiting }) => (
                <TableRow
                  key={acc.id}
                  onClick={
                    expandable ? () => toggleExpand(acc.id) : undefined
                  }
                  className={`transition-colors ${expandable ? "cursor-pointer hover:bg-slate-50/70" : ""} ${expandable && isExpanded ? "bg-slate-50/50" : ""} ${
                    exiting
                      ? "pointer-events-none animate-out fade-out duration-200 fill-mode-forwards"
                      : depth > 0 && !query
                        ? "animate-in fade-in duration-200"
                        : ""
                  }`}
                >
                  <TableCell>
                    <div
                      className="flex items-center"
                      style={{ paddingLeft: depth * 20 }}
                    >
                      {expandable ? (
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleExpand(acc.id);
                          }}
                          title={
                            isExpanded ? "Tutup anak akun" : "Buka anak akun"
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
                      <span className="font-mono text-xs font-bold text-slate-800">
                        {acc.code}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center">
                      {acc.is_header ? (
                        <FolderOpen className="size-3.5 text-blue-500 shrink-0 mr-1.5" />
                      ) : (
                        <FileText className="size-3.5 text-slate-400 shrink-0 mr-1.5" />
                      )}
                      <span className="text-sm font-medium text-foreground">
                        {acc.name}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={TYPE_VARIANT[acc.account_type]}>
                      {acc.account_type}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <span
                      className={`text-xs font-bold ${acc.normal_balance === "DEBIT" ? "text-blue-600" : "text-amber-600"}`}
                    >
                      {acc.normal_balance === "DEBIT" ? "D" : "C"}
                    </span>
                  </TableCell>
                  <TableCell>
                    {acc.parent_id ? (
                      <span className="text-xs text-slate-500 flex items-center gap-1">
                        <CornerUpLeft className="size-3" />
                        <span className="font-mono">
                          {acc.parent_code || ""}
                        </span>
                      </span>
                    ) : (
                      <span className="text-xs text-slate-400">Root</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={acc.is_active ? "success" : "danger"}>
                      {acc.is_active ? "AKTIF" : "NONAKTIF"}
                    </Badge>
                  </TableCell>
                  {showActionColumn && (
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1.5">
                        {canUpdate && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 cursor-pointer"
                            onClick={(e) => {
                              e.stopPropagation();
                              onEditClick(acc);
                            }}
                            title="Edit akun"
                          >
                            <Edit2 className="size-3.5" />
                          </Button>
                        )}
                        {canDelete && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 text-rose-500 hover:text-rose-600 hover:bg-rose-50 cursor-pointer"
                            onClick={(e) => {
                              e.stopPropagation();
                              onDeleteClick(acc);
                            }}
                            title="Nonaktifkan akun"
                          >
                            <Trash2 className="size-3.5" />
                          </Button>
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

      <p className="text-xs text-slate-400 text-center">
        {roots.length} akun utama · klik baris akun untuk melihat/menyembunyikan
        anak akun.
      </p>
    </div>
  );
}
