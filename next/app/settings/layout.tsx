import { RouteShell } from "@/components/layout/route-shell"

export default async function SettingsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <RouteShell title="Pengaturan Pajak">
      {children}
    </RouteShell>
  )
}
