import { getSessionData } from "@/lib/session"
import { PosTopbar } from "@/components/pos-topbar"

export const metadata = {
  title: "Axion POS",
}

export default async function PosLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { user } = await getSessionData()

  return (
    <div className="flex min-h-svh flex-col bg-muted/30">
      <PosTopbar user={user} />
      <main className="flex-1">{children}</main>
    </div>
  )
}
