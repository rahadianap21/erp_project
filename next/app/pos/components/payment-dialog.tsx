"use client";

import * as React from "react";
import { BanknoteIcon, WalletIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { MoneyInput } from "@/components/ui/money-input";
import { Label } from "@/components/ui/label";
import { PaymentDraft, PaymentMethod, formatRupiah, round2 } from "./types";

interface PaymentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  total: number;
  methods: PaymentMethod[];
  submitting: boolean;
  onConfirm: (payment: PaymentDraft, notes: string) => void;
}

const CHANNEL_ICON: Record<string, React.ReactNode> = {
  CASH: <BanknoteIcon className="size-5" />,
  BANK: <WalletIcon className="size-5" />,
  EWALLET: <WalletIcon className="size-5" />,
};

export function PaymentDialog({
  open,
  onOpenChange,
  total,
  methods,
  submitting,
  onConfirm,
}: PaymentDialogProps) {
  const [selectedMethod, setSelectedMethod] = React.useState<number | null>(
    null
  );
  const [amount, setAmount] = React.useState("");
  const [reference, setReference] = React.useState("");
  const [notes, setNotes] = React.useState("");

  React.useEffect(() => {
    if (open) {
      setSelectedMethod(null);
      setAmount(String(round2(total)));
      setReference("");
      setNotes("");
    }
  }, [open, total]);

  const payMethods = methods.filter((m) => m.channel !== "CREDIT");
  const selected = payMethods.find((m) => m.id === selectedMethod) ?? null;
  const isCash = selected?.channel === "CASH" || selected?.is_cash === true;

  const received = isCash
    ? Number((amount || "0").replace(/[^\d]/g, "")) || 0
    : total;
  const change = round2(Math.max(received - total, 0));
  const lunas =
    total <= 0 ||
    received >= total ||
    (total - received <= 0.01 && received > 0);
  const canConfirm = selectedMethod !== null && lunas && !submitting;

  const handleConfirm = () => {
    if (!canConfirm || selectedMethod === null) return;
    onConfirm(
      {
        paymentMethodId: selectedMethod,
        amount: Math.max(received, total),
        referenceNumber: reference.trim(),
      },
      notes.trim()
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="md:max-w-2xl 4xl:max-w-4xl">
        <DialogHeader>
          <DialogTitle className="4xl:text-2xl">Pembayaran</DialogTitle>
          <DialogDescription className="4xl:text-base">
            Penjualan wajib lunas. Pilih satu metode, masukkan uang diterima
            (untuk tunai), lalu simpan transaksi.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 4xl:gap-5">
          <div className="grid grid-cols-1 gap-2 rounded-lg border bg-muted/40 p-3 text-center md:grid-cols-3 4xl:p-4">
            <div>
              <p className="text-xs text-muted-foreground 4xl:text-sm">Total</p>
              <p className="text-sm font-semibold 4xl:text-base">
                {formatRupiah(total)}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground 4xl:text-sm">
                Uang Diterima
              </p>
              <p className="text-sm font-semibold text-emerald-600 4xl:text-base">
                {formatRupiah(received)}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground 4xl:text-sm">
                Kembalian
              </p>
              <p className="text-sm font-semibold text-amber-600 4xl:text-base">
                {formatRupiah(change)}
              </p>
            </div>
          </div>

          <div className="space-y-2">
            <Label className="text-xs text-muted-foreground 4xl:text-sm">
              Metode pembayaran
            </Label>
            <div className="grid grid-cols-1 gap-2 md:grid-cols-3 4xl:gap-3">
              {payMethods.map((m) => (
                <Button
                  key={m.id}
                  type="button"
                  variant={selectedMethod === m.id ? "default" : "outline"}
                  className="h-14 cursor-pointer justify-start gap-2 4xl:h-20 4xl:gap-3 4xl:text-lg"
                  onClick={() => {
                    setSelectedMethod(m.id);
                    setAmount(String(round2(total)));
                  }}
                >
                  {CHANNEL_ICON[m.channel] ?? <WalletIcon className="size-5" />}
                  <span className="truncate">{m.name}</span>
                </Button>
              ))}
            </div>
          </div>

          {selectedMethod !== null && isCash && (
            <div className="grid gap-3 rounded-lg border p-3 4xl:p-4">
              <div className="grid gap-1">
                <Label htmlFor="payment-amount" className="4xl:text-base">
                  Uang diterima
                </Label>
                <MoneyInput
                  id="payment-amount"
                  value={Number(amount) || 0}
                  onValueChange={(value) => setAmount(String(value))}
                  placeholder="0"
                  className="h-12 px-3 text-base 4xl:h-16 4xl:text-xl"
                  inputClassName="text-base 4xl:text-xl"
                />
              </div>
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  variant="outline"
                  className="cursor-pointer 4xl:h-12 4xl:px-5 4xl:text-base"
                  onClick={() => setAmount(String(round2(total)))}
                >
                  Uang Pas
                </Button>
                {[20000, 50000, 100000, 200000].map((denom) => (
                  <Button
                    key={denom}
                    type="button"
                    variant="outline"
                    className="cursor-pointer 4xl:h-12 4xl:px-5 4xl:text-base"
                    onClick={() => setAmount(String(denom))}
                  >
                    {formatRupiah(denom)}
                  </Button>
                ))}
              </div>
              {!lunas && (
                <p className="text-xs font-medium text-rose-600 4xl:text-sm">
                  Kurang {formatRupiah(round2(total - received))} — penjualan
                  wajib lunas.
                </p>
              )}
            </div>
          )}

          {selectedMethod !== null && !isCash && (
            <div className="grid gap-3 rounded-lg border p-3 4xl:p-4">
              <div className="grid gap-1">
                <Label htmlFor="payment-reference" className="4xl:text-base">
                  Nomor referensi (opsional)
                </Label>
                <Input
                  id="payment-reference"
                  value={reference}
                  onChange={(e) => setReference(e.target.value)}
                  placeholder="No. transfer / e-wallet"
                  className="4xl:h-12 4xl:text-base"
                />
              </div>
              <p className="text-xs text-muted-foreground 4xl:text-sm">
                Nominal mengikuti total nota {formatRupiah(total)}.
              </p>
            </div>
          )}

          <div className="grid gap-1">
            <Label htmlFor="payment-notes" className="4xl:text-base">
              Catatan (opsional)
            </Label>
            <Input
              id="payment-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Catatan nota"
              className="4xl:h-12 4xl:text-base"
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            className="cursor-pointer 4xl:h-12 4xl:px-6 4xl:text-base"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            Batal
          </Button>
          <Button
            className="cursor-pointer 4xl:h-12 4xl:px-6 4xl:text-base"
            onClick={handleConfirm}
            disabled={!canConfirm}
          >
            {submitting ? "Menyimpan..." : "Simpan Transaksi"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
