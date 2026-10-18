"use client"

import * as React from "react"
import Cookies from "js-cookie"
import {
  MinusIcon,
  PlusIcon,
  ScanLineIcon,
  SearchIcon,
  ShoppingCartIcon,
  Trash2Icon,
  UserIcon,
  XIcon,
} from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { PaymentDialog } from "./payment-dialog"
import { ReceiptDialog } from "./receipt-dialog"
import {
  CartLine,
  Member,
  POSBootstrap,
  POSProduct,
  PaymentDraft,
  Sale,
  formatRupiah,
  round2,
} from "./types"

const DISCOUNT_OPTIONS = [
  { value: "NONE", label: "Tanpa Diskon" },
  { value: "PERCENT", label: "Persen (%)" },
  { value: "AMOUNT", label: "Nominal (Rp)" },
]

interface Totals {
  subtotal: number
  itemDiscount: number
  invoiceDiscount: number
  dpp: number
  tax: number
  total: number
  redeemAmount: number
  usedPoints: number
  maxRedeemPoints: number
  pointsEarned: number
}

const POINT_VALUE = 100
const POINT_EARN_PER = 100000
const REDEEM_MAX_RATIO = 0.5

function calculateTotals(
  cart: CartLine[],
  taxMode: string,
  discountType: string,
  discountValue: number,
  taxRate: number,
  redeemPoints = 0,
  availablePoints = 0
): Totals {
  const lineNet: number[] = []
  let subtotal = 0
  let itemDiscount = 0
  for (const line of cart) {
    const gross = round2(line.qty * line.unitPrice)
    const disc = lineDiscount(line.discountType, line.discountValue, gross)
    subtotal += gross
    itemDiscount += disc
    lineNet.push(round2(gross - disc))
  }
  subtotal = round2(subtotal)
  itemDiscount = round2(itemDiscount)

  const netBefore = round2(subtotal - itemDiscount)
  const invoiceDiscount = lineDiscount(discountType, discountValue, netBefore)
  const invAlloc = allocateProportional(invoiceDiscount, lineNet)
  const postInvNet = lineNet.map((n, i) => round2(n - invAlloc[i]))

  // Pajak dihitung PER BARIS: hanya produk ber-flag is_taxable yang dikenai PPN.
  const lineTax = (net: number, taxable: boolean) => {
    if (!taxable) return { dpp: round2(net), tax: 0 }
    if (taxMode === "EXCLUSIVE") {
      const dpp = round2(net)
      return { dpp, tax: round2(dpp * taxRate) }
    }
    const dpp = round2(net / (1 + taxRate))
    return { dpp, tax: round2(net - dpp) }
  }

  // Batas tukar poin: tidak melebihi 50% total nota DAN tidak melebihi saldo member.
  let preTotal = 0
  for (let i = 0; i < cart.length; i++) {
    const t = lineTax(postInvNet[i], cart[i].product.is_taxable)
    preTotal += t.dpp + t.tax
  }
  preTotal = round2(preTotal)
  const notaCapPoints = Math.floor((preTotal * REDEEM_MAX_RATIO) / POINT_VALUE)
  const cappedAvailable = Math.floor(Math.max(availablePoints, 0))
  const maxRedeemPoints = Math.min(notaCapPoints, cappedAvailable)
  const requestedPoints = Math.max(0, Math.floor(redeemPoints))
  const usedPoints = Math.min(requestedPoints, maxRedeemPoints)
  const redeemAmount = round2(usedPoints * POINT_VALUE)
  const redeemAlloc = allocateProportional(redeemAmount, postInvNet)

  let dpp = 0
  let tax = 0
  for (let i = 0; i < cart.length; i++) {
    const finalNet = round2(postInvNet[i] - redeemAlloc[i])
    const t = lineTax(finalNet, cart[i].product.is_taxable)
    dpp += t.dpp
    tax += t.tax
  }
  dpp = round2(dpp)
  tax = round2(tax)
  const total = round2(dpp + tax)
  return {
    subtotal,
    itemDiscount,
    invoiceDiscount,
    dpp,
    tax,
    total,
    redeemAmount,
    usedPoints,
    maxRedeemPoints,
    pointsEarned: Math.floor(total / POINT_EARN_PER),
  }
}

// allocateProportional membagi total ke tiap baris sebanding bobot dengan
// koreksi pembulatan largest-remainder, agar jumlah alokasi tepat = total.
function allocateProportional(total: number, weights: number[]): number[] {
  const out = weights.map(() => 0)
  if (!weights.length) return out
  const totalR = round2(total)
  if (totalR === 0) return out
  let sumW = 0
  for (const w of weights) if (w > 0) sumW += w
  if (sumW <= 0) return out

  const fracs: { idx: number; rem: number }[] = []
  let allocated = 0
  for (let i = 0; i < weights.length; i++) {
    const w = weights[i]
    if (w <= 0) continue
    const exact = (totalR * w) / sumW
    const floorCents = Math.floor(exact * 100 + 1e-9) / 100
    out[i] = floorCents
    allocated = round2(allocated + floorCents)
    fracs.push({ idx: i, rem: exact - floorCents })
  }
  let remaining = Math.round((totalR - allocated) * 100)
  if (remaining > fracs.length) remaining = fracs.length
  for (let k = 0; k < remaining && fracs.length > 0; k++) {
    let best = 0
    for (let j = 1; j < fracs.length; j++) {
      if (fracs[j].rem > fracs[best].rem) best = j
    }
    out[fracs[best].idx] = round2(out[fracs[best].idx] + 0.01)
    fracs[best].rem -= 0.01
  }
  return out
}

function lineDiscount(type: string, value: number, base: number): number {
  if (type === "PERCENT") {
    const pct = Math.min(Math.max(value, 0), 100)
    return round2((base * pct) / 100)
  }
  if (type === "AMOUNT") {
    return round2(Math.min(Math.max(value, 0), base))
  }
  return 0
}

export function PosTerminal() {
  const authToken = Cookies.get("session_active")

  const [bootstrap, setBootstrap] = React.useState<POSBootstrap | null>(null)
  const [products, setProducts] = React.useState<POSProduct[]>([])
  const [search, setSearch] = React.useState("")
  const [loadingProducts, setLoadingProducts] = React.useState(false)

  const [cart, setCart] = React.useState<CartLine[]>([])
  const [member, setMember] = React.useState<Member | null>(null)
  const [memberQuery, setMemberQuery] = React.useState("")
  const [memberResults, setMemberResults] = React.useState<Member[]>([])
  const [showMemberResults, setShowMemberResults] = React.useState(false)
  const [redeemPoints, setRedeemPoints] = React.useState(0)

  const [discountType, setDiscountType] = React.useState("NONE")
  const [discountValue, setDiscountValue] = React.useState(0)

  const [paymentOpen, setPaymentOpen] = React.useState(false)
  const [submitting, setSubmitting] = React.useState(false)
  const [receipt, setReceipt] = React.useState<Sale | null>(null)

  const searchRef = React.useRef<HTMLInputElement>(null)

  const showProducts = search.trim().length > 0

  const headers = React.useMemo(
    () => ({
      "Content-Type": "application/json",
      Authorization: `Bearer ${authToken}`,
    }),
    [authToken]
  )

  const loadBootstrap = React.useCallback(async () => {
    try {
      const res = await fetch(`/api/pos/bootstrap`, { headers })
      const body = await res.json()
      if (!res.ok) throw new Error(body.message ?? "Gagal memuat data kasir")
      setBootstrap(body.data)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat data kasir")
    }
  }, [headers])

  const loadProducts = React.useCallback(async () => {
    setLoadingProducts(true)
    try {
      const params = new URLSearchParams()
      if (search) params.set("search", search)
      params.set("limit", "120")
      const res = await fetch(`/api/pos/products?${params.toString()}`, { headers })
      const body = await res.json()
      if (!res.ok) throw new Error(body.message ?? "Gagal memuat produk")
      setProducts(body.data ?? [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat produk")
    } finally {
      setLoadingProducts(false)
    }
  }, [headers, search])

  React.useEffect(() => {
    loadBootstrap()
  }, [loadBootstrap])

  React.useEffect(() => {
    if (!showProducts) {
      setProducts([])
      return
    }
    const timer = setTimeout(() => {
      loadProducts()
    }, 250)
    return () => clearTimeout(timer)
  }, [showProducts, loadProducts])

  React.useEffect(() => {
    searchRef.current?.focus()
  }, [])

  React.useEffect(() => {
    if (!memberQuery.trim()) {
      setMemberResults([])
      return
    }
    const timer = setTimeout(async () => {
      try {
        const res = await fetch(
          `/api/pos/members?search=${encodeURIComponent(memberQuery)}&limit=10`,
          { headers }
        )
        const body = await res.json()
        if (res.ok) {
          setMemberResults(body.data ?? [])
          setShowMemberResults(true)
        }
      } catch {
        /* abaikan pencarian member */
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [memberQuery, headers])

  const taxRate = bootstrap?.tax_rate ?? 0.12
  const taxMode = bootstrap?.tax_mode ?? "INCLUSIVE"
  const totals = React.useMemo(
    () =>
      calculateTotals(
        cart,
        taxMode,
        discountType,
        discountValue,
        taxRate,
        redeemPoints,
        member?.points ?? 0
      ),
    [cart, taxMode, discountType, discountValue, taxRate, redeemPoints, member]
  )

  // Tukar poin tidak sah bila jumlah yang diminta melebihi batas (saldo/50% nota).
  const redeemInvalid = member !== null && Math.floor(redeemPoints) > totals.maxRedeemPoints

  const addToCart = (product: POSProduct) => {
    if (product.current_stock <= 0) {
      toast.error("Stok barang habis")
      return
    }
    setCart((prev) => {
      const existing = prev.find((l) => l.product.id === product.id)
      if (existing) {
        if (existing.qty >= product.current_stock) {
          toast.error(`Stok ${product.name} tersisa ${product.current_stock}`)
          return prev
        }
        return prev.map((l) =>
          l.product.id === product.id ? { ...l, qty: l.qty + 1 } : l
        )
      }
      return [
        ...prev,
        {
          product,
          qty: 1,
          unitPrice: product.selling_price,
          discountType: product.discount_type || "NONE",
          discountValue: product.discount_value || 0,
        },
      ]
    })
  }

  const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") return
    e.preventDefault()
    if (products.length > 0) {
      addToCart(products[0])
      setSearch("")
    }
  }

  const changeQty = (productID: number, delta: number) => {
    setCart((prev) =>
      prev
        .map((l) => {
          if (l.product.id !== productID) return l
          const qty = l.qty + delta
          return { ...l, qty }
        })
        .filter((l) => l.qty > 0)
    )
  }

  const clearCart = () => {
    setCart([])
    setDiscountType("NONE")
    setDiscountValue(0)
    setMember(null)
    setRedeemPoints(0)
    setMemberQuery("")
    setMemberResults([])
  }

  const buildItems = () =>
    cart.map((l) => ({
      product_id: l.product.id,
      qty: l.qty,
      unit_price: l.unitPrice,
      discount_type: l.discountType,
      discount_value: l.discountValue,
    }))

  const submitSale = async (payments: PaymentDraft[], notes: string) => {
    if (redeemInvalid) {
      toast.error("Tukar poin tidak valid — periksa saldo poin member")
      return
    }
    if (!payments.length) {
      toast.error("Pilih dan masukkan pembayaran terlebih dahulu")
      return
    }
    setSubmitting(true)
    try {
      const payload = {
        member_id: member?.id ?? 0,
        redeem_points: totals.usedPoints,
        discount_type: discountType,
        discount_value: discountValue,
        due_date: null,
        notes,
        items: buildItems(),
        payments: payments.slice(0, 1).map((p) => ({
          payment_method_id: p.paymentMethodId,
          amount: p.amount,
          reference_number: p.referenceNumber,
        })),
      }
      const res = await fetch(`/api/sales`, {
        method: "POST",
        headers,
        body: JSON.stringify(payload),
      })
      const body = await res.json()
      if (!res.ok) throw new Error(body.message ?? "Gagal menyimpan penjualan")
      setPaymentOpen(false)
      setReceipt(body.data)
      clearCart()
      setSearch("")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menyimpan penjualan")
    } finally {
      setSubmitting(false)
    }
  }

  const selectMember = (m: Member) => {
    setMember(m)
    setMemberQuery("")
    setMemberResults([])
    setShowMemberResults(false)
    setRedeemPoints(0)
  }

  return (
    <div className="mx-auto grid h-[calc(100svh-3.5rem)] w-full max-w-[2560px] grid-cols-1 lg:grid-cols-[minmax(0,1fr)_370px] xl:grid-cols-[minmax(0,1fr)_400px] 2xl:grid-cols-[minmax(0,1fr)_420px] 3xl:grid-cols-[minmax(0,1fr)_500px] 4xl:h-[calc(100svh-4rem)] 4xl:grid-cols-[minmax(0,1fr)_600px]">
      <section className="flex min-h-0 flex-col border-r">
        <div className="space-y-3 border-b p-4 3xl:p-5 4xl:p-6">
          <div className="relative">
            <SearchIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground 4xl:size-5" />
            <Input
              ref={searchRef}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={handleSearchKeyDown}
              placeholder="Cari nama / SKU / scan barcode lalu Enter"
              className="h-10 pl-9 4xl:h-12 4xl:pl-10 4xl:text-base"
              autoFocus
            />
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4 3xl:p-5 4xl:p-6">
          {!showProducts ? (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-muted-foreground">
              <ScanLineIcon className="size-10 opacity-40 4xl:size-14" />
              <p className="text-sm 4xl:text-base">
                Scan barcode atau ketik nama produk untuk mulai.
              </p>
            </div>
          ) : loadingProducts && products.length === 0 ? (
            <p className="py-10 text-center text-sm text-muted-foreground">Mencari produk...</p>
          ) : products.length === 0 ? (
            <p className="py-10 text-center text-sm text-muted-foreground">
              Produk tidak ditemukan.
            </p>
          ) : (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 3xl:gap-4 4xl:gap-5">
              {products.map((p) => {
                const habis = p.current_stock <= 0
                const stokTipis = !habis && p.current_stock <= 5
                const adaDiskon = p.discount_type !== "NONE" && p.discount_value > 0
                return (
                  <button
                    key={p.id}
                    type="button"
                    disabled={habis}
                    onClick={() => addToCart(p)}
                    className="group relative flex min-h-40 cursor-pointer flex-col justify-between gap-3 overflow-hidden rounded-2xl border bg-card p-3.5 text-left shadow-xs transition-all duration-150 hover:-translate-y-0.5 hover:border-primary hover:shadow-md active:translate-y-0 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 focus-visible:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-55 disabled:hover:translate-y-0 disabled:hover:border-border disabled:hover:shadow-xs 3xl:min-h-48 3xl:p-4 4xl:min-h-56 4xl:gap-4 4xl:p-5"
                  >
                    {adaDiskon && (
                      <span className="absolute right-0 top-0 rounded-bl-xl bg-amber-500 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wide text-white shadow-sm 4xl:text-xs">
                        {p.discount_type === "PERCENT"
                          ? `-${p.discount_value}%`
                          : `-${formatRupiah(p.discount_value)}`}
                      </span>
                    )}

                    <div className={`min-w-0 space-y-1 ${adaDiskon ? "pr-12" : ""}`}>
                      <p className="line-clamp-2 text-sm font-semibold leading-snug text-slate-900 3xl:text-base 4xl:text-lg">
                        {p.name}
                      </p>
                      <p className="truncate font-mono text-[10px] text-muted-foreground 3xl:text-xs 4xl:text-sm">
                        {p.sku}
                        {p.category_name ? ` · ${p.category_name}` : ""}
                      </p>
                    </div>

                    <div className="flex items-end justify-between gap-2 border-t border-dashed pt-2.5">
                      <span className="text-lg font-bold leading-none tracking-tight text-primary 3xl:text-xl 4xl:text-2xl">
                        {formatRupiah(p.selling_price)}
                      </span>
                      <span
                        className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold 3xl:text-xs 4xl:text-sm ${habis
                            ? "bg-destructive/10 text-destructive"
                            : stokTipis
                              ? "bg-amber-100 text-amber-700"
                              : "bg-muted text-muted-foreground"
                          }`}
                      >
                        {habis
                          ? "Habis"
                          : stokTipis
                            ? `Sisa ${p.current_stock}`
                            : `Stok ${p.current_stock}`}
                      </span>
                    </div>
                  </button>
                )
              })}
            </div>
          )}
        </div>
      </section>

      <aside className="flex min-h-0 flex-col bg-background">
        <div className="space-y-3 border-b p-4 3xl:p-5 4xl:p-6">
          <div className="relative">
            <UserIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground 4xl:size-5" />
            {member ? (
              <div className="flex items-center justify-between gap-2 rounded-lg border bg-muted/40 py-2 pl-9 pr-2 4xl:py-3">
                <div className="min-w-0">
                  <span className="block truncate text-sm font-medium 4xl:text-base">{member.name}</span>
                  <span className="block text-xs text-muted-foreground 4xl:text-sm">
                    {member.code} · {member.points} poin
                  </span>
                </div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="cursor-pointer"
                  onClick={() => {
                    setMember(null)
                    setRedeemPoints(0)
                  }}
                >
                  <XIcon />
                </Button>
              </div>
            ) : (
              <Input
                value={memberQuery}
                onChange={(e) => setMemberQuery(e.target.value)}
                onFocus={() => setShowMemberResults(memberResults.length > 0)}
                placeholder="Cari member (opsional)..."
                className="h-10 pl-9 4xl:h-12 4xl:pl-10 4xl:text-base"
              />
            )}
            {showMemberResults && !member && memberResults.length > 0 && (
              <div className="absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-lg border bg-popover shadow-md">
                {memberResults.map((m) => (
                  <button
                    key={m.id}
                    type="button"
                    className="flex w-full cursor-pointer justify-between px-3 py-2 text-left text-sm hover:bg-muted"
                    onClick={() => selectMember(m)}
                  >
                    <span>{m.name}</span>
                    <span className="text-xs text-muted-foreground">{m.code}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
          {member && (
            <div className="space-y-1 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2">
              <div className="flex items-center justify-between gap-2">
                <div className="min-w-0">
                  <Label className="text-xs text-amber-800">Tukar Poin</Label>
                  <p className="text-[10px] text-amber-700">
                    Maks {totals.maxRedeemPoints} poin · saldo {member.points}
                  </p>
                </div>
                <Input
                  type="number"
                  min={0}
                  max={totals.maxRedeemPoints}
                  value={redeemPoints}
                  onChange={(e) => setRedeemPoints(Math.max(0, Number(e.target.value)))}
                  className="h-9 w-24 text-sm 4xl:h-11 4xl:w-28 4xl:text-base"
                />
              </div>
              {member.points <= 0 ? (
                <p className="text-[10px] font-medium text-rose-600">
                  Saldo poin 0 — tidak ada poin untuk ditukar.
                </p>
              ) : redeemInvalid ? (
                <p className="text-[10px] font-medium text-rose-600">
                  Jumlah tukar poin melebihi batas — maksimal {totals.maxRedeemPoints} poin.
                </p>
              ) : totals.maxRedeemPoints <= 0 ? (
                <p className="text-[10px] font-medium text-rose-600">
                  Total nota terlalu kecil untuk menukar poin.
                </p>
              ) : null}
            </div>
          )}
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4 3xl:p-5 4xl:p-6">
          {cart.length === 0 ? (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-muted-foreground">
              <ShoppingCartIcon className="size-10 opacity-40 4xl:size-14" />
              <p className="text-sm 4xl:text-base">Keranjang kosong. Ketuk produk untuk menambah.</p>
            </div>
          ) : (
            <div className="space-y-2 4xl:space-y-3">
              {cart.map((line) => (
                <div key={line.product.id} className="rounded-lg border p-3 4xl:p-4">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium 4xl:text-base">{line.product.name}</p>
                      <p className="text-xs text-muted-foreground 4xl:text-sm">
                        {formatRupiah(line.unitPrice)} / {line.product.selling_unit || "pcs"}
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="cursor-pointer text-destructive"
                      onClick={() => changeQty(line.product.id, -line.qty)}
                    >
                      <Trash2Icon />
                    </Button>
                  </div>
                  <div className="mt-2 flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="icon-sm"
                        className="cursor-pointer"
                        onClick={() => changeQty(line.product.id, -1)}
                      >
                        <MinusIcon />
                      </Button>
                      <span className="w-8 text-center text-sm font-medium 4xl:w-10 4xl:text-base">{line.qty}</span>
                      <Button
                        variant="outline"
                        size="icon-sm"
                        className="cursor-pointer"
                        onClick={() => changeQty(line.product.id, 1)}
                      >
                        <PlusIcon />
                      </Button>
                    </div>
                    <span className="text-sm font-semibold 4xl:text-base">
                      {formatRupiah(round2(line.qty * line.unitPrice - lineDiscount(line.discountType, line.discountValue, round2(line.qty * line.unitPrice))))}
                    </span>
                  </div>
                  {line.discountType !== "NONE" && line.discountValue > 0 && (
                    <p className="mt-2 text-xs text-muted-foreground 4xl:text-sm">
                      Diskon item:{" "}
                      {line.discountType === "PERCENT"
                        ? `${line.discountValue}%`
                        : formatRupiah(line.discountValue)}
                    </p>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="space-y-3 border-t p-4 3xl:p-5 4xl:space-y-4 4xl:p-6">
          <div className="grid gap-1">
            <Label className="text-xs text-muted-foreground 4xl:text-sm">Diskon Nota</Label>
            <div className="flex gap-1">
              <select
                value={discountType}
                onChange={(e) => setDiscountType(e.target.value)}
                className="h-9 flex-1 rounded-md border bg-background px-2 text-sm 4xl:h-11 4xl:text-base"
              >
                {DISCOUNT_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
              {discountType !== "NONE" && (
                <Input
                  type="number"
                  min={0}
                  value={discountValue}
                  onChange={(e) => setDiscountValue(Number(e.target.value))}
                  className="h-9 w-24 text-sm 4xl:h-11 4xl:w-28 4xl:text-base"
                />
              )}
            </div>
          </div>

          <div className="space-y-1 text-sm 4xl:space-y-2 4xl:text-base">
            <div className="flex justify-between text-muted-foreground">
              <span>Subtotal</span>
              <span>{formatRupiah(totals.subtotal)}</span>
            </div>
            <div className="flex justify-between text-muted-foreground">
              <span>Diskon</span>
              <span>-{formatRupiah(totals.itemDiscount + totals.invoiceDiscount)}</span>
            </div>
            <div className="flex justify-between text-muted-foreground">
              <span>DPP</span>
              <span>{formatRupiah(totals.dpp)}</span>
            </div>
            {totals.tax > 0 && (
              <div className="flex justify-between text-muted-foreground">
                <span>PPN</span>
                <span>{formatRupiah(totals.tax)}</span>
              </div>
            )}
            {totals.redeemAmount > 0 && (
              <div className="flex justify-between text-amber-700">
                <span>Tukar Poin ({totals.usedPoints})</span>
                <span>-{formatRupiah(totals.redeemAmount)}</span>
              </div>
            )}
            <div className="flex justify-between text-lg font-semibold 4xl:text-2xl">
              <span>Total</span>
              <span>{formatRupiah(totals.total)}</span>
            </div>
            {member && totals.pointsEarned > 0 && (
              <div className="flex justify-between text-xs text-muted-foreground 4xl:text-sm">
                <span>Poin didapat</span>
                <span>+{totals.pointsEarned}</span>
              </div>
            )}
          </div>

          <div>
            {/* <Button
              variant="outline"
              className="h-14 cursor-pointer"
              onClick={clearCart}
              disabled={cart.length === 0}
            >
              <Trash2Icon /> Kosongkan
            </Button> */}
            <Button
              className="w-full h-14 cursor-pointer text-base 4xl:h-16 4xl:text-lg"
              disabled={cart.length === 0 || redeemInvalid}
              onClick={() => setPaymentOpen(true)}
            >
              Bayar · {formatRupiah(totals.total)}
            </Button>
          </div>
        </div>
      </aside>

      <PaymentDialog
        open={paymentOpen}
        onOpenChange={setPaymentOpen}
        total={totals.total}
        methods={bootstrap?.payment_methods ?? []}
        submitting={submitting}
        onConfirm={(payment, notes) => submitSale([payment], notes)}
      />

      <ReceiptDialog sale={receipt} onClose={() => setReceipt(null)} />
    </div>
  )
}
