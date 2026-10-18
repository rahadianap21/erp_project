"use client"

import { apiDelete, apiGet, apiPost, apiPut } from "@/lib/api-client";
import * as React from "react"
import { CreditCardIcon, PencilIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

interface PaymentMethod {
  id: number
  code: string
  name: string
  channel: string
  account_id?: number | null
  is_cash: boolean
  sort_order: number
  is_active: boolean
}

const CHANNELS = ["CASH", "BANK", "EWALLET", "CREDIT"]

const emptyForm = {
  code: "",
  name: "",
  channel: "CASH",
  account_id: "" as string,
  is_cash: false,
  sort_order: 0,
  is_active: true,
}

export default function PaymentMethodsPage() {

  const [methods, setMethods] = React.useState<PaymentMethod[]>([])
  const [loading, setLoading] = React.useState(false)
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [editing, setEditing] = React.useState<PaymentMethod | null>(null)
  const [form, setForm] = React.useState(emptyForm)
  const [saving, setSaving] = React.useState(false)

  const load = React.useCallback(async () => {
    setLoading(true)
    try {
      const body = await apiGet<PaymentMethod[]>(`/api/payment-methods`)
      setMethods(body.data ?? [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat metode pembayaran")
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    load()
  }, [load])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = (m: PaymentMethod) => {
    setEditing(m)
    setForm({
      code: m.code,
      name: m.name,
      channel: m.channel,
      account_id: m.account_id ? String(m.account_id) : "",
      is_cash: m.is_cash,
      sort_order: m.sort_order,
      is_active: m.is_active,
    })
    setDialogOpen(true)
  }

  const save = async () => {
    if (!form.code.trim() || !form.name.trim()) {
      toast.error("Kode dan nama wajib diisi")
      return
    }
    setSaving(true)
    try {
      const payload = {
        code: form.code,
        name: form.name,
        channel: form.channel,
        account_id: form.account_id ? Number(form.account_id) : null,
        is_cash: form.is_cash,
        sort_order: form.sort_order,
        is_active: form.is_active,
      }
      const url = editing
        ? `/api/payment-methods?id=${editing.id}`
        : `/api/payment-methods`
      await (editing ? apiPut(url, payload) : apiPost(url, payload))
      toast.success(editing ? "Metode diperbarui" : "Metode ditambahkan")
      setDialogOpen(false)
      load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menyimpan metode pembayaran")
    } finally {
      setSaving(false)
    }
  }

  const remove = async (m: PaymentMethod) => {
    if (!window.confirm(`Hapus metode ${m.name}?`)) return
    try {
      await apiDelete(`/api/payment-methods?id=${m.id}`)
      toast.success("Metode dihapus")
      load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus metode")
    }
  }

  return (
    <main className="flex flex-1 flex-col gap-4 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <h1 className="flex items-center gap-2 text-xl font-semibold tracking-tight">
            <CreditCardIcon className="size-5 text-blue-600" /> Metode Pembayaran
          </h1>
          <p className="text-xs text-muted-foreground">
            Kanal pembayaran POS dan akun kas/bank/E-wallet tujuannya.
          </p>
        </div>
        <Button className="cursor-pointer" onClick={openCreate}>
          <PlusIcon /> Metode
        </Button>
      </div>

      <div className="overflow-hidden rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-3 py-2">Kode</th>
              <th className="px-3 py-2">Nama</th>
              <th className="px-3 py-2">Kanal</th>
              <th className="px-3 py-2 text-right">Akun CoA</th>
              <th className="px-3 py-2 text-right">Urutan</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {loading && methods.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-3 py-10 text-center text-muted-foreground">
                  Memuat...
                </td>
              </tr>
            ) : methods.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-3 py-10 text-center text-muted-foreground">
                  Belum ada metode pembayaran.
                </td>
              </tr>
            ) : (
              methods.map((m) => (
                <tr key={m.id} className="border-t">
                  <td className="px-3 py-2 font-medium">{m.code}</td>
                  <td className="px-3 py-2">{m.name}</td>
                  <td className="px-3 py-2">
                    <Badge variant="outline">{m.channel}</Badge>
                  </td>
                  <td className="px-3 py-2 text-right">{m.account_id ?? "-"}</td>
                  <td className="px-3 py-2 text-right">{m.sort_order}</td>
                  <td className="px-3 py-2">
                    {m.is_active ? (
                      <Badge variant="secondary">Aktif</Badge>
                    ) : (
                      <Badge variant="outline">Nonaktif</Badge>
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
            <DialogTitle>{editing ? "Ubah Metode Pembayaran" : "Tambah Metode Pembayaran"}</DialogTitle>
            <DialogDescription>
              Kanal CREDIT tidak memerlukan akun (sisa menjadi piutang).
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 sm:grid-cols-2">
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
                onChange={(e) => setForm({ ...form, channel: e.target.value })}
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
              <Label>ID Akun CoA</Label>
              <Input
                type="number"
                value={form.account_id}
                onChange={(e) => setForm({ ...form, account_id: e.target.value })}
                placeholder="Kosongkan untuk kredit"
              />
            </div>
            <div className="grid gap-1">
              <Label>Urutan</Label>
              <Input
                type="number"
                value={form.sort_order}
                onChange={(e) => setForm({ ...form, sort_order: Number(e.target.value) })}
              />
            </div>
            <div className="flex items-end gap-4">
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.is_cash}
                  onChange={(e) => setForm({ ...form, is_cash: e.target.checked })}
                />
                Kas
              </label>
              <label className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.is_active}
                  onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                />
                Aktif
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
  )
}
