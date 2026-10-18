import React from "react";

const ACCENTS: Record<string, string> = {
    emerald: "bg-emerald-50 text-emerald-600 border-emerald-100",
    blue: "bg-blue-50 text-blue-600 border-blue-100",
    violet: "bg-violet-50 text-violet-600 border-violet-100",
    amber: "bg-amber-50 text-amber-600 border-amber-100",
};

interface ReportHeaderProps {
    icon: React.ComponentType<{ className?: string }>;
    title: string;
    subtitle: string;
    accent?: keyof typeof ACCENTS;
    children?: React.ReactNode;
}

// ReportHeader adalah judul halaman laporan yang seragam dengan modul lain.
export function ReportHeader({ icon: Icon, title, subtitle, accent = "emerald", children }: ReportHeaderProps) {
    return (
        <div className="flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2">
                <div className={`rounded-xl border p-2 ${ACCENTS[accent] ?? ACCENTS.emerald}`}>
                    <Icon className="size-5" />
                </div>
                <div>
                    <h1 className="text-sm font-black uppercase tracking-wider text-slate-900">{title}</h1>
                    <p className="text-xs text-slate-500">{subtitle}</p>
                </div>
            </div>
            {children}
        </div>
    );
}
