"use client";

import { apiGet, apiPost, apiPut, messageFromError } from "@/lib/api-client";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Plus,
  Trash2,
  ShoppingCart,
  Loader2,
  Package,
  Layers,
  DollarSign,
  FileText,
  ChevronDown,
  ChevronUp,
  Percent,
} from "lucide-react";
import { toast } from "sonner";
import { ComboboxSelect } from "@/components/combobox-select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PurchaseOrder } from "./purchase-order-table";

interface Supplier {
  id: number;
  name: string;
  default_tax_type?: string;
}

// Bentuk data dari /api/po-lookup/products (lihat go/query.sql LookupProductsForPO)
interface Product {
  id: number;
  sku: string;
  name: string;
  supplier_id: number;
  purchase_price: number;
  purchase_unit_kode?: string;
}

interface SelectedItem {
  product_id: string;
  qty_ordered: number;
  purchase_price: number;
  unit_kode: string;
  // Nama barang yang dibawa response detail PO. Dipakai sebagai fallback label
  // agar baris tidak tampak kosong walau barangnya absen dari daftar lookup.
  product_label: string;
}

// Bentuk baris dari response detail PO (GET /api/purchase-orders?id=...).
interface PORowDetail {
  product_id: number | string;
  qty_ordered: number;
  unit_price: number;
  unit_kode: string;
  product_sku: string;
  product_name: string;
}

// Susun label barang dengan format yang sama dengan lookup endpoint, supaya
// opsi "dibawa record" dan opsi "dari lookup" tampil identik.
function formatProductLabel(p: {
  sku?: string;
  name?: string;
  unit_kode?: string;
}) {
  const name = String(p.name ?? "")
    .trim()
    .toUpperCase();
  const sku = String(p.sku ?? "").trim();
  const unit = p.unit_kode ? ` (${p.unit_kode})` : "";
  if (!name) return "";
  return `[${sku || "?"}] - ${name}${unit}`;
}

interface PurchaseOrderFormProps {
  token: string | undefined;
  // wasEdit memberi tahu induk form ini apakah suksesnya itu penerbitan baru
  // atau koreksi, supaya pesan toast-nya tidak sama untuk dua hal berbeda.
  onSuccess: (poNumber: string, wasEdit: boolean) => void;
  editingPO: PurchaseOrder | null;
  onCancelEdit: () => void;
}

export function PurchaseOrderForm({
  token,
  onSuccess,
  editingPO,
  onCancelEdit,
}: PurchaseOrderFormProps) {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isLoadingRelations, setIsLoadingRelations] = useState<boolean>(true); // 🌟 KUNCI SYNC
  const [isLoadingProducts, setIsLoadingProducts] = useState<boolean>(false);

  // State Array Opsi Master Data (diisi dari /api/po-lookup/*, bukan modul master)
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [products, setProducts] = useState<Product[]>([]);

  // State Data Binding Header PO
  const [supplierId, setSupplierId] = useState<string>("");
  const [taxType, setTaxType] = useState<string>("NON_PPN");
  const [notes, setNotes] = useState<string>("");

  // State Data Binding Multi-Row Detail Items PO
  const [items, setItems] = useState<SelectedItem[]>([
    {
      product_id: "",
      qty_ordered: 1,
      purchase_price: 0,
      unit_kode: "",
      product_label: "",
    },
  ]);

  const isEditMode = !!editingPO;

  const supplierOptions = useMemo(() => {
    return suppliers.map((s) => ({
      value: String(s.id),
      label: s.name.toUpperCase(),
    }));
  }, [suppliers]);

  // Opsi barang = hasil lookup supplier aktif + barang yang sedang disunting pada
  // PO ini. Baris bawaan record memakai product_label dari response detail, sehingga
  // nama tetap tampil walau barangnya tidak lolos filter supplier atau endpoint
  // lookup tidak berhasil dimuat.
  const productOptions = useMemo(() => {
    const fromLookup = products
      .filter((p) => String(p.supplier_id) === String(supplierId)) // ✨ Konversi paksa kedua belah ID ke String
      .map((p) => ({
        value: String(p.id),
        label: formatProductLabel({
          sku: p.sku,
          name: p.name,
          unit_kode: p.purchase_unit_kode,
        }),
      }))
      .filter((o) => o.label);

    const carried = items
      .filter((it) => it.product_id && it.product_label)
      .filter(
        (it) => !fromLookup.some((o) => o.value === String(it.product_id))
      )
      .map((it) => ({ value: String(it.product_id), label: it.product_label }));

    return [...carried, ...fromLookup];
  }, [products, supplierId, items]);

  // Indeks resolusi harga & satuan. Harus memuat barang bawaan record juga, karena
  // saat user memilih ulang baris tersebut datanya tidak ada di 'products'.
  const productIndex = useMemo(() => {
    const map = new Map<
      string,
      { purchase_price: number; unit_kode: string; label: string }
    >();
    products.forEach((p) => {
      map.set(String(p.id), {
        purchase_price: p.purchase_price,
        unit_kode: p.purchase_unit_kode || "",
        label: formatProductLabel({
          sku: p.sku,
          name: p.name,
          unit_kode: p.purchase_unit_kode,
        }),
      });
    });
    items.forEach((it) => {
      if (it.product_id && it.product_label) {
        map.set(String(it.product_id), {
          purchase_price: it.purchase_price,
          unit_kode: it.unit_kode,
          label: it.product_label,
        });
      }
    });
    return map;
  }, [products, items]);

  // Perhitungan Akuntansi Reaktif (DPP, PPN 12%, & Grand Total)
  const financialSummary = useMemo(() => {
    const totalGrossItems = items.reduce(
      (sum, item) => sum + item.qty_ordered * item.purchase_price,
      0
    );
    const taxRate = 0.12;

    let dpp = 0;
    let tax = 0;
    let grandTotal = 0;

    switch (taxType) {
      case "TAX_EXCLUSIVE":
        dpp = totalGrossItems;
        tax = Math.round(dpp * taxRate);
        grandTotal = dpp + tax;
        break;
      case "TAX_INCLUSIVE":
        grandTotal = totalGrossItems;
        dpp = Math.round(grandTotal / (1.0 + taxRate));
        tax = grandTotal - dpp;
        break;
      default:
        dpp = totalGrossItems;
        tax = 0;
        grandTotal = totalGrossItems;
        break;
    }
    return { dpp, tax, grandTotal };
  }, [items, taxType]);

  // Helper: endpoint lookup PO memakai gate 'po:read', bukan 'suppliers:read' /
  // 'products:read', jadi user PO bisa memakai form ini tanpa akses master.
  const lookupFetch = useCallback(async <T,>(path: string): Promise<T[]> => {
    const body = await apiGet<T[]>(path);
    return Array.isArray(body.data) ? body.data : [];
  }, []);

  // Ambil daftar supplier (dipakai untuk mode create dan edit alike)
  const fetchFormSuppliers = useCallback(async () => {
    if (!token) return;
    setIsLoadingRelations(true);
    try {
      const rows = await lookupFetch<Supplier>("/api/po-lookup/suppliers");
      setSuppliers(
        rows.map((s) => ({
          id: s.id,
          name: s.name,
          default_tax_type: s.default_tax_type,
        }))
      );
    } catch (err) {
      console.error("Gagal memuat daftar supplier PO:", err);
      // Jangan diamkan: form akan tampil kosong dan ini penyebab utama bug.
      toast.warning("Daftar supplier tidak termuat", {
        description:
          "Periksa hak akses 'po:read' atau koneksi ke server. Nama supplier pada mode edit tetap ditampilkan dari data PO.",
      });
    } finally {
      setIsLoadingRelations(false); // 🌟 Nyatakan master data katalog resmi siap dikonsumsi
    }
  }, [token, lookupFetch]);

  // Ambil barang milik supplier terpilih (hanya saat form sudah Create/Edit)
  const fetchFormProducts = useCallback(
    async (targetSupplierId: string) => {
      if (!token) return;
      setIsLoadingProducts(true);
      try {
        const rows = await lookupFetch<Product>(
          `/api/po-lookup/products?supplier_id=${targetSupplierId}`
        );
        setProducts(
          rows.map((p) => ({
            id: p.id,
            sku: p.sku,
            name: p.name,
            supplier_id: p.supplier_id,
            purchase_price: p.purchase_price,
            purchase_unit_kode: p.purchase_unit_kode,
          }))
        );
      } catch (err) {
        console.error("Gagal memuat daftar barang PO:", err);
        setProducts([]);
        toast.warning("Daftar barang tidak termuat", {
          description:
            "Baris yang sedang disunting tetap menampilkan nama dari data PO, namun menambah baris baru butuh daftar barang.",
        });
      } finally {
        setIsLoadingProducts(false);
      }
    },
    [token, lookupFetch]
  );

  useEffect(() => {
    if (token) fetchFormSuppliers();
  }, [token, fetchFormSuppliers]);

  // Katalog barang mengikuti supplier yang aktif. Sengaja tidak menunggu detail PO:
  // baris bawaan record membawa labelnya sendiri, jadi tidak ada ketergantungan.
  // Daftar lama sengaja dibiarkan apa adanya; semua konsumen memfilter berdasarkan
  // supplierId, jadi data basi tidak akan tampil dan tidak perlu di-reset di sini.
  useEffect(() => {
    if (supplierId) fetchFormProducts(supplierId);
  }, [supplierId, token, fetchFormProducts]);

  // 🌟 SINKRONISASI KAKU EDIT MODE: Tunggu katalog selesai unduh baru injeksikan nilai lama
  useEffect(() => {
    if (editingPO && !isLoadingRelations) {
      setSupplierId(String(editingPO.supplier_id));
      setTaxType(editingPO.tax_type);
      setNotes(editingPO.notes || "");
      setIsExpanded(true);

      const fetchPODetailsForEdit = async () => {
        try {
          const result = await apiGet<{ items?: PORowDetail[] }>(
            `/api/purchase-orders?id=${editingPO.id}`
          );
          if (result.data?.items) {
            const mappedItems = result.data.items.map((item) => ({
              product_id: String(item.product_id), // ✨ Wajib dipaksa menjadi String agar ComboboxSelect mengenali kecocokan nilainya
              qty_ordered: item.qty_ordered,
              purchase_price: item.unit_price,
              unit_kode: item.unit_kode,
              // Bawa nama dari server: inilah yang membuat kolom barang
              // tetap terbaca walau /api/po-lookup/products tidak mengembalikan ID ini.
              product_label: formatProductLabel({
                sku: item.product_sku,
                name: item.product_name,
                unit_kode: item.unit_kode,
              }),
            }));
            setItems(mappedItems);
          }
        } catch (err) {
          console.error(
            "Gagal memuat sub-item untuk proses koreksi edit:",
            err
          );
        }
      };
      fetchPODetailsForEdit();
    }
  }, [editingPO, isLoadingRelations, token]);

  // Efek Proteksi Supplier Dinamis (Hanya aktif untuk pembuatan baru)
  // Pasang blok ini di bawah useEffect pencari data editingPO Anda:

  // 🌟 FUNGSI PENYELAMAT RELASI: Detektor Validasi Hubungan Item & Auto-Clean Cross Supplier
  useEffect(() => {
    if (isEditMode) return; // Kunci efek pembersihan massal jika sedang dalam mode koreksi edit

    if (!supplierId) {
      setTaxType("NON_PPN");
      setItems([
        {
          product_id: "",
          qty_ordered: 1,
          purchase_price: 0,
          unit_kode: "",
          product_label: "",
        },
      ]);
      return;
    }

    // Cari tahu tipe pajak bawaan supplier terpilih untuk otomatisasi form
    const selectedSupplier = suppliers.find(
      (s) => String(s.id) === String(supplierId)
    );
    if (selectedSupplier && selectedSupplier.default_tax_type) {
      setTaxType(selectedSupplier.default_tax_type);
    }

    // Penting: jangan bersihkan baris selagi daftar barang masih dimuat, atau
    // satu-satunya barang milik supplier itu akan terhapus sebelum sempat tampil.
    if (isLoadingProducts) return;

    // Menyisir baris item agar patuh kaku pada katalog supplier yang aktif.
    // Sengaja hanya memakai 'products' (hasil lookup), bukan 'productIndex',
    // karena productIndex memuat juga baris bawaan record sehingga pengaman
    // silang antar supplier ini tidak akan pernah menyala.
    setItems((prevItems) => {
      return prevItems.map((item) => {
        if (!item.product_id) return item;

        // Kunci pencocokan tipe data kaku berbasis string di kedua sisi
        const isStillValid = products.some(
          (p) =>
            String(p.id) === String(item.product_id) &&
            String(p.supplier_id) === String(supplierId)
        );

        return isStillValid
          ? item
          : {
              product_id: "",
              qty_ordered: 1,
              purchase_price: 0,
              unit_kode: "",
              product_label: "",
            };
      });
    });
  }, [supplierId, suppliers, products, isLoadingProducts, isEditMode]); // Mengawasi pergeseran state secara reaktif

  const handleAddItemRow = () => {
    setItems([
      ...items,
      {
        product_id: "",
        qty_ordered: 1,
        purchase_price: 0,
        unit_kode: "",
        product_label: "",
      },
    ]);
  };

  const handleRemoveItemRow = (index: number) => {
    if (items.length === 1) {
      toast.warning("Aturan Dokumen kaku", {
        description: "Minimal dokumen PO berisi 1 baris item barang!",
      });
      return;
    }
    setItems(items.filter((_, i) => i !== index));
  };

  const handleUpdateRowValue = <K extends keyof SelectedItem>(
    index: number,
    field: K,
    value: SelectedItem[K]
  ) => {
    const updated = [...items];
    if (field === "product_id") {
      // 'productIndex' memuat barang bawaan record dan hasil lookup, jadi
      // memilih ulang baris tersebut tidak menghapus harga & satuannya.
      const matched = productIndex.get(String(value));
      updated[index] = {
        ...updated[index],
        product_id: String(value),
        purchase_price: matched?.purchase_price ?? 0,
        unit_kode: matched?.unit_kode ?? "",
        product_label: matched?.label ?? "",
      };
    } else {
      updated[index] = { ...updated[index], [field]: value };
    }
    setItems(updated);
  };

  const resetForm = () => {
    setSupplierId("");
    setTaxType("NON_PPN");
    setNotes("");
    setItems([
      {
        product_id: "",
        qty_ordered: 1,
        purchase_price: 0,
        unit_kode: "",
        product_label: "",
      },
    ]);
    if (isEditMode) onCancelEdit();
  };

  // Batal menutup seluruh alur, bukan sekadar mengosongkan isian.
  // Tombol lama ("Reset Kolom") hanya membersihkan isi form, sehingga panel
  // tetap terbuka dengan form kosong. Bagi pengguna itu tidak berbeda dari
  // form yang siap diisi ulang, jadi tombolnya menyesatkan.
  //
  // Urutannya penting: onCancelEdit() membuat editingPO di-induk menjadi
  // null, dan efek sinkronisasi edit mode ikut mencoba mengisi form lagi
  // dari dokumen yang baru dibatalkan. Menutup panel di akhir memastikan
  // tidak ada sisa state yang terlihat setelah form ditutup.
  const handleCancel = () => {
    resetForm();
    setIsExpanded(false);
  };

  const handleSubmitPO = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!supplierId) {
      toast.error("Validasi Gagal", {
        description: "Wajib memilih rekanan Supplier!",
      });
      return;
    }
    if (items.some((item) => !item.product_id)) {
      toast.error("Validasi Gagal", {
        description: "Terdapat baris barang yang belum dipilih!",
      });
      return;
    }

    setIsLoading(true);
    const targetUrl = isEditMode
      ? `/api/purchase-orders?id=${editingPO.id}`
      : `/api/purchase-orders`;

    try {
      // Payload differs by mode. Saat koreksi, supplier_id SENGAJA tidak
      // dikirim: kolom rekanan terkunci dan backend juga tidak pernah
      // menyentuh kolom itu. Harga pun tidak dikirim sama sekali karena
      // server menurunkannya sendiri dari master produk.
      const payload: Record<string, unknown> = {
        tax_type: taxType,
        notes: notes.trim(),
        items: items.map((item) => ({
          product_id: Number(item.product_id),
          qty_ordered: Number(item.qty_ordered),
        })),
      };
      if (!isEditMode) {
        payload.supplier_id = Number(supplierId);
      }

      const result = await (isEditMode
        ? apiPut<{ po_number?: string }>(targetUrl, payload)
        : apiPost<{ po_number?: string }>(targetUrl, payload));
      onSuccess(
        isEditMode ? editingPO.po_number : result.data?.po_number || "PO BARU",
        isEditMode
      );
      // resetForm() menutup mode koreksi lewat onCancelEdit, jadi
      // form perlu disetel ulang lebih dulu supaya state edit tidak
      // bocor ke render berikutnya.
      resetForm();
      setIsExpanded(false);
    } catch (err) {
      toast.error(isEditMode ? "Gagal Menyimpan Koreksi" : "Gagal Menyimpan", {
        description: messageFromError(err),
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div
      className={`bg-white rounded-xl shadow-2xs border ${isEditMode ? "border-amber-300 ring-2 ring-amber-500/10" : "border-slate-200"} overflow-hidden`}
    >
      <button
        type="button"
        onClick={() => !editingPO && setIsExpanded(!isExpanded)}
        disabled={!!editingPO}
        className="w-full px-6 py-3.5 flex items-center justify-between bg-slate-50/80 hover:bg-slate-100/60 transition-colors text-left cursor-pointer"
      >
        <span className="text-xs font-bold uppercase tracking-wider text-slate-700 flex items-center gap-2">
          <Package className="size-4 text-blue-600" />
          {editingPO
            ? `Koreksi Spesifikasi Dokumen PO: ${editingPO.po_number}`
            : "Penerbitan Dokumen Purchase Order (PO) Baru"}
        </span>
        {!editingPO &&
          (isExpanded ? (
            <ChevronUp className="h-4 w-4" />
          ) : (
            <ChevronDown className="h-4 w-4" />
          ))}
      </button>

      <div
        className={`transition-all duration-300 ease-in-out ${isExpanded ? "max-h-[1600px] p-6 border-t border-slate-100" : "max-h-0 opacity-0 pointer-events-none"}`}
      >
        {isLoadingRelations ? (
          <div className="py-12 flex flex-col items-center justify-center text-xs text-slate-400 gap-2 font-medium">
            <Loader2 className="size-5 animate-spin text-blue-600" />
            <span>Sinkronisasi katalog gudang Axion...</span>
          </div>
        ) : (
          <form onSubmit={handleSubmitPO} className="space-y-5">
            {/* Header Parameter Dokumen */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 bg-slate-50/60 p-4 rounded-xl border">
              <div className="space-y-1.5 md:col-span-1">
                <label className="text-[11px] font-bold text-slate-600 uppercase flex items-center gap-1">
                  <Package className="size-3 text-blue-600" /> Rekanan Supplier
                </label>
                <ComboboxSelect
                  key={`supplier-combo-${editingPO?.id ?? "new"}`}
                  options={supplierOptions}
                  value={supplierId}
                  placeholder="Pilih Supplier..."
                  onChange={(val) => setSupplierId(val)}
                  emptyText="Supplier tidak terindeks..."
                  disabled={isEditMode}
                  // Nama ini dibawa dokumen PO. Tanpa ini, input kosong
                  // saat daftar supplier gagal dimuat / tidak memuat ID ini.
                  fallbackLabel={editingPO?.supplier_name}
                />
              </div>
              <div className="space-y-1.5 md:col-span-1">
                <label className="text-[11px] font-bold text-slate-600 uppercase flex items-center gap-1">
                  <Percent className="size-3 text-amber-600" /> Pengaturan PPN
                </label>
                <select
                  value={taxType}
                  onChange={(e) => setTaxType(e.target.value)}
                  className="w-full text-xs px-3 h-8 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer font-bold text-slate-700 focus:border-blue-500"
                >
                  <option value="NON_PPN">NON-PPN</option>
                  <option value="TAX_EXCLUSIVE">
                    TAX EXCLUSIVE (Belum Termasuk PPN 12%)
                  </option>
                  <option value="TAX_INCLUSIVE">
                    TAX INCLUSIVE (Sudah Termasuk PPN 12%)
                  </option>
                </select>
              </div>
              <div className="space-y-1.5 md:col-span-2">
                <label className="text-[11px] font-bold text-slate-600 uppercase flex items-center gap-1">
                  <FileText className="size-3" /> Memo Instruksi Pengiriman
                  Vendor
                </label>
                <input
                  type="text"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg outline-none focus:border-blue-500 h-8 font-medium text-slate-900"
                  placeholder="Kirim armada box tertutup..."
                />
              </div>
            </div>
            <div className="space-y-3">
              <div className="flex items-center justify-between border-b pb-2">
                <h3 className="text-[11px] font-bold uppercase tracking-wider text-slate-800 flex items-center gap-1.5">
                  <Layers className="size-3.5 text-blue-600" /> Baris Komoditas
                  Detail Item Barang
                </h3>
                <Button
                  type="button"
                  size="sm"
                  onClick={handleAddItemRow}
                  className="bg-blue-50 text-blue-600 hover:bg-blue-100 border border-blue-200 text-[10px] font-bold h-7 cursor-pointer px-2.5 shadow-2xs"
                >
                  <Plus className="size-3 mr-1" /> Tambah Baris Item
                </Button>
              </div>
              <div className="border border-slate-200 rounded-xl bg-white shadow-2xs overflow-hidden">
                <Table>
                  <TableHeader className="bg-slate-50/70 border-b">
                    <TableRow className="h-9">
                      <TableHead className="font-bold text-[11px] text-slate-700 text-center w-12 py-1">
                        No
                      </TableHead>
                      <TableHead className="font-bold text-[11px] text-slate-700 py-1">
                        Pilih Produk / Komoditas Barang
                      </TableHead>
                      <TableHead className="font-bold text-[11px] text-slate-700 text-center w-36 py-1">
                        Kuantitas (Qty)
                      </TableHead>
                      <TableHead className="font-bold text-[11px] text-slate-700 text-right w-36 py-1">
                        {isEditMode
                          ? "Harga Satuan (dari master)"
                          : "Harga Satuan (HPP)"}
                      </TableHead>
                      <TableHead className="font-bold text-[11px] text-slate-700 text-right w-40 py-1">
                        Sub-Total
                      </TableHead>
                      <TableHead className="font-bold text-[11px] text-slate-700 text-center w-16 py-1">
                        Hapus
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item, idx) => (
                      <TableRow
                        key={`po-form-row-${idx}`}
                        className="hover:bg-slate-50/30 border-b last:border-b-0 h-12"
                      >
                        <TableCell className="text-center font-mono font-bold text-slate-400 py-1.5 w-12">
                          {idx + 1}
                        </TableCell>
                        <TableCell className="py-1.5 min-w-[280px]">
                          <div className="w-full">
                            <ComboboxSelect
                              key={`product-combo-${idx}-${item.product_id || "empty"}`}
                              options={productOptions}
                              value={item.product_id}
                              placeholder={
                                supplierId
                                  ? "Ketik nama barang..."
                                  : "🔒 Pilih Supplier terlebih dahulu"
                              }
                              onChange={(val) =>
                                handleUpdateRowValue(idx, "product_id", val)
                              }
                              emptyText="Barang tidak terindeks..."
                              disabled={!supplierId}
                              // Nama bawaan baris detail PO; menahankan tampilan
                              // walau ID ini absen dari hasil lookup.
                              fallbackLabel={item.product_label}
                            />
                          </div>
                        </TableCell>
                        <TableCell className="py-1.5 w-36 text-center">
                          <div className="flex items-center gap-1.5 bg-white border rounded-lg px-2 h-8 max-w-[130px] mx-auto shadow-2xs focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-500/10">
                            <input
                              type="number"
                              required
                              min={1}
                              disabled={!item.product_id}
                              value={item.qty_ordered}
                              onChange={(e) =>
                                handleUpdateRowValue(
                                  idx,
                                  "qty_ordered",
                                  Math.max(1, Number(e.target.value))
                                )
                              }
                              className="w-full text-xs outline-none text-center font-mono font-bold text-slate-800 disabled:opacity-50"
                            />
                            <span className="text-[9px] font-black text-slate-400 font-mono shrink-0 border-l pl-1 uppercase">
                              {item.unit_kode || "UoM"}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="py-1.5 text-right font-mono text-xs font-semibold text-slate-500 w-36">
                          {item.product_id ? (
                            `Rp ${new Intl.NumberFormat("id-ID").format(item.purchase_price)}`
                          ) : (
                            <span className="text-slate-300 italic font-sans text-[10px]">
                              Menunggu barang
                            </span>
                          )}
                        </TableCell>
                        <TableCell className="py-1.5 text-right font-mono text-xs font-bold text-slate-900 w-40">
                          Rp{" "}
                          {new Intl.NumberFormat("id-ID").format(
                            item.qty_ordered * item.purchase_price
                          )}
                        </TableCell>
                        <TableCell className="py-1.5 text-center w-16">
                          <button
                            type="button"
                            onClick={() => handleRemoveItemRow(idx)}
                            className="p-1.5 border border-slate-200 rounded-lg text-rose-500 bg-white hover:bg-rose-50 hover:text-rose-600 transition-colors shadow-3xs cursor-pointer"
                          >
                            <Trash2 className="size-3.5" />
                          </button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
            {/* Ringkasan Finansial Bawah */}
            <div className="flex flex-col sm:flex-row justify-between items-end p-5 bg-slate-50 border border-slate-200 rounded-xl gap-4">
              <div className="text-[11px] font-medium text-slate-400 flex items-start gap-1 self-start sm:self-center">
                <DollarSign className="size-3.5 text-blue-500 shrink-0" />
                <span>
                  Nilai DPP dan PPN 12% dihitung otomatis secara ketat
                  berdasarkan standar aturan akuntansi perpajakan.
                  {isEditMode && (
                    <span className="block text-amber-700 font-semibold mt-0.5">
                      Mode koreksi: rekanan supplier terkunci dan harga satuan
                      dihitung ulang server dari master barang, jadi angka di
                      bawah bisa berbeda dari yang tampil sebelum disimpan.
                    </span>
                  )}
                </span>
              </div>
              <div className="w-full sm:w-80 space-y-1.5 text-xs font-mono font-medium text-slate-600">
                <div className="flex justify-between border-b border-dashed pb-1">
                  <span>TOTAL DPP (NETTO):</span>
                  <span className="text-slate-900">
                    Rp{" "}
                    {new Intl.NumberFormat("id-ID").format(
                      financialSummary.dpp
                    )}
                  </span>
                </div>
                <div className="flex justify-between border-b border-dashed pb-1">
                  <span>PPN MASUKAN (12%):</span>
                  <span className="text-slate-900">
                    Rp{" "}
                    {new Intl.NumberFormat("id-ID").format(
                      financialSummary.tax
                    )}
                  </span>
                </div>
                <div className="flex justify-between text-sm pt-0.5">
                  <span className="font-bold text-slate-700">
                    GRAND TOTAL PO:
                  </span>
                  <span className="font-black text-blue-700 text-base">
                    {new Intl.NumberFormat("id-ID", {
                      style: "currency",
                      currency: "IDR",
                      maximumFractionDigits: 0,
                    }).format(financialSummary.grandTotal)}
                  </span>
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-2 pt-3 border-t">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleCancel}
                disabled={isLoading}
                className="text-xs h-8 cursor-pointer"
              >
                Batal
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={isLoading}
                className={`text-white text-xs h-8 flex items-center gap-1 cursor-pointer ${isEditMode ? "bg-amber-600 hover:bg-amber-700" : "bg-blue-600 hover:bg-blue-700"}`}
              >
                {isLoading ? (
                  <Loader2 className="size-3 animate-spin" />
                ) : (
                  <ShoppingCart className="size-3" />
                )}
                <span>
                  {isEditMode
                    ? "Simpan Koreksi Perubahan PO"
                    : "Kunci & Terbitkan Dokumen PO"}
                </span>
              </Button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
