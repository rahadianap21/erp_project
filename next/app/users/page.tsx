"use client";

import { apiDelete } from "@/lib/api-client";
import { ensureSession, readPermissionFlags } from "@/lib/permissions";
import React, { useState, useEffect } from "react";
import dynamic from "next/dynamic";
import UserForm from "./components/user-form";
import { UserTableSkeleton } from "./components/user-table";
import { ForbiddenView } from "@/components/forbidden-view";
import { toast } from "sonner";

const UserTable = dynamic(
  () => import("./components/user-table").then((mod) => mod.UserTable),
  {
    ssr: false,
    loading: () => <UserTableSkeleton />,
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

interface User {
  id: number;
  name: string;
  email: string;
  role_name: string;
}

export default function UsersCRUDPage() {
  const [httpStatus, setHttpStatus] = useState<number>(200);
  const [permissions, setPermissions] = useState({
    create: false,
    update: false,
    delete: false,
  });
  const [refreshTrigger, setRefreshTrigger] = useState(0);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [roleOptions, setRoleOptions] = useState<string[]>([]);
  const [userToDelete, setUserToDelete] = useState<User | null>(null);
  const [isDeleting, setIsDeleting] = useState<boolean>(false);

  const loadPermissions = () => {
    if (!ensureSession()) return;
    setPermissions(readPermissionFlags("users"));
  };

  const handleDeleteConfirm = async () => {
    if (!userToDelete) return;

    setIsDeleting(true);
    const targetedName = userToDelete.name;

    try {
      await apiDelete(`/api/users?id=${userToDelete.id}`);

      setUserToDelete(null);
      setTimeout(() => {
        toast.success("Pengguna Berhasil Dihapus", {
          description: `Akun milik ${targetedName} telah dihapus dari sistem.`,
          duration: 2000,
        });
      }, 100);

      // Trigger refresh tabel pengguna secara reaktif
      setRefreshTrigger((prev) => prev + 1);
    } catch (error: any) {
      toast.error("Gagal Menghapus Pengguna", {
        description:
          error.message ||
          "Terjadi kesalahan koneksi atau masalah internal server.",
      });
    } finally {
      setIsDeleting(false);
    }
  };

  // Callback terpusat untuk kesuksesan mutasi form (POST / PUT)
  const handleFormSuccess = (mode: "create" | "update", name: string) => {
    setEditingUser(null);
    setRefreshTrigger((prev) => prev + 1);

    toast.success(
      mode === "create" ? "Pengguna Ditambahkan" : "Pengguna Diperbarui",
      {
        description: `Akun milik ${name} berhasil ${mode === "create" ? "disimpan ke" : "diperbarui dalam"} sistem.`,
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
      {/* 🔄 LANGKAH PERBAIKAN 2: Form harus dirender jika user punya hak 'create' ATAU sedang meng-edit data */}
      {(permissions.create || editingUser) && (
        <UserForm
          onSuccess={handleFormSuccess} // Gunakan callback hander terpusat agar toast muncul
          roleOptions={roleOptions}
          editingUser={editingUser}
          onCancelEdit={() => setEditingUser(null)}
        />
      )}

      <UserTable
        canUpdate={permissions.update}
        canDelete={permissions.delete}
        refreshTrigger={refreshTrigger}
        onEditClick={(user) => setEditingUser(user)}
        onDeleteClick={(user) => setUserToDelete(user)}
        onRoleOptionsChange={setRoleOptions}
        onForbidden={() => setHttpStatus(403)}
      />

      <AlertDialog
        open={userToDelete !== null}
        onOpenChange={(open) => {
          if (!open && !isDeleting) setUserToDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Apakah Anda benar-benar yakin?</AlertDialogTitle>
            <AlertDialogDescription>
              Tindakan ini tidak dapat dibatalkan. Pengguna dengan identitas
              berikut akan dihapus secara permanen dari database:
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="my-2 p-3 bg-muted rounded-md border text-xs">
            <p className="font-semibold text-foreground">
              {userToDelete?.name}
            </p>
            <p className="text-xs text-muted-foreground">
              {userToDelete?.email}
            </p>
          </div>

          <AlertDialogFooter>
            <Button
              variant="outline"
              disabled={isDeleting}
              onClick={() => setUserToDelete(null)}
            >
              Batal
            </Button>

            <Button
              variant="destructive"
              disabled={isDeleting}
              onClick={handleDeleteConfirm}
            >
              {isDeleting ? "Menghapus..." : "Ya, Hapus Pengguna"}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  );
}
