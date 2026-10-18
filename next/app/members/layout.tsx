import { RouteShell } from "@/components/layout/route-shell"

export default async function MembersLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <RouteShell title="Master Member">
      {children}
    </RouteShell>
  )
}
