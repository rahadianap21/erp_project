"use client";

import React from "react";
import { BarChart3 } from "lucide-react";
import { ForbiddenView } from "@/components/forbidden-view";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ReportHeader } from "./components/report-header";
import { useReportAccess } from "./components/use-report-access";
import { BalanceSheetView } from "./components/balance-sheet-view";
import { IncomeStatementView } from "./components/income-statement-view";
import { TrialBalanceView } from "./components/trial-balance-view";
import { GeneralLedgerView } from "./components/general-ledger-view";

const TABS = [
    { value: "balance-sheet", label: "Neraca" },
    { value: "income-statement", label: "Laba Rugi" },
    { value: "trial-balance", label: "Neraca Saldo" },
    { value: "general-ledger", label: "Buku Besar" },
];

export default function ReportsPage() {
    const { ready, canRead, token, httpStatus, setHttpStatus, reset } = useReportAccess();

    if (!ready) {
        return <div className="p-8 text-center text-xs text-slate-400">Memuat&hellip;</div>;
    }
    if (httpStatus === 403 || !canRead) {
        return <ForbiddenView reset={reset} />;
    }

    return (
        <main className="flex w-full flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8">
            <ReportHeader
                icon={BarChart3}
                accent="emerald"
                title="Laporan Keuangan (Financial Reports)"
                subtitle="Neraca, laba rugi, neraca saldo, dan buku besar yang dihitung dari jurnal terposting."
            />

            <Tabs defaultValue="balance-sheet" className="gap-4">
                <TabsList className="grid w-full grid-cols-2 sm:grid-cols-4">
                    {TABS.map((tab) => (
                        <TabsTrigger key={tab.value} value={tab.value}>
                            {tab.label}
                        </TabsTrigger>
                    ))}
                </TabsList>

                <TabsContent value="balance-sheet">
                    <BalanceSheetView token={token} onForbidden={() => setHttpStatus(403)} />
                </TabsContent>
                <TabsContent value="income-statement">
                    <IncomeStatementView token={token} onForbidden={() => setHttpStatus(403)} />
                </TabsContent>
                <TabsContent value="trial-balance">
                    <TrialBalanceView token={token} onForbidden={() => setHttpStatus(403)} />
                </TabsContent>
                <TabsContent value="general-ledger">
                    <GeneralLedgerView token={token} onForbidden={() => setHttpStatus(403)} />
                </TabsContent>
            </Tabs>
        </main>
    );
}
