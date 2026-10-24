import { RouteShell } from "@/components/layout/route-shell";

export default async function ProductCategoriesLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RouteShell
      title="Product Categories Management"
      permission="categories:read"
    >
      {children}
    </RouteShell>
  );
}
