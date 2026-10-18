"use client";

import { apiGet, apiPost, messageFromError } from "@/lib/api-client";
import * as React from "react";
import {
  Combobox,
  ComboboxInput,
  ComboboxContent,
  ComboboxList,
  ComboboxItem,
  ComboboxEmpty,
} from "@/components/ui/combobox";
import { cn } from "@/lib/utils";
import { Plus, Loader2, Save, Link2, FolderTree, FileText } from "lucide-react";
import Cookies from "js-cookie";
import { toast } from "sonner";

// IMPOR KOMPONEN DIALOG PREMIUM SHADCN/UI
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useEffect } from "react";

export interface ComboboxOption {
  value: string;
  label: string;
}

interface ComboboxSelectProps {
  options: ComboboxOption[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  emptyText?: string;
  className?: string;
  id?: string;
  permissionCode?: string; // Penanda kaku modal: "units:create", "categories:create", atau "suppliers:create"
  onRefreshRelations?: () => Promise<void>; // Callback pembaru list array form induk
  popoverClassName?: string;
  // Label yang dibawa record itu sendiri. Dipakai saat nilai terpilih tidak ada
  // di dalam 'options' (mis. daftar master terpotong / endpoint lookup 403), agar
  // input tetap menampilkan nama aslinya dan tidak tampak kosong.
  fallbackLabel?: string;
}

export function ComboboxSelect({
  options,
  value,
  onChange,
  placeholder = "Pilih opsi...",
  disabled = false,
  emptyText = "Tidak ada data ditemukan",
  className,
  id,
  permissionCode,
  onRefreshRelations,
  popoverClassName,
  fallbackLabel,
}: ComboboxSelectProps) {
  const [searchValue, setSearchValue] = React.useState<string>("");
  const [hasPermission, setHasPermission] = React.useState<boolean>(false);

  // KONTROL STATE MODAL FORM DINAMIS
  const [isModalOpen, setIsModalOpen] = React.useState<boolean>(false);
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const [parentCategoryOptions, setParentCategoryOptions] = React.useState<
    { id: number | string; name: string }[]
  >([]);

  // Buffer State Kolektif Penampung Nilai Formulir Dinamis
  const [formFields, setFormFields] = React.useState({
    kode: "",
    name: "",
    description: "",
    slug: "", // 🌟 Tambahan untuk Kategori
    parentId: "",
    picName: "",
    phone: "",
    email: "",
    address: "",
  });

  const authToken = Cookies.get("session_active") || "";

  // Opsi efektif: bila nilai terpilih tidak ada di 'options' tapi kita punya
  // fallbackLabel dari record aslinya, suntikkan opsi itu di depan daftar.
  // Base UI akan menerima selectedOption yang bukan null sehingga nama tetap
  // tampil di input, dan nilainya tetap sah saat user memilih ulang.
  const effectiveOptions = React.useMemo(() => {
    const hasValue = value !== undefined && value !== null && value !== "";
    if (!hasValue || !fallbackLabel) return options;
    if (options.some((opt) => opt.value === value)) return options;
    return [{ value, label: fallbackLabel }, ...options];
  }, [options, value, fallbackLabel]);

  const displayedOptions = React.useMemo(() => {
    // Jika kolom pencarian kosong murni, tampilkan 10 data pertama sebagai opsi default awal
    if (searchValue.trim() === "") {
      return effectiveOptions.slice(0, 10);
    }

    // Jika user mengetik kata kunci, saring dari seluruh isi 'effectiveOptions' di memori, lalu potong maksimal 10 baris saja
    return effectiveOptions
      .filter((opt) =>
        opt.label.toLowerCase().includes(searchValue.toLowerCase())
      )
      .slice(0, 10); // ✨ Kunci Efisiensi DOM: Batasi maksimal 10 data yang di-render di layar
  }, [effectiveOptions, searchValue]);

  const selectedOption = React.useMemo(() => {
    if (value === undefined || value === null || value === "") return null;
    return effectiveOptions.find((opt) => opt.value === value) ?? null;
  }, [effectiveOptions, value]);

  const fetchModalParentCategories = React.useCallback(async () => {
    if (!authToken || !permissionCode?.startsWith("categories")) return;
    try {
      const result = await apiGet<{ id: number | string; name: string }[]>(
        `/api/categories?limit=100`
      );
      if (result.data) {
        setParentCategoryOptions(result.data);
        console.log("Parent Category Options:", result.data);
      }
    } catch (err) {
      console.error("Gagal memuat opsi induk kategori di dalam modal:", err);
    }
  }, [authToken, permissionCode]);

  // Evaluasi Otorisasi Mandiri Level Komponen (Self-Contained RBAC Guard)
  React.useEffect(() => {
    if (!permissionCode) {
      setHasPermission(false);
      return;
    }
    const savedPermissions = Cookies.get("user_permissions");
    if (savedPermissions) {
      try {
        const parsed: string[] = JSON.parse(
          decodeURIComponent(savedPermissions)
        );
        if (Array.isArray(parsed)) {
          setHasPermission(parsed.includes(permissionCode));
        }
      } catch (e) {
        console.error("ComboboxSelect RBAC Evaluation Error:", e);
        setHasPermission(false);
      }
    }
  }, [permissionCode]);

  // 🌲 METADATA DINAMIS: Deteksi Jenis Modul Berdasarkan Teks permissionCode secara Otomatis
  const moduleMeta = React.useMemo(() => {
    const code = permissionCode || "";
    if (code.startsWith("units")) {
      return {
        title: "Satuan Ukur / UoM Baru",
        endpoint: "/api/units",
        placeholder: "Contoh: PACK, ROLL, CRT",
      };
    }
    if (code.startsWith("categories")) {
      return {
        title: "Kategori Produk Baru",
        endpoint: "/api/categories",
        placeholder: "Contoh: Bahan Baku Logistik",
      };
    }
    if (code.startsWith("suppliers")) {
      return {
        title: "Rekanan Vendor / Supplier Baru",
        endpoint: "/api/suppliers",
        placeholder: "Contoh: PT. Sumber Jaya",
      };
    }
    return { title: "Data Baru", endpoint: "", placeholder: "" };
  }, [permissionCode]);

  useEffect(() => {
    if (isModalOpen && permissionCode?.startsWith("categories")) {
      fetchModalParentCategories(); // <-- Pastikan baris pemicu ini aktif & tidak terhapus!
    }
  }, [isModalOpen, permissionCode, fetchModalParentCategories]);

  // EKSEKUSI PENYIMPANAN DATA TRANSAKSIONAL KE GO BACKEND
  const handleDynamicQuickSubmit = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    if (!authToken || !moduleMeta.endpoint) return;

    // Bersihkan format kode khusus untuk modul master satuan ukuran
    const cleanKode = formFields.kode
      .trim()
      .toUpperCase()
      .replace(/[^A-Z0-9]/g, "");

    if (permissionCode?.startsWith("units") && cleanKode.length < 2) {
      toast.error("Validasi Gagal", {
        description:
          "Kode satuan minimal terdiri dari 2 karakter alfanumerik murni!",
      });
      return;
    }
    if (!formFields.name.trim()) {
      toast.error("Validasi Gagal", {
        description: "Nama bidang wajib diisi secara lengkap!",
      });
      return;
    }

    setIsSubmitting(true);
    const toastId = toast.loading(`Mendaftarkan ${moduleMeta.title}...`);

    try {
      // Pembentukan skema Payload JSON adaptif mengikuti target endpoint backend Go
      let payload: Record<string, unknown> = {};
      if (permissionCode?.startsWith("units")) {
        payload = {
          kode: cleanKode,
          name: formFields.name.trim(),
          description: formFields.description,
        };
      } else if (permissionCode?.startsWith("categories")) {
        payload = {
          name: formFields.name.trim(),
          slug: formFields.slug.trim() || undefined,
          description: formFields.description.trim() || "",
          parent_id: formFields.parentId ? Number(formFields.parentId) : null,
          is_active: true,
        };
      } else if (permissionCode?.startsWith("suppliers")) {
        payload = {
          name: formFields.name.trim(),
          pic_name: formFields.picName.trim(),
          phone: formFields.phone.trim(),
          email: formFields.email.trim(),
          address: formFields.address.trim(),
          // village_id: formFields.villageId.trim(), // Kunci kelurahan penyeimbang foreign key
          is_active: true,
        };
      }

      const result = await apiPost<{ id?: number }>(
        moduleMeta.endpoint,
        payload,
        { redirectOn403: false }
      );

      if (result.ok) {
        toast.success(`${moduleMeta.title} Sukses Ditambahkan`, {
          id: toastId,
        });

        // Reset seluruh form buffer ke string kosong
        setFormFields({
          kode: "",
          name: "",
          description: "",
          slug: "",
          parentId: "",
          picName: "",
          phone: "",
          email: "",
          address: "",
        });
        setIsModalOpen(false);

        // Memicu trigger pembaruan list dropdown di form induk Next.js secara reaktif
        if (onRefreshRelations) {
          await onRefreshRelations();
        }

        // Otomatis pasang nilai data baru tersebut ke kolom form yang sedang diketik
        if (result.data?.id) {
          onChange(String(result.data.id));
        }
      }
    } catch (err) {
      toast.error("Gagal Menyimpan data", {
        id: toastId,
        description: messageFromError(err),
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <>
      <Combobox<ComboboxOption>
        id={id}
        items={effectiveOptions}
        value={selectedOption}
        onValueChange={(val) => {
          onChange(val ? val.value : "");
        }}
        itemToStringLabel={(item) => (item ? item.label : "")}
        itemToStringValue={(item) => (item ? item.value : "")}
        isItemEqualToValue={(item, val) => item?.value === val?.value}
        autoHighlight
        disabled={disabled}
      >
        <ComboboxInput
          placeholder={placeholder}
          disabled={disabled}
          showClear={!disabled && !!value}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            setSearchValue(e.target.value)
          }
          className={cn(
            "w-full text-xs bg-white border-slate-200 focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-500/20 [&_input]:text-xs [&_input]:placeholder:text-slate-400 h-8",
            disabled && "opacity-50 cursor-not-allowed bg-slate-50",
            className
          )}
        />
        <ComboboxContent
          className={cn(
            "z-50 w-[var(--anchor-width)] max-w-[var(--anchor-width)] min-w-[var(--anchor-width)] bg-white border border-slate-200 shadow-lg rounded-lg text-slate-800 p-1 flex flex-col",
            popoverClassName // ✨ Menyisipkan kelas pemaksa lebar luar jika berada di dalam modal
          )}
          sideOffset={4}
          align="start"
        >
          <ComboboxEmpty className="py-2 px-3 text-xs text-slate-400 text-center select-none">
            {emptyText}
          </ComboboxEmpty>

          <ComboboxList className="max-h-48 overflow-y-auto text-xs flex-1">
            {displayedOptions.map((item: ComboboxOption) => (
              <ComboboxItem
                key={item.value}
                value={item}
                className="text-xs px-2.5 py-1.5 rounded-lg hover:bg-blue-50 hover:text-blue-700 data-[highlighted]:bg-blue-50 data-[highlighted]:text-blue-700 cursor-pointer flex items-center justify-between transition-colors font-medium"
              >
                <span className="truncate block w-full text-left uppercase tracking-tight">
                  {item.label}
                </span>
              </ComboboxItem>
            ))}
          </ComboboxList>

          {/* TOMBOL MANIFOLD PEMICU DIALOG */}
          {hasPermission && (
            <button
              type="button"
              onClick={() => {
                // Tangkap kata kunci ketikan pencarian terakhir sebagai input otomatis nama/kode awal
                if (permissionCode?.startsWith("units")) {
                  setFormFields((prev) => ({
                    ...prev,
                    kode: searchValue.trim().toUpperCase(),
                  }));
                } else {
                  setFormFields((prev) => ({
                    ...prev,
                    name: searchValue.trim(),
                  }));
                }
                setIsModalOpen(true);
              }}
              className="w-full text-left px-2.5 py-2 mt-1 border-t border-slate-100 text-[11px] font-bold text-blue-600 hover:bg-blue-50 rounded-md transition-colors flex items-center gap-1.5 cursor-pointer shrink-0 bg-white sticky bottom-0"
            >
              <Plus className="size-3" />
              <span>
                {searchValue.trim() !== ""
                  ? `Buat Master Baru: "${searchValue.toUpperCase()}"`
                  : "Buat Master Baru"}
              </span>
            </button>
          )}
        </ComboboxContent>
      </Combobox>

      <Dialog
        open={isModalOpen}
        onOpenChange={(open) => {
          if (!open && !isSubmitting) {
            setIsModalOpen(false);
            setFormFields({
              kode: "",
              name: "",
              description: "",
              slug: "",
              parentId: "",
              picName: "",
              phone: "",
              email: "",
              address: "",
            });
          }
        }}
      >
        <DialogContent className="w-full !max-w-2xl sm:!max-w-2xl p-6 bg-white rounded-xl border shadow-lg z-50 !overflow-visible">
          <DialogHeader>
            <DialogTitle className="text-sm font-bold text-slate-900 flex items-center gap-1.5 uppercase">
              <Plus className="size-4 text-blue-600" /> Registrasi{" "}
              {moduleMeta.title}
            </DialogTitle>
            <DialogDescription className="text-xs text-slate-500">
              Formulir pembuatan kilat (*On the Fly*). Mohon lengkapi seluruh
              parameter logistik berikut dengan valid.
            </DialogDescription>
          </DialogHeader>

          {/* 🌟 PERBAIKAN ABSOLUT: Menggunakan tag <div> biasa, bukan <form> agar tidak bentrok dengan form induk */}
          <div className="space-y-4 pt-2">
            {/* 🌲 STRUCT FORM 1: KHUSUS TIPE SATUAN (UNITS) */}
            {permissionCode?.startsWith("units") && (
              <div className="grid grid-cols-1 gap-3">
                <div className="space-y-1">
                  <label className="text-[11px] font-bold text-slate-600 uppercase">
                    Kode Satuan
                  </label>
                  <input
                    type="text"
                    required
                    value={formFields.kode}
                    onChange={(e) =>
                      setFormFields({ ...formFields, kode: e.target.value })
                    }
                    className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none uppercase font-mono"
                    placeholder={moduleMeta.placeholder}
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-[11px] font-bold text-slate-600 uppercase">
                    Nama Lengkap Satuan
                  </label>
                  <input
                    type="text"
                    required
                    value={formFields.name}
                    onChange={(e) =>
                      setFormFields({ ...formFields, name: e.target.value })
                    }
                    className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none font-medium"
                    placeholder="Contoh: GULUNGAN KABEL BESAR"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-[11px] font-bold text-slate-600 uppercase">
                    Deskripsi Satuan
                  </label>
                  <textarea
                    rows={3}
                    required
                    value={formFields.description}
                    onChange={(e) =>
                      setFormFields({
                        ...formFields,
                        description: e.target.value,
                      })
                    }
                    className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none font-medium"
                    placeholder="Contoh: GULUNGAN KABEL BESAR"
                  />
                </div>
              </div>
            )}

            {/* 🌲 STRUCT FORM 2: KHUSUS TIPE KATEGORI (CATEGORIES) */}
            {permissionCode?.startsWith("categories") && (
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-1.5">
                    <label className="text-[11px] font-bold text-slate-600 uppercase flex items-center gap-1">
                      <FolderTree className="size-3 text-blue-600" /> Nama
                      Kategori
                    </label>
                    <input
                      type="text"
                      required
                      value={formFields.name}
                      onChange={(e) =>
                        setFormFields({ ...formFields, name: e.target.value })
                      }
                      className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none font-medium focus:border-blue-500 focus:bg-white"
                      placeholder={moduleMeta.placeholder}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-[11px] font-bold text-slate-600 uppercase flex items-center gap-1">
                      <Link2 className="size-3 text-slate-400" /> Slug Kustom
                      (Opsional)
                    </label>
                    <input
                      type="text"
                      value={formFields.slug}
                      onChange={(e) =>
                        setFormFields({ ...formFields, slug: e.target.value })
                      }
                      className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none font-mono focus:border-blue-500 focus:bg-white"
                      placeholder="Otomatis jika kosong"
                    />
                  </div>
                </div>

                <div className="space-y-1.5 relative">
                  <label className="text-[11px] font-bold text-slate-600 uppercase flex items-center gap-1">
                    <FolderTree className="size-3 text-blue-600" /> Induk
                    Kategori (Hierarki)
                  </label>
                  <ComboboxSelect
                    // 🌟 PERBAIKAN ABSOLUT: Sisipkan opsi kosong di urutan pertama array sebelum data database di-mapping
                    options={[
                      {
                        value: "",
                        label: "--- JADIKAN KATEGORI UTAMA / TANPA INDUK ---",
                      },
                      ...parentCategoryOptions.map((opt) => ({
                        value: String(opt.id),
                        label: opt.name.toUpperCase(),
                      })),
                    ]}
                    value={formFields.parentId}
                    placeholder="Pilih Kategori Induk..."
                    onChange={(val) =>
                      setFormFields({ ...formFields, parentId: val })
                    }
                    emptyText="Kategori induk tidak ditemukan..."
                    popoverClassName="!w-[620px] !min-w-[620px] !max-w-2xl !z-50"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="text-[11px] font-bold text-slate-600 uppercase flex items-center gap-1">
                    <FileText className="size-3" /> Deskripsi Kategori
                  </label>
                  <textarea
                    value={formFields.description}
                    onChange={(e) =>
                      setFormFields({
                        ...formFields,
                        description: e.target.value,
                      })
                    }
                    rows={3}
                    className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none text-slate-800 font-medium resize-none focus:border-blue-500 focus:bg-white"
                    placeholder="Tulis catatan penjelasan klasifikasi barang di sini..."
                  />
                </div>
              </div>
            )}

            {/* 🌲 STRUCT FORM 3: KHUSUS TIPE VENDOR VITAL (SUPPLIERS) */}
            {permissionCode?.startsWith("suppliers") && (
              <div className="grid grid-cols-1 gap-3 max-h-80 overflow-y-auto pr-1">
                <div className="space-y-1">
                  <label className="text-[11px] font-bold text-slate-600 uppercase">
                    Nama Resmi Perusahaan Vendor
                  </label>
                  <input
                    type="text"
                    required
                    value={formFields.name}
                    onChange={(e) =>
                      setFormFields({ ...formFields, name: e.target.value })
                    }
                    className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none font-medium"
                    placeholder={moduleMeta.placeholder}
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-[11px] font-bold text-slate-600 uppercase">
                    Nama Lengkap PIC Narahubung
                  </label>
                  <input
                    type="text"
                    required
                    value={formFields.picName}
                    onChange={(e) =>
                      setFormFields({ ...formFields, picName: e.target.value })
                    }
                    className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none font-medium"
                    placeholder="Contoh: Hendra Wijaya"
                  />
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div className="space-y-1">
                    <label className="text-[11px] font-bold text-slate-600 uppercase">
                      No. Telepon / Fax
                    </label>
                    <input
                      type="text"
                      required
                      value={formFields.phone}
                      onChange={(e) =>
                        setFormFields({ ...formFields, phone: e.target.value })
                      }
                      className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none font-mono"
                      placeholder="021-555xxxx"
                    />
                  </div>
                  <div className="space-y-1">
                    <label className="text-[11px] font-bold text-slate-600 uppercase">
                      Email Korespondensi
                    </label>
                    <input
                      type="email"
                      required
                      value={formFields.email}
                      onChange={(e) =>
                        setFormFields({ ...formFields, email: e.target.value })
                      }
                      className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none font-medium"
                      placeholder="vendor@company.com"
                    />
                  </div>
                </div>
                <div className="space-y-1">
                  <label className="text-[11px] font-bold text-slate-600 uppercase">
                    Alamat Operasional Pabrik / Kantor Utama
                  </label>
                  <textarea
                    required
                    value={formFields.address}
                    onChange={(e) =>
                      setFormFields({ ...formFields, address: e.target.value })
                    }
                    rows={2}
                    className="w-full text-xs px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg outline-none text-slate-800 font-medium resize-none"
                    placeholder="Jl. Raya Industri Barat Blok H No. 12..."
                  />
                </div>
              </div>
            )}

            {/* Tombol Kontrol Bawah Dialog */}
            <DialogFooter className="flex justify-end gap-2 pt-2 border-t border-slate-50">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={isSubmitting}
                onClick={() => {
                  setIsModalOpen(false);
                  setFormFields({
                    kode: "",
                    name: "",
                    description: "",
                    slug: "",
                    parentId: "",
                    picName: "",
                    phone: "",
                    email: "",
                    address: "",
                  });
                }}
                className="text-xs h-8 cursor-pointer"
              >
                Batal
              </Button>
              {/* 🌟 FIX: Ubah type menjadi "button" (bukan submit) dan jalankan aksi via onClick */}
              <Button
                type="button"
                size="sm"
                disabled={isSubmitting}
                onClick={handleDynamicQuickSubmit}
                className="bg-blue-600 hover:bg-blue-700 text-white text-xs h-8 flex items-center gap-1 cursor-pointer"
              >
                {isSubmitting ? (
                  <Loader2 className="size-3 animate-spin" />
                ) : (
                  <Save className="size-3" />
                )}
                <span>Daftarkan Data</span>
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
