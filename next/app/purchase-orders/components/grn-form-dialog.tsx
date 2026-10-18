"use client";

import { apiGet, apiPost } from "@/lib/api-client";
import { useEffect, useMemo, useState } from "react";
import Cookies from "js-cookie";
import { toast } from "sonner";
import { Loader2, PackageCheck, TriangleAlert } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

export interface ReceiveLine {
  id: number;
  product_name: string;
  product_sku: string;
  qty_ordered: number;
  qty_received: number;
  unit_kode: string;
  conversion_rate: number;
  selling_unit_kode?: string;
}

interface GRNFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  poId: number;
  poNumber: string;
  onSuccess: () => void;
}

/**
 * Dialog pencatatan penerimaan barang untuk satu PO.
 *
 * Satu PO boleh diterima bertahap, jadi baris hanya receivable selama masih
 * ada sisa. Qty yang di sini dihitung dalam satuan beli; dampaknya ke stok
 * selalu dihitung ulang dari conversion_rate produk supaya tidak ada angka
 * yang diketik ulang secara manual oleh pengguna.
 */
export function GRNFormDialog({
  open,
  onOpenChange,
  poId,
  poNumber,
  onSuccess,
}: GRNFormDialogProps) {
  const token = Cookies.get("session_active") || "";

  const [lines, setLines] = useState<ReceiveLine[]>([]);
  const [qtyInput, setQtyInput] = useState<Record<number, string>>({});
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  // Sengaja kosong, bukan new Date(). Nilai "hari ini" diisi di efek pemuatan
  // saat dialog dibuka. Memakai tanggal langsung di useState membuat render
  // server dan client bisa berbeda, dan itu pemicu hydration mismatch.
  const [receivedDate, setReceivedDate] = useState("");
  const [notes, setNotes] = useState("");

  useEffect(() => {
    if (!open || !token) return;
    let cancelled = false;

    const load = async () => {
      setLoading(true);
      try {
        const result = await apiGet<{ items?: ReceiveLine[] }>(
          `/api/purchase-orders?id=${poId}`
        );
        if (cancelled) return;

        const items: ReceiveLine[] = result.data?.items ?? [];
        setLines(items);
        setQtyInput({});
        setNotes("");
        setReceivedDate(new Date().toISOString().slice(0, 10));
      } catch (error: any) {
        if (!cancelled)
          toast.error("Gagal Memuat Data", { description: error.message });
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    load();
    return () => {
      cancelled = true;
    };
  }, [open, poId, token]);

  // Baris yang sudah penuh disembunyikan supaya tidak ada ruang input yang
  // pasti ditolak backend.
  const receivable = useMemo(
    () => lines.filter((l) => l.qty_ordered > l.qty_received),
    [lines]
  );

  const rows = useMemo(
    () =>
      receivable.map((l) => {
        const remaining = l.qty_ordered - l.qty_received;
        const raw = qtyInput[l.id] ?? "";
        const parsed = Number(raw);
        const qty = raw === "" || Number.isNaN(parsed) ? 0 : parsed;
        // Rate efektif mengikuti aturan yang sama dengan backend: satuan
        // beli yang sama dengan satuan jual produk berarti 1:1.
        const rate = Math.max(1, l.conversion_rate || 1);
        const sameUnit =
          Boolean(l.selling_unit_kode) && l.selling_unit_kode === l.unit_kode;
        const effectiveRate = sameUnit ? 1 : rate;
        return {
          line: l,
          remaining,
          qty,
          over: qty > remaining,
          stockDelta: qty * effectiveRate,
          effectiveRate,
          sellingUnit: sameUnit
            ? l.unit_kode
            : l.selling_unit_kode || l.unit_kode,
        };
      }),
    [receivable, qtyInput]
  );

  const totalStockDelta = rows.reduce(
    (sum, r) => sum + (r.over ? 0 : r.stockDelta),
    0
  );
  const totalReceived = rows.reduce((sum, r) => sum + (r.over ? 0 : r.qty), 0);
  const hasOver = rows.some((r) => r.over);
  const canSubmit = !loading && !hasOver && totalReceived > 0 && !submitting;

  const setQty = (id: number, value: string) => {
    setQtyInput((prev) => ({ ...prev, [id]: value }));
  };

  const handleSubmit = async () => {
    if (!canSubmit) return;

    const items = rows
      .filter((r) => r.qty > 0)
      .map((r) => ({
        purchase_order_detail_id: r.line.id,
        qty_received: r.qty,
      }));

    if (items.length === 0) return;

    setSubmitting(true);
    try {
      const result = await apiPost<{ grn_number?: string }>(
        `/api/goods-receipt-notes`,
        {
          purchase_order_id: poId,
          received_date: receivedDate,
          notes,
          items,
        }
      );

      toast.success("Penerimaan Barang Tersimpan", {
        description: `Dokumen ${result.data?.grn_number ?? "GRN"} tercatat untuk PO ${poNumber}.`,
      });
      onOpenChange(false);
      onSuccess();
    } catch (error: any) {
      toast.error("Penerimaan Ditolak", { description: error.message });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !submitting && onOpenChange(o)}>
      <DialogContent className="w-full !max-w-3xl sm:!max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <PackageCheck className="size-5 text-emerald-600" />
            Terima Barang — PO {poNumber}
          </DialogTitle>
          <DialogDescription>
            Masukkan jumlah yang diterima pada batch ini. Stok baru ditambahkan
            ke gudang saat penerimaan PO ditutup lewat tombol &quot;Selesai
            Terima&quot;.
          </DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className="space-y-2 py-2">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        ) : receivable.length === 0 ? (
          <div className="py-10 text-center border border-dashed rounded-xl">
            <p className="text-sm font-semibold text-slate-700">
              Semua barang sudah diterima
            </p>
            <p className="text-xs text-slate-500 mt-1">
              Tidak ada sisa yang bisa dicatat. Gunakan tombol &quot;Selesai
              Terima&quot; untuk menutup PO.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="overflow-x-auto border rounded-lg">
              <table className="w-full text-xs">
                <thead className="bg-slate-50 text-slate-600">
                  <tr>
                    <th className="text-left font-semibold px-3 py-2">
                      Barang
                    </th>
                    <th className="text-center font-semibold px-2 py-2">
                      Sisa
                    </th>
                    <th className="text-center font-semibold px-2 py-2 w-28">
                      Diterima
                    </th>
                    <th className="text-center font-semibold px-2 py-2 w-32">
                      Stok Bertambah
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.line.id} className="border-t">
                      <td className="px-3 py-2">
                        <div className="font-medium text-slate-800">
                          {r.line.product_name}
                        </div>
                        <div className="font-mono text-[10px] text-slate-500">
                          {r.line.product_sku}
                        </div>
                      </td>
                      <td className="px-2 py-2 text-center font-mono font-bold text-slate-700">
                        {r.remaining} {r.line.unit_kode}
                      </td>
                      <td className="px-2 py-2">
                        <Input
                          type="number"
                          min={0}
                          max={r.remaining}
                          value={qtyInput[r.line.id] ?? ""}
                          onChange={(e) => setQty(r.line.id, e.target.value)}
                          placeholder="0"
                          className={`h-8 text-xs text-center font-mono font-bold ${
                            r.over
                              ? "border-rose-400 focus-visible:ring-rose-500/20 focus-visible:border-rose-500"
                              : ""
                          }`}
                        />
                        {/* <div className="text-[9px] text-center text-slate-400 mt-0.5">
                                                    {r.line.unit_kode}
                                                </div> */}
                      </td>
                      <td className="px-2 py-2 text-center font-mono">
                        {r.qty > 0 && !r.over ? (
                          <span className="font-bold text-emerald-600">
                            +{r.stockDelta} {r.sellingUnit}
                          </span>
                        ) : (
                          <span className="text-slate-300">—</span>
                        )}
                        {r.effectiveRate > 1 && (
                          <div className="text-[9px] text-slate-400">
                            1 {r.line.unit_kode} = {r.effectiveRate}{" "}
                            {r.sellingUnit}
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {hasOver && (
              <div className="flex items-start gap-2 bg-rose-50 border border-rose-200 text-rose-800 rounded-lg p-3 text-xs">
                <TriangleAlert className="size-4 shrink-0 mt-0.5" />
                <span>
                  Ada jumlah yang melebihi sisa. Perbaiki dulu sebelum
                  menyimpan; backend juga akan menolak nilai seperti ini.
                </span>
              </div>
            )}

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase">
                  Tanggal Terima
                </label>
                <Input
                  type="date"
                  value={receivedDate}
                  onChange={(e) => setReceivedDate(e.target.value)}
                  className="h-9 text-xs"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase">
                  Catatan
                </label>
                <Input
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="Opsional, mis. nama penerima"
                  className="h-9 text-xs"
                />
              </div>
            </div>

            <div className="flex items-center justify-between bg-emerald-50 border border-emerald-200 rounded-lg px-3 py-2 font-mono text-xs font-bold">
              <span className="text-emerald-800">
                Total diterima: {totalReceived} (satuan beli)
              </span>
              <span className="text-emerald-700">
                Stok bertambah saat ditutup: +{totalStockDelta}
              </span>
            </div>
          </div>
        )}

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
            className="text-xs"
          >
            Batal
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!canSubmit}
            className="bg-emerald-600 hover:bg-emerald-700 text-xs"
          >
            {submitting ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <PackageCheck className="size-3.5" />
            )}
            Simpan Penerimaan
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
