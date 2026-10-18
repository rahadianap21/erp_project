"use client";

import React, { useCallback, useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ReportState } from "./report-state";
import { AccountTable } from "./account-table";
import {
    IncomeStatement,
    awalTahunISO,
    formatIDR,
    formatTanggal,
    todayISO,
} from "./report-types";
import { fetchLaporan } from "@/lib/api-client";

interface IncomeStatementViewProps {
    token?: string;
    onForbidden: () => void;
}

export function IncomeStatementView({ token, onForbidden }: IncomeStatementViewProps) {
    const [from, setFrom] = useState<string>(awalTahunISO());
    const [to, setTo] = useState<string>(todayISO());
    const [report, setReport] = useState<IncomeStatement | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [error, setError] = useState<string | null>(null);

    const load = useCallback(async () => {
        if (!token || !from || !to) return;
        setLoading(true);
        setError(null);
        const result = await fetchLaporan<IncomeStatement>(
            `/api/accounting/reports/income-statement?from=${from}&to=${to}`,
);
        if (result.status === 403) {
            onForbidden();
            return;
        }
        if (!result.data) {
            setReport(null);
            setError(result.message || "Gagal memuat laporan laba rugi.");
        } else {
            setReport(result.data);
        }
        setLoading(false);
    }, [token, from, to, onForbidden]);

    useEffect(() => {
        load();
    }, [load]);

    return (
        <div className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center gap-2">
                <label className="text-[11px] font-bold uppercase tracking-wide text-slate-600">Dari</label>
                <input
                    type="date"
                    value={from}
                    onChange={(e) => setFrom(e.target.value)}
                    className="rounded-lg border border-slate-200 bg-slate-50/50 px-3 py-2 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                />
                <label className="text-[11px] font-bold uppercase tracking-wide text-slate-600">Sampai</label>
                <input
                    type="date"
                    value={to}
                    onChange={(e) => setTo(e.target.value)}
                    className="rounded-lg border border-slate-200 bg-slate-50/50 px-3 py-2 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
                />
            </div>

            {report && (
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                        <p className="text-[10px] font-bold uppercase tracking-wider text-slate-500">Total Pendapatan</p>
                        <p className="mt-1 font-mono text-lg font-black text-slate-900">{formatIDR(report.total_revenue)}</p>
                    </div>
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                        <p className="text-[10px] font-bold uppercase tracking-wider text-slate-500">Total Beban</p>
                        <p className="mt-1 font-mono text-lg font-black text-slate-900">{formatIDR(report.total_expenses)}</p>
                    </div>
                    <div className={`rounded-xl border p-4 ${report.net_income >= 0 ? "border-emerald-200 bg-emerald-50" : "border-rose-200 bg-rose-50"}`}>
                        <p className="text-[10px] font-bold uppercase tracking-wider text-slate-500">
                            {report.net_income >= 0 ? "Laba Bersih" : "Rugi Bersih"}
                        </p>
                        <p className={`mt-1 font-mono text-lg font-black ${report.net_income >= 0 ? "text-emerald-700" : "text-rose-700"}`}>
                            {formatIDR(report.net_income)}
                        </p>
                        <p className="text-[11px] text-slate-500">per {formatTanggal(report.to)}</p>
                    </div>
                </div>
            )}

            {loading || error ? (
                <Card>
                    <CardContent className="p-0">
                        <ReportState loading={loading} error={error} />
                    </CardContent>
                </Card>
            ) : (
                report && (
                    <>
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-xs font-bold uppercase tracking-wider text-slate-700">Pendapatan</CardTitle>
                            </CardHeader>
                            <CardContent className="p-0">
                                <AccountTable
                                    accounts={report.revenue}
                                    variant="balance"
                                    totalLabel="Total Pendapatan"
                                    total={{ debit: 0, credit: 0, balance: report.total_revenue }}
                                    emptyText="Belum ada pendapatan pada periode ini."
                                />
                            </CardContent>
                        </Card>

                        <Card>
                            <CardHeader>
                                <CardTitle className="text-xs font-bold uppercase tracking-wider text-slate-700">Beban</CardTitle>
                            </CardHeader>
                            <CardContent className="p-0">
                                <AccountTable
                                    accounts={report.expenses}
                                    variant="balance"
                                    totalLabel="Total Beban"
                                    total={{ debit: 0, credit: 0, balance: report.total_expenses }}
                                    emptyText="Belum ada beban pada periode ini."
                                />
                            </CardContent>
                        </Card>
                    </>
                )
            )}
        </div>
    );
}
