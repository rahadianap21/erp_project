import { Skeleton } from "@/components/ui/skeleton";
import { SupplierTableSkeleton } from "./components/supplier-table";

export default function SuppliersLoading() {
  return (
    <main className="flex flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8 w-full animate-in fade-in duration-300">
      {/* Form Collapsible Header Skeleton */}
      <div className="bg-card rounded-lg shadow-sm border overflow-hidden">
        <div className="px-6 py-4 flex items-center justify-between bg-muted/40">
          <Skeleton className="h-5 w-48" />
          <Skeleton className="h-5 w-5 rounded-full" />
        </div>
      </div>

      {/* Table Area Skeleton (Identik & Konsisten) */}
      <SupplierTableSkeleton />
    </main>
  );
}
