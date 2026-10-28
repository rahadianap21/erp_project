import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { GeneralLedgerLine, formatIDR, formatTanggal } from "./report-types";

interface LedgerTableProps {
  openingBalance: number;
  lines: GeneralLedgerLine[];
  totalDebit: number;
  totalCredit: number;
  closingBalance: number;
}

// LedgerTable menyajikan mutasi buku besar satu akun: saldo awal, baris
// transaksi dengan saldo berjalan, lalu saldo akhir.
export function LedgerTable({
  openingBalance,
  lines,
  totalDebit,
  totalCredit,
  closingBalance,
}: LedgerTableProps) {
  return (
    <>
      {/* Mobile: stacked ledger rows */}
      <div className="md:hidden">
        <div className="flex items-center justify-between gap-2 border-b bg-slate-50/60 px-4 py-2.5">
          <span className="text-xs font-medium tracking-wide text-muted-foreground">
            Saldo Awal
          </span>
          <span className="font-mono text-xs font-bold text-slate-900">
            {formatIDR(openingBalance)}
          </span>
        </div>

        {lines.length === 0 ? (
          <p className="px-4 py-8 text-center text-xs text-slate-400">
            Tidak ada mutasi pada periode ini.
          </p>
        ) : (
          <ul className="divide-y">
            {lines.map((line, index) => (
              <li
                key={`m-${line.entry_number}-${index}`}
                className="px-4 py-3"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="text-xs text-slate-500">
                      {formatTanggal(line.entry_date)}
                    </p>
                    <p className="font-mono text-xs text-muted-foreground">
                      {line.entry_number}
                    </p>
                    <p className="mt-0.5 text-xs text-slate-800">
                      {line.description || "-"}
                    </p>
                  </div>
                  <p
                    className={cn(
                      "shrink-0 text-right font-mono text-xs font-bold",
                      line.balance < 0 ? "text-rose-600" : "text-slate-900"
                    )}
                  >
                    {formatIDR(line.balance)}
                  </p>
                </div>
                <div className="mt-1.5 flex gap-4 font-mono text-xs text-slate-600">
                  <span>D: {line.debit ? formatIDR(line.debit) : "-"}</span>
                  <span>K: {line.credit ? formatIDR(line.credit) : "-"}</span>
                </div>
              </li>
            ))}
          </ul>
        )}

        <div className="border-t bg-slate-50 px-4 py-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium tracking-wide text-muted-foreground">
              Mutasi Periode
            </span>
            <span className="font-mono text-xs text-slate-900">
              D {formatIDR(totalDebit)} · K {formatIDR(totalCredit)}
            </span>
          </div>
        </div>

        <div className="border-t bg-slate-100/80 px-4 py-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium tracking-wide text-muted-foreground">
              Saldo Akhir
            </span>
            <span
              className={cn(
                "font-mono text-sm",
                closingBalance < 0 ? "text-rose-600" : "text-slate-900"
              )}
            >
              {formatIDR(closingBalance)}
            </span>
          </div>
        </div>
      </div>

      {/* Desktop: table */}
      <div className="hidden md:block">
        <Table>
          <TableHeader>
            <TableRow className="bg-slate-50/80">
              <TableHead className="w-32">
                Tanggal
              </TableHead>
              <TableHead className="w-36">
                No. Jurnal
              </TableHead>
              <TableHead>
                Keterangan
              </TableHead>
              <TableHead className="text-right">
                Debit
              </TableHead>
              <TableHead className="text-right">
                Kredit
              </TableHead>
              <TableHead className="text-right">
                Saldo
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow className="bg-slate-50/60">
              <TableCell
                colSpan={5}
                className="text-xs font-medium tracking-wide text-muted-foreground"
              >
                Saldo Awal
              </TableCell>
              <TableCell className="text-right font-mono text-xs font-bold text-slate-900">
                {formatIDR(openingBalance)}
              </TableCell>
            </TableRow>

            {lines.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="py-8 text-center text-xs text-slate-400"
                >
                  Tidak ada mutasi pada periode ini.
                </TableCell>
              </TableRow>
            ) : (
              lines.map((line, index) => (
                <TableRow key={`${line.entry_number}-${index}`}>
                  <TableCell className="text-xs text-slate-500">
                    {formatTanggal(line.entry_date)}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {line.entry_number}
                  </TableCell>
                  <TableCell className="text-xs text-slate-800">
                    {line.description || "-"}
                  </TableCell>
                  <TableCell className="text-right font-mono text-xs text-slate-600">
                    {line.debit ? formatIDR(line.debit) : "-"}
                  </TableCell>
                  <TableCell className="text-right font-mono text-xs text-slate-600">
                    {line.credit ? formatIDR(line.credit) : "-"}
                  </TableCell>
                  <TableCell
                    className={cn(
                      "text-right font-mono text-xs font-bold",
                      line.balance < 0 ? "text-rose-600" : "text-slate-900"
                    )}
                  >
                    {formatIDR(line.balance)}
                  </TableCell>
                </TableRow>
              ))
            )}

            <TableRow className="bg-slate-50 font-bold">
              <TableCell
                colSpan={3}
                className="text-xs font-medium tracking-wide text-muted-foreground"
              >
                Mutasi Periode
              </TableCell>
              <TableCell className="text-right font-mono text-xs text-slate-900">
                {formatIDR(totalDebit)}
              </TableCell>
              <TableCell className="text-right font-mono text-xs text-slate-900">
                {formatIDR(totalCredit)}
              </TableCell>
              <TableCell className="text-right font-mono text-xs text-slate-900">
                -
              </TableCell>
            </TableRow>

            <TableRow className="bg-slate-100/80 font-bold">
              <TableCell
                colSpan={5}
                className="text-xs font-medium tracking-wide text-muted-foreground"
              >
                Saldo Akhir
              </TableCell>
              <TableCell
                className={cn(
                  "text-right font-mono text-sm",
                  closingBalance < 0 ? "text-rose-600" : "text-slate-900"
                )}
              >
                {formatIDR(closingBalance)}
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </>
  );
}
