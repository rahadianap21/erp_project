import { RouteShell } from "@/components/layout/route-shell"

export default async function UsersLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <RouteShell title="Users Management">
      {children}
    </RouteShell>
  )
}
