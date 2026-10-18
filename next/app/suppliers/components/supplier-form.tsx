"use client";

import { apiGet, apiPost, apiPut, messageFromError } from "@/lib/api-client";
import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  ChevronDown,
  ChevronUp,
  Loader2,
  MapPin,
  Building2,
  User,
  Phone,
  Mail,
  CreditCard,
} from "lucide-react";
import { toast } from "sonner";
import { ComboboxSelect } from "@/components/combobox-select";

interface Region {
  id: string;
  name: string;
}

interface VillageRegion extends Region {
  postal_code: string;
}

interface BankOption {
  id: number;
  kode: string;
  name: string;
  short_name: string;
}

interface Supplier {
  id: number;
  name: string;
  pic_name: string;
  phone: string;
  email: string;
  address: string;
  village_id: string;
  province_id?: string;
  regency_id?: string;
  district_id?: string;
  bank_id?: number | null;
  account_number?: string;
  account_name?: string;
  is_active: boolean;
}

interface SupplierFormProps {
  token: string | undefined;
  editingSupplier: Supplier | null;
  onSuccess: (mode: "create" | "update", name: string) => void;
  onCancelEdit: () => void;
}

export default function SupplierForm({
  token,
  editingSupplier,
  onSuccess,
  onCancelEdit,
}: SupplierFormProps) {
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  // State Input Form Utama
  const [name, setName] = useState<string>("");
  const [picName, setPicName] = useState<string>("");
  const [phone, setPhone] = useState<string>("");
  const [email, setEmail] = useState<string>("");
  const [address, setAddress] = useState<string>("");
  const [isActive, setIsActive] = useState<boolean>(true);

  const [bankOptions, setBankOptions] = useState<BankOption[]>([]);
  const [selectedBank, setSelectedBank] = useState<string>("");
  const [accountNumber, setAccountNumber] = useState<string>("");
  const [accountName, setAccountName] = useState<string>("");

  // State Cascading Dropdown Alamat
  const [provinces, setProvinces] = useState<Region[]>([]);
  const [regencies, setRegencies] = useState<Region[]>([]);
  const [districts, setDistricts] = useState<Region[]>([]);
  const [villages, setVillageRegions] = useState<VillageRegion[]>([]);

  const [selectedProvince, setSelectedProvince] = useState<string>("");
  const [selectedRegency, setSelectedRegency] = useState<string>("");
  const [selectedDistrict, setSelectedDistrict] = useState<string>("");
  const [selectedVillage, setSelectedVillage] = useState<string>("");
  const [postalCode, setPostalCode] = useState<string>("");

  // Load Data Provinsi Pertama Kali (On Mount)
  useEffect(() => {
    if (token) fetchRegions("", setProvinces);
    fetchMasterBanks();
  }, [token]);

  // Jalur Reaktif Membaca Mode Edit
  useEffect(() => {
    if (editingSupplier) {
      setName(editingSupplier.name);
      setPicName(editingSupplier.pic_name);
      setPhone(editingSupplier.phone);
      setEmail(editingSupplier.email);
      setAddress(editingSupplier.address);
      setIsActive(editingSupplier.is_active);
      setSelectedBank(
        editingSupplier.bank_id ? String(editingSupplier.bank_id) : ""
      );
      setAccountNumber(editingSupplier.account_number || "");
      setAccountName(editingSupplier.account_name || "");

      // Set ID wilayah lama agar dropdown cascading terkunci otomatis
      setSelectedProvince(editingSupplier.province_id || "");
      setSelectedRegency(editingSupplier.regency_id || "");
      setSelectedDistrict(editingSupplier.district_id || "");
      setSelectedVillage(editingSupplier.village_id || "");
      setIsExpanded(true);
    } else {
      resetForm();
    }
  }, [editingSupplier]);

  // Trigger Fetching Regional Bertingkat (Cascading Select Logic)
  useEffect(() => {
    if (selectedProvince && token)
      fetchRegions(`?province_id=${selectedProvince}`, setRegencies);
    else setRegencies([]);
  }, [selectedProvince, token]);

  useEffect(() => {
    if (selectedRegency && token)
      fetchRegions(`?regency_id=${selectedRegency}`, setDistricts);
    else setDistricts([]);
  }, [selectedRegency, token]);

  useEffect(() => {
    if (selectedDistrict && token)
      fetchRegions(`?district_id=${selectedDistrict}`, setVillageRegions);
    else setVillageRegions([]);
  }, [selectedDistrict, token]);

  useEffect(() => {
    const villageObj = villages.find((v) => v.id === selectedVillage);
    if (villageObj) setPostalCode(villageObj.postal_code || "");
    else setPostalCode("");
  }, [selectedVillage, villages]);

  const fetchRegions = async <T,>(
    query: string,
    setter: (data: T[]) => void
  ) => {
    try {
      const result = await apiGet<T[]>(`/api/regions${query}`);
      if (result.data) setter(result.data);
    } catch (err) {
      console.error("Gagal memuat data regional:", err);
    }
  };

  const resetForm = () => {
    setName("");
    setPicName("");
    setPhone("");
    setEmail("");
    setAddress("");
    setSelectedProvince("");
    setSelectedRegency("");
    setSelectedDistrict("");
    setSelectedVillage("");
    setPostalCode("");
    setSelectedBank("");
    setAccountNumber("");
    setAccountName("");
    setIsActive(true);
  };

  const fetchMasterBanks = async () => {
    try {
      const result = await apiGet<BankOption[]>(`/api/banks`);
      if (result.data) setBankOptions(result.data);
    } catch (err) {
      console.error("Gagal memuat master bank:", err);
    }
  };

  // Memoized options untuk Combobox
  const provinceOptions = useMemo(() => {
    return provinces.map((p) => ({ value: p.id, label: p.name }));
  }, [provinces]);

  const regencyOptions = useMemo(() => {
    return regencies.map((r) => ({ value: r.id, label: r.name }));
  }, [regencies]);

  const districtOptions = useMemo(() => {
    return districts.map((d) => ({ value: d.id, label: d.name }));
  }, [districts]);

  const villageOptions = useMemo(() => {
    return villages.map((v) => ({ value: v.id, label: v.name }));
  }, [villages]);

  const bankSelectOptions = useMemo(() => {
    return bankOptions.map((b) => ({
      value: String(b.id),
      label: `[${b.kode}] ${b.short_name} - ${b.name}`,
    }));
  }, [bankOptions]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!selectedVillage) {
      toast.error("Validasi Gagal", {
        description:
          "Mohon lengkapi alamat regional hingga tingkat Kelurahan / Desa.",
      });
      return;
    }

    setIsLoading(true);

    const isEditMode = editingSupplier !== null;
    const url = isEditMode
      ? `/api/suppliers?id=${editingSupplier.id}`
      : `/api/suppliers`;

    try {
      const payload = {
        name,
        pic_name: picName,
        phone,
        email,
        address,
        village_id: selectedVillage,
        is_active: isActive,
        bank_id: selectedBank ? Number(selectedBank) : null, // 🌟 Kirim Bank ID
        account_number: accountNumber || null, // 🌟 Kirim No Rekening
        account_name: accountName || null,
      };
      if (isEditMode) await apiPut(url, payload);
      else await apiPost(url, payload);

      resetForm();
      onSuccess(isEditMode ? "update" : "create", name);
    } catch (error) {
      toast.error("Gagal Menyimpan", { description: messageFromError(error) });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden transition-all duration-300">
      {/* Header Tombol Collapsible */}
      <button
        type="button"
        onClick={() => !editingSupplier && setIsExpanded(!isExpanded)}
        disabled={!!editingSupplier}
        className="w-full px-6 py-3.5 flex items-center justify-between bg-slate-50/80 hover:bg-slate-100/60 transition-colors text-left cursor-pointer"
      >
        <span className=" text-sm font-semibold text-gray-800">
          {editingSupplier
            ? `Edit Supplier: ${editingSupplier.name}`
            : "Tambah Supplier Baru"}
        </span>
        {!editingSupplier &&
          (isExpanded ? (
            <ChevronUp className="h-4 w-4" />
          ) : (
            <ChevronDown className="h-4 w-4" />
          ))}
      </button>

      {/* Konten Form Ekpansi */}
      <div
        className={`transition-all duration-300 ease-in-out ${isExpanded ? "max-h-[800px] p-6 border-t border-slate-100" : "max-h-0 opacity-0 pointer-events-none"}`}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Baris 1: Informasi Kontak Utama */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <Building2 className="size-3" /> Nama Perusahaan
              </label>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900"
                placeholder="PT. Logistik Jaya"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <User className="size-3" /> Nama PIC Narahubung
              </label>
              <input
                type="text"
                required
                value={picName}
                onChange={(e) => setPicName(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900"
                placeholder="Ahmad Dani"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <Phone className="size-3" /> No. Telepon
              </label>
              <input
                type="text"
                required
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900"
                placeholder="08123456xxx"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <Mail className="size-3" /> Email Bisnis
              </label>
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900"
                placeholder="supplier@company.com"
              />
            </div>
          </div>

          {/* Baris 2: Alamat Jalan Fisik */}
          <div className="space-y-1.5">
            <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
              <MapPin className="size-3" /> Alamat Fisik (Jalan, RT/RW, No.
              Rumah)
            </label>
            <textarea
              required
              value={address}
              onChange={(e) => setAddress(e.target.value)}
              rows={2}
              className="w-full text-xs px-3 py-2 bg-slate-50/50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 resize-none"
              placeholder="Jl. Industri Raya No. 45, Blok B..."
            />
          </div>

          {/* Baris 3: Cascading Dropdown Regional Indonesia */}
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-5 gap-4 bg-slate-50/40 p-4 rounded-xl border border-slate-100">
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-slate-500 uppercase">
                Provinsi
              </label>
              <ComboboxSelect
                options={provinceOptions}
                value={selectedProvince}
                placeholder="Pilih Provinsi..."
                onChange={(val) => {
                  setSelectedProvince(val);
                  setSelectedRegency("");
                  setSelectedDistrict("");
                  setSelectedVillage("");
                }}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-slate-500 uppercase">
                Kota / Kabupaten
              </label>
              <ComboboxSelect
                options={regencyOptions}
                value={selectedRegency}
                placeholder={
                  !selectedProvince
                    ? "Pilih provinsi dulu..."
                    : "Pilih Kota / Kab..."
                }
                disabled={!selectedProvince}
                onChange={(val) => {
                  setSelectedRegency(val);
                  setSelectedDistrict("");
                  setSelectedVillage("");
                }}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-slate-500 uppercase">
                Kecamatan
              </label>
              <ComboboxSelect
                options={districtOptions}
                value={selectedDistrict}
                placeholder={
                  !selectedRegency ? "Pilih kota dulu..." : "Pilih Kecamatan..."
                }
                disabled={!selectedRegency}
                onChange={(val) => {
                  setSelectedDistrict(val);
                  setSelectedVillage("");
                }}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-slate-500 uppercase">
                Kelurahan / Desa
              </label>
              <ComboboxSelect
                options={villageOptions}
                value={selectedVillage}
                placeholder={
                  !selectedDistrict
                    ? "Pilih kecamatan dulu..."
                    : "Pilih Kelurahan..."
                }
                disabled={!selectedDistrict}
                onChange={(val) => {
                  setSelectedVillage(val);
                }}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-slate-500 uppercase">
                Kode Pos
              </label>
              <input
                type="text"
                readOnly
                value={postalCode}
                className="w-full text-xs px-2.5 py-1.5 bg-slate-100 border border-slate-200 rounded-lg text-slate-500 font-mono outline-none h-8"
                placeholder="Otomatis"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 bg-blue-50/20 p-4 rounded-xl border border-blue-100/40">
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <CreditCard className="size-3" /> Bank Tujuan Transfer
              </label>
              <ComboboxSelect
                options={bankSelectOptions}
                value={selectedBank}
                placeholder="Pilih Bank Vendor (Opsional)..."
                emptyText="Bank tidak ditemukan"
                onChange={(val) => {
                  setSelectedBank(val);
                }}
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <CreditCard className="size-3" /> Nomor Rekening Bank
              </label>
              <input
                type="text"
                value={accountNumber}
                onChange={(e) => setAccountNumber(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 font-mono"
                placeholder="Masukkan nomor rekening (Contoh: 801234xxxx)"
              />
            </div>
            <div className="space-y-1.5">
              <label className="text-[11px] font-bold tracking-wide text-slate-600 uppercase flex items-center gap-1">
                <User className="size-3" /> Atas Nama rekening
              </label>
              <input
                type="text"
                value={accountName}
                onChange={(e) => setAccountName(e.target.value)}
                className="w-full text-xs px-3 py-2 bg-white border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-900 font-mono"
                placeholder="Atas Nama rekening"
              />
            </div>
          </div>

          {/* Tombol Kontrol */}
          <div className="flex justify-end space-x-3 pt-3 border-t border-slate-100">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                onCancelEdit();
                resetForm();
                setIsExpanded(false);
              }}
              className="text-xs h-8 cursor-pointer"
            >
              Batal
            </Button>
            <Button
              type="submit"
              size="sm"
              disabled={isLoading}
              className="bg-blue-600 hover:bg-blue-700 text-white text-xs h-8 shadow-xs cursor-pointer"
            >
              {isLoading ? (
                <span className="flex items-center gap-1.5">
                  <Loader2 className="size-3 animate-spin" /> Memproses...
                </span>
              ) : editingSupplier ? (
                "Perbarui Data"
              ) : (
                "Simpan Supplier"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
