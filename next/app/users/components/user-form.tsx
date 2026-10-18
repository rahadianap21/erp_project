"use client";

import { apiPost, apiPut, messageFromError } from "@/lib/api-client";
import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { ChevronDown, ChevronUp, Eye, EyeOff } from "lucide-react";
import { toast } from "sonner";

interface User {
  id: number;
  name: string;
  email: string;
  role_name: string;
}

interface UserFormProps {
  roleOptions: string[];
  editingUser: User | null;
  onSuccess: (mode: "create" | "update", name: string) => void;
  onCancelEdit: () => void;
}

export default function UserForm({
  roleOptions,
  editingUser,
  onSuccess,
  onCancelEdit,
}: UserFormProps) {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  const [name, setName] = useState<string>("");
  const [email, setEmail] = useState<string>("");
  const [selectedRole, setSelectedRole] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [showPassword, setShowPassword] = useState<boolean>(false);

  // Efek Reaktif untuk Mode Edit
  useEffect(() => {
    if (editingUser) {
      setName(editingUser.name);
      setEmail(editingUser.email);
      setSelectedRole(editingUser.role_name || "");
      setPassword("");
      setShowPassword(false);
      setIsExpanded(true);
    } else {
      resetForm();
    }
  }, [editingUser]);

  const resetForm = () => {
    setName("");
    setEmail("");
    setSelectedRole("");
    setPassword("");
    setShowPassword(false);
  };

  const handleCancel = () => {
    resetForm();
    if (editingUser) {
      onCancelEdit();
    } else {
      setIsExpanded(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    const isEditMode = editingUser !== null;
    const targetPath = isEditMode
      ? `/api/users?id=${editingUser.id}`
      : `/api/users`;

    const payload: Record<string, unknown> = {
      name,
      email,
      role_name: selectedRole,
    };

    if (!isEditMode) {
      payload.password = password;
    }

    try {
      if (isEditMode) await apiPut(targetPath, payload);
      else await apiPost(targetPath, payload);

      resetForm();
      setIsExpanded(false);
      onSuccess(isEditMode ? "update" : "create", name);
    } catch (error) {
      toast.error("Gagal Menyimpan Pengguna", {
        description:
          messageFromError(error) ||
          "Email sudah terdaftar atau terjadi kesalahan server.",
        duration: 4000, // Berikan waktu 4 detik agar teks terbaca utuh
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden transition-all duration-300">
      {/* Header Tombol Collapsible */}
      <button
        type="button"
        onClick={() => !editingUser && setIsExpanded(!isExpanded)}
        disabled={!!editingUser}
        className="w-full px-6 py-4 flex items-center justify-between bg-gray-50 hover:bg-gray-100/80 transition-colors text-left cursor-pointer"
      >
        <span className=" text-sm font-semibold text-gray-800">
          {editingUser
            ? `📝 Edit Pengguna: ${editingUser.name}`
            : "Tambah Pengguna Baru"}
        </span>
        {!editingUser &&
          (isExpanded ? (
            <ChevronUp className="h-5 w-5" />
          ) : (
            <ChevronDown className="h-5 w-5" />
          ))}
      </button>

      {/* Konten Form */}
      <div
        className={`transition-all duration-300 ease-in-out ${isExpanded ? "max-h-[500px] p-6 border-t border-gray-100" : "max-h-0 opacity-0 pointer-events-none"}`}
      >
        <form onSubmit={handleSubmit} autoComplete="off" className="space-y-5">
          <div
            className={`grid grid-cols-1 md:grid-cols-2 ${editingUser ? "lg:grid-cols-3" : "lg:grid-cols-4"} gap-5`}
          >
            {/* Input Nama */}
            <div className="space-y-2">
              <label
                htmlFor="user-name"
                className="text-xs font-semibold tracking-wide text-slate-800"
              >
                Nama Lengkap
              </label>
              <input
                id="user-name"
                name="user_full_name"
                type="text"
                required
                autoComplete="off"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full text-xs px-3.5 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 placeholder:text-slate-400 transition-all font-normal"
                placeholder="Masukkan nama lengkap"
              />
            </div>

            {/* Input Email */}
            <div className="space-y-2">
              <label
                htmlFor="user-email"
                className="text-xs font-semibold tracking-wide text-slate-800"
              >
                Alamat Email
              </label>
              <input
                id="user-email"
                name="user_email"
                type="email"
                required
                autoComplete="off"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full text-xs px-3.5 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 placeholder:text-slate-400 transition-all font-normal"
                placeholder="nama@company.com"
              />
            </div>

            {/* Input Kata Sandi (Hanya tampil saat Tambah Pengguna Baru) */}
            {!editingUser && (
              <div className="space-y-2">
                <label
                  htmlFor="user-password"
                  className="text-xs font-semibold tracking-wide text-slate-800"
                >
                  Kata Sandi (Password)
                </label>
                <div className="relative">
                  <input
                    id="user-password"
                    name="new_user_password"
                    type={showPassword ? "text" : "password"}
                    required
                    minLength={6}
                    autoComplete="new-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full text-xs px-3.5 py-2 pr-10 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 placeholder:text-slate-400 transition-all font-normal"
                    placeholder="Minimal 6 karakter"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 focus:outline-none p-1 transition-colors cursor-pointer"
                    tabIndex={-1}
                    aria-label={
                      showPassword
                        ? "Sembunyikan kata sandi"
                        : "Tampilkan kata sandi"
                    }
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </button>
                </div>
              </div>
            )}

            {/* Dropdown Role */}
            <div className="space-y-2">
              <label className="text-xs font-semibold tracking-wide text-slate-800">
                Hak Akses (Role)
              </label>
              <select
                required
                value={selectedRole}
                onChange={(e) => setSelectedRole(e.target.value)}
                className="w-full text-xs px-3.5 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 transition-all font-normal cursor-pointer"
              >
                <option value="" disabled>
                  Pilih tingkat akses...
                </option>
                {roleOptions.map((roleName, index) => (
                  <option key={index} value={roleName}>
                    {roleName}
                  </option>
                ))}
              </select>
            </div>
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
              {isLoading
                ? "Menyimpan..."
                : editingUser
                  ? "Perbarui Data"
                  : "Simpan Pengguna"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
