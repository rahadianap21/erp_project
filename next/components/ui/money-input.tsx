"use client";

import * as React from "react";

import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { formatMoneyInput, parseMoney, roundMoney } from "@/lib/money";

type MoneyInputProps = Omit<
  React.ComponentProps<"input">,
  "type" | "value" | "onChange" | "defaultValue"
> & {
  /** Nominal rupiah saat ini (angka biasa, bukan string terformat). */
  value: number;
  /** Dipanggil dengan nominal rupiah hasil parsing input. */
  onValueChange: (value: number) => void;
  /** Kelas untuk pembungkus (prefiks Rp + input). */
  className?: string;
  /** Kelas tambahan untuk elemen input di dalamnya. */
  inputClassName?: string;
};

/** Menyusun tampilan angka dengan pemisah ribuan sambil mempertahankan koma
 * desimal yang sedang diketik pengguna. */
function formatWhileTyping(raw: string): string {
  const [integerRaw, ...decimalParts] = raw.split(",");
  const digits = integerRaw.replace(/\D/g, "").replace(/^0+(?=\d)/, "");
  const grouped = digits.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  if (decimalParts.length === 0) return grouped;

  const decimals = decimalParts.join("").replace(/\D/g, "").slice(0, 2);
  return `${grouped || "0"},${decimals}`;
}

/**
 * Kolom input nominal rupiah: selalu berprefiks "Rp", memakai pemisah ribuan
 * titik, dan mengirim angka murni ke pemanggil. Semua form yang menerima uang
 * wajib memakai komponen ini agar pola input konsisten.
 */
export function MoneyInput({
  value,
  onValueChange,
  className,
  inputClassName,
  disabled,
  ...props
}: MoneyInputProps) {
  const [text, setText] = React.useState(() => formatMoneyInput(value));
  const lastEmitted = React.useRef(value);

  React.useEffect(() => {
    if (value !== lastEmitted.current) {
      lastEmitted.current = value;
      setText(formatMoneyInput(value));
    }
  }, [value]);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const raw = event.target.value.replace(/[^\d,]/g, "");
    const formatted = formatWhileTyping(raw);
    const nominal = parseMoney(formatted);

    lastEmitted.current = nominal;
    setText(formatted);
    onValueChange(nominal);
  };

  return (
    <div
      className={cn(
        "flex h-8 w-full items-center rounded-lg border border-input bg-transparent px-2.5 transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      data-disabled={disabled ? "true" : undefined}
    >
      <span className="mr-1.5 shrink-0 text-sm font-medium text-muted-foreground">
        Rp
      </span>
      <Input
        {...props}
        type="text"
        inputMode="decimal"
        value={text}
        onChange={handleChange}
        onFocus={(event) => {
          if (roundMoney(value) === 0 && text === "0") {
            setText("");
          }
          props.onFocus?.(event);
        }}
        onBlur={(event) => {
          if (!text.trim()) {
            lastEmitted.current = 0;
            setText(formatMoneyInput(0));
            onValueChange(0);
          }
          props.onBlur?.(event);
        }}
        disabled={disabled}
        className={cn(
          "h-full min-w-0 flex-1 border-0 bg-transparent p-0 font-mono text-right text-sm shadow-none focus-visible:ring-0",
          inputClassName
        )}
      />
    </div>
  );
}
