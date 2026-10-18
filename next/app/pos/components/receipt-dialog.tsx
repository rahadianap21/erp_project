"use client";

import * as React from "react";
import { PrinterIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Sale, formatRupiah } from "./types";

interface ReceiptDialogProps {
  sale: Sale | null;
  onClose: () => void;
}

export function ReceiptDialog({ sale, onClose }: ReceiptDialogProps) {
  if (!sale) return null;

  const handlePrint = () => {
    const win = window.open("", "_blank", "width=380,height=640");
    if (!win) return;
    const rows = (sale.details ?? [])
      .map(
        (d) => `<tr>
          <td>${escapeHtml(d.product_name)}</td>
          <td style="text-align:center">${d.qty}</td>
          <td style="text-align:right">${formatRupiah(d.line_total)}</td>
        </tr>`
      )
      .join("");
    const payments = (sale.payments ?? [])
      .map(
        (p) =>
          `<div class="row"><span>${escapeHtml(p.payment_method_name)}</span><span>${formatRupiah(
            p.amount
          )}</span></div>`
      )
      .join("");
    const unpaid = sale.total_amount - sale.paid_amount;
    win.document
      .write(`<!doctype html><html><head><title>${sale.sale_number}</title>
      <style>
        body{font-family:monospace;font-size:12px;margin:8px}
        h2{text-align:center;margin:4px 0}
        table{width:100%;border-collapse:collapse;margin-top:6px}
        td{padding:1px 0}
        .row{display:flex;justify-content:space-between}
        .sep{border-top:1px dashed #000;margin:6px 0}
      </style></head><body>
      <h2>AXION POS</h2>
      <div>${sale.sale_number}</div>
      <div>${new Date(sale.sale_date).toLocaleString("id-ID")}</div>
      <div>Member: ${escapeHtml(sale.member_name || "Umum")}</div>
      <div class="sep"></div>
      <table>${rows}</table>
      <div class="sep"></div>
      <div class="row"><span>Subtotal</span><span>${formatRupiah(sale.subtotal)}</span></div>
      <div class="row"><span>Diskon</span><span>-${formatRupiah(
        sale.item_discount + sale.discount_amount
      )}</span></div>
      ${
        sale.tax_amount > 0.001
          ? `<div class="row"><span>DPP</span><span>${formatRupiah(sale.dpp_amount)}</span></div>
      <div class="row"><span>PPN</span><span>${formatRupiah(sale.tax_amount)}</span></div>`
          : ""
      }
      ${
        sale.redeem_amount > 0.001
          ? `<div class="row"><span>Tukar Poin (${sale.redeem_points})</span><span>-${formatRupiah(sale.redeem_amount)}</span></div>`
          : ""
      }
      <div class="row"><strong><span>Total</span><span>${formatRupiah(
        sale.total_amount
      )}</span></strong></div>
      ${
        sale.points_earned > 0
          ? `<div class="row"><span>Poin didapat</span><span>+${sale.points_earned}</span></div>`
          : ""
      }
      <div class="sep"></div>
      ${payments}
      ${
        sale.change_amount > 0.001
          ? `<div class="row"><span>Kembalian</span><span>${formatRupiah(sale.change_amount)}</span></div>`
          : ""
      }
      ${
        unpaid > 0.001
          ? `<div class="row"><span>Piutang</span><span>${formatRupiah(unpaid)}</span></div>`
          : ""
      }
      <div class="sep"></div>
      <div style="text-align:center">Terima kasih</div>
      </body></html>`);
    win.document.close();
    win.focus();
    win.print();
  };

  return (
    <Dialog open={!!sale} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center justify-between">
            <span>Transaksi Berhasil</span>
            <Badge variant="secondary">{sale.sale_number}</Badge>
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-2 text-sm">
          <div className="flex justify-between text-muted-foreground">
            <span>{sale.member_name || "Umum"}</span>
            <span>{new Date(sale.sale_date).toLocaleString("id-ID")}</span>
          </div>
          <div className="max-h-56 space-y-1 overflow-auto rounded-md border p-2">
            {(sale.details ?? []).map((d) => (
              <div key={d.id} className="flex justify-between gap-2">
                <span className="truncate">
                  {d.product_name}{" "}
                  <span className="text-muted-foreground">×{d.qty}</span>
                </span>
                <span className="shrink-0">{formatRupiah(d.line_total)}</span>
              </div>
            ))}
          </div>
          <div className="flex justify-between">
            <span>Subtotal</span>
            <span>{formatRupiah(sale.subtotal)}</span>
          </div>
          <div className="flex justify-between">
            <span>Diskon</span>
            <span>
              -{formatRupiah(sale.item_discount + sale.discount_amount)}
            </span>
          </div>
          {sale.tax_amount > 0.001 && (
            <>
              <div className="flex justify-between">
                <span>DPP</span>
                <span>{formatRupiah(sale.dpp_amount)}</span>
              </div>
              <div className="flex justify-between">
                <span>PPN</span>
                <span>{formatRupiah(sale.tax_amount)}</span>
              </div>
            </>
          )}
          {sale.redeem_amount > 0.001 && (
            <div className="flex justify-between text-amber-600">
              <span>Tukar Poin ({sale.redeem_points})</span>
              <span>-{formatRupiah(sale.redeem_amount)}</span>
            </div>
          )}
          <div className="flex justify-between text-base font-semibold">
            <span>Total</span>
            <span>{formatRupiah(sale.total_amount)}</span>
          </div>
          {sale.points_earned > 0 && (
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>Poin didapat</span>
              <span>+{sale.points_earned}</span>
            </div>
          )}
          <div className="flex justify-between text-emerald-600">
            <span>Dibayar</span>
            <span>{formatRupiah(sale.paid_amount)}</span>
          </div>
          {sale.change_amount > 0.001 && (
            <div className="flex justify-between text-amber-600">
              <span>Kembalian</span>
              <span>{formatRupiah(sale.change_amount)}</span>
            </div>
          )}
          {sale.total_amount - sale.paid_amount > 0.001 && (
            <div className="flex justify-between text-rose-600">
              <span>Piutang</span>
              <span>{formatRupiah(sale.total_amount - sale.paid_amount)}</span>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            className="cursor-pointer"
            onClick={handlePrint}
          >
            <PrinterIcon /> Cetak
          </Button>
          <Button className="cursor-pointer" onClick={onClose}>
            Transaksi Baru
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
