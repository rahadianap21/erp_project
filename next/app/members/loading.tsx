import { Skeleton } from "@/components/ui/skeleton";

export default function MembersLoading() {
    return (
        <main className="flex flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8">
            <div className="bg-card rounded-lg shadow-sm border overflow-hidden">
                <div className="px-6 py-4 flex items-center justify-between bg-muted/40">
                    <Skeleton className="h-5 w-48" />
                    <Skeleton className="h-9 w-32" />
                </div>
            </div>
            <div className="space-y-3">
                {Array.from({ length: 6 }).map((_, i) => (
                    <Skeleton key={i} className="h-12 w-full" />
                ))}
            </div>
        </main>
    );
}
