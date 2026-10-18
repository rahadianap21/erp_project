"use client";

import { Edit2, Trash2, Package, Layers, Search, Barcode, ShieldAlert, Building2, Tag } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useDebounceFetch } from "@/hooks/use-api-list";
import { DataTableToolbar } from "@/components/data-table-toolbar";
import { DataTablePagination } from "@/components/data-table-pagination";
import React from "react";

export interface Product {
    id: number;
    sku: string;
    barcode: string;
    name: string;
    description: string;
    category_id: number;
    category_name: string;
    supplier_id: number;
    supplier_name: string;
    selling_unit_id: number;
    selling_unit_kode?: string;
    selling_unit_name?: string;
    purchase_unit_id: number;
    purchase_unit_kode?: string;
    purchase_unit_name?: string;     // Metadata nama dari left join Go backend
    purchase_price: number;
    selling_price: number;
    min_stock_level: number;
    current_stock: number;
    avg_cost: number;
    discount_type?: string;
    discount_value?: number;
    is_taxable?: boolean;
    is_active: boolean;
}

interface ProductTableProps {
    canUpdate: boolean;
    canDelete: boolean;
    refreshTrigger: number;
    onDeleteClick: (prod: Product) => void;
    onEditClick: (prod: Product) => void;
    onForbidden?: () => void;
}

export function ProductTable({ canUpdate, canDelete, refreshTrigger, onDeleteClick, onEditClick, onForbidden }: ProductTableProps) {
    const {
        search, setSearch, page, limit, setLimit, data: products, total, loading, handleReset, isFiltered,
        hasNext, hasPrev, handleNextPage, handlePrevPage, filters, updateFilter, httpStatus
    } = useDebounceFetch<Product>({
        endpoint: "/api/products",
        delay: 500,
        initialFilters: { category_id: "" },
        refreshTrigger: refreshTrigger
    });

    // Teruskan sinyal 403 dari hook ke halaman induk agar ForbiddenView tampil,
    // bukan tabel kosong yang membingungkan pengguna tanpa izin.
    React.useEffect(() => {
        if (httpStatus === 403) onForbidden?.();
    }, [httpStatus, onForbidden]);

    // Helper Pemformat Mata Uang Akuntansi IDR Rupiah
    const formatIDR = (num: number) => {
        return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(num);
    };

    // Ringkasan diskon default produk yang akan terpasang otomatis di POS.
    const formatDiscount = (p: Product) => {
        const type = p.discount_type || "NONE";
        const value = p.discount_value || 0;
        if (type === "NONE" || value <= 0) return null;
        if (type === "PERCENT") return `${value}%`;
        return formatIDR(value);
    };

    const showActionColumn = canUpdate || canDelete;

    return (
        <div className="space-y-4">
            {/* Filter Input Search */}
            <DataTableToolbar
                search={search}
                setSearch={setSearch}
                placeholder="Cari berdasarkan nama barang atau kode SKU..."
                loading={loading}
                isFiltered={isFiltered}
                onReset={handleReset}
                counterIcon={<Package className="size-3.5 text-slate-400" />}
                counterLabel={`Total ${total} item master terdaftar`}
            >
                <div className="flex items-center gap-2">
                    <select
                        value={filters.category_id ?? ""}
                        onChange={(e) => updateFilter("category_id", e.target.value || "")}
                        className="text-xs h-9 px-3 bg-white border border-slate-200 rounded-lg outline-none cursor-pointer focus:border-blue-500 font-medium text-slate-700"
                    >
                        <option value="">Semua Kategori</option>
                        <option value="1">Bahan Baku</option>
                        <option value="2">Kemasan</option>
                        <option value="3">Aksesori Kemasan</option>
                        <option value="5">Label & Coding</option>
                        <option value="6">Pelumas & Bahan Tambahan</option>
                        <option value="7">Spare Part Produksi</option>
                        <option value="8">Tools & Maintenance</option>
                        <option value="9">Finished Goods</option>
                        <option value="10">Komponen Mekanikal</option>
                        <option value="11">Frozen Food</option>
                        <option value="12">Buah & Sayur</option>
                        <option value="13">Daging & Ayam</option>
                        <option value="14">Ikan & Seafood</option>
                        <option value="15">Nugget</option>
                        <option value="16">Sosis</option>
                    </select>
                </div>
            </DataTableToolbar>

            {/* Area Konten Tabel Fisik */}
            <div className="border rounded-xl shadow-xs overflow-hidden bg-white relative">
                <Table>
                    <TableHeader className="bg-slate-50/70">
                        <TableRow>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/4">Spesifikasi Item / SKU</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/5">Rumpun Kategori & Satuan</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/4 text-center">Satuan (UoM)</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-right w-28">Harga Beli (HPP)</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-right w-28">H. Pokok Rata²</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-right w-28">Harga Jual</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-center w-24">Diskon</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-center w-24">PPN</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-center w-28">Stok Real-Time</TableHead>
                            {showActionColumn && <TableHead className="text-right font-bold text-xs w-20">Aksi</TableHead>}
                        </TableRow>
                    </TableHeader>

                    <TableBody>
                        {loading && products.length === 0 ? (
                            Array.from({ length: 6 }).map((_, idx) => (
                                <TableRow key={`prod-row-skeleton-${idx}`}><TableCell colSpan={showActionColumn ? 10 : 9} className="py-4"><Skeleton className="h-4 w-full" /></TableCell></TableRow>
                            ))
                        ) : products.length === 0 ? (
                            <TableRow>
                                <TableCell
                                    colSpan={showActionColumn ? 10 : 9}
                                    className="h-32 text-center text-muted-foreground font-medium text-xs"
                                >
                                    <div className="flex flex-col items-center justify-center gap-2 py-4">
                                        <Search className="size-5 text-muted-foreground/50" />
                                        <p>
                                            {isFiltered
                                                ? "Data tidak ditemukan. Coba sesuaikan kata kunci atau filter pencarian Anda."
                                                : "Tidak ada data produk yang terindeks di sistem."}
                                        </p>
                                        {isFiltered && (
                                            <Button
                                                type="button"
                                                variant="outline"
                                                size="sm"
                                                onClick={handleReset}
                                                className="mt-1 h-7 text-[11px] cursor-pointer"
                                            >
                                                Bersihkan Filter
                                            </Button>
                                        )}
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : (
                            products.map((p) => (
                                <TableRow key={p.id} className={`hover:bg-muted/20 transition-opacity border-b duration-200 ${loading ? "opacity-50 pointer-events-none" : ""}`}>
                                    <TableCell className="text-xs py-3 font-medium">
                                        <div className="space-y-0.5">
                                            <p className="font-bold text-slate-900 uppercase tracking-tight line-clamp-1">{p.name}</p>
                                            <div className="flex flex-wrap items-center gap-2 text-[10px] text-muted-foreground font-mono">
                                                <span className="bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded border border-slate-200">SKU: {p.sku}</span>
                                                {p.barcode && <span className="flex items-center gap-0.5"><Barcode className="size-2.5" /> {p.barcode}</span>}
                                            </div>
                                        </div>
                                    </TableCell>

                                    <TableCell className="text-xs py-3">
                                        <div className="space-y-0.5">
                                            <p className="text-slate-800 font-semibold flex items-center gap-1 uppercase text-[11px]"><Layers className="size-3 text-slate-400" /> {p.category_name || "Kategori Umum"}</p>
                                            <p className="text-[10px] text-muted-foreground flex items-center gap-1"><Building2 className="size-2.5" /> {p.supplier_name || "Vendor Utama"}</p>
                                        </div>
                                    </TableCell>

                                    <TableCell className="text-xs py-3 text-center">
                                        <div className="inline-flex items-center gap-1.5 bg-slate-50 p-1 rounded-lg border border-slate-200/60 font-mono text-[9px] font-bold">
                                            <div className="px-1.5 py-0.5 rounded bg-emerald-50 border border-emerald-200 text-emerald-700" title="Satuan Eceran Transaksi Kasir">
                                                BELI: {p.purchase_unit_kode || "PCS"}
                                            </div>
                                            <div className="text-slate-300 select-none">|</div>
                                            <div className="px-1.5 py-0.5 rounded bg-blue-50 border border-blue-200 text-blue-700" title="Satuan Grosir Pembelian Logistik">
                                                JUAL: {p.selling_unit_kode || "BOX"}
                                            </div>
                                        </div>
                                    </TableCell>

                                    <TableCell className="text-xs py-3 text-right font-mono text-slate-500 font-medium">{formatIDR(p.purchase_price)}</TableCell>
                                    <TableCell className="text-xs py-3 text-right font-mono text-slate-500 font-medium" title="Rata-rata harga pokok per satuan jual dari seluruh penerimaan">{formatIDR(p.avg_cost)}</TableCell>
                                    <TableCell className="text-xs py-3 text-right font-mono font-bold text-slate-900">{formatIDR(p.selling_price)}</TableCell>

                                    <TableCell className="text-xs py-3 text-center">
                                        {formatDiscount(p) ? (
                                            <span className="inline-flex items-center gap-1 text-[11px] bg-amber-50 text-amber-700 border border-amber-200 px-2 py-0.5 rounded-lg font-mono font-bold" title={p.discount_type === "PERCENT" ? "Diskon persen default" : "Diskon nominal default"}>
                                                <Tag className="size-2.5" /> {formatDiscount(p)}
                                            </span>
                                        ) : (
                                            <span className="text-[10px] text-slate-400">-</span>
                                        )}
                                    </TableCell>

                                    {/* Kolom Status Pengenaan PPN */}
                                    <TableCell className="text-xs py-3 text-center">
                                        {p.is_taxable === false ? (
                                            <span className="inline-flex items-center text-[11px] bg-slate-100 text-slate-500 border border-slate-200 px-2 py-0.5 rounded-lg font-semibold" title="Bebas PPN: tidak memisahkan DPP & PPN Keluaran">
                                                Bebas
                                            </span>
                                        ) : (
                                            <span className="inline-flex items-center text-[11px] bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-lg font-semibold" title="Kena PPN: memisahkan DPP & PPN Keluaran pada nota">
                                                Kena
                                            </span>
                                        )}
                                    </TableCell>

                                    {/* Kolom Status Pengawasan Kritis Ambang Safety Stock */}
                                    <TableCell className="text-xs py-3 text-center">
                                        {p.current_stock <= p.min_stock_level ? (
                                            <div className="inline-flex flex-col items-center gap-0.5">
                                                <span className="inline-flex items-center gap-1 text-[11px] bg-rose-50 text-rose-600 border border-rose-200 px-2 py-0.5 rounded-lg font-mono font-bold animate-pulse">
                                                    {p.current_stock} {p.selling_unit_kode || "PCS"}
                                                </span>
                                                <span className="text-[9px] font-bold text-rose-500 flex items-center gap-0.5"><ShieldAlert className="size-2.5" /> KRITIS (Min: {p.min_stock_level})</span>
                                            </div>
                                        ) : (
                                            <span className="inline-flex items-center gap-1 text-[11px] bg-emerald-50 text-emerald-600 border border-emerald-200 px-2 py-0.5 rounded-lg font-mono font-bold">
                                                {p.current_stock} {p.selling_unit_kode || "PCS"}
                                            </span>
                                        )}
                                    </TableCell>

                                    {/* Kolom Kendali Aksi Edit & Hapus */}
                                    {showActionColumn && (
                                        <TableCell className="text-right py-3">
                                            <div className="inline-flex gap-1.5 justify-end">
                                                {canUpdate && <button type="button" onClick={() => onEditClick(p)} className="p-1.5 border border-slate-200 rounded-lg bg-white text-slate-700 hover:bg-slate-50 transition-colors shadow-3xs cursor-pointer" title="Ubah spesifikasi"><Edit2 className="size-3.5" /></button>}
                                                {canDelete && <button type="button" onClick={() => onDeleteClick(p)} className="p-1.5 border border-slate-200 rounded-lg bg-white text-rose-600 hover:bg-rose-50 hover:border-rose-200 transition-colors shadow-3xs cursor-pointer" title="Hapus dari katalog"><Trash2 className="size-3.5" /></button>}
                                            </div>
                                        </TableCell>
                                    )}
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>
            {/* Pagination Controls */}
            <DataTablePagination
                page={page}
                limit={limit}
                setLimit={setLimit}
                total={total}
                rowCount={products.length}
                hasNext={hasNext}
                hasPrev={hasPrev}
                onNextPage={handleNextPage}
                onPrevPage={handlePrevPage}
                loading={loading}
            />
        </div>
    );
}

export default ProductTable;
