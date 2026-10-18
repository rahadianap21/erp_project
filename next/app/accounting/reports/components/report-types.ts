export interface ReportAccount {
    account_id: number;
    code: string;
    name: string;
    account_type: string;
    debit: number;
    credit: number;
    balance: number;
}

export interface IncomeStatement {
    from: string;
    to: string;
    revenue: ReportAccount[];
    expenses: ReportAccount[];
    total_revenue: number;
    total_expenses: number;
    net_income: number;
}

export interface BalanceSheet {
    as_of: string;
    assets: ReportAccount[];
    liabilities: ReportAccount[];
    equity: ReportAccount[];
    total_assets: number;
    total_liabilities: number;
    total_equity: number;
    total_liabilities_and_equity: number;
    balanced: boolean;
}

export interface TrialBalance {
    from: string;
    to: string;
    accounts: ReportAccount[];
    total_debit: number;
    total_credit: number;
    balanced: boolean;
}

export interface GeneralLedgerLine {
    entry_number: string;
    entry_date: string;
    description: string;
    debit: number;
    credit: number;
    balance: number;
}

export interface GeneralLedger {
    account: ReportAccount;
    from: string;
    to: string;
    opening_balance: number;
    lines: GeneralLedgerLine[];
    total_debit: number;
    total_credit: number;
    closing_balance: number;
}

// Label bahasa Indonesia untuk tipe akun dari backend.
export const ACCOUNT_TYPE_LABEL: Record<string, string> = {
    ASSET: "Aset",
    LIABILITY: "Kewajiban",
    EQUITY: "Ekuitas",
    REVENUE: "Pendapatan",
    EXPENSE: "Beban",
};

// Format mata uang Rupiah mengikuti konvensi tabel lain di aplikasi.
export function formatIDR(num: number): string {
    return new Intl.NumberFormat("id-ID", {
        style: "currency",
        currency: "IDR",
        maximumFractionDigits: 0,
    }).format(Number(num) || 0);
}

// Format tanggal ISO dari backend menjadi tanggal Indonesia yang mudah dibaca.
export function formatTanggal(value: string): string {
    if (!value) return "-";
    const d = new Date(value);
    if (isNaN(d.getTime())) return value;
    return d.toLocaleDateString("id-ID", { day: "2-digit", month: "long", year: "numeric" });
}

export function todayISO(): string {
    return new Date().toISOString().slice(0, 10);
}

export function awalTahunISO(): string {
    return `${new Date().getFullYear()}-01-01`;
}

// fetchLaporan membungkus pemanggilan endpoint laporan: menyertakan token,
// membaca amplop {status, data}, dan mengembalikan status HTTP apa adanya agar
// halaman bisa membedakan 403 (sesi/izin) dari kegagalan lain.
export async function fetchLaporan<T>(
    path: string,
    token: string | undefined,
): Promise<{ status: number; data?: T; message?: string }> {
    try {
        const response = await fetch(`${path}`, {
            headers: { Authorization: `Bearer ${token}` },
        });
        const body = await response.json().catch(() => ({}));
        if (!response.ok) {
            return {
                status: response.status,
                message: body?.message || body?.error || "Gagal memuat laporan.",
            };
        }
        return { status: response.status, data: body?.data as T };
    } catch {
        return { status: 0, message: "Tidak dapat menghubungi server." };
    }
}
