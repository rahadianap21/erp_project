"use client";

import { apiDelete } from "@/lib/api-client";
import { ensureSession, readPermissionFlags } from "@/lib/permissions";
import React, { useState, useEffect } from "react";
import Cookies from "js-cookie";
import RoleForm from "./components/role-form";
import { RoleTable } from "./components/role-table";
import { ForbiddenView } from "@/components/forbidden-view";
import { toast } from 'sonner';
import {
    AlertDialog,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

interface Role {
    id: number;
    name: string;
    permissions: string[];
}

export default function RolesCRUDPage() {
    const [roles, setRoles] = useState<Role[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [httpStatus, setHttpStatus] = useState<number>(200);
    const [permissions, setPermissions] = useState({ create: false, update: false, delete: false });
    const [refreshTrigger, setRefreshTrigger] = useState(0);
    const [editingRole, setEditingRole] = useState<Role | null>(null);

    const [roleToDelete, setRoleToDelete] = useState<Role | null>(null);
    const [isDeleting, setIsDeleting] = useState<boolean>(false);

    const authToken = Cookies.get("session_active");

    const loadPermissions = () => {
        if (!ensureSession()) return;
        setPermissions(readPermissionFlags("roles"));
    };

    const handleDeleteConfirm = async () => {
        if (!roleToDelete) return;

        setIsDeleting(true);
        const targetedName = roleToDelete.name;
        try {
            await apiDelete(`/api/roles?id=${roleToDelete.id}`);

            setRoleToDelete(null);
            setTimeout(() => {
                toast.success("Peran Berhasil Dihapus", {
                    description: `Peran ${targetedName} telah dihapus dari sistem.`,
                    duration: 2000,
                });
            }, 100);

            // Trigger refresh tabel peran secara reaktif
            setRefreshTrigger((prev) => prev + 1);
        } catch (error: any) {
            toast.error("Gagal Menghapus Peran", {
                description: error.message || "Terjadi kesalahan koneksi atau masalah internal server.",
            });
        } finally {
            setIsDeleting(false);
        }
    };

    // Callback terpusat untuk kesuksesan mutasi form (POST / PUT)
    const handleFormSuccess = (mode: 'create' | 'update', name: string) => {
        setEditingRole(null);
        setRefreshTrigger((prev) => prev + 1);

        toast.success(mode === 'create' ? "Peran Ditambahkan" : "Peran Diperbarui", {
            description: `Peran ${name} berhasil ${mode === 'create' ? 'disimpan ke' : 'diperbarui dalam'} sistem.`,
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

            {/* 🔄 LANGKAH PERBAIKAN 2: Form harus dirender jika user punya hak 'create' ATAU sedang meng-edit data */}
            {(permissions.create || editingRole) && (
                <RoleForm
                    token={authToken}
                    onSuccess={handleFormSuccess} // Gunakan callback hander terpusat agar toast muncul
                    roleOptions={[]}
                    editingRole={editingRole}
                    onCancelEdit={() => setEditingRole(null)}
                />
            )}

            <RoleTable
                canUpdate={permissions.update}
                canDelete={permissions.delete}
                refreshTrigger={refreshTrigger}
                onEditClick={(role) => setEditingRole(role)}
                onDeleteClick={(role) => setRoleToDelete(role)}
                onForbidden={() => setHttpStatus(403)}
            />

            <AlertDialog open={roleToDelete !== null} onOpenChange={(open) => { if (!open && !isDeleting) setRoleToDelete(null); }}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Apakah Anda benar-benar yakin?</AlertDialogTitle>
                        <AlertDialogDescription>
                            Tindakan ini tidak dapat dibatalkan. Peran dengan identitas berikut akan dihapus secara permanen dari database:
                        </AlertDialogDescription>
                    </AlertDialogHeader>

                    <div className="my-2 p-3 bg-muted rounded-md border text-xs">
                        <p className="font-semibold text-foreground">{roleToDelete?.name}</p>
                    </div>

                    <AlertDialogFooter>
                        <Button
                            variant="outline"
                            disabled={isDeleting}
                            onClick={() => setRoleToDelete(null)}
                        >
                            Batal
                        </Button>

                        <Button
                            variant="destructive"
                            disabled={isDeleting}
                            onClick={handleDeleteConfirm}
                        >
                            {isDeleting ? "Menghapus..." : "Ya, Hapus Peran"}
                        </Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </main>
    );
}
