import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { ReportAccount, formatIDR } from "./report-types";

interface AccountTableProps {
  accounts: ReportAccount[];
  // "full" menampilkan kolom debit & kredit (Neraca Saldo); "balance" hanya
  // kode, nama, dan saldo (Neraca / Laba Rugi).
  variant?: "full" | "balance";
  totalLabel?: string;
  total?: { debit: number; credit: number; balance: number };
  emptyText?: string;
}

// AccountTable menyajikan baris akun laporan dengan format seragam.
export function AccountTable({
  accounts,
  variant = "full",
  totalLabel = "Total",
  total,
  emptyText = "Tidak ada mutasi pada periode ini.",
}: AccountTableProps) {
  const full = variant === "full";

  if (accounts.length === 0) {
    return (
      <div className="px-4 py-8 text-center text-xs text-slate-400">
        {emptyText}
      </div>
    );
  }

  return (
    <>
      {/* Mobile: stacked account rows */}
      <div className="md:hidden">
        <ul className="divide-y">
          {accounts.map((account, index) => (
            <li
              key={`m-${account.account_id}-${index}`}
              className="px-4 py-3"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="font-mono text-xs text-muted-foreground">
                    {account.code}
                  </p>
                  <p className="text-sm font-medium text-foreground">
                    {account.name}
                  </p>
                </div>
                <p
                  className={cn(
                    "shrink-0 text-right font-mono text-xs font-bold",
                    account.balance < 0 ? "text-rose-600" : "text-slate-900"
                  )}
                >
                  {formatIDR(account.balance)}
                </p>
              </div>
              {full && (
                <div className="mt-1.5 flex gap-4 font-mono text-xs text-slate-600">
                  <span>D: {account.debit ? formatIDR(account.debit) : "-"}</span>
                  <span>
                    K: {account.credit ? formatIDR(account.credit) : "-"}
                  </span>
                </div>
              )}
            </li>
          ))}
        </ul>
        {total && (
          <div className="border-t bg-slate-50 px-4 py-3">
            <div className="flex items-center justify-between gap-2">
              <span className="text-xs font-medium tracking-wide text-muted-foreground">
                {totalLabel}
              </span>
              <span className="font-mono text-xs font-bold text-slate-900">
                {formatIDR(total.balance)}
              </span>
            </div>
            {full && (
              <div className="mt-1 flex gap-4 font-mono text-xs text-slate-900">
                <span>D: {formatIDR(total.debit)}</span>
                <span>K: {formatIDR(total.credit)}</span>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Desktop: table */}
      <div className="hidden md:block">
        <Table>
          <TableHeader>
            <TableRow className="bg-slate-50/80">
              <TableHead className="w-28">
                Kode
              </TableHead>
              <TableHead>
                Nama Akun
              </TableHead>
              {full && (
                <TableHead className="text-right">
                  Debit
                </TableHead>
              )}
              {full && (
                <TableHead className="text-right">
                  Kredit
                </TableHead>
              )}
              <TableHead className="text-right">
                Saldo
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {accounts.map((account, index) => (
              <TableRow key={`${account.account_id}-${index}`}>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {account.code}
                </TableCell>
                <TableCell className="text-sm font-medium text-foreground">
                  {account.name}
                </TableCell>
                {full && (
                  <TableCell className="text-right font-mono text-xs text-slate-600">
                    {account.debit ? formatIDR(account.debit) : "-"}
                  </TableCell>
                )}
                {full && (
                  <TableCell className="text-right font-mono text-xs text-slate-600">
                    {account.credit ? formatIDR(account.credit) : "-"}
                  </TableCell>
                )}
                <TableCell
                  className={cn(
                    "text-right font-mono text-xs font-bold",
                    account.balance < 0 ? "text-rose-600" : "text-slate-900"
                  )}
                >
                  {formatIDR(account.balance)}
                </TableCell>
              </TableRow>
            ))}
            {total && (
              <TableRow className="bg-slate-50 font-bold">
                <TableCell
                  colSpan={2}
                  className="text-xs font-medium tracking-wide text-muted-foreground"
                >
                  {totalLabel}
                </TableCell>
                {full && (
                  <TableCell className="text-right font-mono text-xs text-slate-900">
                    {formatIDR(total.debit)}
                  </TableCell>
                )}
                {full && (
                  <TableCell className="text-right font-mono text-xs text-slate-900">
                    {formatIDR(total.credit)}
                  </TableCell>
                )}
                <TableCell className="text-right font-mono text-xs text-slate-900">
                  {formatIDR(total.balance)}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </>
  );
}
