import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
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
export function LedgerTable({ openingBalance, lines, totalDebit, totalCredit, closingBalance }: LedgerTableProps) {
    return (
        <Table>
            <TableHeader>
                <TableRow className="bg-slate-50/80">
                    <TableHead className="w-32 text-[10px] font-bold uppercase tracking-wider">Tanggal</TableHead>
                    <TableHead className="w-36 text-[10px] font-bold uppercase tracking-wider">No. Jurnal</TableHead>
                    <TableHead className="text-[10px] font-bold uppercase tracking-wider">Keterangan</TableHead>
                    <TableHead className="text-right text-[10px] font-bold uppercase tracking-wider">Debit</TableHead>
                    <TableHead className="text-right text-[10px] font-bold uppercase tracking-wider">Kredit</TableHead>
                    <TableHead className="text-right text-[10px] font-bold uppercase tracking-wider">Saldo</TableHead>
                </TableRow>
            </TableHeader>
            <TableBody>
                <TableRow className="bg-slate-50/60">
                    <TableCell colSpan={5} className="text-[11px] font-bold uppercase tracking-wider text-slate-700">
                        Saldo Awal
                    </TableCell>
                    <TableCell className="text-right font-mono text-xs font-bold text-slate-900">
                        {formatIDR(openingBalance)}
                    </TableCell>
                </TableRow>

                {lines.length === 0 ? (
                    <TableRow>
                        <TableCell colSpan={6} className="py-8 text-center text-xs text-slate-400">
                            Tidak ada mutasi pada periode ini.
                        </TableCell>
                    </TableRow>
                ) : (
                    lines.map((line, index) => (
                        <TableRow key={`${line.entry_number}-${index}`}>
                            <TableCell className="text-[11px] text-slate-500">{formatTanggal(line.entry_date)}</TableCell>
                            <TableCell className="font-mono text-[11px] text-slate-500">{line.entry_number}</TableCell>
                            <TableCell className="text-xs text-slate-800">{line.description || "-"}</TableCell>
                            <TableCell className="text-right font-mono text-xs text-slate-600">
                                {line.debit ? formatIDR(line.debit) : "-"}
                            </TableCell>
                            <TableCell className="text-right font-mono text-xs text-slate-600">
                                {line.credit ? formatIDR(line.credit) : "-"}
                            </TableCell>
                            <TableCell
                                className={cn(
                                    "text-right font-mono text-xs font-bold",
                                    line.balance < 0 ? "text-rose-600" : "text-slate-900",
                                )}
                            >
                                {formatIDR(line.balance)}
                            </TableCell>
                        </TableRow>
                    ))
                )}

                <TableRow className="bg-slate-50 font-bold">
                    <TableCell colSpan={3} className="text-[11px] uppercase tracking-wider text-slate-700">
                        Mutasi Periode
                    </TableCell>
                    <TableCell className="text-right font-mono text-xs text-slate-900">{formatIDR(totalDebit)}</TableCell>
                    <TableCell className="text-right font-mono text-xs text-slate-900">{formatIDR(totalCredit)}</TableCell>
                    <TableCell className="text-right font-mono text-xs text-slate-900">-</TableCell>
                </TableRow>

                <TableRow className="bg-slate-100/80 font-bold">
                    <TableCell colSpan={5} className="text-[11px] uppercase tracking-wider text-slate-700">
                        Saldo Akhir
                    </TableCell>
                    <TableCell
                        className={cn(
                            "text-right font-mono text-sm",
                            closingBalance < 0 ? "text-rose-600" : "text-slate-900",
                        )}
                    >
                        {formatIDR(closingBalance)}
                    </TableCell>
                </TableRow>
            </TableBody>
        </Table>
    );
}
