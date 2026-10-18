"use client";

import React, { useCallback, useEffect, useState } from "react";
import { ComboboxSelect } from "@/components/combobox-select";
import { Card, CardContent } from "@/components/ui/card";
import { ReportState } from "./report-state";
import { LedgerTable } from "./ledger-table";
import {
    ACCOUNT_TYPE_LABEL,
    GeneralLedger,
    awalTahunISO,
    todayISO,
} from "./report-types";
import { fetchLaporan } from "@/lib/api-client";
import { ChartOfAccount } from "../../chart-of-accounts/components/chart-of-account-table";

interface GeneralLedgerViewProps {
    token?: string;
    onForbidden: () => void;
}

export function GeneralLedgerView({ token, onForbidden }: GeneralLedgerViewProps) {
    const [accounts, setAccounts] = useState<ChartOfAccount[]>([]);
    const [accountId, setAccountId] = useState<string>("");
    const [from, setFrom] = useState<string>(awalTahunISO());
    const [to, setTo] = useState<string>(todayISO());
    const [report, setReport] = useState<GeneralLedger | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [error, setError] = useState<string | null>(null);

    const fetchAccounts = useCallback(async () => {
        if (!token) return;
        try {
            const res = await fetch(`/api/chart-of-accounts?limit=500`, {
                headers: { Authorization: `Bearer ${token}` },
            });
            const result = await res.json();
            if (res.ok && result.data) {
                setAccounts(result.data.filter((a: ChartOfAccount) => !a.is_header && a.is_active));
            }
        } catch (err) {
            console.error("Gagal memuat daftar akun buku besar:", err);
        }
    }, [token]);

    useEffect(() => {
        fetchAccounts();
    }, [fetchAccounts]);

    const load = useCallback(async () => {
        if (!token || !accountId || !from || !to) return;
        setLoading(true);
        setError(null);
        const result = await fetchLaporan<GeneralLedger>(
            `/api/accounting/reports/general-ledger?account_id=${accountId}&from=${from}&to=${to}`,
);
        if (result.status === 403) {
            onForbidden();
            return;
        }
        if (!result.data) {
            setReport(null);
            setError(result.message || "Gagal memuat buku besar.");
        } else {
            setReport(result.data);
        }
        setLoading(false);
    }, [token, accountId, from, to, onForbidden]);

    useEffect(() => {
        load();
    }, [load]);

    const selectedAccount = accounts.find((a) => String(a.id) === accountId);

    return (
        <div className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center gap-2">
                <div className="w-64">
                    <ComboboxSelect
                        options={accounts.map((a) => ({ value: String(a.id), label: `${a.code} · ${a.name}` }))}
                        value={accountId}
                        onChange={(val) => setAccountId(val)}
                        placeholder="-- Pilih Akun --"
                        emptyText="Akun tidak ditemukan..."
                    />
                </div>
                <input
                    type="date"
                    value={from}
                    onChange={(e) => setFrom(e.target.value)}
                    className="rounded-lg border border-slate-200 bg-slate-50/50 px-3 py-2 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                />
                <input
                    type="date"
                    value={to}
                    onChange={(e) => setTo(e.target.value)}
                    className="rounded-lg border border-slate-200 bg-slate-50/50 px-3 py-2 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                />
            </div>

            {selectedAccount && (
                <div className="inline-flex w-fit flex-wrap items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2 text-xs">
                    <span className="font-mono font-bold text-slate-700">{selectedAccount.code}</span>
                    <span className="font-medium text-slate-800">{selectedAccount.name}</span>
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-slate-500">
                        {ACCOUNT_TYPE_LABEL[selectedAccount.account_type] || selectedAccount.account_type}
                    </span>
                </div>
            )}

            <Card>
                <CardContent className="p-0">
                    {!accountId ? (
                        <div className="px-4 py-10 text-center text-xs text-slate-400">
                            Pilih satu akun untuk menampilkan buku besarnya.
                        </div>
                    ) : loading || error || !report ? (
                        <ReportState loading={loading} error={error} />
                    ) : (
                        <LedgerTable
                            openingBalance={report.opening_balance}
                            lines={report.lines}
                            totalDebit={report.total_debit}
                            totalCredit={report.total_credit}
                            closingBalance={report.closing_balance}
                        />
                    )}
                </CardContent>
            </Card>
        </div>
    );
}
