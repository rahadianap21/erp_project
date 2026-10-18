"use client";

import { apiGet, apiPost } from "@/lib/api-client";
import * as React from "react";
import { readPermissionList } from "@/lib/permissions";
import {
  BanIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ReceiptTextIcon,
  SearchIcon,
} from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { formatRupiah } from "@/app/pos/components/types";

interface Sale {
  id: number;
  sale_number: string;
  sale_date: string;
  member_id: number;
  member_name: string;
  tax_type: string;
  subtotal: number;
  item_discount: number;
  discount_amount: number;
  dpp_amount: number;
  tax_amount: number;
  total_amount: number;
  paid_amount: number;
  change_amount: number;
  payment_status: string;
  status: string;
  due_date?: string | null;
  details?: {
    id: number;
    product_name: string;
    qty: number;
    unit_price: number;
    line_total: number;
  }[];
  payments?: { id: number; payment_method_name: string; amount: number }[];
}

const LIMIT = 20;

export function SalesView() {
  const permissions = React.useMemo<string[]>(
    () => readPermissionList() ?? [],
    []
  );
  const canVoid = permissions.includes("sales:void");

  const [sales, setSales] = React.useState<Sale[]>([]);
  const [total, setTotal] = React.useState(0);
  const [search, setSearch] = React.useState("");
  const [status, setStatus] = React.useState("ALL");
  const [cursor, setCursor] = React.useState(0);
  const [direction, setDirection] = React.useState<"next" | "prev">("next");
  const [history, setHistory] = React.useState<number[]>([]);
  const [loading, setLoading] = React.useState(false);

  const [detail, setDetail] = React.useState<Sale | null>(null);
  const [voidReason, setVoidReason] = React.useState("");
  const [voiding, setVoiding] = React.useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (search) params.set("search", search);
      if (status !== "ALL") params.set("status", status);
      params.set("limit", String(LIMIT));
      params.set("direction", direction);
      if (cursor) params.set("cursor_id", String(cursor));
      const body = await apiGet<Sale[]>(`/api/sales?${params.toString()}`);
      setSales(body.data ?? []);
      setTotal(Number((body.body as { total?: number })?.total ?? 0));
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal memuat penjualan"
      );
    } finally {
      setLoading(false);
    }
  }, [search, status, cursor, direction]);

  React.useEffect(() => {
    load();
  }, [load]);

  const openDetail = async (id: number) => {
    try {
      const body = await apiGet<Sale>(`/api/sales?id=${id}`);
      setDetail(body.data);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat detail");
    }
  };

  const resetToFirstPage = () => {
    setCursor(0);
    setDirection("next");
    setHistory([]);
  };

  const goNext = () => {
    if (sales.length === 0) return;
    setHistory((h) => [...h, cursor]);
    setCursor(sales[sales.length - 1].id);
    setDirection("next");
  };

  const goPrev = () => {
    if (history.length === 0) {
      resetToFirstPage();
      return;
    }
    const prevCursor = history[history.length - 1];
    setHistory((h) => h.slice(0, -1));
    setDirection("prev");
    setCursor(prevCursor || (sales.length > 0 ? sales[0].id : 0));
  };

  const handleVoid = async () => {
    if (!detail) return;
    setVoiding(true);
    try {
      await apiPost(`/api/sales/void?id=${detail.id}`, { reason: voidReason });
      toast.success("Penjualan dibatalkan");
      setDetail(null);
      setVoidReason("");
      resetToFirstPage();
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal membatalkan");
    } finally {
      setVoiding(false);
    }
  };

  const statusBadge = (s: string) =>
    s === "VOID" ? (
      <Badge variant="destructive">VOID</Badge>
    ) : (
      <Badge variant="secondary">POSTED</Badge>
    );

  const paymentBadge = (s: string) => {
    if (s === "PAID")
      return <Badge className="bg-emerald-600 text-white">PAID</Badge>;
    if (s === "PARTIAL") return <Badge variant="outline">PARTIAL</Badge>;
    return <Badge variant="outline">UNPAID</Badge>;
  };

  return (
    <main className="flex flex-1 flex-col gap-4 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <h1 className="flex items-center gap-2 text-xl font-semibold tracking-tight">
            <ReceiptTextIcon className="size-5 text-blue-600" /> Riwayat
            Penjualan
          </h1>
          <p className="text-xs text-muted-foreground">
            {total} transaksi penjualan tercatat.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <SearchIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                resetToFirstPage();
              }}
              placeholder="Cari nomor / pelanggan"
              className="h-9 w-56 pl-9"
            />
          </div>
          <select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              resetToFirstPage();
            }}
            className="h-9 rounded-md border bg-background px-2 text-sm"
          >
            <option value="ALL">Semua status</option>
            <option value="POSTED">POSTED</option>
            <option value="VOID">VOID</option>
          </select>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-3 py-2">Nomor</th>
              <th className="px-3 py-2">Tanggal</th>
              <th className="px-3 py-2">Member</th>
              <th className="px-3 py-2 text-right">Total</th>
              <th className="px-3 py-2 text-right">Dibayar</th>
              <th className="px-3 py-2">Bayar</th>
              <th className="px-3 py-2">Status</th>
            </tr>
          </thead>
          <tbody>
            {loading && sales.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-10 text-center text-muted-foreground"
                >
                  Memuat...
                </td>
              </tr>
            ) : sales.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-10 text-center text-muted-foreground"
                >
                  Belum ada transaksi.
                </td>
              </tr>
            ) : (
              sales.map((s) => (
                <tr
                  key={s.id}
                  className="cursor-pointer border-t hover:bg-muted/40"
                  onClick={() => openDetail(s.id)}
                >
                  <td className="px-3 py-2 font-medium">{s.sale_number}</td>
                  <td className="px-3 py-2">
                    {new Date(s.sale_date).toLocaleDateString("id-ID")}
                  </td>
                  <td className="px-3 py-2">{s.member_name || "Umum"}</td>
                  <td className="px-3 py-2 text-right">
                    {formatRupiah(s.total_amount)}
                  </td>
                  <td className="px-3 py-2 text-right">
                    {formatRupiah(s.paid_amount)}
                  </td>
                  <td className="px-3 py-2">
                    {paymentBadge(s.payment_status)}
                  </td>
                  <td className="px-3 py-2">{statusBadge(s.status)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-end gap-2">
        <Button
          variant="outline"
          className="cursor-pointer"
          onClick={goPrev}
          disabled={loading || (history.length === 0 && cursor === 0)}
        >
          <ChevronLeftIcon /> Sebelumnya
        </Button>
        <Button
          variant="outline"
          className="cursor-pointer"
          onClick={goNext}
          disabled={loading || sales.length < LIMIT}
        >
          Berikutnya <ChevronRightIcon />
        </Button>
      </div>

      <Dialog open={!!detail} onOpenChange={(open) => !open && setDetail(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center justify-between">
              <span>{detail?.sale_number}</span>
              {detail && statusBadge(detail.status)}
            </DialogTitle>
            <DialogDescription>
              {detail?.member_name || "Umum"} ·{" "}
              {detail ? new Date(detail.sale_date).toLocaleString("id-ID") : ""}
            </DialogDescription>
          </DialogHeader>

          {detail && (
            <div className="space-y-3 text-sm">
              <div className="max-h-52 space-y-1 overflow-auto rounded-md border p-2">
                {(detail.details ?? []).map((d) => (
                  <div key={d.id} className="flex justify-between gap-2">
                    <span className="truncate">
                      {d.product_name}{" "}
                      <span className="text-muted-foreground">×{d.qty}</span>
                    </span>
                    <span className="shrink-0">
                      {formatRupiah(d.line_total)}
                    </span>
                  </div>
                ))}
              </div>
              <div className="flex justify-between">
                <span>Subtotal</span>
                <span>{formatRupiah(detail.subtotal)}</span>
              </div>
              <div className="flex justify-between">
                <span>Diskon</span>
                <span>
                  -{formatRupiah(detail.item_discount + detail.discount_amount)}
                </span>
              </div>
              <div className="flex justify-between">
                <span>PPN</span>
                <span>{formatRupiah(detail.tax_amount)}</span>
              </div>
              <div className="flex justify-between text-base font-semibold">
                <span>Total</span>
                <span>{formatRupiah(detail.total_amount)}</span>
              </div>
              <div className="flex justify-between text-emerald-600">
                <span>Dibayar</span>
                <span>{formatRupiah(detail.paid_amount)}</span>
              </div>
              {detail.change_amount > 0.001 && (
                <div className="flex justify-between text-amber-600">
                  <span>Kembalian</span>
                  <span>{formatRupiah(detail.change_amount)}</span>
                </div>
              )}

              {canVoid && detail.status === "POSTED" && (
                <div className="space-y-2 rounded-md border border-rose-200 bg-rose-50 p-3">
                  <p className="text-xs font-medium text-rose-700">
                    Alasan pembatalan (stok dikembalikan, jurnal dibalik)
                  </p>
                  <Input
                    value={voidReason}
                    onChange={(e) => setVoidReason(e.target.value)}
                    placeholder="Mis. salah input / pelanggan batal"
                    className="h-9"
                  />
                  <Button
                    variant="destructive"
                    className="cursor-pointer"
                    onClick={handleVoid}
                    disabled={voiding}
                  >
                    <BanIcon />{" "}
                    {voiding ? "Membatalkan..." : "Batalkan Penjualan"}
                  </Button>
                </div>
              )}
            </div>
          )}

          <DialogFooter>
            <Button
              variant="outline"
              className="cursor-pointer"
              onClick={() => setDetail(null)}
            >
              Tutup
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  );
}
