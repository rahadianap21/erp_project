import { RouteShell } from "@/components/layout/route-shell"

export default async function PaymentMethodsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <RouteShell title="Metode Pembayaran">
      {children}
    </RouteShell>
  )
}
