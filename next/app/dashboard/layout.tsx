import { RouteShell } from "@/components/layout/route-shell";
import { Input } from "@/components/ui/input";
import { SearchIcon } from "lucide-react";

export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RouteShell
      title="Enterprise Overview"
      permission="dashboard:read"
      headerActions={
        <div className="flex items-center gap-2 sm:gap-3">
          <div className="relative hidden md:block w-56 lg:w-72">
            <SearchIcon className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
            <Input
              placeholder="Search orders, SKUs, invoices..."
              className="h-9 pl-8 text-xs bg-muted/40"
            />
          </div>
        </div>
      }
    >
      {children}
    </RouteShell>
  );
}
