import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
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
        return <div className="px-4 py-8 text-center text-xs text-slate-400">{emptyText}</div>;
    }

    return (
        <Table>
            <TableHeader>
                <TableRow className="bg-slate-50/80">
                    <TableHead className="w-28 text-[10px] font-bold uppercase tracking-wider">Kode</TableHead>
                    <TableHead className="text-[10px] font-bold uppercase tracking-wider">Nama Akun</TableHead>
                    {full && (
                        <TableHead className="text-right text-[10px] font-bold uppercase tracking-wider">Debit</TableHead>
                    )}
                    {full && (
                        <TableHead className="text-right text-[10px] font-bold uppercase tracking-wider">Kredit</TableHead>
                    )}
                    <TableHead className="text-right text-[10px] font-bold uppercase tracking-wider">Saldo</TableHead>
                </TableRow>
            </TableHeader>
            <TableBody>
                {accounts.map((account, index) => (
                    <TableRow key={`${account.account_id}-${index}`}>
                        <TableCell className="font-mono text-[11px] text-slate-500">{account.code}</TableCell>
                        <TableCell className="text-xs font-medium text-slate-800">{account.name}</TableCell>
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
                                account.balance < 0 ? "text-rose-600" : "text-slate-900",
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
                            className="text-[11px] uppercase tracking-wider text-slate-700"
                        >
                            {totalLabel}
                        </TableCell>
                        {full && (
                            <TableCell className="text-right font-mono text-xs text-slate-900">{formatIDR(total.debit)}</TableCell>
                        )}
                        {full && (
                            <TableCell className="text-right font-mono text-xs text-slate-900">{formatIDR(total.credit)}</TableCell>
                        )}
                        <TableCell className="text-right font-mono text-xs text-slate-900">{formatIDR(total.balance)}</TableCell>
                    </TableRow>
                )}
            </TableBody>
        </Table>
    );
}
