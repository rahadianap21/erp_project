import { RouteShell } from "@/components/layout/route-shell"

export default async function SuppliersLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <RouteShell title="Suppliers Management">
      {children}
    </RouteShell>
  )
}
