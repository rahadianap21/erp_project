import { RouteShell } from "@/components/layout/route-shell";

export default async function ProductsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RouteShell title="Products Management" permission="products:read">
      {children}
    </RouteShell>
  );
}
