import { RouteShell } from "@/components/layout/route-shell";

export default async function VouchersLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RouteShell title="Master Voucher" permission="vouchers:read">
      {children}
    </RouteShell>
  );
}
