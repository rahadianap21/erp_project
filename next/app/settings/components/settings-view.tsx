"use client"

import { apiGet, apiPut } from "@/lib/api-client";
import { readPermissionList } from "@/lib/permissions";
import * as React from "react"
import { Loader2Icon, PercentIcon, SaveIcon, ReceiptTextIcon } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"

interface SettingsView {
  ppn_rate: number
  ppn_price_mode: string
}

const MODE_OPTIONS = [
  {
    value: "INCLUSIVE",
    title: "Harga Sudah Termasuk PPN",
    desc: "Harga jual di master produk dianggap sudah mengandung PPN. Total nota tidak berubah; DPP & PPN hanya dipisahkan pada jurnal.",
  },
  {
    value: "EXCLUSIVE",
    title: "Harga Belum Termasuk PPN",
    desc: "PPN ditambahkan di atas harga jual. Total nota akan meningkat sebesar PPN dari baris yang dikenai pajak.",
  },
]

export function SettingsPageView() {

  const canUpdate = React.useMemo(
    () => (readPermissionList() ?? []).includes("settings:update"),
    [],
  )

  const [loading, setLoading] = React.useState(true)
  const [saving, setSaving] = React.useState(false)
  const [ratePercent, setRatePercent] = React.useState<number>(12)
  const [mode, setMode] = React.useState<string>("INCLUSIVE")

  const load = React.useCallback(async () => {
    setLoading(true)
    try {
      const body = await apiGet<SettingsView>(`/api/settings`)
      const view: SettingsView = body.data ?? ({} as SettingsView)
      setRatePercent(Math.round((view.ppn_rate ?? 0.12) * 10000) / 100)
      setMode(view.ppn_price_mode || "INCLUSIVE")
    } catch (err: any) {
      toast.error("Gagal Memuat Pengaturan", { description: err.message })
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    load()
  }, [load])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (ratePercent < 0 || ratePercent > 100) {
      toast.error("Tarif PPN Tidak Valid", {
        description: "Tarif PPN harus berada di antara 0% dan 100%.",
      })
      return
    }
    setSaving(true)
    try {
      const body = await apiPut<SettingsView>(`/api/settings`, {
        ppn_rate: ratePercent / 100,
        ppn_price_mode: mode,
      })
      const view: SettingsView = body.data ?? ({} as SettingsView)
      setRatePercent(Math.round((view.ppn_rate ?? ratePercent / 100) * 10000) / 100)
      setMode(view.ppn_price_mode || mode)
      toast.success("Pengaturan Tersimpan", {
        description: "Perubahan tarif & mode PPN langsung berlaku untuk transaksi POS berikutnya.",
      })
    } catch (err: any) {
      toast.error("Gagal Menyimpan", { description: err.message })
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-4 md:p-6">
      <div>
        <h1 className="text-lg font-bold text-slate-900">Pengaturan Pajak (PPN)</h1>
        <p className="text-sm text-muted-foreground">
          Atur tarif PPN global dan perlakuan harga jual terhadap pajak untuk transaksi POS.
        </p>
      </div>

      {loading ? (
        <div className="max-w-2xl space-y-4 rounded-xl border bg-white p-6 shadow-xs">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-24 w-full" />
        </div>
      ) : (
        <form
          onSubmit={handleSubmit}
          className="max-w-2xl space-y-6 rounded-xl border bg-white p-6 shadow-xs"
        >
          <div className="space-y-2">
            <Label className="flex items-center gap-1.5 text-sm font-semibold text-slate-700">
              <PercentIcon className="size-4 text-blue-600" /> Tarif PPN (%)
            </Label>
            <div className="relative w-44">
              <Input
                type="number"
                min={0}
                max={100}
                step={0.01}
                value={ratePercent}
                disabled={!canUpdate || saving}
                onChange={(e) => setRatePercent(Number(e.target.value))}
                className="pr-8"
              />
              <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-muted-foreground">
                %
              </span>
            </div>
            <p className="text-xs text-muted-foreground">
              Contoh: 12 untuk tarif 12%. Nilai ini dipakai saat memisahkan DPP &amp; PPN Keluaran
              pada nota POS.
            </p>
          </div>

          <div className="space-y-2">
            <Label className="flex items-center gap-1.5 text-sm font-semibold text-slate-700">
              <ReceiptTextIcon className="size-4 text-blue-600" /> Perlakuan Harga Jual
            </Label>
            <div className="grid gap-3">
              {MODE_OPTIONS.map((opt) => {
                const active = mode === opt.value
                return (
                  <label
                    key={opt.value}
                    className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${
                      active ? "border-blue-500 bg-blue-50/50" : "border-slate-200 hover:bg-slate-50"
                    } ${!canUpdate ? "cursor-not-allowed opacity-70" : ""}`}
                  >
                    <input
                      type="radio"
                      name="ppn_price_mode"
                      value={opt.value}
                      checked={active}
                      disabled={!canUpdate || saving}
                      onChange={() => setMode(opt.value)}
                      className="mt-1 size-4 accent-blue-600"
                    />
                    <div className="space-y-0.5">
                      <p className="text-sm font-semibold text-slate-800">{opt.title}</p>
                      <p className="text-xs text-muted-foreground">{opt.desc}</p>
                    </div>
                  </label>
                )
              })}
            </div>
          </div>

          <div className="flex items-center justify-end gap-2 border-t pt-4">
            {!canUpdate && (
              <span className="mr-auto text-xs text-amber-600">
                Anda tidak memiliki izin untuk mengubah pengaturan ini.
              </span>
            )}
            <Button type="submit" size="sm" disabled={!canUpdate || saving}>
              {saving ? (
                <span className="flex items-center gap-1.5">
                  <Loader2Icon className="size-3.5 animate-spin" /> Menyimpan...
                </span>
              ) : (
                <span className="flex items-center gap-1.5">
                  <SaveIcon className="size-3.5" /> Simpan Pengaturan
                </span>
              )}
            </Button>
          </div>
        </form>
      )}
    </div>
  )
}
