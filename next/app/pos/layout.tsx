import { requirePermission } from "@/lib/dal";
import { PosTopbar } from "@/components/pos-topbar";

export const metadata = {
  title: "Axion POS",
};

export default async function PosLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await requirePermission("pos:read");

  return (
    <div className="flex min-h-svh flex-col bg-muted/30">
      <PosTopbar user={{ name: session.name, email: session.email }} />
      <main className="flex-1">{children}</main>
    </div>
  );
}
