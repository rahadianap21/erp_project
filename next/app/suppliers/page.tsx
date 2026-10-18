"use client";

import React, { useState, useEffect } from "react";
import Cookies from "js-cookie";
import { useRouter } from "next/navigation";
import dynamic from "next/dynamic";
import SupplierForm from "./components/supplier-form";
import { toast } from 'sonner';
import { ForbiddenView } from "@/components/forbidden-view";

const SupplierTable = dynamic(
    () => import("./components/supplier-table").then((mod) => mod.SupplierTable),
    {
        ssr: false,
        loading: () => (
            <div className="w-full min-h-[40vh] flex items-center justify-center gap-2 text-slate-500 font-medium text-xs">
                <span className="animate-pulse">Menyiapkan komponen data vendor...</span>
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

interface Supplier {
    id: number;
    name: string;
    pic_name: string;
    phone: string;
    email: string;
    address: string;
    village_id: string;
    village_name: string;
    postal_code: string;
    district_name: string;
    regency_name: string;
    province_name: string;
    is_active: boolean;
}

export default function SuppliersCRUDPage() {
    const [suppliers, setSuppliers] = useState<Supplier[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [httpStatus, setHttpStatus] = useState<number>(200);
    const [permissions, setPermissions] = useState({ create: false, update: false, delete: false });
    const [refreshTrigger, setRefreshTrigger] = useState(0);
    const [editingSupplier, setEditingSupplier] = useState<Supplier | null>(null);
    const [supplierToDelete, setSupplierToDelete] = useState<Supplier | null>(null);
    const [isDeleting, setIsDeleting] = useState<boolean>(false);

    const authToken = Cookies.get("session_active");

    // Memuat izin fitur modular berdasarkan kuki penugasan JWT
    const loadPermissions = () => {
        const savedPermissions = Cookies.get("user_permissions");
        const authToken = Cookies.get("session_active");

        // 🌟 PERBAIKAN DEFENSIF MUTLAK:
        // Jika salah satu komponen kuki otorisasi hilang/kedaluwarsa saat idle, paksa bersihkan dan tendang ke halaman login
        if (!savedPermissions || !authToken) {
            Cookies.remove("session_active");
            Cookies.remove("user_permissions");

            toast.error("Sesi Anda Telah Berakhir", {
                description: "Silakan lakukan login ulang untuk mengamankan kredensial rantai pasok Anda.",
                duration: 4000,
            });

            // Alihkan paksa rute navigasi jendela browser secara mutlak ke pintu login utama
            window.location.href = "/";
            return;
        }

        try {
            const parsed: string[] = JSON.parse(decodeURIComponent(savedPermissions));
            if (Array.isArray(parsed)) {
                setPermissions({
                    create: parsed.includes("suppliers:create"),
                    update: parsed.includes("suppliers:update"),
                    delete: parsed.includes("suppliers:delete"),
                });
            }
        } catch (e) {
            console.error("Gagal membaca cookie permissions:", e);
        }
    };

    // Eksekusi Konfirmasi Penghapusan Data Soft-Delete
    const handleDeleteConfirm = async () => {
        if (!supplierToDelete) return;

        setIsDeleting(true);
        const targetedName = supplierToDelete.name;
        try {
            const response = await fetch(`/api/suppliers?id=${supplierToDelete.id}`, {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`,
                },
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.message || 'Gagal menghapus data supplier');
            }

            setSupplierToDelete(null);
            setTimeout(() => {
                toast.success("Vendor Berhasil Dihapus", {
                    description: `Data dari mitra ${targetedName} telah dihapus dari sistem.`,
                    duration: 2000,
                });
            }, 100);

            setRefreshTrigger((prev) => prev + 1);
        } catch (error: any) {
            toast.error("Gagal Menghapus Vendor", {
                description: error.message || "Terjadi kesalahan koneksi atau masalah internal server.",
            });
        } finally {
            setIsDeleting(false);
        }
    };

    // Akses Balik Sukses Mutasi Data Form
    const handleFormSuccess = (mode: 'create' | 'update', name: string) => {
        setEditingSupplier(null);
        setRefreshTrigger((prev) => prev + 1);

        toast.success(mode === 'create' ? "Supplier Ditambahkan" : "Supplier Diperbarui", {
            description: `Data dari vendor ${name} berhasil ${mode === 'create' ? 'disimpan ke' : 'diperbarui dalam'} sistem.`,
            duration: 2000,
        });
    };

    useEffect(() => {
        // Jalankan pembacaan kuki secara instan saat pertama kali dimuat
        loadPermissions();

        // Antisipasi Hot Module Replacement (HMR):
        // Jika Next.js melakukan Fast Refresh saat Anda menekan Ctrl+S di VS Code,
        // interval pendek ini akan mengembalikan state perizinan dalam hitungan milidetik
        const hmrGuardInterval = setInterval(() => {
            const savedPermissions = Cookies.get("user_permissions");
            if (savedPermissions && !permissions.create && !permissions.update) {
                loadPermissions();
            }
        }, 1000);

        // Pasang pendeteksi interaksi jendela fokus harian karyawan
        window.addEventListener("focus", loadPermissions);

        // Bersihkan seluruh listener saat komponen dihancurkan (Memory-Leak Safety)
        return () => {
            clearInterval(hmrGuardInterval);
            window.removeEventListener("focus", loadPermissions);
        };
    }, [refreshTrigger]);

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

            {/* Panel Form Pengisian Data Vendor */}
            {(permissions.create || editingSupplier) && (
                <SupplierForm
                    token={authToken}
                    editingSupplier={editingSupplier}
                    onSuccess={handleFormSuccess}
                    onCancelEdit={() => setEditingSupplier(null)}
                />
            )}

            {/* 🌟 INTEGRASI PARALEL: Alirkan state search dan setSearch langsung ke komponen SupplierTable */}
            <SupplierTable
                canUpdate={permissions.update}
                canDelete={permissions.delete}
                refreshTrigger={refreshTrigger}
                onEditClick={(user) => setEditingSupplier(user)}
                onDeleteClick={(user) => setSupplierToDelete(user)}
                onForbidden={() => setHttpStatus(403)}
            />
            {/* Dialog Pop-Up Peringatan Penghapusan Permanen */}
            <AlertDialog open={supplierToDelete !== null} onOpenChange={(open) => { if (!open && !isDeleting) setSupplierToDelete(null); }}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Apakah Anda benar-benar yakin?</AlertDialogTitle>
                        <AlertDialogDescription>
                            Tindakan ini tidak dapat dibatalkan. Vendor kemitraan dengan identitas berikut akan dihapus secara permanen dari database:
                        </AlertDialogDescription>
                    </AlertDialogHeader>

                    <div className="my-2 p-3 bg-muted rounded-md border text-xs font-mono">
                        <p className="font-semibold text-slate-900 uppercase">{supplierToDelete?.name}</p>
                        <p className="text-xs text-muted-foreground mt-0.5">PIC: {supplierToDelete?.pic_name} | {supplierToDelete?.email}</p>
                    </div>

                    <AlertDialogFooter>
                        <Button variant="outline" disabled={isDeleting} onClick={() => setSupplierToDelete(null)}>Batal</Button>
                        <Button variant="destructive" disabled={isDeleting} onClick={handleDeleteConfirm}>{isDeleting ? "Menghapus..." : "Ya, Hapus Vendor"}</Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </main>
    );
}