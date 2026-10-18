"use client";

import { apiDelete } from "@/lib/api-client";
import { ensureSession, readPermissionFlags } from "@/lib/permissions";
import React, { useState, useEffect } from "react";
import Cookies from "js-cookie";
import dynamic from "next/dynamic";
import { ProductCategoryForm } from "./components/product-category-form";
import { toast } from 'sonner';
import { ForbiddenView } from "@/components/forbidden-view";
import { FolderTree } from "lucide-react";

// 🌟 INTEGRASI PORTAL: Ekstrak secara kaku Named Export dari komponen anak
const ProductCategoryTable = dynamic(
    () => import("./components/product-category-table").then((mod) => mod.ProductCategoryTable),
    {
        ssr: false,
        loading: () => (
            <div className="w-full min-h-[30vh] flex items-center justify-center text-slate-500 text-xs font-medium">
                <span className="animate-pulse">Menyiapkan panel klasifikasi kategori barang...</span>
            </div>
        ),
    }
);

import {
    AlertDialog,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

interface ProductCategory {
    id: number;
    name: string;
    slug: string;
    description: string;
    parent_id: number | null;
    parent_name: string;
    is_active: boolean;
}

export default function ProductCategoriesCRUDPage() {
    const [loading, setLoading] = useState<boolean>(true);
    const [httpStatus, setHttpStatus] = useState<number>(200);
    const [permissions, setPermissions] = useState({ create: false, update: false, delete: false });
    const [refreshTrigger, setRefreshTrigger] = useState<number>(0);
    const [editingCategory, setEditingCategory] = useState<ProductCategory | null>(null);
    const [categoryToDelete, setCategoryToDelete] = useState<ProductCategory | null>(null);
    const [isDeleting, setIsDeleting] = useState<boolean>(false);
    const [error, setError] = useState<string | null>(null);

    const authToken = Cookies.get("session_active");

    const loadPermissions = () => {
        if (!ensureSession()) return;
        setPermissions(readPermissionFlags("categories"));
    };

    const handleDeleteConfirm = async () => {
        if (!categoryToDelete) return;
        setIsDeleting(true);
        const targetedName = categoryToDelete.name;

        try {
            await apiDelete(`/api/categories?id=${categoryToDelete.id}`);

            setCategoryToDelete(null);
            setTimeout(() => {
                toast.success("Kategori Berhasil Dihapus", {
                    description: `Klasifikasi barang ${targetedName} resmi dicabut.`,
                    duration: 2000,
                });
            }, 100);
            setRefreshTrigger((prev) => prev + 1);
        } catch (error: any) {
            toast.error("Gagal Menghapus Kategori", { description: error.message });
        } finally {
            setIsDeleting(false);
        }
    };

    const handleFormSuccess = (mode: 'create' | 'update', name: string) => {
        setEditingCategory(null);
        setRefreshTrigger((prev) => prev + 1);
        toast.success(mode === 'create' ? "Kategori Ditambahkan" : "Kategori Diperbarui", {
            description: `Kelompok barang ${name} berhasil disimpan ke basis sistem.`,
            duration: 2000,
        });
    };
    useEffect(() => {
        loadPermissions();

        // Setiap kali pengguna kembali fokus membuka tab aplikasi setelah idle, 
        // sistem otomatis memvalidasi apakah tokennya masih hidup atau sudah hangus
        window.addEventListener("focus", loadPermissions);
        return () => window.removeEventListener("focus", loadPermissions);
    }, []);

    if (httpStatus === 403) {
        const handleReset = () => {
            setHttpStatus(200);
            loadPermissions();
            setRefreshTrigger((prev) => prev + 1);
        };
        return <ForbiddenView reset={handleReset} />;
    }

    return (
        <main className="flex flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8 w-full">
            {error && (
                <div className="bg-destructive/10 border border-destructive/20 text-destructive p-4 rounded-xl text-xs font-medium">
                    {error}
                </div>
            )}

            {(permissions.create || editingCategory) && (
                <ProductCategoryForm
                    token={authToken}
                    editingCategory={editingCategory}
                    onSuccess={handleFormSuccess}
                    onCancelEdit={() => setEditingCategory(null)}
                />
            )}

            <ProductCategoryTable
                canUpdate={permissions.update}
                canDelete={permissions.delete}
                refreshTrigger={refreshTrigger}
                onEditClick={(c) => setEditingCategory(c)}
                onDeleteClick={(c) => setCategoryToDelete(c)}
                onForbidden={() => setHttpStatus(403)}
            />

            <AlertDialog open={categoryToDelete !== null} onOpenChange={(open) => { if (!open && !isDeleting) setCategoryToDelete(null); }}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Apakah Anda benar-benar yakin?</AlertDialogTitle>
                        <AlertDialogDescription>
                            Tindakan ini tidak dapat dibatalkan. Kategori dengan rujukan identitas berikut akan dicabut secara permanen:
                        </AlertDialogDescription>
                    </AlertDialogHeader>

                    <div className="my-2 p-3 bg-muted rounded-md border text-xs font-mono">
                        <p className="font-semibold text-slate-900 uppercase">{categoryToDelete?.name}</p>
                        <p className="text-xs text-muted-foreground mt-0.5">Slug: {categoryToDelete?.slug}</p>
                    </div>

                    <AlertDialogFooter>
                        <Button variant="outline" disabled={isDeleting} onClick={() => setCategoryToDelete(null)}>Batal</Button>
                        <Button variant="destructive" disabled={isDeleting} onClick={handleDeleteConfirm}>{isDeleting ? "Menghapus..." : "Ya, Hapus Kategori"}</Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </main>
    );
}
