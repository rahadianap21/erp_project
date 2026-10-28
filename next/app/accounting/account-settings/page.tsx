"use client";

import { ApiError, apiGet, apiPut, messageFromError } from "@/lib/api-client";
import { ensureSession, readPermissionList } from "@/lib/permissions";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { ListTree, RefreshCw, Save } from "lucide-react";
import { ComboboxSelect } from "@/components/combobox-select";
import { ForbiddenView } from "@/components/forbidden-view";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type { ChartOfAccount } from "@/app/accounting/chart-of-accounts/components/chart-of-account-table";

interface AccountingSettingDetail {
  mapping_key: string;
  description?: string;
  account_id: number;
  account_code: string;
  account_name: string;
  account_active: boolean;
}

interface SettingField {
  key: string;
  label: string;
  hint: string;
}

interface SettingGroup {
  title: string;
  description: string;
  fields: SettingField[];
}

// Urutan dan penjelasan sengaja ditulis di UI agar pengguna tahu jurnal mana
// yang memakai tiap akun; deskripsi di database hanya menjadi cadangan.
const SETTING_GROUPS: SettingGroup[] = [
  {
    title: "Penjualan",
    description:
      "Dipakai saat nota penjualan/POS diterbitkan dan jurnalnya dibuat otomatis.",
    fields: [
      {
        key: "SALES_REVENUE",
        label: "Pendapatan Penjualan",
        hint: "Kredit sebesar DPP saat nota penjualan diterbitkan.",
      },
      {
        key: "VAT_OUTPUT",
        label: "PPN Keluaran",
        hint: "Kredit PPN dari penjualan yang dikenakan pajak.",
      },
      {
        key: "AR",
        label: "Piutang Usaha",
        hint: "Debit sisa tagihan penjualan yang belum dibayar.",
      },
      {
        key: "COGS",
        label: "Harga Pokok Penjualan",
        hint: "Debit total modal (HPP) barang yang terjual.",
      },
    ],
  },
  {
    title: "Pembelian & Persediaan",
    description:
      "Dipakai saat penerimaan barang (GRN) mencatat persediaan dan hutang.",
    fields: [
      {
        key: "INVENTORY",
        label: "Persediaan",
        hint: "Debit nilai persediaan saat penerimaan barang.",
      },
      {
        key: "AP",
        label: "Hutang Usaha",
        hint: "Kredit kewajiban ke supplier atas barang yang diterima.",
      },
      {
        key: "VAT_INPUT",
        label: "PPN Masukan",
        hint: "Debit PPN masukan dari pembelian.",
      },
    ],
  },
  {
    title: "Kas",
    description: "Akun kas bawaan sistem.",
    fields: [
      {
        key: "CASH",
        label: "Kas Default",
        hint: "Akun kas cadangan. Pembayaran penjualan memakai akun pada menu Metode Pembayaran.",
      },
    ],
  },
  {
    title: "Tutup Buku",
    description: "Dipakai saat periode fiskal ditutup.",
    fields: [
      {
        key: "RETAINED_EARNINGS",
        label: "Laba Ditahan",
        hint: "Tujuan pemindahan laba/rugi periode saat tutup buku.",
      },
    ],
  },
];

const ALL_FIELDS = SETTING_GROUPS.flatMap((group) => group.fields);

export default function AccountingSettingsPage() {
  const [httpStatus, setHttpStatus] = useState<number>(200);
  const [canUpdate, setCanUpdate] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [details, setDetails] = useState<AccountingSettingDetail[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [initialValues, setInitialValues] = useState<Record<string, string>>(
    {}
  );

  const [accounts, setAccounts] = useState<ChartOfAccount[]>([]);
  const [accountsLoading, setAccountsLoading] = useState<boolean>(true);
  const [accountsError, setAccountsError] = useState<string | null>(null);

  const loadPermissions = useCallback(() => {
    if (!ensureSession()) return;
    const list = readPermissionList() ?? [];
    setCanUpdate(list.includes("accounting:update"));
  }, []);

  const applyDetails = useCallback((list: AccountingSettingDetail[]) => {
    const next: Record<string, string> = {};
    ALL_FIELDS.forEach((field) => {
      next[field.key] = "";
    });
    list.forEach((item) => {
      next[item.mapping_key] = String(item.account_id);
    });
    setDetails(list);
    setValues(next);
    setInitialValues(next);
  }, []);

  const loadSettings = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const body = await apiGet<AccountingSettingDetail[]>(
        `/api/accounting/settings`,
        { redirectOn403: false }
      );
      applyDetails(body.data ?? []);
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setHttpStatus(403);
        return;
      }
      setLoadError(messageFromError(err, "Gagal memuat pengaturan akun"));
    } finally {
      setLoading(false);
    }
  }, [applyDetails]);

  const loadAccounts = useCallback(async () => {
    setAccountsLoading(true);
    setAccountsError(null);
    try {
      const body = await apiGet<ChartOfAccount[]>(
        `/api/chart-of-accounts?limit=1000`,
        { redirectOn403: false }
      );
      setAccounts(body.data ?? []);
    } catch (err) {
      setAccountsError(
        messageFromError(err, "Gagal memuat daftar akun CoA")
      );
    } finally {
      setAccountsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadPermissions();
    loadSettings();
    loadAccounts();
    window.addEventListener("focus", loadPermissions);
    return () => window.removeEventListener("focus", loadPermissions);
  }, [loadPermissions, loadSettings, loadAccounts]);

  const accountOptions = useMemo(
    () =>
      accounts
        .filter((acc) => !acc.is_header && acc.is_active)
        .map((acc) => ({
          value: String(acc.id),
          label: `${acc.code} · ${acc.name}`,
        })),
    [accounts]
  );

  const detailByKey = useMemo(
    () => new Map(details.map((item) => [item.mapping_key, item])),
    [details]
  );

  const dirty = useMemo(
    () =>
      ALL_FIELDS.some(
        (field) => (values[field.key] ?? "") !== (initialValues[field.key] ?? "")
      ),
    [values, initialValues]
  );

  const resetValues = () => {
    setValues(initialValues);
    toast.info("Perubahan dibatalkan", {
      description: "Nilai dikembalikan ke pengaturan yang tersimpan.",
    });
  };

  const save = async () => {
    const missing = ALL_FIELDS.filter((field) => !values[field.key]);
    if (missing.length > 0) {
      toast.error("Akun Belum Lengkap", {
        description: `Pilih akun untuk: ${missing
          .map((field) => field.label)
          .join(", ")}.`,
      });
      return;
    }

    setSaving(true);
    try {
      const body = await apiPut<AccountingSettingDetail[]>(
        `/api/accounting/settings`,
        {
          settings: ALL_FIELDS.map((field) => ({
            mapping_key: field.key,
            account_id: Number(values[field.key]),
          })),
        },
        { redirectOn403: false }
      );
      applyDetails(body.data ?? []);
      toast.success("Pengaturan Akun Disimpan", {
        description:
          "Jurnal otomatis berikutnya akan memakai akun yang baru.",
      });
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        toast.error("Akses Ditolak", {
          description: "Anda tidak memiliki izin accounting:update.",
        });
        return;
      }
      toast.error("Gagal Menyimpan", { description: messageFromError(err) });
    } finally {
      setSaving(false);
    }
  };

  if (httpStatus === 403) {
    return (
      <ForbiddenView
        reset={() => {
          setHttpStatus(200);
          loadPermissions();
          loadSettings();
        }}
      />
    );
  }

  return (
    <main className="flex w-full flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8">
      <PageHeader
        icon={ListTree}
        title="Pengaturan Akun Otomatis"
        description="Tentukan akun CoA yang dipakai sistem saat menerbitkan jurnal penjualan, penerimaan barang, dan tutup buku."
        actions={
          canUpdate ? (
            <>
              <Button
                variant="outline"
                size="sm"
                className="cursor-pointer"
                onClick={resetValues}
                disabled={!dirty || saving}
              >
                <RefreshCw className="size-3.5" /> Batalkan
              </Button>
              <Button
                size="sm"
                className="cursor-pointer bg-emerald-600 text-white hover:bg-emerald-700"
                onClick={save}
                disabled={!dirty || saving || !!accountsError}
              >
                {saving ? (
                  <RefreshCw className="size-3.5 animate-spin" />
                ) : (
                  <Save className="size-3.5" />
                )}{" "}
                Simpan
              </Button>
            </>
          ) : null
        }
      />

      <div className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-xs text-amber-800">
        Perubahan hanya berlaku untuk jurnal otomatis yang dibuat setelah
        disimpan; jurnal yang sudah terbit tidak diubah. Akun header dan akun
        nonaktif tidak dapat dipakai.
      </div>

      {loadError && (
        <div className="flex flex-col items-start gap-2 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-xs text-rose-700 sm:flex-row sm:items-center sm:justify-between">
          <span>{loadError}</span>
          <Button
            variant="outline"
            size="sm"
            className="cursor-pointer border-rose-200 text-rose-700 hover:bg-rose-100"
            onClick={() => {
              loadSettings();
              loadAccounts();
            }}
          >
            <RefreshCw className="size-3.5" /> Coba Lagi
          </Button>
        </div>
      )}

      {accountsError && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-xs text-rose-700">
          Daftar akun CoA tidak dapat dimuat: {accountsError}
        </div>
      )}

      {loading
        ? Array.from({ length: 3 }).map((_, i) => (
            <div
              key={`setting-skeleton-${i}`}
              className="space-y-3 rounded-xl border bg-white p-4"
            >
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-3 w-64" />
              <Skeleton className="h-9 w-full" />
            </div>
          ))
        : SETTING_GROUPS.map((group) => (
            <section
              key={group.title}
              className="overflow-hidden rounded-xl border bg-white shadow-2xs"
            >
              <header className="border-b bg-slate-50/80 px-5 py-3.5">
                <h2 className="text-sm font-semibold text-foreground">
                  {group.title}
                </h2>
                <p className="mt-0.5 text-xs text-slate-500">
                  {group.description}
                </p>
              </header>

              <div className="divide-y">
                {group.fields.map((field) => {
                  const current = detailByKey.get(field.key);
                  const inactive = current ? !current.account_active : false;
                  const fallbackLabel = current
                    ? `${current.account_code} · ${current.account_name}`
                    : undefined;

                  return (
                    <div
                      key={field.key}
                      className="grid gap-2.5 px-5 py-4 md:grid-cols-[minmax(0,1fr)_320px] md:items-center md:gap-6"
                    >
                      <div className="min-w-0">
                        <p className="flex flex-wrap items-center gap-2 text-sm font-medium text-foreground">
                          {field.label}
                          <Badge
                            variant="neutral"
                            className="font-mono text-[10px]"
                          >
                            {field.key}
                          </Badge>
                          {inactive && <Badge variant="danger">NONAKTIF</Badge>}
                        </p>
                        <p className="mt-0.5 text-xs text-slate-500">
                          {field.hint}
                        </p>
                      </div>
                      <ComboboxSelect
                        options={accountOptions}
                        value={values[field.key] ?? ""}
                        onChange={(val) =>
                          setValues((prev) => ({ ...prev, [field.key]: val }))
                        }
                        placeholder={
                          accountsLoading ? "Memuat akun..." : "-- Pilih akun --"
                        }
                        emptyText="Akun tidak ditemukan..."
                        disabled={
                          !canUpdate || accountsLoading || !!accountsError
                        }
                        fallbackLabel={fallbackLabel}
                        className="h-9 text-sm [&_input]:text-sm"
                        popoverClassName="w-[calc(100vw-2rem)] min-w-0 max-w-[calc(100vw-2rem)] md:w-[380px] md:min-w-[380px] md:max-w-[380px]"
                      />
                    </div>
                  );
                })}
              </div>
            </section>
          ))}

      {!canUpdate && !loading && (
        <p className="text-center text-xs text-slate-400">
          Anda hanya dapat melihat pengaturan ini. Perubahan membutuhkan izin
          accounting:update.
        </p>
      )}

      <p className="text-xs text-slate-400">
        Menyimpan pengaturan akan memperbarui tabel{" "}
        <span className="font-mono">s_accounting_settings</span> dan langsung
        dipakai jurnal otomatis berikutnya.
      </p>
    </main>
  );
}
