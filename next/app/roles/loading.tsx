import { Skeleton } from "@/components/ui/skeleton";

export default function RolesLoading() {
  return (
    <main className="flex flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8 w-full animate-in fade-in duration-300">
      {/* Form Collapsible Header Skeleton */}
      <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 flex items-center justify-between bg-gray-50">
          <Skeleton className="h-5 w-48" />
          <Skeleton className="h-5 w-5 rounded-full" />
        </div>
      </div>

      {/* Table Area Skeleton */}
      <div className="space-y-4">
        {/* Filter & Search Bar Skeleton */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-card p-4 rounded-xl border shadow-sm">
          <div className="flex flex-1 flex-col sm:flex-row items-stretch sm:items-center gap-3">
            <Skeleton className="h-9 flex-1 max-w-md rounded-md" />
          </div>
          <Skeleton className="h-5 w-36" />
        </div>

        {/* Table Skeleton */}
        <div className="border rounded-xl shadow-sm overflow-hidden bg-card">
          <div className="bg-muted/50 border-b px-4 py-3 flex items-center justify-between">
            <Skeleton className="h-4 w-10" />
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-4 w-16" />
          </div>
          <div className="divide-y">
            {Array.from({ length: 5 }).map((_, idx) => (
              <div
                key={`role-page-skel-${idx}`}
                className="px-4 py-3.5 flex items-center justify-between"
              >
                <Skeleton className="h-4 w-5 rounded" />
                <Skeleton className="h-4 w-36" />
                <div className="flex gap-2">
                  <Skeleton className="h-6 w-6 rounded" />
                  <Skeleton className="h-6 w-6 rounded" />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Pagination Skeleton */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-card p-4 rounded-xl border shadow-2xs">
          <Skeleton className="h-5 w-48" />
          <div className="flex items-center gap-2">
            <Skeleton className="h-8 w-24 rounded-md" />
            <Skeleton className="h-8 w-8 rounded-md" />
            <Skeleton className="h-8 w-24 rounded-md" />
          </div>
        </div>
      </div>
    </main>
  );
}
