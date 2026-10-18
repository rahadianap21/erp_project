"use client";

import React, { useCallback, useEffect, useState } from "react";
import { BadgeCheck, TriangleAlert } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { ReportState } from "./report-state";
import { AccountTable } from "./account-table";
import { fetchLaporan } from "@/lib/api-client";
import { TrialBalance, awalTahunISO, todayISO } from "./report-types";

interface TrialBalanceViewProps {
  token?: string;
  onForbidden: () => void;
}

export function TrialBalanceView({
  token,
  onForbidden,
}: TrialBalanceViewProps) {
  const [from, setFrom] = useState<string>(awalTahunISO());
  const [to, setTo] = useState<string>(todayISO());
  const [report, setReport] = useState<TrialBalance | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!token || !from || !to) return;
    setLoading(true);
    setError(null);
    const result = await fetchLaporan<TrialBalance>(
      `/api/accounting/reports/trial-balance?from=${from}&to=${to}`
    );
    if (result.status === 403) {
      onForbidden();
      return;
    }
    if (!result.data) {
      setReport(null);
      setError(result.message || "Gagal memuat neraca saldo.");
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
        <label className="text-[11px] font-bold uppercase tracking-wide text-slate-600">
          Dari
        </label>
        <input
          type="date"
          value={from}
          onChange={(e) => setFrom(e.target.value)}
          className="rounded-lg border border-slate-200 bg-slate-50/50 px-3 py-2 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
        />
        <label className="text-[11px] font-bold uppercase tracking-wide text-slate-600">
          Sampai
        </label>
        <input
          type="date"
          value={to}
          onChange={(e) => setTo(e.target.value)}
          className="rounded-lg border border-slate-200 bg-slate-50/50 px-3 py-2 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
        />
      </div>

      {report && (
        <div
          className={`inline-flex w-fit items-center gap-2 rounded-xl border px-4 py-3 ${report.balanced ? "border-emerald-200 bg-emerald-50" : "border-rose-200 bg-rose-50"}`}
        >
          {report.balanced ? (
            <BadgeCheck className="size-5 text-emerald-600" />
          ) : (
            <TriangleAlert className="size-5 text-rose-600" />
          )}
          <span
            className={`text-xs font-bold uppercase tracking-wider ${report.balanced ? "text-emerald-700" : "text-rose-700"}`}
          >
            {report.balanced
              ? "Neraca Saldo Seimbang"
              : "Neraca Saldo Tidak Seimbang"}
          </span>
        </div>
      )}

      <Card>
        <CardContent className="p-0">
          {loading || error || !report ? (
            <ReportState loading={loading} error={error} />
          ) : (
            <AccountTable
              accounts={report.accounts}
              variant="full"
              totalLabel="Total"
              total={{
                debit: report.total_debit,
                credit: report.total_credit,
                balance: report.total_debit - report.total_credit,
              }}
              emptyText="Belum ada aktivitas jurnal pada periode ini."
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
