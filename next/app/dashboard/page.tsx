"use client";

import { apiGet } from "@/lib/api-client";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  AlertTriangleIcon,
  ArrowUpRightIcon,
  BoxesIcon,
  ClockIcon,
  DollarSignIcon,
  FileText,
  Package,
  PackageCheck,
  TruckIcon,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { formatIDR } from "@/lib/money";

interface DashboardStats {
  total_po_value: number;
  total_po: number;
  po_pending: number;
  po_approved: number;
  po_received: number;
  total_grn: number;
  total_products: number;
  total_suppliers: number;
  low_stock_count: number;
}

interface LowStockItem {
  id: number;
  sku: string;
  name: string;
  current_stock: number;
  min_stock_level: number;
  selling_unit: string;
}

interface RecentPO {
  id: number;
  po_number: string;
  supplier_name: string;
  total_amount: number;
  status: string;
  po_date: string;
}

interface RecentGRN {
  id: number;
  grn_number: string;
  po_number: string;
  total_qty: number;
  received_date: string;
}

interface DashboardOverview {
  stats: DashboardStats;
  low_stock: LowStockItem[];
  recent_pos: RecentPO[];
  recent_grns: RecentGRN[];
}

function formatDate(value: string) {
  if (!value) return "-";
  return new Date(value).toLocaleDateString("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

function statusVariant(status: string) {
  switch (status) {
    case "APPROVED":
      return "info";
    case "RECEIVED":
      return "success";
    case "REJECTED":
      return "danger";
    default:
      return "warning";
  }
}

const QUICK_ACTIONS = [
  {
    title: "Purchase Order",
    description: "Buat & kelola dokumen PO",
    href: "/purchase-orders",
    icon: <FileText className="size-4" />,
    color: "bg-primary/10 text-primary",
  },
  {
    title: "Penerimaan Barang",
    description: "Catat & lihat riwayat GRN",
    href: "/goods-receipt-notes",
    icon: <PackageCheck className="size-4" />,
    color: "bg-blue-500/10 text-blue-600",
  },
  {
    title: "Produk",
    description: "Master barang & stok gudang",
    href: "/products",
    icon: <Package className="size-4" />,
    color: "bg-emerald-500/10 text-emerald-600",
  },
  {
    title: "Supplier",
    description: "Data vendor & rekening",
    href: "/suppliers",
    icon: <TruckIcon className="size-4" />,
    color: "bg-amber-500/10 text-amber-600",
  },
];

export default function Page() {
  const [overview, setOverview] = useState<DashboardOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadDashboard = useCallback(async () => {
    try {
      const res = await apiGet<DashboardOverview>("/api/dashboard");
      setOverview(res.data);
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Gagal memuat data dashboard"
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadDashboard();
  }, [loadDashboard]);

  const retry = () => {
    setLoading(true);
    loadDashboard();
  };

  const stats = overview?.stats;
  const statCards = [
    {
      title: "Nilai Purchase Order",
      value: formatIDR(stats?.total_po_value ?? 0),
      subtitle: `${stats?.total_po ?? 0} dokumen diterbitkan`,
      icon: <DollarSignIcon className="size-5 text-primary" />,
    },
    {
      title: "Menunggu Persetujuan",
      value: `${stats?.po_pending ?? 0} PO`,
      subtitle: `${stats?.po_approved ?? 0} disetujui, ${stats?.po_received ?? 0} diterima`,
      icon: <ClockIcon className="size-5 text-blue-500" />,
    },
    {
      title: "Stok Menipis",
      value: `${stats?.low_stock_count ?? 0} SKU`,
      subtitle: "Perlu reorder segera",
      icon: <AlertTriangleIcon className="size-5 text-amber-500" />,
    },
    {
      title: "Produk Active",
      value: `${stats?.total_products ?? 0} item`,
      subtitle: `${stats?.total_suppliers ?? 0} supplier aktif`,
      icon: <BoxesIcon className="size-5 text-emerald-500" />,
    },
  ];

  return (
    <main className="flex flex-1 flex-col gap-6 p-4 sm:p-6 lg:p-8">
      {/* Welcome Banner */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 rounded-2xl bg-gradient-to-r from-primary/10 via-primary/5 to-transparent p-6 border border-primary/15 shadow-xs">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            Axion ERP Command Center
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            Visibilitas operasional pengadaan, penerimaan barang, dan stok
            gudang secara real-time.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant="success" className="gap-1.5 px-3 py-1">
            <span className="size-2 rounded-full bg-emerald-500 animate-pulse" />
            Sistem Berjalan Normal
          </Badge>
        </div>
      </div>

      {/* Error State */}
      {error && (
        <div className="flex items-center justify-between gap-4 rounded-xl border border-destructive/30 bg-destructive/5 px-5 py-4">
          <p className="text-sm text-destructive">Gagal memuat data: {error}</p>
          <Button
            variant="outline"
            size="sm"
            className="h-8 text-xs"
            onClick={retry}
          >
            Coba Lagi
          </Button>
        </div>
      )}

      {/* KPI Stat Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {loading
          ? Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="rounded-xl border bg-card p-5 shadow-2xs">
              <div className="flex items-center justify-between">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="size-8 rounded-lg" />
              </div>
              <div className="mt-4 space-y-2">
                <Skeleton className="h-6 w-28" />
                <Skeleton className="h-3 w-32" />
              </div>
            </div>
          ))
          : statCards.map((stat, i) => (
            <div
              key={i}
              className="rounded-xl border bg-card p-5 shadow-2xs hover:shadow-sm transition-all"
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground">
                  {stat.title}
                </span>
                <div className="p-2 rounded-lg bg-muted/60">{stat.icon}</div>
              </div>
              <div className="mt-3">
                <div className="text-2xl font-bold tracking-tight text-foreground">
                  {stat.value}
                </div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {stat.subtitle}
                </div>
              </div>
            </div>
          ))}
      </div>

      {/* Quick Actions Shortcuts */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
        {QUICK_ACTIONS.map((action) => (
          <Link
            key={action.href}
            href={action.href}
            className="flex items-center gap-3 p-3.5 rounded-xl border bg-card hover:bg-muted/50 transition-colors text-left group"
          >
            <div
              className={`p-2.5 rounded-lg group-hover:scale-105 transition-transform ${action.color}`}
            >
              {action.icon}
            </div>
            <div>
              <div className="text-xs font-semibold text-foreground">
                {action.title}
              </div>
              <div className="text-xs text-muted-foreground">
                {action.description}
              </div>
            </div>
          </Link>
        ))}
      </div>

      {/* Main Grid: Recent Orders + Side Widgets */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column: Recent Purchase Orders (2 cols) */}
        <div className="lg:col-span-2 rounded-xl border bg-card p-5 shadow-2xs">
          <div className="flex items-center justify-between pb-4 border-b">
            <div>
              <h2 className="text-base font-semibold text-foreground">
                Purchase Order Terbaru
              </h2>
              <p className="text-xs text-muted-foreground mt-0.5">
                Dokumen pengadaan terakhir yang tercatat
              </p>
            </div>
            <Link
              href="/purchase-orders"
              className="inline-flex h-8 max-md:min-h-12 items-center gap-1 rounded-lg border border-transparent px-2.5 text-xs font-medium text-primary hover:bg-muted/60 transition-colors"
            >
              Lihat Semua
              <ArrowUpRightIcon className="size-3" />
            </Link>
          </div>

          <div className="overflow-x-auto mt-4">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <th className="pb-3 font-medium">Nomor PO</th>
                  <th className="pb-3 font-medium">Supplier</th>
                  <th className="pb-3 font-medium">Total</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium">Tanggal</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/60">
                {loading ? (
                  Array.from({ length: 4 }).map((_, i) => (
                    <tr key={i}>
                      <td className="py-3" colSpan={5}>
                        <Skeleton className="h-4 w-full" />
                      </td>
                    </tr>
                  ))
                ) : overview && overview.recent_pos.length > 0 ? (
                  overview.recent_pos.map((po) => (
                    <tr
                      key={po.id}
                      className="hover:bg-muted/40 transition-colors"
                    >
                      <td className="py-3 font-semibold text-foreground whitespace-nowrap">
                        {po.po_number}
                      </td>
                      <td className="py-3">
                        <div className="font-medium text-foreground">
                          {po.supplier_name}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {formatDate(po.po_date)}
                        </div>
                      </td>
                      <td className="py-3 font-semibold text-foreground whitespace-nowrap">
                        {formatIDR(po.total_amount)}
                      </td>
                      <td className="py-3">
                        <Badge variant={statusVariant(po.status)}>
                          {po.status}
                        </Badge>
                      </td>
                      <td className="py-3 text-muted-foreground whitespace-nowrap">
                        {formatDate(po.po_date)}
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan={5}
                      className="py-8 text-center text-muted-foreground"
                    >
                      Belum ada dokumen Purchase Order.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right Column: Low Stock Alerts & Recent Receipts */}
        <div className="space-y-6">
          {/* Low Stock Alerts */}
          <div className="rounded-xl border bg-card p-5 shadow-2xs">
            <div className="flex items-center justify-between pb-3 border-b">
              <div className="flex items-center gap-2">
                <AlertTriangleIcon className="size-4 text-amber-500" />
                <h2 className="text-sm font-semibold text-foreground">
                  Stok Kritis
                </h2>
              </div>
              <Badge variant="warning">
                {stats?.low_stock_count ?? 0} item
              </Badge>
            </div>

            <div className="space-y-3 mt-4">
              {loading ? (
                Array.from({ length: 2 }).map((_, i) => (
                  <Skeleton key={i} className="h-14 w-full rounded-lg" />
                ))
              ) : overview && overview.low_stock.length > 0 ? (
                overview.low_stock.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between p-3 rounded-lg border bg-muted/30"
                  >
                    <div>
                      <div className="text-xs font-semibold text-foreground">
                        {item.name}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {item.sku} • {item.selling_unit}
                      </div>
                    </div>
                    <div className="text-right">
                      <span className="text-xs font-bold text-destructive">
                        {item.current_stock} tersisa
                      </span>
                      <div className="text-xs text-muted-foreground">
                        Min: {item.min_stock_level}
                      </div>
                    </div>
                  </div>
                ))
              ) : (
                <p className="text-xs text-muted-foreground py-2">
                  Semua stok di atas ambang minimum.
                </p>
              )}
            </div>
          </div>

          {/* Recent Receipts */}
          <div className="rounded-xl border bg-card p-5 shadow-2xs">
            <div className="flex items-center justify-between pb-3 border-b">
              <h2 className="text-sm font-semibold text-foreground">
                Penerimaan Terbaru
              </h2>
              <Link
                href="/goods-receipt-notes"
                className="text-xs text-primary hover:underline max-md:inline-flex max-md:min-h-12 max-md:items-center"
              >
                Lihat Semua
              </Link>
            </div>

            <div className="space-y-4 mt-4">
              {loading ? (
                Array.from({ length: 2 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))
              ) : overview && overview.recent_grns.length > 0 ? (
                overview.recent_grns.map((grn) => (
                  <div key={grn.id} className="flex items-start gap-3">
                    <div className="mt-0.5 rounded-full p-1.5 bg-muted">
                      <PackageCheck className="size-4 text-emerald-500" />
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-medium text-foreground">
                          {grn.grn_number}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          {formatDate(grn.received_date)}
                        </span>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {grn.total_qty} unit diterima • PO {grn.po_number}
                      </p>
                    </div>
                  </div>
                ))
              ) : (
                <p className="text-xs text-muted-foreground py-2">
                  Belum ada penerimaan barang.
                </p>
              )}
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
