"use client";

import { apiGet, apiPost, apiPut, messageFromError } from "@/lib/api-client";
import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  ChevronDown,
  ChevronUp,
  Loader2,
  Package,
  Tag,
  Building2,
  Layers,
  DollarSign,
  AlertCircle,
} from "lucide-react";
import { toast } from "sonner";
import { ComboboxSelect } from "@/components/combobox-select";
import { MoneyInput } from "@/components/ui/money-input";

interface Category {
  id: number;
  name: string;
}
interface Supplier {
  id: number;
  name: string;
}
interface Unit {
  id: number;
  kode: string;
  name: string;
}

interface Product {
  id: number;
  sku: string;
  barcode: string;
  name: string;
  description: string;
  category_id: number;
  supplier_id: number;
  selling_unit_id: number; // 🌟 Dual Unit Schema
  purchase_unit_id: number;
  conversion_rate?: number;
  purchase_price: number;
  selling_price: number;
  min_stock_level: number;
  current_stock: number;
  discount_type?: string;
  discount_value?: number;
  is_taxable?: boolean;
  is_active: boolean;
}

interface ProductFormProps {
  token: string | undefined;
  editingProduct: Product | null;
  onSuccess: (mode: "create" | "update", name: string) => void;
  onCancelEdit: () => void;
}

export function ProductForm({
  token,
  editingProduct,
  onSuccess,
  onCancelEdit,
}: ProductFormProps) {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  // State Opsi Pilihan Dropdown Dinamis Relasional
  const [categories, setCategories] = useState<Category[]>([]);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [units, setUnits] = useState<Unit[]>([]);

  // State Form Utama
  const [sku, setSku] = useState<string>("");
  const [barcode, setBarcode] = useState<string>("");
  const [name, setName] = useState<string>("");
  const [description, setDescription] = useState<string>("");
  const [categoryId, setCategoryId] = useState<string>("");
  const [supplierId, setSupplierId] = useState<string>("");
  const [sellingUnitId, setSellingUnitId] = useState<string>("");
  const [purchaseUnitId, setPurchaseUnitId] = useState<string>("");
  const [conversionRate, setConversionRate] = useState<number>(1);
  const [purchasePrice, setPurchasePrice] = useState<number>(0);
  const [sellingPrice, setSellingPrice] = useState<number>(0);
  const [minStock, setMinStock] = useState<number>(5);
  const [currentStock, setCurrentStock] = useState<number>(0);
  const [discountType, setDiscountType] = useState<string>("NONE");
  const [discountValue, setDiscountValue] = useState<number>(0);
  const [isTaxable, setIsTaxable] = useState<boolean>(true);
  const [isActive, setIsActive] = useState<boolean>(true);

  const categoryOptions = useMemo(() => {
    return categories.map((c) => ({ value: String(c.id), label: c.name }));
  }, [categories]);

  const supplierOptions = useMemo(() => {
    return suppliers.map((s) => ({ value: String(s.id), label: s.name }));
  }, [suppliers]);

  const unitOptions = useMemo(() => {
    return units.map((u) => ({
      value: String(u.id),
      label: `[${u.kode}] - ${u.name}`,
    }));
  }, [units]);

  // Mengambil Opsi Relasional dari API Kategori, Supplier, & Master Satuan Baru
  const fetchRelations = async () => {
    try {
      const [catData, supData, unitData] = await Promise.all([
        apiGet<Category[]>(`/api/categories?limit=100`),
        apiGet<Supplier[]>(`/api/suppliers?limit=100`),
        apiGet<Unit[]>(`/api/units?limit=100`),
      ]);

      if (catData.data) setCategories(catData.data);
      if (supData.data) setSuppliers(supData.data);
      if (unitData.data) setUnits(unitData.data);
    } catch (err) {
      console.error("Gagal memuat data opsi relasi:", err);
    }
  };

  useEffect(() => {
    if (token) fetchRelations();
  }, [token]);

  const resetForm = () => {
    setSku("");
    setBarcode("");
    setName("");
    setDescription("");
    setCategoryId("");
    setSupplierId("");
    setSellingUnitId("");
    setPurchaseUnitId("");
    setPurchasePrice(0);
    setSellingPrice(0);
    setMinStock(5);
    setCurrentStock(0);
    setIsActive(true);
    setDiscountType("NONE");
    setDiscountValue(0);
    setIsTaxable(true);
  };

  // Jalur Sinkronisasi Mode Edit Data
  useEffect(() => {
    if (editingProduct) {
      setSku(editingProduct.sku);
      setBarcode(editingProduct.barcode || "");
      setName(editingProduct.name);
      setDescription(editingProduct.description || "");
      setCategoryId(String(editingProduct.category_id));
      setSupplierId(String(editingProduct.supplier_id));
      setSellingUnitId(String(editingProduct.selling_unit_id));
      setPurchaseUnitId(String(editingProduct.purchase_unit_id));
      setConversionRate(editingProduct.conversion_rate || 1);
      setPurchasePrice(editingProduct.purchase_price);
      setSellingPrice(editingProduct.selling_price);
      setMinStock(editingProduct.min_stock_level);
      setCurrentStock(editingProduct.current_stock || 0);
      setDiscountType(editingProduct.discount_type || "NONE");
      setDiscountValue(editingProduct.discount_value || 0);
      setIsTaxable(editingProduct.is_taxable ?? true);
      setIsActive(editingProduct.is_active);
      setIsExpanded(true);
    } else {
      resetForm();
    }
  }, [editingProduct]);

  // useEffect(() => {
  //     if (categoryId && token) fetchCategories(`?category_id=${categoryId}`, setCategories);
  //     else setCategories([]);
  // }, [categoryId, token]);

  // useEffect(() => {
  //     if (supplierId && token) fetchSuppliers(`?supplier_id=${supplierId}`, setSuppliers);
  //     else setSuppliers([]);
  // }, [supplierId, token]);

  // useEffect(() => {
  //     if (unitId && token) fetchUnits(`?unit_id=${unitId}`, setUnits);
  //     else setUnits([]);
  // }, [unitId, token]);

  // const fetchCategories = async (query: string, setter: Function) => {
  //     try {
  //         const apiBaseUrl = API_BASE_URL;
  //         const res = await fetch(`/api/categories${query}`, {
  //             headers: { "Authorization": `Bearer ${token}` }
  //         });
  //         const result = await res.json();
  //         if (res.ok && result.data) setter(result.data);
  //     } catch (err) {
  //         console.error("Gagal memuat data regional:", err);
  //     }
  // };

  // const fetchSuppliers = async (query: string, setter: Function) => {
  //     try {
  //         const apiBaseUrl = API_BASE_URL;
  //         const res = await fetch(`/api/suppliers${query}`, {
  //             headers: { "Authorization": `Bearer ${token}` }
  //         });
  //         const result = await res.json();
  //         if (res.ok && result.data) setter(result.data);
  //     } catch (err) {
  //         console.error("Gagal memuat data regional:", err);
  //     }
  // };

  // const fetchUnits = async (query: string, setter: Function) => {
  //     try {
  //         const apiBaseUrl = API_BASE_URL;
  //         const res = await fetch(`/api/units${query}`, {
  //             headers: { "Authorization": `Bearer ${token}` }
  //         });
  //         const result = await res.json();
  //         if (res.ok && result.data) setter(result.data);
  //     } catch (err) {
  //         console.error("Gagal memuat data regional:", err);
  //     }
  // };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validasi Akuntansi di Sisi Klien
    if (sellingPrice < purchasePrice) {
      toast.error("Kesalahan Aturan Akuntansi", {
        description:
          "Harga jual standar tidak boleh lebih rendah dari harga beli modal (HPP dasar)!",
      });
      return;
    }

    if (!sellingUnitId || !purchaseUnitId) {
      toast.error("Validasi Gagal", {
        description:
          "Mohon lengkapi pengaturan Satuan Jual dan Satuan Beli produk!",
      });
      return;
    }

    if (
      discountType === "PERCENT" &&
      (discountValue < 0 || discountValue > 100)
    ) {
      toast.error("Validasi Diskon Gagal", {
        description: "Diskon persen harus berada di antara 0 dan 100!",
      });
      return;
    }

    if (discountType === "AMOUNT" && discountValue < 0) {
      toast.error("Validasi Diskon Gagal", {
        description: "Nilai diskon tidak boleh negatif!",
      });
      return;
    }

    setIsLoading(true);
    const isEditMode = editingProduct !== null;
    const url = isEditMode
      ? `/api/products?id=${editingProduct.id}`
      : `/api/products`;
    try {
      const payload = {
        sku: sku || undefined,
        barcode: barcode || null,
        name,
        description,
        category_id: Number(categoryId),
        supplier_id: Number(supplierId),
        selling_unit_id: Number(sellingUnitId),
        purchase_unit_id: Number(purchaseUnitId),
        conversion_rate: Number(conversionRate),
        purchase_price: Number(purchasePrice),
        selling_price: Number(sellingPrice),
        min_stock_level: Number(minStock),
        current_stock: Number(currentStock),
        discount_type: discountType,
        discount_value: discountType === "NONE" ? 0 : Number(discountValue),
        is_taxable: isTaxable,
        is_active: isActive,
      };
      if (isEditMode) await apiPut(url, payload);
      else await apiPost(url, payload);

      resetForm();
      setIsExpanded(false);
      onSuccess(isEditMode ? "update" : "create", name);
    } catch (error) {
      toast.error("Gagal Menyimpan", { description: messageFromError(error) });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-2xs border border-slate-200 overflow-hidden transition-all duration-300">
      {/* Header Tombol Collapsible */}
      <button
        type="button"
        onClick={() => !editingProduct && setIsExpanded(!isExpanded)}
        disabled={!!editingProduct}
        className="w-full px-6 py-3.5 flex items-center justify-between bg-slate-50/80 hover:bg-slate-100/60 transition-colors text-left cursor-pointer"
      >
        <span className="text-sm font-semibold text-foreground flex items-center gap-2">
          <Package className="size-4 text-blue-600" />
          {editingProduct
            ? `Edit Spesifikasi Produk: ${editingProduct.name}`
            : "Tambah Barang / Produk Baru"}
        </span>
        {!editingProduct &&
          (isExpanded ? (
            <ChevronUp className="h-4 w-4" />
          ) : (
            <ChevronDown className="h-4 w-4" />
          ))}
      </button>

      <div
        className={`transition-all duration-300 ease-in-out ${isExpanded ? "max-h-none p-6 border-t border-slate-100" : "max-h-0 opacity-0 pointer-events-none"}`}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Baris 1: Identifikasi Unik SKU, Barcode, & Dropdown Master Satuan */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Tag className="size-3" /> Kode SKU Internal
              </label>
              <input
                type="text"
                value={sku}
                onChange={(e) => setSku(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 font-mono"
                placeholder="Kosongkan untuk auto-generate"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Tag className="size-3" /> Kode Barcode / EAN-13
              </label>
              <input
                type="text"
                value={barcode}
                onChange={(e) => setBarcode(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 font-mono"
                placeholder="Contoh: 8991234xxxx"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Layers className="size-3" /> Satuan Beli (UoM)
              </label>
              <ComboboxSelect
                options={unitOptions}
                value={purchaseUnitId}
                placeholder="Pilih Satuan Beli..."
                onChange={(val) => setPurchaseUnitId(val)}
                emptyText="Satuan tidak ditemukan, ketik kode baru..."
                permissionCode="units:create"
                onRefreshRelations={fetchRelations}
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Layers className="size-3" /> Satuan Jual (UoM)
              </label>
              <ComboboxSelect
                options={unitOptions}
                value={sellingUnitId}
                placeholder="Pilih Satuan..."
                onChange={(val) => setSellingUnitId(val)}
                emptyText="Satuan tidak ditemukan, ketik kode baru..."
                permissionCode="units:create"
                onRefreshRelations={fetchRelations}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Layers className="size-3" /> Konversi (1 Satuan Beli = ? Satuan
                Jual)
              </label>
              <input
                type="number"
                min={1}
                step={1}
                value={conversionRate}
                onChange={(e) =>
                  setConversionRate(Math.max(1, Number(e.target.value)))
                }
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900"
                placeholder="1"
              />
              <p className="text-xs text-slate-500">
                Stok dihitung dalam satuan jual. Kosongkan/biarkan 1 jika satuan
                beli sama dengan satuan jual.
              </p>
            </div>
          </div>

          {/* Baris 2: Nama & Deskripsi */}
          <div className="grid grid-cols-1 gap-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Package className="size-3" /> Nama Lengkap Barang
              </label>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900"
                placeholder="Contoh: Router Hub Pro 8-Port"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                Deskripsi Item
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 resize-none"
                placeholder="Tulis spesifikasi teknis komponen barang di sini..."
              />
            </div>
          </div>

          {/* Baris 3: Relasi Dropdown Dinamik */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 bg-slate-50/50 p-4 rounded-xl border border-slate-100">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Layers className="size-3" /> Klasifikasi Rumpun Kategori
              </label>
              {/* <select required value={categoryId} onChange={(e) => setCategoryId(e.target.value)} className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer">
                                <option value="">--- Pilih Golongan Kategori Barang ---</option>
                                {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                            </select> */}
              <ComboboxSelect
                options={categoryOptions}
                value={categoryId}
                placeholder="Pilih Kategori..."
                onChange={(val) => {
                  setCategoryId(val);
                }}
                emptyText="Kategori produk tidak ditemukan, ketik kode baru..."
                permissionCode="categories:create"
                onRefreshRelations={fetchRelations}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Building2 className="size-3" /> Rekanan Vendor / Supplier Utama
              </label>
              {/* <select required value={supplierId} onChange={(e) => setSupplierId(e.target.value)} className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer">
                                <option value="">--- Pilih Perusahaan Supplier Penyuplai ---</option>
                                {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                            </select> */}
              <ComboboxSelect
                options={supplierOptions}
                value={supplierId}
                placeholder="Pilih Supplier..."
                onChange={(val) => {
                  setSupplierId(val);
                }}
              />
            </div>
          </div>

          {/* Baris 4: Metrika Finansial Harga & Batas Stok */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 bg-blue-50/10 p-4 rounded-xl border border-blue-100/30">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <DollarSign className="size-3" /> Harga Beli Modal (HPP)
              </label>
              <MoneyInput
                required
                value={purchasePrice}
                onValueChange={setPurchasePrice}
                className="h-auto border-slate-200 bg-white px-3 py-2"
                inputClassName="text-xs"
                placeholder="0"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <DollarSign className="size-3" /> Harga Jual Standar
              </label>
              <MoneyInput
                required
                value={sellingPrice}
                onValueChange={setSellingPrice}
                className="h-auto border-slate-200 bg-white px-3 py-2"
                inputClassName="text-xs"
                placeholder="0"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <AlertCircle className="size-3" /> Batas Minimum Stok (Safety)
              </label>
              <input
                type="number"
                required
                min={1}
                value={minStock}
                onChange={(e) => setMinStock(Number(e.target.value))}
                className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg focus:outline-none font-mono"
                placeholder="5"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Package className="size-3" /> Saldo Stok Awal (Qty)
              </label>
              <input
                type="number"
                required
                min={0}
                value={currentStock}
                onChange={(e) => setCurrentStock(Number(e.target.value))}
                disabled={editingProduct !== null}
                title={
                  editingProduct !== null
                    ? "Stok diisi otomatis oleh penerimaan barang (GRN)"
                    : "Saldo stok awal dalam satuan jual"
                }
                className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg focus:outline-none font-mono cursor-not-allowed text-slate-500"
                placeholder="0"
              />
            </div>
          </div>

          {/* Baris 5: Diskon Default Produk untuk Kasir POS */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 bg-amber-50/10 p-4 rounded-xl border border-amber-100/40">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Tag className="size-3" /> Tipe Diskon Default
              </label>
              <select
                value={discountType}
                onChange={(e) => {
                  const val = e.target.value;
                  setDiscountType(val);
                  if (val === "NONE") setDiscountValue(0);
                }}
                className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer text-slate-900"
              >
                <option value="NONE">Tanpa Diskon</option>
                <option value="PERCENT">Persen (%)</option>
                <option value="AMOUNT">Nominal (Rp)</option>
              </select>
              <p className="text-xs text-slate-500">
                Diskon ini otomatis terpasang saat kasir memilih produk di POS.
              </p>
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <DollarSign className="size-3" /> Nilai Diskon
              </label>
              {discountType === "AMOUNT" ? (
                <MoneyInput
                  value={discountValue}
                  onValueChange={setDiscountValue}
                  className="h-auto border-slate-200 bg-white px-3 py-2"
                  inputClassName="text-xs"
                  placeholder="0"
                />
              ) : (
                <input
                  type="number"
                  min={0}
                  max={discountType === "PERCENT" ? 100 : undefined}
                  step={discountType === "PERCENT" ? 1 : 100}
                  disabled={discountType === "NONE"}
                  value={discountValue}
                  onChange={(e) => setDiscountValue(Number(e.target.value))}
                  className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg focus:outline-none font-mono text-slate-900 disabled:bg-slate-50 disabled:text-slate-400 disabled:cursor-not-allowed"
                  placeholder={discountType === "PERCENT" ? "0 - 100" : "0"}
                />
              )}
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <DollarSign className="size-3" /> Kena PPN
              </label>
              <label className="flex items-center gap-2 h-[34px] px-3 bg-white border border-slate-200 rounded-lg cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={isTaxable}
                  onChange={(e) => setIsTaxable(e.target.checked)}
                  className="size-4 accent-blue-600 cursor-pointer"
                />
                <span className="text-xs text-slate-700">
                  {isTaxable ? "Dikenai PPN" : "Bebas PPN"}
                </span>
              </label>
              <p className="text-xs text-slate-500">
                Produk ber-flag PPN memisahkan DPP &amp; PPN Keluaran pada nota
                POS.
              </p>
            </div>
          </div>

          {/* Tombol Kontrol */}
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
              className="bg-blue-600 hover:bg-blue-700 text-white text-xs h-8 shadow-xs cursor-pointer"
            >
              {isLoading ? (
                <span className="flex items-center gap-1.5">
                  <Loader2 className="size-3 animate-spin" /> Memproses...
                </span>
              ) : editingProduct ? (
                "Perbarui Spesifikasi"
              ) : (
                "Simpan Barang"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
