"use client";

import { apiDelete, apiGet, apiPost, apiPut } from "@/lib/api-client";
import * as React from "react";
import {
  HistoryIcon,
  PencilIcon,
  PlusIcon,
  SearchIcon,
  Trash2Icon,
  Users2Icon,
} from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface Member {
  id: number;
  code: string;
  name: string;
  phone: string;
  email: string;
  address: string;
  default_term_days: number;
  points: number;
  is_active: boolean;
}

interface PointEntry {
  id: number;
  entry_type: string;
  points: number;
  balance_after: number;
  sale_number?: string;
  note: string;
  created_at: string;
}

const emptyForm = {
  code: "",
  name: "",
  phone: "",
  email: "",
  address: "",
  default_term_days: 0,
  is_active: true,
};

const POINT_TYPE_LABEL: Record<string, string> = {
  EARN: "Perolehan",
  REDEEM: "Tukar Poin",
  EARN_REVERSAL: "Batal Perolehan",
  REDEEM_REVERSAL: "Batal Tukar",
  ADJUST: "Penyesuaian",
};

export function MembersView() {
  const [members, setMembers] = React.useState<Member[]>([]);
  const [search, setSearch] = React.useState("");
  const [loading, setLoading] = React.useState(false);
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<Member | null>(null);
  const [form, setForm] = React.useState(emptyForm);
  const [saving, setSaving] = React.useState(false);

  const [historyMember, setHistoryMember] = React.useState<Member | null>(null);
  const [entries, setEntries] = React.useState<PointEntry[]>([]);
  const [loadingHistory, setLoadingHistory] = React.useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (search) params.set("search", search);
      params.set("limit", "100");
      const body = await apiGet<Member[]>(`/api/members?${params.toString()}`);
      setMembers(body.data ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat member");
    } finally {
      setLoading(false);
    }
  }, [search]);

  React.useEffect(() => {
    const timer = setTimeout(load, 250);
    return () => clearTimeout(timer);
  }, [load]);

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setDialogOpen(true);
  };

  const openEdit = (m: Member) => {
    setEditing(m);
    setForm({
      code: m.code,
      name: m.name,
      phone: m.phone,
      email: m.email,
      address: m.address,
      default_term_days: m.default_term_days,
      is_active: m.is_active,
    });
    setDialogOpen(true);
  };

  const save = async () => {
    if (!form.code.trim() || !form.name.trim()) {
      toast.error("Kode dan nama wajib diisi");
      return;
    }
    setSaving(true);
    try {
      const url = editing ? `/api/members?id=${editing.id}` : `/api/members`;
      await (editing ? apiPut(url, form) : apiPost(url, form));
      toast.success(editing ? "Member diperbarui" : "Member ditambahkan");
      setDialogOpen(false);
      load();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal menyimpan member"
      );
    } finally {
      setSaving(false);
    }
  };

  const remove = async (m: Member) => {
    if (!window.confirm(`Hapus member ${m.name}?`)) return;
    try {
      await apiDelete(`/api/members?id=${m.id}`);
      toast.success("Member dihapus");
      load();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal menghapus member"
      );
    }
  };

  const openHistory = async (m: Member) => {
    setHistoryMember(m);
    setEntries([]);
    setLoadingHistory(true);
    try {
      const body = await apiGet<PointEntry[]>(
        `/api/members/points?id=${m.id}&limit=50`
      );
      setEntries(body.data ?? []);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal memuat riwayat poin"
      );
    } finally {
      setLoadingHistory(false);
    }
  };

  return (
    <main className="flex flex-1 flex-col gap-4 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-col gap-3 border-b pb-4 md:flex-row md:items-center md:justify-between">
        <div className="space-y-1">
          <h1 className="flex items-center gap-2 text-xl font-semibold tracking-tight">
            <Users2Icon className="size-5 text-primary" /> Master Member
          </h1>
          <p className="text-sm text-muted-foreground">
            Data member, saldo poin, dan tempo pembayaran bawaan.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <SearchIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari kode / nama / telepon"
              className="h-9 w-56 pl-9"
            />
          </div>
          <Button className="cursor-pointer" onClick={openCreate}>
            <PlusIcon /> Member
          </Button>
        </div>
      </div>

      {/* Mobile: card list */}
      <div className="space-y-3 md:hidden">
        {loading && members.length === 0 ? (
          <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
            Memuat...
          </div>
        ) : members.length === 0 ? (
          <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
            Belum ada member.
          </div>
        ) : (
          members.map((m) => (
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
                  <dt className="text-muted-foreground">Telepon</dt>
                  <dd className="mt-0.5 truncate font-medium">
                    {m.phone || "-"}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Saldo Poin</dt>
                  <dd className="mt-0.5 font-semibold text-amber-700">
                    {m.points}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Tempo</dt>
                  <dd className="mt-0.5 font-medium">
                    {m.default_term_days} hari
                  </dd>
                </div>
              </dl>

              <div className="mt-3 flex justify-end gap-1 border-t pt-3">
                <Button
                  variant="ghost"
                  size="sm"
                  className="cursor-pointer text-sm"
                  onClick={() => openHistory(m)}
                  title="Riwayat poin"
                >
                  <HistoryIcon /> Riwayat
                </Button>
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
              <th className="px-3 py-2">Telepon</th>
              <th className="px-3 py-2 text-right">Saldo Poin</th>
              <th className="px-3 py-2 text-right">Tempo (hari)</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {loading && members.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-10 text-center text-muted-foreground"
                >
                  Memuat...
                </td>
              </tr>
            ) : members.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-10 text-center text-muted-foreground"
                >
                  Belum ada member.
                </td>
              </tr>
            ) : (
              members.map((m) => (
                <tr key={m.id} className="border-t">
                  <td className="px-3 py-2 font-mono font-medium">{m.code}</td>
                  <td className="px-3 py-2">{m.name}</td>
                  <td className="px-3 py-2">{m.phone || "-"}</td>
                  <td className="px-3 py-2 text-right font-mono font-medium text-amber-700">
                    {m.points}
                  </td>
                  <td className="px-3 py-2 text-right font-mono">
                    {m.default_term_days}
                  </td>
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
                        onClick={() => openHistory(m)}
                        title="Riwayat poin"
                      >
                        <HistoryIcon />
                      </Button>
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
              {editing ? "Ubah Member" : "Tambah Member"}
            </DialogTitle>
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
              <Label>Telepon</Label>
              <Input
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
              />
            </div>
            <div className="grid gap-1">
              <Label>Email</Label>
              <Input
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
              />
            </div>
            <div className="grid gap-1 sm:col-span-2">
              <Label>Alamat</Label>
              <Input
                value={form.address}
                onChange={(e) => setForm({ ...form, address: e.target.value })}
              />
            </div>
            <div className="grid gap-1">
              <Label>Tempo bawaan (hari)</Label>
              <Input
                type="number"
                min={0}
                value={form.default_term_days}
                onChange={(e) =>
                  setForm({
                    ...form,
                    default_term_days: Number(e.target.value),
                  })
                }
              />
            </div>
            <div className="flex items-end gap-2">
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

      <Dialog
        open={!!historyMember}
        onOpenChange={(open) => !open && setHistoryMember(null)}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Riwayat Poin</DialogTitle>
            <DialogDescription>
              {historyMember?.name} · saldo {historyMember?.points} poin
            </DialogDescription>
          </DialogHeader>
          <div className="max-h-80 space-y-1 overflow-auto">
            {loadingHistory ? (
              <p className="py-6 text-center text-sm text-muted-foreground">
                Memuat...
              </p>
            ) : entries.length === 0 ? (
              <p className="py-6 text-center text-sm text-muted-foreground">
                Belum ada transaksi poin.
              </p>
            ) : (
              entries.map((e) => (
                <div
                  key={e.id}
                  className="flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm"
                >
                  <div className="min-w-0">
                    <p className="font-medium">
                      {POINT_TYPE_LABEL[e.entry_type] ?? e.entry_type}
                    </p>
                    <p className="truncate text-sm text-muted-foreground">
                      {new Date(e.created_at).toLocaleString("id-ID")}
                      {e.sale_number ? ` · ${e.sale_number}` : ""}
                      {e.note ? ` · ${e.note}` : ""}
                    </p>
                  </div>
                  <div className="shrink-0 text-right">
                    <p
                      className={
                        e.points >= 0 ? "text-emerald-600" : "text-rose-600"
                      }
                    >
                      {e.points >= 0 ? `+${e.points}` : e.points}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      saldo {e.balance_after}
                    </p>
                  </div>
                </div>
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>
    </main>
  );
}
