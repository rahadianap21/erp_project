"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { ChevronDown, ChevronUp, Loader2, FolderTree, FileText, Link2 } from "lucide-react";
import { toast } from "sonner";

interface ProductCategory {
    id: number;
    name: string;
    slug: string;
    description: string;
    parent_id: number | null;
    is_active: boolean;
}

interface ProductCategoryFormProps {
    token: string | undefined;
    editingCategory: ProductCategory | null;
    onSuccess: (mode: "create" | "update", name: string) => void;
    onCancelEdit: () => void;
}

export function ProductCategoryForm({ token, editingCategory, onSuccess, onCancelEdit }: ProductCategoryFormProps) {
    const [isExpanded, setIsExpanded] = useState<boolean>(false);
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [parentOptions, setParentOptions] = useState<ProductCategory[]>([]);

    // State Form
    const [name, setName] = useState<string>("");
    const [slug, setSlug] = useState<string>("");
    const [description, setDescription] = useState<string>("");
    const [parentId, setParentId] = useState<string>("");
    const [isActive, setIsActive] = useState<boolean>(true);

    // Ambil daftar kategori induk untuk pilihan dropdown
    const fetchParentOptions = async () => {
        try {
            const res = await fetch(`/api/categories?limit=100`, {
                headers: { "Authorization": `Bearer ${token}` }
            });
            const result = await res.json();
            if (res.ok && result.data) {
                // Saring agar kategori yang sedang diedit tidak bisa memilih dirinya sendiri sebagai induk
                const options = editingCategory
                    ? result.data.filter((c: ProductCategory) => c.id !== editingCategory.id)
                    : result.data;
                setParentOptions(options);
            }
        } catch (err) {
            console.error("Gagal memuat opsi induk kategori:", err);
        }
    };

    useEffect(() => {
        if (token) fetchParentOptions();
    }, [token, editingCategory]);

    useEffect(() => {
        if (editingCategory) {
            setName(editingCategory.name);
            setSlug(editingCategory.slug);
            setDescription(editingCategory.description || "");
            setParentId(editingCategory.parent_id ? String(editingCategory.parent_id) : "");
            setIsActive(editingCategory.is_active);
            setIsExpanded(true);
        } else {
            resetForm();
        }
    }, [editingCategory]);

    const resetForm = () => {
        setName(""); setSlug(""); setDescription(""); setParentId(""); setIsActive(true);
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsLoading(true);

        const isEditMode = editingCategory !== null;
        const url = isEditMode ? `/api/categories?id=${editingCategory.id}` : `/api/categories`;
        const method = isEditMode ? 'PUT' : 'POST';

        try {
            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({
                    name,
                    slug: slug || undefined, // Kosongkan agar backend meng-generate otomatis lewat usecase Go
                    description,
                    parent_id: parentId ? Number(parentId) : null,
                    is_active: isActive
                }),
            });

            const errorData = await response.json().catch(() => ({}));
            if (!response.ok) throw new Error(errorData.message || 'Gagal memproses data kategori.');

            resetForm();
            setIsExpanded(false);
            onSuccess(isEditMode ? 'update' : 'create', name);
        } catch (error: any) {
            toast.error("Gagal Menyimpan", { description: error.message });
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="bg-white rounded-xl shadow-2xs border border-slate-200 overflow-hidden transition-all duration-300">
            <button
                type="button"
                onClick={() => !editingCategory && setIsExpanded(!isExpanded)}
                disabled={!!editingCategory}
                className="w-full px-6 py-3.5 flex items-center justify-between bg-slate-50/80 hover:bg-slate-100/60 transition-colors text-left cursor-pointer"
            >
                <span className="text-xs font-bold uppercase tracking-wider text-slate-700 flex items-center gap-2">
                    <FolderTree className="size-4 text-blue-600" />
                    {editingCategory ? `Edit Kategori: ${editingCategory.name}` : "Tambah Kategori Baru"}
                </span>
                {!editingCategory && (isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />)}
            </button>

            <div className={`transition-all duration-300 ease-in-out ${isExpanded ? "max-h-[800px] p-6 border-t border-slate-100" : "max-h-0 opacity-0 pointer-events-none"}`}>
                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1"><FolderTree className="size-3" /> Nama Kategori</label>
                            <input type="text" required value={name} onChange={(e) => setName(e.target.value)} className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900" placeholder="Contoh: Bahan Baku" />
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1"><Link2 className="size-3" /> Slug Kustom (Opsional)</label>
                            <input type="text" value={slug} onChange={(e) => setSlug(e.target.value)} className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 font-mono" placeholder="Otomatis jika kosong" />
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1"><FolderTree className="size-3" /> Induk Kategori (Hierarki)</label>
                            <select value={parentId} onChange={(e) => setParentId(e.target.value)} className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 cursor-pointer">
                                <option value="">--- Jadikan Kategori Utama / Tanpa Induk ---</option>
                                {parentOptions.map(opt => (
                                    <option key={opt.id} value={opt.id}>{opt.name}</option>
                                ))}
                            </select>
                        </div>
                    </div>

                    <div className="space-y-1.5">
                        <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1"><FileText className="size-3" /> Deskripsi Kategori</label>
                        <textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={2} className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 resize-none" placeholder="Tulis catatan penjelasan klasifikasi barang di sini..." />
                    </div>

                    <div className="flex justify-end space-x-3 pt-3 border-t border-slate-100">
                        <Button type="button" variant="outline" size="sm" onClick={() => { onCancelEdit(); resetForm(); setIsExpanded(false); }} className="text-xs h-8 cursor-pointer">Batal</Button>
                        <Button type="submit" size="sm" disabled={isLoading} className="bg-blue-600 hover:bg-blue-700 text-white text-xs h-8 shadow-xs cursor-pointer">
                            {isLoading ? <span className="flex items-center gap-1.5"><Loader2 className="size-3 animate-spin" /> Memproses...</span> : editingCategory ? "Perbarui Kategori" : "Simpan Kategori"}
                        </Button>
                    </div>
                </form>
            </div>
        </div>
    );
}
