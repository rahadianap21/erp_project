import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="flex flex-1 flex-col gap-4 p-4 sm:p-6 lg:p-8">
      <div className="flex items-center justify-between border-b pb-4">
        <Skeleton className="h-7 w-48" />
        <Skeleton className="h-9 w-28" />
      </div>
      <Skeleton className="h-96 w-full" />
    </div>
  );
}
