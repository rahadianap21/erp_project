import { RouteShell } from "@/components/layout/route-shell";

export default async function PurchaseOrdersLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <RouteShell title="Purchase Orders">{children}</RouteShell>;
}
