"use client";

import { ensureSession, readPermissionList } from "@/lib/permissions";
import React, { useState, useEffect } from "react";
import Cookies from "js-cookie";
import { toast } from "sonner";
import { BookOpenCheck } from "lucide-react";
import { ChartOfAccountForm } from "./components/chart-of-account-form";
import { ChartOfAccountTable, ChartOfAccount } from "./components/chart-of-account-table";
import { ForbiddenView } from "@/components/forbidden-view";
import {
    AlertDialog,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

export default function ChartOfAccountsPage() {
    const [httpStatus, setHttpStatus] = useState<number>(200);
    const [permissions, setPermissions] = useState({ create: false, read: false, update: false, delete: false });
    const [refreshTrigger, setRefreshTrigger] = useState(0);

    const [accountToDeactivate, setAccountToDeactivate] = useState<ChartOfAccount | null>(null);
    const [editingAccount, setEditingAccount] = useState<ChartOfAccount | null>(null);
    const [isProcessing, setIsProcessing] = useState<boolean>(false);

    const authToken = Cookies.get("session_active");

    const loadPermissions = () => {
        if (!ensureSession()) return;
        const list = readPermissionList() ?? [];
        setPermissions({
            create: list.includes("accounting:create"),
            read: list.includes("accounting:read"),
            update: list.includes("accounting:update"),
            delete: list.includes("accounting:delete"),
        });
    };

    const handleDeactivateConfirm = async () => {
        if (!accountToDeactivate) return;

        setIsProcessing(true);
        const targetedCode = accountToDeactivate.code;

        try {
            const response = await fetch(`/api/chart-of-accounts?id=${accountToDeactivate.id}`, {
                method: "DELETE",
                headers: {
                    "Content-Type": "application/json",
                    "Authorization": `Bearer ${authToken}`,
                },
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                if (response.status === 403) setHttpStatus(403);
                throw new Error(errorData.message || "Gagal menonaktifkan akun.");
            }

            setAccountToDeactivate(null);
            toast.success("Akun Dinonaktifkan", {
                description: `Akun ${targetedCode} dikeluarkan dari daftar akun aktif.`,
                duration: 3000,
            });
            setRefreshTrigger((prev) => prev + 1);
        } catch (error: any) {
            toast.error("Operasi Gagal", {
                description: error.message || "Masalah internal server.",
            });
        } finally {
            setIsProcessing(false);
        }
    };

    const handleFormSuccess = (message: string) => {
        setRefreshTrigger((prev) => prev + 1);
        toast.success(message, { duration: 3000 });
    };

    useEffect(() => {
        loadPermissions();
        window.addEventListener("focus", loadPermissions);
        return () => window.removeEventListener("focus", loadPermissions);
    }, []);

    if (httpStatus === 403) {
        const handleResetForbidden = () => {
            setHttpStatus(200);
            loadPermissions();
            setRefreshTrigger((prev) => prev + 1);
        };
        return <ForbiddenView reset={handleResetForbidden} />;
    }

    return (
        <main className="flex flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8 w-full">
            <div className="flex items-center gap-2 border-b pb-4">
                <div className="p-2 bg-emerald-50 text-emerald-600 rounded-xl border border-emerald-100">
                    <BookOpenCheck className="size-5" />
                </div>
                <div>
                    <h1 className="text-sm font-black uppercase tracking-wider text-slate-900">
                        Bagan Akun (Chart of Accounts)
                    </h1>
                    <p className="text-xs text-slate-500">
                        Fondasi struktur buku besar: aset, kewajiban, ekuitas, pendapatan, dan beban.
                    </p>
                </div>
            </div>

            {permissions.create && (
                <ChartOfAccountForm
                    token={authToken}
                    editingAccount={editingAccount}
                    onSuccess={handleFormSuccess}
                    onCancelEdit={() => setEditingAccount(null)}
                />
            )}

            <ChartOfAccountTable
                canUpdate={permissions.update}
                canDelete={permissions.delete}
                refreshTrigger={refreshTrigger}
                onEditClick={(acc) => setEditingAccount(acc)}
                onDeleteClick={(acc) => setAccountToDeactivate(acc)}
            />

            <AlertDialog open={accountToDeactivate !== null} onOpenChange={(open) => { if (!open && !isProcessing) setAccountToDeactivate(null); }}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Nonaktifkan Akun Ini?</AlertDialogTitle>
                        <AlertDialogDescription>
                            Akun berikut dikeluarkan dari daftar aktif sehingga tidak bisa lagi dipakai pada jurnal baru.
                        </AlertDialogDescription>
                    </AlertDialogHeader>

                    <div className="my-2 p-3 bg-rose-50 text-rose-700 rounded-md border border-rose-100 font-mono">
                        <p className="font-bold">{accountToDeactivate?.code}</p>
                        <p className="text-[11px] mt-0.5">{accountToDeactivate?.name}</p>
                    </div>

                    <AlertDialogFooter>
                        <Button variant="outline" disabled={isProcessing} onClick={() => setAccountToDeactivate(null)} className="text-xs h-8 cursor-pointer">Batal</Button>
                        <Button variant="destructive" disabled={isProcessing} onClick={handleDeactivateConfirm} className="text-xs h-8 cursor-pointer">
                            {isProcessing ? "Memproses..." : "Ya, Nonaktifkan"}
                        </Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </main>
    );
}