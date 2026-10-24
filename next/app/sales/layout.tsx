import { RouteShell } from "@/components/layout/route-shell";

export default async function SalesLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RouteShell title="Penjualan" permission="sales:read">
      {children}
    </RouteShell>
  );
}
