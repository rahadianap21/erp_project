"use client";

import { Loader2, ShoppingBag } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

// 🌟 PERBAIKAN ABSOLUT: Pastikan wajib tertulis "export default function"
export default function PurchaseOrdersLoading() {
    return (
        <div className="p-6 space-y-6 max-w-7xl mx-auto animate-fade-in">
            {/* Header Judul Section Skeleton */}
            <div className="flex items-center gap-2 border-b pb-4">
                <div className="p-2 bg-slate-100 text-slate-400 rounded-xl border border-slate-200">
                    <ShoppingBag className="size-5" />
                </div>
                <div className="space-y-2">
                    <Skeleton className="h-5 w-64 rounded" />
                    <Skeleton className="h-3 w-96 rounded" />
                </div>
            </div>

            {/* Form Box Skeleton */}
            <div className="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-2xs">
                <div className="px-6 py-4 bg-slate-50 flex items-center justify-between border-b">
                    <div className="flex items-center gap-2">
                        <Loader2 className="size-4 animate-spin text-blue-600" />
                        <Skeleton className="h-4 w-48 rounded" />
                    </div>
                    <Skeleton className="h-4 w-4 rounded" />
                </div>
            </div>

            {/* Filter Toolbar Control Skeleton */}
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-white p-4 rounded-xl border border-slate-200 shadow-2xs">
                <Skeleton className="h-9 flex-1 max-w-md rounded-lg" />
                <div className="flex items-center gap-2">
                    <Skeleton className="h-9 w-28 rounded-lg" />
                    <Skeleton className="h-9 w-36 rounded-lg" />
                </div>
            </div>

            {/* Table Area Grid Skeleton */}
            <div className="border border-slate-200 rounded-xl bg-white shadow-2xs overflow-hidden">
                <Table>
                    <TableHeader className="bg-slate-50/70 border-b">
                        <TableRow>
                            <TableHead className="w-10"></TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/4">Dokumen Pemesanan / Vendor</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/5">Tanggal PO</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 w-1/5 text-center">Status</TableHead>
                            <TableHead className="font-bold text-xs text-slate-700 text-right w-32">Total Nilai PO</TableHead>
                            <TableHead className="text-right font-bold text-xs text-slate-700 w-32">Aksi</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {Array.from({ length: 5 }).map((_, idx) => (
                            <TableRow key={`po-root-skeleton-${idx}`} className="border-b">
                                <TableCell className="w-10 py-3.5"><Skeleton className="h-3.5 w-3.5 rounded mx-auto" /></TableCell>
                                <TableCell className="py-3.5">
                                    <div className="space-y-1.5">
                                        <Skeleton className="h-4 w-36 rounded" />
                                        <Skeleton className="h-3 w-48 rounded" />
                                    </div>
                                </TableCell>
                                <TableCell className="py-3.5"><Skeleton className="h-4 w-24 rounded" /></TableCell>
                                <TableCell className="py-3.5 text-center"><Skeleton className="h-5 w-16 mx-auto rounded-full" /></TableCell>
                                <TableCell className="py-3.5 text-right"><Skeleton className="h-4 w-20 ml-auto rounded" /></TableCell>
                                <TableCell className="text-right py-3.5"><Skeleton className="h-7 w-14 rounded ml-auto" /></TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>

            {/* Pagination Controls Skeleton */}
            <div className="flex flex-col sm:flex-row items-center justify-between gap-4 bg-white p-4 rounded-xl border border-slate-200 shadow-2xs mt-4">
                <Skeleton className="h-4 w-48 rounded" />
                <div className="flex items-center gap-3">
                    <Skeleton className="h-8 w-16 rounded-lg" />
                    <div className="flex items-center gap-1">
                        <Skeleton className="size-8 rounded-lg" />
                        <Skeleton className="h-8 w-14 rounded-lg" />
                        <Skeleton className="size-8 rounded-lg" />
                    </div>
                </div>
            </div>
        </div>
    );
}
