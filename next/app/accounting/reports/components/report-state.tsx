// ReportState menampilkan status muat/gagal secara seragam di dalam kartu
// laporan. Mengembalikan null bila laporan siap ditampilkan.
export function ReportState({ loading, error }: { loading: boolean; error: string | null }) {
    if (loading) {
        return <div className="px-4 py-10 text-center text-xs text-slate-400">Memuat laporan&hellip;</div>;
    }
    if (error) {
        return <div className="px-4 py-10 text-center text-xs text-rose-600">{error}</div>;
    }
    return null;
}
