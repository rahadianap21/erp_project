"use client";

import React, { useCallback, useEffect, useState } from "react";
import { BadgeCheck, TriangleAlert } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ReportState } from "./report-state";
import { AccountTable } from "./account-table";
import {
  BalanceSheet,
  formatIDR,
  formatTanggal,
  todayISO,
} from "./report-types";
import { fetchLaporan } from "@/lib/api-client";

interface BalanceSheetViewProps {
  token?: string;
  onForbidden: () => void;
}

export function BalanceSheetView({
  token,
  onForbidden,
}: BalanceSheetViewProps) {
  const [asOf, setAsOf] = useState<string>(todayISO());
  const [report, setReport] = useState<BalanceSheet | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!token || !asOf) return;
    setLoading(true);
    setError(null);
    const result = await fetchLaporan<BalanceSheet>(
      `/api/accounting/reports/balance-sheet?as_of=${asOf}`
    );
    if (result.status === 403) {
      onForbidden();
      return;
    }
    if (!result.data) {
      setReport(null);
      setError(result.message || "Gagal memuat neraca.");
    } else {
      setReport(result.data);
    }
    setLoading(false);
  }, [token, asOf, onForbidden]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <label className="text-[11px] font-bold uppercase tracking-wide text-slate-600">
          Per Tanggal
        </label>
        <input
          type="date"
          value={asOf}
          onChange={(e) => setAsOf(e.target.value)}
          className="rounded-lg border border-slate-200 bg-slate-50/50 px-3 py-2 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20"
        />
      </div>

      {report && (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
          <div className="rounded-xl border border-slate-200 bg-white p-4">
            <p className="text-[10px] font-bold uppercase tracking-wider text-slate-500">
              Total Aset
            </p>
            <p className="mt-1 font-mono text-lg font-black text-slate-900">
              {formatIDR(report.total_assets)}
            </p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white p-4">
            <p className="text-[10px] font-bold uppercase tracking-wider text-slate-500">
              Total Kewajiban + Ekuitas
            </p>
            <p className="mt-1 font-mono text-lg font-black text-slate-900">
              {formatIDR(report.total_liabilities_and_equity)}
            </p>
          </div>
          <div
            className={`flex items-center gap-2 rounded-xl border p-4 ${report.balanced ? "border-emerald-200 bg-emerald-50" : "border-rose-200 bg-rose-50"}`}
          >
            {report.balanced ? (
              <BadgeCheck className="size-5 text-emerald-600" />
            ) : (
              <TriangleAlert className="size-5 text-rose-600" />
            )}
            <div>
              <p
                className={`text-[10px] font-bold uppercase tracking-wider ${report.balanced ? "text-emerald-700" : "text-rose-700"}`}
              >
                {report.balanced ? "Seimbang" : "Tidak Seimbang"}
              </p>
              <p className="text-[11px] text-slate-500">
                per {formatTanggal(report.as_of)}
              </p>
            </div>
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
                <CardTitle className="text-xs font-bold uppercase tracking-wider text-slate-700">
                  Aset
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <AccountTable
                  accounts={report.assets}
                  variant="balance"
                  totalLabel="Total Aset"
                  total={{ debit: 0, credit: 0, balance: report.total_assets }}
                />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-xs font-bold uppercase tracking-wider text-slate-700">
                  Kewajiban
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <AccountTable
                  accounts={report.liabilities}
                  variant="balance"
                  totalLabel="Total Kewajiban"
                  total={{
                    debit: 0,
                    credit: 0,
                    balance: report.total_liabilities,
                  }}
                />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-xs font-bold uppercase tracking-wider text-slate-700">
                  Ekuitas
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <AccountTable
                  accounts={report.equity}
                  variant="balance"
                  totalLabel="Total Ekuitas"
                  total={{ debit: 0, credit: 0, balance: report.total_equity }}
                />
              </CardContent>
            </Card>
          </>
        )
      )}
    </div>
  );
}
