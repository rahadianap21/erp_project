"use client";

import { apiDelete, apiGet, apiPost, apiPut } from "@/lib/api-client";
import JsBarcode from "jsbarcode";
import * as React from "react";
import {
  PencilIcon,
  PlusIcon,
  PrinterIcon,
  SearchIcon,
  TicketPercentIcon,
  Trash2Icon,
} from "lucide-react";
import { toast } from "sonner";
import { formatIDR } from "@/lib/money";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { MoneyInput } from "@/components/ui/money-input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface Voucher {
  id: number;
  code: string;
  name: string;
  description: string;
  discount_type: string;
  discount_value: number;
  min_purchase: number;
  max_discount: number;
  valid_from?: string | null;
  valid_until?: string | null;
  is_active: boolean;
  used_at?: string | null;
  created_at: string;
}

const emptyForm = {
  code: "",
  name: "",
  description: "",
  discount_type: "AMOUNT",
  discount_value: 0,
  min_purchase: 0,
  max_discount: 0,
  valid_from: "",
  valid_until: "",
  is_active: true,
};

const toDateInput = (value?: string | null): string =>
  value ? value.slice(0, 10) : "";

const discountLabel = (v: Voucher): string =>
  v.discount_type === "PERCENT"
    ? `${v.discount_value}%${v.max_discount > 0 ? ` (maks ${formatIDR(v.max_discount)})` : ""}`
    : formatIDR(v.discount_value);

export function VouchersView() {
  const [vouchers, setVouchers] = React.useState<Voucher[]>([]);
  const [search, setSearch] = React.useState("");
  const [loading, setLoading] = React.useState(false);
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<Voucher | null>(null);
  const [form, setForm] = React.useState(emptyForm);
  const [saving, setSaving] = React.useState(false);
  const [printTarget, setPrintTarget] = React.useState<Voucher | null>(null);
  const barcodeRef = React.useRef<SVGSVGElement>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (search) params.set("search", search);
      params.set("limit", "100");
      const body = await apiGet<Voucher[]>(`/api/vouchers?${params.toString()}`);
      setVouchers(body.data ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat voucher");
    } finally {
      setLoading(false);
    }
  }, [search]);

  React.useEffect(() => {
    const timer = setTimeout(load, 250);
    return () => clearTimeout(timer);
  }, [load]);

  React.useEffect(() => {
    if (!printTarget || !barcodeRef.current) return;
    try {
      JsBarcode(barcodeRef.current, printTarget.code, {
        format: "CODE128",
        width: 2,
        height: 80,
        displayValue: true,
        font: "monospace",
        fontSize: 16,
        margin: 8,
      });
    } catch {
      toast.error("Kode voucher tidak dapat dibuat menjadi barcode");
    }
  }, [printTarget]);

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setDialogOpen(true);
  };

  const openEdit = (v: Voucher) => {
    setEditing(v);
    setForm({
      code: v.code,
      name: v.name,
      description: v.description,
      discount_type: v.discount_type,
      discount_value: v.discount_value,
      min_purchase: v.min_purchase,
      max_discount: v.max_discount,
      valid_from: toDateInput(v.valid_from),
      valid_until: toDateInput(v.valid_until),
      is_active: v.is_active,
    });
    setDialogOpen(true);
  };

  const save = async () => {
    if (!form.code.trim() || !form.name.trim()) {
      toast.error("Kode dan nama wajib diisi");
      return;
    }
    if (form.discount_value <= 0) {
      toast.error("Nilai diskon harus lebih dari 0");
      return;
    }
    setSaving(true);
    try {
      const payload = {
        ...form,
        code: form.code.trim(),
        name: form.name.trim(),
        description: form.description.trim(),
        max_discount:
          form.discount_type === "PERCENT" ? form.max_discount : 0,
        valid_from: form.valid_from || null,
        valid_until: form.valid_until || null,
      };
      const url = editing ? `/api/vouchers?id=${editing.id}` : `/api/vouchers`;
      await (editing ? apiPut(url, payload) : apiPost(url, payload));
      toast.success(editing ? "Voucher diperbarui" : "Voucher ditambahkan");
      setDialogOpen(false);
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menyimpan voucher");
    } finally {
      setSaving(false);
    }
  };

  const remove = async (v: Voucher) => {
    if (!window.confirm(`Hapus voucher ${v.name} (${v.code})?`)) return;
    try {
      await apiDelete(`/api/vouchers?id=${v.id}`);
      toast.success("Voucher dihapus");
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus voucher");
    }
  };

  const printBarcode = () => {
    if (!printTarget || !barcodeRef.current) return;
    const svg = barcodeRef.current.outerHTML;
    const win = window.open("", "_blank", "width=420,height=320");
    if (!win) return;
    win.document.write(`<!doctype html><html><head><title>${printTarget.code}</title>
      <style>
        body{font-family:system-ui,sans-serif;text-align:center;margin:16px}
        h2{margin:4px 0;font-size:16px}
        p{margin:2px 0;font-size:12px;color:#333}
        svg{max-width:100%}
      </style></head><body>
      <h2>${printTarget.name}</h2>
      <p>${discountLabel(printTarget)}</p>
      ${svg}
      <p>${printTarget.code}</p>
      </body></html>`);
    win.document.close();
    win.focus();
    win.print();
  };

  return (
    <main className="flex flex-1 flex-col gap-4 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-col gap-3 border-b pb-4 md:flex-row md:items-center md:justify-between">
        <div className="space-y-1">
          <h1 className="flex items-center gap-2 text-xl font-semibold tracking-tight">
            <TicketPercentIcon className="size-5 text-primary" /> Master Voucher
          </h1>
          <p className="text-sm text-muted-foreground">
            Kupon sekali pakai dengan barcode Code128 untuk diskon di POS.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <SearchIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari kode / nama"
              className="h-9 w-56 pl-9"
            />
          </div>
          <Button className="cursor-pointer" onClick={openCreate}>
            <PlusIcon /> Voucher
          </Button>
        </div>
      </div>

      {/* Mobile: card list */}
      <div className="space-y-3 md:hidden">
        {loading && vouchers.length === 0 ? (
          <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
            Memuat...
          </div>
        ) : vouchers.length === 0 ? (
          <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
            Belum ada voucher.
          </div>
        ) : (
          vouchers.map((v) => (
            <article
              key={v.id}
              className="rounded-lg border bg-card p-4 shadow-sm"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h3 className="truncate font-mono text-sm font-medium">
                    {v.code}
                  </h3>
                  <p className="mt-0.5 truncate text-sm text-muted-foreground">
                    {v.name}
                  </p>
                </div>
                {v.used_at ? (
                  <Badge variant="neutral" className="shrink-0">
                    Terpakai
                  </Badge>
                ) : v.is_active ? (
                  <Badge variant="success" className="shrink-0">
                    Active
                  </Badge>
                ) : (
                  <Badge variant="neutral" className="shrink-0">
                    Nonaktif
                  </Badge>
                )}
              </div>

              <dl className="mt-3 grid grid-cols-2 gap-2 rounded-md bg-muted/40 p-2.5 text-sm">
                <div>
                  <dt className="text-muted-foreground">Diskon</dt>
                  <dd className="mt-0.5 font-semibold">{discountLabel(v)}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Min. Belanja</dt>
                  <dd className="mt-0.5 font-medium">
                    {formatIDR(v.min_purchase)}
                  </dd>
                </div>
                <div className="col-span-2">
                  <dt className="text-muted-foreground">Masa Berlaku</dt>
                  <dd className="mt-0.5 text-muted-foreground">
                    {v.valid_from || v.valid_until
                      ? `${toDateInput(v.valid_from) || "..."} s.d. ${toDateInput(v.valid_until) || "..."
                      }`
                      : "Tanpa batas"}
                  </dd>
                </div>
              </dl>

              <div className="mt-3 flex justify-end gap-1 border-t pt-3">
                <Button
                  variant="ghost"
                  size="sm"
                  className="cursor-pointer text-sm"
                  onClick={() => setPrintTarget(v)}
                >
                  <PrinterIcon /> Cetak
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="cursor-pointer text-sm"
                  onClick={() => openEdit(v)}
                >
                  <PencilIcon /> Edit
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="cursor-pointer text-sm text-destructive"
                  onClick={() => remove(v)}
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
              <th className="px-3 py-2">Diskon</th>
              <th className="px-3 py-2 text-right">Min. Belanja</th>
              <th className="px-3 py-2">Masa Berlaku</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {loading && vouchers.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-10 text-center text-muted-foreground"
                >
                  Memuat...
                </td>
              </tr>
            ) : vouchers.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-10 text-center text-muted-foreground"
                >
                  Belum ada voucher.
                </td>
              </tr>
            ) : (
              vouchers.map((v) => (
                <tr key={v.id} className="border-t">
                  <td className="px-3 py-2 font-mono font-medium">{v.code}</td>
                  <td className="px-3 py-2">{v.name}</td>
                  <td className="px-3 py-2">{discountLabel(v)}</td>
                  <td className="px-3 py-2 text-right font-mono font-medium">
                    {formatIDR(v.min_purchase)}
                  </td>
                  <td className="px-3 py-2 text-sm text-muted-foreground">
                    {v.valid_from || v.valid_until
                      ? `${toDateInput(v.valid_from) || "..."} s.d. ${toDateInput(v.valid_until) || "..."
                      }`
                      : "Tanpa batas"}
                  </td>
                  <td className="px-3 py-2">
                    {v.used_at ? (
                      <Badge variant="neutral">Terpakai</Badge>
                    ) : v.is_active ? (
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
                        onClick={() => setPrintTarget(v)}
                        title="Cetak barcode"
                      >
                        <PrinterIcon />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="cursor-pointer"
                        onClick={() => openEdit(v)}
                      >
                        <PencilIcon />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="cursor-pointer text-destructive"
                        onClick={() => remove(v)}
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
            <DialogTitle>{editing ? "Ubah Voucher" : "Tambah Voucher"}</DialogTitle>
            <DialogDescription>
              Kode voucher menjadi isi barcode dan tidak dapat diubah setelah
              disimpan.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 md:grid-cols-2">
            <div className="grid gap-1">
              <Label>Kode</Label>
              <Input
                value={form.code}
                onChange={(e) => setForm({ ...form, code: e.target.value })}
                disabled={!!editing}
                placeholder="PROMO10"
                className="font-mono"
              />
            </div>
            <div className="grid gap-1">
              <Label>Nama</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="Promo Akhir Tahun"
              />
            </div>
            <div className="grid gap-1">
              <Label>Jenis Diskon</Label>
              <select
                value={form.discount_type}
                onChange={(e) =>
                  setForm({ ...form, discount_type: e.target.value })
                }
                className="h-9 rounded-md border bg-background px-2 text-sm"
              >
                <option value="AMOUNT">Nominal (Rp)</option>
                <option value="PERCENT">Persen (%)</option>
              </select>
            </div>
            <div className="grid gap-1">
              <Label>
                Nilai Diskon{" "}
                {form.discount_type === "PERCENT" ? "(%)" : "(Rp)"}
              </Label>
              {form.discount_type === "AMOUNT" ? (
                <MoneyInput
                  value={form.discount_value}
                  onValueChange={(value) =>
                    setForm({ ...form, discount_value: value })
                  }
                  className="h-9"
                />
              ) : (
                <Input
                  type="number"
                  min={0}
                  max={100}
                  value={form.discount_value}
                  onChange={(e) =>
                    setForm({ ...form, discount_value: Number(e.target.value) })
                  }
                />
              )}
            </div>
            <div className="grid gap-1">
              <Label>Minimal Belanja</Label>
              <MoneyInput
                value={form.min_purchase}
                onValueChange={(value) =>
                  setForm({ ...form, min_purchase: value })
                }
                className="h-9"
              />
            </div>
            <div className="grid gap-1">
              <Label>Maksimal Diskon</Label>
              <MoneyInput
                value={form.max_discount}
                onValueChange={(value) =>
                  setForm({ ...form, max_discount: value })
                }
                disabled={form.discount_type !== "PERCENT"}
                className="h-9"
                placeholder="0 = tanpa batas"
              />
            </div>
            <div className="grid gap-1">
              <Label>Berlaku Dari</Label>
              <Input
                type="date"
                value={form.valid_from}
                onChange={(e) =>
                  setForm({ ...form, valid_from: e.target.value })
                }
              />
            </div>
            <div className="grid gap-1">
              <Label>Berlaku Sampai</Label>
              <Input
                type="date"
                value={form.valid_until}
                onChange={(e) =>
                  setForm({ ...form, valid_until: e.target.value })
                }
              />
            </div>
            <div className="grid gap-1 sm:col-span-2">
              <Label>Keterangan</Label>
              <Input
                value={form.description}
                onChange={(e) =>
                  setForm({ ...form, description: e.target.value })
                }
                placeholder="Opsional"
              />
            </div>
            <div className="flex items-end">
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
        open={!!printTarget}
        onOpenChange={(open) => !open && setPrintTarget(null)}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Cetak Barcode Voucher</DialogTitle>
            <DialogDescription>
              {printTarget?.name} · {printTarget ? discountLabel(printTarget) : ""}
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col items-center gap-2 rounded-lg border bg-white p-4">
            <svg ref={barcodeRef} />
            <p className="font-mono text-sm font-semibold">{printTarget?.code}</p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              className="cursor-pointer"
              onClick={() => setPrintTarget(null)}
            >
              Tutup
            </Button>
            <Button className="cursor-pointer" onClick={printBarcode}>
              <PrinterIcon /> Cetak
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  );
}
