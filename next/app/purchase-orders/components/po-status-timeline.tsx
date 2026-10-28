import { Badge } from "@/components/ui/badge";
import { History, Ban, PackageCheck, FileClock, Stamp } from "lucide-react";

// Tahap happy path, mengikuti transisi yang ada di backend
// (purchase_order_usecase.go). Warna tiap tahap sesuai statusnya:
// PENDING kuning, APPROVED hijau, RECEIVED biru.
const TAHAP_UTAMA = [
  { status: "PENDING", label: "PENDING", Icon: FileClock },
  { status: "APPROVED", label: "APPROVED", Icon: Stamp },
  { status: "RECEIVED", label: "RECEIVED", Icon: PackageCheck },
] as const;

const WARNA: Record<string, string> = {
  PENDING: "border-amber-500 bg-amber-50 text-amber-500",
  APPROVED: "border-emerald-600 bg-emerald-50 text-emerald-600",
  RECEIVED: "border-blue-600 bg-blue-50 text-blue-600",
  REJECTED: "border-rose-600 bg-rose-50 text-rose-600",
};

// Tahap yang belum dilewati dibiarkan abu-abu: warna hanya dipakai untuk tahap
// yang benar-benar sudah terjadi.
const BELUM = "border-slate-200 bg-slate-50 text-slate-300";

const KOTAK =
  "inline-flex size-8 shrink-0 items-center justify-center rounded-md border";
const IKON = "size-3";

function TooltipStatus({ children }: { children: string }) {
  return (
    <span className="pointer-events-none absolute bottom-full left-1/2 z-10 mb-1 -translate-x-1/2 whitespace-nowrap rounded bg-foreground px-1.5 py-0.5 text-xs font-semibold text-background opacity-0 transition-opacity duration-100 group-hover:opacity-100">
      {children}
    </span>
  );
}

export function POStatusTimeline({ status }: { status: string }) {
  // REJECTED adalah cabang keluar, bukan tahap keempat: dokumen berhenti di
  // sini, bukan melaju ke arah sana.
  if (status === "REJECTED") {
    return (
      <div
        className="flex items-center"
        role="list"
        aria-label="Status dokumen: REJECTED"
      >
        <div className="group relative flex items-center" role="listitem">
          <span className={cn(KOTAK, WARNA.PENDING)}>
            <History className={IKON} strokeWidth={2.5} />
          </span>
          <TooltipStatus>PENDING</TooltipStatus>
        </div>

        <span
          aria-hidden="true"
          className="mx-1 h-[2px] w-4 rounded-sm bg-slate-300"
        />

        <div className="group relative flex items-center" role="listitem">
          <span className={cn(KOTAK, WARNA.REJECTED)}>
            <Ban className={IKON} strokeWidth={2.5} />
          </span>
          <TooltipStatus>REJECTED</TooltipStatus>
        </div>
      </div>
    );
  }

  const indexSekarang = TAHAP_UTAMA.findIndex((t) => t.status === status);

  // Status di luar jalur utama, mis. DRAFT. Menampilkan timeline dengan semua
  // tahap terlewati akan menyesatkan, jadi jatuh ke label polos.
  if (indexSekarang === -1) {
    return (
      <Badge variant="neutral">{String(status ?? "PENDING")}</Badge>
    );
  }

  return (
    <div
      className="flex items-center"
      role="list"
      aria-label={`Status dokumen: ${status}`}
    >
      {TAHAP_UTAMA.map((tahap, i) => {
        const sudahLewat = i < indexSekarang;
        // Kotak berwarna berarti tahap ini sudah terjadi; warna diambil
        // dari status tahap itu sendiri, bukan dari posisi sekarang.
        const warna = i <= indexSekarang ? WARNA[tahap.status] : BELUM;

        return (
          <div
            key={tahap.status}
            className="group relative flex items-center"
            role="listitem"
          >
            {i > 0 && (
              <span
                aria-hidden="true"
                className="mx-1 h-[2px] w-4 rounded-sm bg-slate-300"
              />
            )}
            <span className={cn(KOTAK, warna)}>
              {/* Yang sudah dilewati memakai ikon history, bukan centang:
                                centang berarti "tuntas", history hanya berarti
                                "pernah dilewati". Kalau centang, tahap terakhir akan
                                terbaca sudah diterima padahal belum terjadi. */}
              {sudahLewat ? (
                <History className={IKON} strokeWidth={2.5} />
              ) : (
                <tahap.Icon className={IKON} strokeWidth={2.5} />
              )}
            </span>
            <TooltipStatus>{tahap.label}</TooltipStatus>
          </div>
        );
      })}
    </div>
  );
}

function cn(...classes: Array<string | false | undefined>): string {
  return classes.filter(Boolean).join(" ");
}
