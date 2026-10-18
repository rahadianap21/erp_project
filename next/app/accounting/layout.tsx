import { RouteShell } from "@/components/layout/route-shell";

export default async function AccountingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <RouteShell title="Accounting">{children}</RouteShell>;
}
