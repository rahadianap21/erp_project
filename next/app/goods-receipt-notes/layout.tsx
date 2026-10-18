import { RouteShell } from "@/components/layout/route-shell";

export default async function GoodsReceiptNotesLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <RouteShell title="Goods Receipt Notes">{children}</RouteShell>;
}
