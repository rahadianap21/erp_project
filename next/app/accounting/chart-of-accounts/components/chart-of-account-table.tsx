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
import React, { useEffect, useMemo, useState } from "react";

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

interface ChartOfAccountTableProps {
  canUpdate: boolean;
  canDelete: boolean;
  refreshTrigger: number;
  onDeleteClick: (acc: ChartOfAccount) => void;
  onEditClick: (acc: ChartOfAccount) => void;
}

const TYPE_BADGE: Record<ChartOfAccount["account_type"], string> = {
  ASSET: "bg-blue-50 text-blue-700 border-blue-200",
  LIABILITY: "bg-amber-50 text-amber-700 border-amber-200",
  EQUITY: "bg-violet-50 text-violet-700 border-violet-200",
  REVENUE: "bg-emerald-50 text-emerald-700 border-emerald-200",
  EXPENSE: "bg-rose-50 text-rose-700 border-rose-200",
};

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

  const toggleExpand = (id: number) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const hasChildren = (id: number) => (childrenMap.get(id)?.length ?? 0) > 0;

  const renderRows = (
    nodes: ChartOfAccount[],
    depth: number
  ): React.ReactNode[] => {
    return nodes.flatMap((acc) => {
      if (!visibleIds.has(acc.id)) return [];

      const kids = (childrenMap.get(acc.id) ?? []).filter((c) =>
        visibleIds.has(c.id)
      );
      const expandable = kids.length > 0;
      const isExpanded = query ? true : expanded.has(acc.id);

      const row = (
        <TableRow
          key={acc.id}
          onClick={expandable ? () => toggleExpand(acc.id) : undefined}
          className={`transition-colors ${expandable ? "cursor-pointer hover:bg-slate-50/70" : ""} ${expandable && isExpanded ? "bg-slate-50/50" : ""}`}
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
                  title={isExpanded ? "Tutup anak akun" : "Buka anak akun"}
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
              <span className="text-xs font-bold text-slate-700">
                {acc.name}
              </span>
            </div>
          </TableCell>
          <TableCell>
            <Badge
              variant="outline"
              className={`text-[10px] font-bold border ${TYPE_BADGE[acc.account_type]}`}
            >
              {acc.account_type}
            </Badge>
          </TableCell>
          <TableCell>
            <span
              className={`text-[11px] font-bold ${acc.normal_balance === "DEBIT" ? "text-blue-600" : "text-amber-600"}`}
            >
              {acc.normal_balance === "DEBIT" ? "D" : "C"}
            </span>
          </TableCell>
          <TableCell>
            {acc.parent_id ? (
              <span className="text-[11px] text-slate-500 flex items-center gap-1">
                <CornerUpLeft className="size-3" />
                <span className="font-mono">{acc.parent_code || ""}</span>
              </span>
            ) : (
              <span className="text-[11px] text-slate-400">Root</span>
            )}
          </TableCell>
          <TableCell>
            {acc.is_active ? (
              <span className="text-[10px] font-bold text-emerald-600 bg-emerald-50 border border-emerald-200 rounded-full px-2 py-0.5">
                AKTIF
              </span>
            ) : (
              <span className="text-[10px] font-bold text-rose-600 bg-rose-50 border border-rose-200 rounded-full px-2 py-0.5">
                NONAKTIF
              </span>
            )}
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
      );

      const childRows =
        expandable && isExpanded ? renderRows(kids, depth + 1) : [];
      return [row, ...childRows];
    });
  };

  const showActionColumn = canUpdate || canDelete;
  const columnCount = showActionColumn ? 7 : 6;
  const visibleCount = visibleIds.size;

  const bodyRows = renderRows(roots, 0);

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

      <div className="border rounded-xl shadow-xs overflow-hidden bg-white relative">
        {loading && accounts.length > 0 && (
          <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] z-10 flex items-center justify-center">
            <Skeleton className="h-5 w-40" />
          </div>
        )}

        <Table>
          <TableHeader>
            <TableRow className="bg-slate-50/80 hover:bg-slate-50/80">
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Kode
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Nama Akun
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Tipe
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Saldo Normal
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Induk
              </TableHead>
              <TableHead className="text-[11px] font-bold uppercase tracking-wider text-slate-500">
                Status
              </TableHead>
              {showActionColumn && (
                <TableHead className="text-right text-[11px] font-bold uppercase tracking-wider text-slate-500">
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
            ) : bodyRows.length === 0 ? (
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
              bodyRows
            )}
          </TableBody>
        </Table>
      </div>

      <p className="text-[11px] text-slate-400 text-center">
        {roots.length} akun utama · klik baris akun untuk melihat/menyembunyikan
        anak akun.
      </p>
    </div>
  );
}
