"use client";

import { apiGet, apiPost, apiPut, messageFromError } from "@/lib/api-client";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  ChevronDown,
  ChevronUp,
  Loader2,
  BookOpenCheck,
  Hash,
  FolderOpen,
  Scale,
  Power,
} from "lucide-react";
import { toast } from "sonner";
import { ComboboxSelect } from "@/components/combobox-select";
import { ChartOfAccount } from "./chart-of-account-table";

interface ChartOfAccountFormProps {
  token: string | undefined;
  editingAccount: ChartOfAccount | null;
  onSuccess: (message: string) => void;
  onCancelEdit: () => void;
}

type AccountType = "ASSET" | "LIABILITY" | "EQUITY" | "REVENUE" | "EXPENSE";
type NormalBalance = "DEBIT" | "CREDIT";

const ACCOUNT_TYPES: AccountType[] = [
  "ASSET",
  "LIABILITY",
  "EQUITY",
  "REVENUE",
  "EXPENSE",
];

export function ChartOfAccountForm({
  token,
  editingAccount,
  onSuccess,
  onCancelEdit,
}: ChartOfAccountFormProps) {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [parentOptions, setParentOptions] = useState<ChartOfAccount[]>([]);
  const isEditMode = editingAccount !== null;

  const [code, setCode] = useState<string>("");
  const [name, setName] = useState<string>("");
  const [accountType, setAccountType] = useState<AccountType>("ASSET");
  const [normalBalance, setNormalBalance] = useState<NormalBalance>("DEBIT");
  const [parentId, setParentId] = useState<string>("");
  const [isHeader, setIsHeader] = useState<boolean>(false);
  const [isActive, setIsActive] = useState<boolean>(true);

  const fetchParentOptions = async () => {
    try {
      const result = await apiGet<ChartOfAccount[]>(
        `/api/chart-of-accounts?limit=500`
      );
      if (result.data) {
        const options: ChartOfAccount[] = result.data.filter(
          (a: ChartOfAccount) => a.is_header
        );
        setParentOptions(
          editingAccount
            ? options.filter((a) => a.id !== editingAccount.id)
            : options
        );
      }
    } catch (err) {
      console.error("Gagal memuat opsi induk CoA:", err);
    }
  };

  useEffect(() => {
    if (token) fetchParentOptions();
  }, [token, editingAccount]);

  useEffect(() => {
    if (editingAccount) {
      setCode(editingAccount.code);
      setName(editingAccount.name);
      setAccountType(editingAccount.account_type);
      setNormalBalance(editingAccount.normal_balance);
      setParentId(
        editingAccount.parent_id ? String(editingAccount.parent_id) : ""
      );
      setIsHeader(editingAccount.is_header);
      setIsActive(editingAccount.is_active);
      setIsExpanded(true);
    } else {
      resetForm();
    }
  }, [editingAccount]);

  const resetForm = () => {
    setCode("");
    setName("");
    setAccountType("ASSET");
    setNormalBalance("DEBIT");
    setParentId("");
    setIsHeader(false);
    setIsActive(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    const url = isEditMode
      ? `/api/chart-of-accounts?id=${editingAccount.id}`
      : `/api/chart-of-accounts`;

    try {
      const payload = {
        code,
        name,
        account_type: accountType,
        normal_balance: normalBalance,
        parent_id: parentId ? Number(parentId) : null,
        is_header: isHeader,
        is_active: isActive,
      };
      await (isEditMode ? apiPut(url, payload) : apiPost(url, payload));

      resetForm();
      setIsExpanded(false);
      onSuccess(
        isEditMode
          ? `Akun ${code} berhasil dikoreksi.`
          : `Akun ${code} berhasil dibuat.`
      );
    } catch (error) {
      toast.error("Gagal Menyimpan Akun", {
        description: messageFromError(error),
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-2xs border border-slate-200 overflow-hidden transition-all duration-300">
      <button
        type="button"
        onClick={() => !editingAccount && setIsExpanded(!isExpanded)}
        disabled={!!editingAccount}
        className="w-full px-6 py-3.5 flex items-center justify-between bg-slate-50/80 hover:bg-slate-100/60 transition-colors text-left cursor-pointer"
      >
        <span className="text-xs font-bold uppercase tracking-wider text-slate-700 flex items-center gap-2">
          <BookOpenCheck className="size-4 text-emerald-600" />
          {editingAccount
            ? `Edit Akun: ${editingAccount.code} - ${editingAccount.name}`
            : "Tambah Akun Bagan Baru"}
        </span>
        {!editingAccount &&
          (isExpanded ? (
            <ChevronUp className="h-4 w-4" />
          ) : (
            <ChevronDown className="h-4 w-4" />
          ))}
      </button>

      <div
        className={`transition-all duration-300 ease-in-out ${isExpanded ? "max-h-[1200px] p-6 border-t border-slate-100" : "max-h-0 opacity-0 pointer-events-none"}`}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <Hash className="size-3" /> Kode Akun
              </label>
              <input
                type="text"
                required
                value={code}
                onChange={(e) => setCode(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 text-slate-900 font-mono"
                placeholder="Contoh: 1-1103"
              />
            </div>
            <div className="md:col-span-2 space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <BookOpenCheck className="size-3" /> Nama Akun
              </label>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 text-slate-900"
                placeholder="Contoh: Kas Opname"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <FolderOpen className="size-3" /> Tipe Akun
              </label>
              <ComboboxSelect
                options={ACCOUNT_TYPES.map((t) => ({ value: t, label: t }))}
                value={accountType}
                placeholder="Pilih Tipe Akun..."
                emptyText="Tipe akun tidak ditemukan..."
                onChange={(val) =>
                  setAccountType((val || "ASSET") as AccountType)
                }
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <Scale className="size-3" /> Saldo Normal
              </label>
              <ComboboxSelect
                options={[
                  { value: "DEBIT", label: "DEBIT" },
                  { value: "CREDIT", label: "CREDIT" },
                ]}
                value={normalBalance}
                placeholder="Pilih Saldo Normal..."
                emptyText="Saldo normal tidak ditemukan..."
                onChange={(val) =>
                  setNormalBalance((val || "DEBIT") as NormalBalance)
                }
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="md:col-span-2 space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <FolderOpen className="size-3" /> Induk Akun (Hierarki)
              </label>
              <ComboboxSelect
                options={[
                  { value: "", label: "--- AKUN UTAMA / ROOT ---" },
                  ...parentOptions.map((opt) => ({
                    value: String(opt.id),
                    label: `${opt.code} · ${opt.name}`,
                  })),
                ]}
                value={parentId}
                placeholder="Pilih Akun Induk..."
                emptyText="Akun induk tidak ditemukan..."
                onChange={(val) => setParentId(val || "")}
                fallbackLabel={
                  editingAccount?.parent_id
                    ? `${editingAccount.parent_code || ""} · ${editingAccount.parent_name || ""}`
                    : undefined
                }
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <Power className="size-3" /> Header / Posting
              </label>
              <label className="flex items-center gap-2 pt-1.5 cursor-pointer">
                <input
                  type="checkbox"
                  checked={isHeader}
                  onChange={(e) => setIsHeader(e.target.checked)}
                  className="accent-emerald-600 size-4 cursor-pointer"
                />
                <span className="text-xs text-slate-600">
                  Akun header (grup, tidak diposting)
                </span>
              </label>
            </div>
          </div>

          {isEditMode && (
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <Power className="size-3" /> Status Akun
              </label>
              <label className="flex items-center gap-2 pt-1.5 cursor-pointer">
                <input
                  type="checkbox"
                  checked={isActive}
                  onChange={(e) => setIsActive(e.target.checked)}
                  className="accent-emerald-600 size-4 cursor-pointer"
                />
                <span className="text-xs text-slate-600">
                  Akun aktif (dapat digunakan dalam jurnal)
                </span>
              </label>
            </div>
          )}

          <div className="flex justify-end space-x-3 pt-3 border-t border-slate-100">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                onCancelEdit();
                resetForm();
                setIsExpanded(false);
              }}
              className="text-xs h-8 cursor-pointer"
            >
              Batal
            </Button>
            <Button
              type="submit"
              size="sm"
              disabled={isLoading}
              className="bg-emerald-600 hover:bg-emerald-700 text-white text-xs h-8 shadow-xs cursor-pointer"
            >
              {isLoading ? (
                <span className="flex items-center gap-1.5">
                  <Loader2 className="size-3 animate-spin" /> Memproses...
                </span>
              ) : editingAccount ? (
                "Perbarui Akun"
              ) : (
                "Simpan Akun"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
