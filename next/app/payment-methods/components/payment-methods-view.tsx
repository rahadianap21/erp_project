"use client";

import { apiDelete, apiGet, apiPost, apiPut } from "@/lib/api-client";
import * as React from "react";
import { CreditCardIcon, PencilIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ComboboxSelect } from "@/components/combobox-select";
import type { ChartOfAccount } from "@/app/accounting/chart-of-accounts/components/chart-of-account-table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface PaymentMethod {
  id: number;
  code: string;
  name: string;
  channel: string;
  account_id?: number | null;
  is_cash: boolean;
  sort_order: number;
  is_active: boolean;
}

const CHANNELS = ["CASH", "BANK", "EWALLET", "CREDIT"];

const emptyForm = {
  code: "",
  name: "",
  channel: "CASH",
  account_id: "" as string,
  is_cash: false,
  sort_order: 0,
  is_active: true,
};

export function PaymentMethodsView() {
  const [methods, setMethods] = React.useState<PaymentMethod[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<PaymentMethod | null>(null);
  const [form, setForm] = React.useState(emptyForm);
  const [saving, setSaving] = React.useState(false);

  // Daftar akun CoA untuk combobox akun kas/bank metode pembayaran.
  const [accounts, setAccounts] = React.useState<ChartOfAccount[]>([]);
  const [accountsLoading, setAccountsLoading] = React.useState(false);
  const [accountsError, setAccountsError] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const body = await apiGet<PaymentMethod[]>(`/api/payment-methods`);
      setMethods(body.data ?? []);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal memuat metode pembayaran"
      );
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    load();
  }, [load]);

  const loadAccounts = React.useCallback(async () => {
    setAccountsLoading(true);
    setAccountsError(null);
    try {
      const body = await apiGet<ChartOfAccount[]>(
        `/api/chart-of-accounts?limit=1000`,
        { redirectOn403: false }
      );
      // Simpan semua akun untuk label; opsi yang bisa dipilih hanya akun
      // detail yang aktif (akun header tidak bisa dipakai posting).
      setAccounts(body.data ?? []);
    } catch (err) {
      setAccountsError(
        err instanceof Error ? err.message : "Gagal memuat daftar akun CoA"
      );
    } finally {
      setAccountsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    loadAccounts();
  }, [loadAccounts]);

  const accountOptions = React.useMemo(
    () =>
      accounts
        .filter((a) => !a.is_header && a.is_active)
        .map((a) => ({
          value: String(a.id),
          label: `${a.code} · ${a.name}`,
        })),
    [accounts]
  );

  const accountById = React.useMemo(
    () => new Map(accounts.map((a) => [a.id, a])),
    [accounts]
  );

  const accountLabel = React.useCallback(
    (id?: number | null) => {
      if (!id) return null;
      const acc = accountById.get(id);
      return acc ? `${acc.code} · ${acc.name}` : `#${id}`;
    },
    [accountById]
  );

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setDialogOpen(true);
  };

  const openEdit = (m: PaymentMethod) => {
    setEditing(m);
    setForm({
      code: m.code,
      name: m.name,
      channel: m.channel,
      account_id: m.account_id ? String(m.account_id) : "",
      is_cash: m.is_cash,
      sort_order: m.sort_order,
      is_active: m.is_active,
    });
    setDialogOpen(true);
  };

  const save = async () => {
    if (!form.code.trim() || !form.name.trim()) {
      toast.error("Kode dan nama wajib diisi");
      return;
    }
    if (form.channel !== "CREDIT" && !form.account_id) {
      toast.error(
        accountsError
          ? "Daftar akun CoA tidak tersedia"
          : "Akun CoA wajib dipilih",
        {
          description:
            accountsError ??
            "Pilih akun kas/bank yang dipakai kanal pembayaran ini.",
        }
      );
      return;
    }
    setSaving(true);
    try {
      const payload = {
        code: form.code,
        name: form.name,
        channel: form.channel,
        account_id:
          form.channel === "CREDIT" || !form.account_id
            ? null
            : Number(form.account_id),
        is_cash: form.is_cash,
        sort_order: form.sort_order,
        is_active: form.is_active,
      };
      const url = editing
        ? `/api/payment-methods?id=${editing.id}`
        : `/api/payment-methods`;
      await (editing ? apiPut(url, payload) : apiPost(url, payload));
      toast.success(editing ? "Metode diperbarui" : "Metode ditambahkan");
      setDialogOpen(false);
      load();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal menyimpan metode pembayaran"
      );
    } finally {
      setSaving(false);
    }
  };

  const remove = async (m: PaymentMethod) => {
    if (!window.confirm(`Hapus metode ${m.name}?`)) return;
    try {
      await apiDelete(`/api/payment-methods?id=${m.id}`);
      toast.success("Metode dihapus");
      load();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal menghapus metode"
      );
    }
  };

  return (
    <main className="flex flex-1 flex-col gap-4 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-col gap-3 border-b pb-4 md:flex-row md:items-center md:justify-between">
        <div className="space-y-1">
          <h1 className="flex items-center gap-2 text-xl font-semibold tracking-tight">
            <CreditCardIcon className="size-5 text-primary" /> Metode
            Pembayaran
          </h1>
          <p className="text-sm text-muted-foreground">
            Kanal pembayaran POS dan akun kas/bank/E-wallet tujuannya.
          </p>
        </div>
        <Button className="cursor-pointer" onClick={openCreate}>
          <PlusIcon /> Metode
        </Button>
      </div>

      {/* Mobile: card list */}
      <div className="space-y-3 md:hidden">
        {loading && methods.length === 0 ? (
          <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
            Memuat...
          </div>
        ) : methods.length === 0 ? (
          <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
            Belum ada metode pembayaran.
          </div>
        ) : (
          methods.map((m) => (
            <article
              key={m.id}
              className="rounded-lg border bg-card p-4 shadow-sm"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h3 className="truncate text-sm font-medium">{m.name}</h3>
                  <p className="mt-0.5 font-mono text-sm text-muted-foreground">
                    {m.code}
                  </p>
                </div>
                {m.is_active ? (
                  <Badge variant="success" className="shrink-0">
                    Active
                  </Badge>
                ) : (
                  <Badge variant="neutral" className="shrink-0">
                    Nonaktif
                  </Badge>
                )}
              </div>

              <dl className="mt-3 grid grid-cols-3 gap-2 rounded-md bg-muted/40 p-2.5 text-sm">
                <div>
                  <dt className="text-muted-foreground">Kanal</dt>
                  <dd className="mt-0.5">
                    <Badge variant="info">{m.channel}</Badge>
                  </dd>
                </div>
                <div className="min-w-0">
                  <dt className="text-muted-foreground">Akun CoA</dt>
                  <dd
                    className="mt-0.5 truncate font-medium"
                    title={accountLabel(m.account_id) ?? undefined}
                  >
                    {accountLabel(m.account_id) ?? "-"}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Urutan</dt>
                  <dd className="mt-0.5 font-medium">{m.sort_order}</dd>
                </div>
              </dl>

              <div className="mt-3 flex justify-end gap-1 border-t pt-3">
                <Button
                  variant="ghost"
                  size="sm"
                  className="cursor-pointer text-sm"
                  onClick={() => openEdit(m)}
                >
                  <PencilIcon /> Edit
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="cursor-pointer text-sm text-destructive"
                  onClick={() => remove(m)}
                >
                  <Trash2Icon /> Hapus
                </Button>
              </div>
            </article>
          ))
        )}
      </div>

      {/* Desktop: table */}
      <div className="hidden overflow-hidden rounded-lg border md:block">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-sm font-medium tracking-wide text-muted-foreground">
            <tr>
              <th className="px-3 py-2">Kode</th>
              <th className="px-3 py-2">Nama</th>
              <th className="px-3 py-2">Kanal</th>
              <th className="px-3 py-2">Akun CoA</th>
              <th className="px-3 py-2 text-right">Urutan</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {loading && methods.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-10 text-center text-muted-foreground"
                >
                  Memuat...
                </td>
              </tr>
            ) : methods.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-10 text-center text-muted-foreground"
                >
                  Belum ada metode pembayaran.
                </td>
              </tr>
            ) : (
              methods.map((m) => (
                <tr key={m.id} className="border-t">
                  <td className="px-3 py-2 font-medium">{m.code}</td>
                  <td className="px-3 py-2">{m.name}</td>
                  <td className="px-3 py-2">
                    <Badge variant="info">{m.channel}</Badge>
                  </td>
                  <td className="max-w-[260px] px-3 py-2">
                    <span
                      className="block truncate"
                      title={accountLabel(m.account_id) ?? undefined}
                    >
                      {accountLabel(m.account_id) ?? (
                        <span className="text-muted-foreground">-</span>
                      )}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-right">{m.sort_order}</td>
                  <td className="px-3 py-2">
                    {m.is_active ? (
                      <Badge variant="success">Active</Badge>
                    ) : (
                      <Badge variant="neutral">Nonaktif</Badge>
                    )}
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="cursor-pointer"
                        onClick={() => openEdit(m)}
                      >
                        <PencilIcon />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="cursor-pointer text-destructive"
                        onClick={() => remove(m)}
                      >
                        <Trash2Icon />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editing ? "Ubah Metode Pembayaran" : "Tambah Metode Pembayaran"}
            </DialogTitle>
            <DialogDescription>
              Kanal CREDIT tidak memerlukan akun (sisa menjadi piutang).
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 md:grid-cols-2">
            <div className="grid gap-1">
              <Label>Kode</Label>
              <Input
                value={form.code}
                onChange={(e) => setForm({ ...form, code: e.target.value })}
                disabled={!!editing}
              />
            </div>
            <div className="grid gap-1">
              <Label>Nama</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </div>
            <div className="grid gap-1">
              <Label>Kanal</Label>
              <select
                value={form.channel}
                onChange={(e) =>
                  setForm({
                    ...form,
                    channel: e.target.value,
                    // Kanal kredit tidak memakai akun kas/bank.
                    account_id:
                      e.target.value === "CREDIT" ? "" : form.account_id,
                  })
                }
                className="h-9 rounded-md border bg-background px-2 text-sm"
              >
                {CHANNELS.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
            </div>
            <div className="grid gap-1">
              <Label>Akun CoA (Kas/Bank)</Label>
              <ComboboxSelect
                options={accountOptions}
                value={form.account_id}
                onChange={(val) => setForm({ ...form, account_id: val })}
                placeholder={
                  accountsLoading ? "Memuat akun..." : "-- Pilih akun CoA --"
                }
                emptyText="Akun tidak ditemukan..."
                disabled={
                  accountsLoading || !!accountsError || form.channel === "CREDIT"
                }
                fallbackLabel={
                  editing?.account_id
                    ? (accountLabel(editing.account_id) ?? undefined)
                    : undefined
                }
                className="h-9 text-sm [&_input]:text-sm"
                popoverClassName="w-[calc(100vw-2rem)] min-w-0 max-w-[calc(100vw-2rem)] md:w-[360px] md:min-w-[360px] md:max-w-[360px]"
              />
              <p
                className={`text-xs ${accountsError ? "text-rose-600" : "text-muted-foreground"}`}
              >
                {accountsError
                  ? accountsError
                  : form.channel === "CREDIT"
                    ? "Kanal CREDIT tidak memerlukan akun; sisa tagihan menjadi piutang."
                    : "Akun yang didebit saat pembayaran diterima."}
              </p>
            </div>
            <div className="grid gap-1">
              <Label>Urutan</Label>
              <Input
                type="number"
                value={form.sort_order}
                onChange={(e) =>
                  setForm({ ...form, sort_order: Number(e.target.value) })
                }
              />
            </div>
            <div className="flex items-end gap-4">
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.is_cash}
                  onChange={(e) =>
                    setForm({ ...form, is_cash: e.target.checked })
                  }
                />
                Kas
              </label>
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.is_active}
                  onChange={(e) =>
                    setForm({ ...form, is_active: e.target.checked })
                  }
                />
                Active
              </label>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              className="cursor-pointer"
              onClick={() => setDialogOpen(false)}
              disabled={saving}
            >
              Batal
            </Button>
            <Button className="cursor-pointer" onClick={save} disabled={saving}>
              {saving ? "Menyimpan..." : "Simpan"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  );
}
