import { RouteShell } from "@/components/layout/route-shell";

export default async function RolesLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RouteShell title="Roles Management" permission="roles:read">
      {children}
    </RouteShell>
  );
}
