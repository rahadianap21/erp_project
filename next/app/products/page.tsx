"use client";

import { apiDelete, messageFromError } from "@/lib/api-client";
import { ensureSession, readPermissionFlags } from "@/lib/permissions";
import React, { useState, useEffect } from "react";
import Cookies from "js-cookie";
import dynamic from "next/dynamic";
import { ProductForm } from "./components/product-form";
import { toast } from "sonner";
import { ForbiddenView } from "@/components/forbidden-view";
import { Package } from "lucide-react";

const ProductTable = dynamic(
  () => import("./components/product-table").then((mod) => mod.ProductTable),
  {
    ssr: false,
    loading: () => (
      <div className="w-full min-h-[30vh] flex items-center justify-center text-slate-500 text-xs font-medium">
        <span className="animate-pulse">
          Menyiapkan katalog data master barang logistik...
        </span>
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

interface Product {
  id: number;
  sku: string;
  barcode: string;
  name: string;
  description: string;
  category_id: number;
  supplier_id: number;
  selling_unit_id: number;
  selling_unit_kode?: string;
  selling_unit_name?: string;
  purchase_unit_id: number;
  purchase_unit_kode?: string;
  purchase_unit_name?: string;
  conversion_rate?: number;
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

export default function MasterProductsCRUDPage() {
  const [httpStatus, setHttpStatus] = useState<number>(200);
  const [permissions, setPermissions] = useState({
    create: false,
    update: false,
    delete: false,
  });
  const [refreshTrigger, setRefreshTrigger] = useState<number>(0);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);
  const [productToDelete, setProductToDelete] = useState<Product | null>(null);
  const [isDeleting, setIsDeleting] = useState<boolean>(false);

  const authToken = Cookies.get("session_active");

  const loadPermissions = () => {
    if (!ensureSession()) return;
    setPermissions(readPermissionFlags("products"));
  };

  const handleDeleteConfirm = async () => {
    if (!productToDelete) return;
    setIsDeleting(true);
    const targetedName = productToDelete.name;

    try {
      await apiDelete(`/api/products?id=${productToDelete.id}`);

      setProductToDelete(null);
      setTimeout(() => {
        toast.success("Barang Berhasil Dihapus", {
          description: `Item master ${targetedName} resmi dinonaktifkan dari katalog gudang.`,
          duration: 2000,
        });
      }, 100);
      setRefreshTrigger((prev) => prev + 1);
    } catch (error) {
      toast.error("Gagal Menghapus Item", {
        description: messageFromError(error),
      });
    } finally {
      setIsDeleting(false);
    }
  };

  const handleFormSuccess = (mode: "create" | "update", name: string) => {
    setEditingProduct(null);
    setRefreshTrigger((prev) => prev + 1);
    toast.success(
      mode === "create" ? "Barang Ditambahkan" : "Spesifikasi Diperbarui",
      {
        description: `Item master ${name} sukses dimasukkan ke kluster inventaris.`,
        duration: 2000,
      }
    );
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
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 border-b pb-4">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold tracking-tight text-slate-900 flex items-center gap-2">
            <Package className="size-5 text-blue-600" /> Katalog Master Barang &
            Inventaris
          </h1>
          <p className="text-xs text-muted-foreground">
            Pusat kontrol standarisasi penomoran SKU, nilai HPP, serta
            pengawasan ambang safety stock gudang pusat.
          </p>
        </div>
      </div>

      {(permissions.create || editingProduct) && (
        <ProductForm
          token={authToken}
          editingProduct={editingProduct}
          onSuccess={handleFormSuccess}
          onCancelEdit={() => setEditingProduct(null)}
        />
      )}

      <ProductTable
        canUpdate={permissions.update}
        canDelete={permissions.delete}
        refreshTrigger={refreshTrigger}
        onEditClick={(p) => setEditingProduct(p)}
        onDeleteClick={(p) => setProductToDelete(p)}
        onForbidden={() => setHttpStatus(403)}
      />

      <AlertDialog
        open={productToDelete !== null}
        onOpenChange={(open) => {
          if (!open && !isDeleting) setProductToDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Apakah Anda benar-benar yakin?</AlertDialogTitle>
            <AlertDialogDescription>
              Tindakan ini tidak dapat dibatalkan. Item logistik dengan rujukan
              kaku berikut akan dicabut permanen dari status inventaris aktif:
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="my-2 p-3 bg-muted rounded-md border text-xs font-mono">
            <p className="font-semibold text-slate-900 uppercase">
              {productToDelete?.name}
            </p>
            <p className="text-xs text-muted-foreground mt-0.5">
              SKU: {productToDelete?.sku} | Barcode:{" "}
              {productToDelete?.barcode || "-"}
            </p>
          </div>

          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isDeleting}
              onClick={() => setProductToDelete(null)}
            >
              Batal
            </Button>
            <Button
              variant="destructive"
              disabled={isDeleting}
              onClick={handleDeleteConfirm}
            >
              {isDeleting ? "Menghapus..." : "Ya, Hapus Barang"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  );
}
