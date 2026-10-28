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

// Formatter uang dipusatkan di @/lib/money agar seluruh laporan memakai
// pemisah dan jumlah desimal yang sama dengan halaman lain.
export { formatIDR } from "@/lib/money";

// Format tanggal ISO dari backend menjadi tanggal Indonesia yang mudah dibaca.
export function formatTanggal(value: string): string {
  if (!value) return "-";
  const d = new Date(value);
  if (isNaN(d.getTime())) return value;
  return d.toLocaleDateString("id-ID", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  });
}

export function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

export function awalTahunISO(): string {
  return `${new Date().getFullYear()}-01-01`;
}
