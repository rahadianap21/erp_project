'use client';

import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronUp, Folder, KeyRound, Loader2 } from 'lucide-react';
import { toast } from 'sonner';

interface Role {
    id: number;
    name: string;
    permissions: string[];
}

interface RoleFormProps {
    token: string | undefined; // Tangkap token dari parent
    roleOptions: string[];      // Tangkap opsi role riil dari database (string[])
    editingRole: Role | null;
    onSuccess: (mode: 'create' | 'update', name: string) => void;
    onCancelEdit: () => void;
}

export default function RoleForm({ token, roleOptions, editingRole, onSuccess, onCancelEdit }: RoleFormProps) {
    const [isExpanded, setIsExpanded] = useState<boolean>(false);
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [errorMessage, setErrorMessage] = useState<string>('');

    const [name, setName] = useState<string>('');

    // 🌟 STATE BARU: Menampung daftar master permission dari backend dan permission yang dicentang
    const [masterPermissions, setMasterPermissions] = useState<string[]>([]);
    const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
    const [loadingFetch, setLoadingFetch] = useState<boolean>(false);

    useEffect(() => {
        if (!isExpanded && !editingRole) return;

        const fetchMasterPermissions = async () => {
            setLoadingFetch(true);
            try {
                const response = await fetch(`/api/permissions`, {
                    headers: { "Authorization": `Bearer ${token}` }
                });
                if (!response.ok) {
                    throw new Error(`Server error: ${response.status}`);
                }
                const result = await response.json();
                setMasterPermissions(Array.isArray(result.data) ? result.data : []);
            } catch (err) {
                console.error("Gagal mengambil master permissions:", err);
                setMasterPermissions([]);
            } finally {
                setLoadingFetch(false);
            }
        };

        if (token) {
            fetchMasterPermissions();
        }
    }, [isExpanded, editingRole, token]);

    // Efek Reaktif untuk Mode Edit
    useEffect(() => {
        if (editingRole) {
            setName(editingRole.name);
            setIsExpanded(true);
            setSelectedPermissions(editingRole.permissions || []);
        } else {
            resetForm();
        }
    }, [editingRole]);

    const resetForm = () => {
        setName('');
        setSelectedPermissions([]);
    };

    const handleCancel = () => {
        resetForm();
        if (editingRole) {
            onCancelEdit();
        } else {
            setIsExpanded(false);
        }
    };

    const groupPermissionsByPrefix = (permissions: string[]) => {
        if (!permissions) return {};
        return permissions.reduce((acc, perm) => {
            const parts = perm.split(":"); //
            const prefix = parts[0] || "Lainnya"; //
            if (!acc[prefix]) {
                acc[prefix] = [];
            }
            acc[prefix].push(perm);
            return acc;
        }, {} as Record<string, string[]>);
    };

    // 🌟 LOGIKA BARU: Fungsi toggle memasukkan/mengeluarkan permission yang dicentang
    const handleCheckboxChange = (perm: string) => {
        setSelectedPermissions((prev) =>
            prev.includes(perm) ? prev.filter((p) => p !== perm) : [...prev, perm]
        );
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsLoading(true);

        const isEditMode = editingRole !== null;
        // Gunakan URL absolute backend sesuai arsitektur fetch pada page.tsx Anda
        const url = isEditMode ? `/api/roles?id=${editingRole.id}` : `/api/roles`;
        const method = isEditMode ? 'PUT' : 'POST';

        try {
            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}` // Sertakan token otentikasi
                },
                body: JSON.stringify({
                    name,
                    permissions: selectedPermissions,
                }),
            });

            const errorData = await response.json().catch(() => ({}));
            if (!response.ok) {
                throw new Error(errorData.message || errorData.error || 'Gagal menyimpan data pengguna.');
            }

            resetForm();
            setIsExpanded(false);
            onSuccess(isEditMode ? 'update' : 'create', name);
        } catch (error: any) {
            setErrorMessage(error.message || "Terjadi kesalahan internal koneksi server.");
            toast.error("Gagal Menyimpan Pengguna", {
                description: error.message || "Email sudah terdaftar atau terjadi kesalahan server.",
                duration: 4000, // Berikan waktu 4 detik agar teks terbaca utuh
            });
        } finally {
            setIsLoading(false);
        }
    };

    const groupedPermissions = groupPermissionsByPrefix(masterPermissions);
    const prefixes = Object.keys(groupedPermissions);

    return (
        <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden transition-all duration-300">
            {/* Header Tombol Collapsible */}
            <button
                type="button"
                onClick={() => !editingRole && setIsExpanded(!isExpanded)}
                disabled={!!editingRole}
                className="w-full px-6 py-4 flex items-center justify-between bg-gray-50 hover:bg-gray-100/80 transition-colors text-left cursor-pointer"
            >
                <span className=" text-sm font-semibold text-gray-800">
                    {editingRole ? `📝 Edit Role: ${editingRole.name}` : 'Tambah Role Baru'}
                </span>
                {!editingRole && (isExpanded ? <ChevronUp className="h-5 w-5" /> : <ChevronDown className="h-5 w-5" />)}
            </button>

            {/* Konten Form */}
            <div className={`transition-all duration-300 ease-in-out ${isExpanded ? 'max-h-[850px] p-6 border-t border-gray-100' : 'max-h-0 opacity-0 pointer-events-none'}`}>
                <form onSubmit={handleSubmit} className="space-y-5">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
                        {/* Input Nama */}
                        <div className="space-y-2">
                            <label className="text-xs font-semibold tracking-wide text-slate-800">
                                Nama Role *
                            </label>
                            <input
                                type="text"
                                required
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                className="w-full text-xs px-3.5 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 placeholder:text-slate-400 transition-all font-normal"
                                placeholder="Masukkan nama role"
                            />
                        </div>
                    </div>

                    <div className="space-y-2">
                        <label className="text-xs font-semibold tracking-wide text-slate-800 flex items-center gap-1.5">
                            <KeyRound className="size-3.5 text-blue-500" /> Atur Hak Akses Granular (Permissions)
                        </label>

                        {loadingFetch ? (
                            <div className="border border-slate-100 rounded-lg flex items-center justify-center bg-slate-50/30 h-24">
                                <Loader2 className="size-4 animate-spin text-muted-foreground" />
                                <span className="text-xs text-muted-foreground ml-2">Memuat daftar hak akses...</span>
                            </div>
                        ) : (
                            <div className="border border-slate-200/60 rounded-xl bg-slate-50/30 p-4 space-y-4 max-h-[45vh] overflow-y-auto shadow-inner">
                                {prefixes.length > 0 ? (
                                    <div className="grid grid-cols-2 md:grid-cols-4 gap-3 pt-1">
                                        {prefixes.map((prefix) => {
                                            const currentGroupPerms = groupedPermissions[prefix];
                                            // 🌟 Logika memeriksa apakah semua permission di modul ini sudah dipilih
                                            const isAllGroupChecked = currentGroupPerms.every((perm) =>
                                                selectedPermissions.includes(perm)
                                            );

                                            // 🌟 Fungsi untuk toggle select all khusus modul ini
                                            const handleSelectAllGroup = () => {
                                                if (isAllGroupChecked) {
                                                    // Jika sudah centang semua, maka hapus semua dari modul ini
                                                    setSelectedPermissions((prev) =>
                                                        prev.filter((perm) => !currentGroupPerms.includes(perm))
                                                    );
                                                } else {
                                                    // Jika belum semua, gabungkan yang belum dicentang ke state
                                                    setSelectedPermissions((prev) => {
                                                        const uniqueNewPerms = currentGroupPerms.filter(
                                                            (perm) => !prev.includes(perm)
                                                        );
                                                        return [...prev, ...uniqueNewPerms];
                                                    });
                                                }
                                            };

                                            return (
                                                <div
                                                    key={`form-group-${prefix}`}
                                                    className="bg-white border border-slate-200/80 p-3 rounded-lg shadow-3xs flex flex-col gap-2.5"
                                                >
                                                    {/* Judul Kop Modul + Tombol Select All */}
                                                    <div className="flex items-center justify-between border-b pb-1.5 border-slate-100">
                                                        <div className="text-[10px] font-bold text-slate-500 uppercase tracking-wide flex items-center gap-1.5">
                                                            <Folder className="size-3 text-slate-400" />
                                                            <span>Modul {prefix}</span>
                                                        </div>

                                                        {/* 🌟 TOMBOL SELECT ALL DINAMIS */}
                                                        <button
                                                            type="button"
                                                            onClick={handleSelectAllGroup}
                                                            className="text-[10px] font-semibold text-blue-600 hover:text-blue-800 transition-colors cursor-pointer select-none"
                                                        >
                                                            {isAllGroupChecked ? "Deselect All" : "Select All"}
                                                        </button>
                                                    </div>

                                                    {/* Kumpulan Grid Pilihan Berbentuk Badge Interaktif */}
                                                    <div className="flex flex-wrap gap-2">
                                                        {currentGroupPerms.map((perm) => {
                                                            const isChecked = selectedPermissions.includes(perm);

                                                            const lowerPerm = perm.toLowerCase();
                                                            let badgeColorClass = "bg-slate-500/10 text-slate-700 border-slate-500/20";

                                                            if (lowerPerm.includes("read") || lowerPerm.includes("view")) {
                                                                badgeColorClass = "bg-emerald-500/10 text-emerald-700 border-emerald-500/30";
                                                            } else if (lowerPerm.includes("create") || lowerPerm.includes("write") || lowerPerm.includes("add")) {
                                                                badgeColorClass = "bg-blue-500/10 text-blue-700 border-blue-500/30";
                                                            } else if (lowerPerm.includes("update") || lowerPerm.includes("edit") || lowerPerm.includes("modify")) {
                                                                badgeColorClass = "bg-amber-500/10 text-amber-700 border-amber-500/30";
                                                            } else if (lowerPerm.includes("delete") || lowerPerm.includes("remove") || lowerPerm.includes("destroy")) {
                                                                badgeColorClass = "bg-rose-500/10 text-rose-700 border-rose-500/30";
                                                            }

                                                            const stateClasses = isChecked
                                                                ? "border-slate-500 ring-2 ring-blue-500/10 font-bold scale-[1.01]"
                                                                : "opacity-75 hover:opacity-100 border-opacity-30";

                                                            return (
                                                                <label
                                                                    key={`cb-${perm}`}
                                                                    className={`inline-flex items-center gap-2 px-2.5 py-1 rounded-md border font-mono text-[10px] font-medium transition-all shadow-3xs cursor-pointer select-none active:scale-95 ${badgeColorClass} ${stateClasses}`}
                                                                >
                                                                    <input
                                                                        type="checkbox"
                                                                        checked={isChecked}
                                                                        onChange={() => handleCheckboxChange(perm)}
                                                                        className="size-3 text-blue-600 rounded border-slate-300 cursor-pointer focus:ring-0 focus:ring-offset-0"
                                                                    />
                                                                    <span>{perm}</span>
                                                                </label>
                                                            );
                                                        })}
                                                    </div>
                                                </div>
                                            );
                                        })}
                                    </div>
                                ) : (
                                    <p className="text-xs text-muted-foreground italic pl-0.5 pt-0.5">
                                        Tidak ada opsi master permission yang tersedia di sistem.
                                    </p>
                                )}
                            </div>
                        )}
                    </div>

                    {/* Tombol Aksi Form */}
                    <div className="flex justify-end space-x-3 pt-3 border-t border-slate-100">
                        <Button
                            type="button"
                            variant="outline"
                            disabled={isLoading}
                            onClick={handleCancel}
                            className="text-xs font-medium px-4 py-2 text-slate-600 border-slate-200 hover:bg-slate-50 cursor-pointer"
                        >
                            Batal
                        </Button>

                        <Button
                            type="submit"
                            disabled={isLoading}
                            className="bg-blue-600 hover:bg-blue-700 text-white text-xs font-medium px-5 py-2 shadow-sm shadow-blue-500/10 transition-colors cursor-pointer"
                        >
                            {isLoading ? 'Menyimpan...' : editingRole ? 'Perbarui Data' : 'Simpan Pengguna'}
                        </Button>
                    </div>
                </form>
            </div>
        </div>
    );
}
