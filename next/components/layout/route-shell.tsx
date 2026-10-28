import { cookies } from "next/headers";
import { requirePermission } from "@/lib/dal";
import { AppSidebar } from "@/components/app-sidebar";
import { ProfileDropdown } from "@/components/layout/profile-dropdown";
import { ThemeSwitch } from "@/components/layout/theme-switch";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { Separator } from "@/components/ui/separator";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

// RouteShell adalah satu-satunya layout halaman ber-sidebar: sidebar + header
// sticky + breadcrumb. Halaman cukup meneruskan judul dan permission yang
// diperlukan; layout akan memverifikasinya ke server sebelum merender konten.
interface RouteShellProps {
  /** Label breadcrumb terakhir, mis. "Users Management". */
  title: string;
  /** Permission yang wajib dimiliki untuk membuka halaman ini. */
  permission: string;
  children: React.ReactNode;
  /** Konten opsional di ujung kanan header (mis. kotak pencarian). */
  headerActions?: React.ReactNode;
}

export async function RouteShell({
  title,
  permission,
  children,
  headerActions,
}: RouteShellProps) {
  const session = await requirePermission(permission);
  const cookieStore = await cookies();
  const defaultSidebarOpen =
    cookieStore.get("sidebar_state")?.value !== "false";

  return (
    <SidebarProvider defaultOpen={defaultSidebarOpen}>
      <AppSidebar
        initialPermissions={session.permissions}
        initialUser={{
          name: session.name,
          email: session.email,
          avatar: "",
        }}
      />
      <SidebarInset className="@container/content min-h-svh">
        <header className="sticky top-0 z-20 flex h-16 shrink-0 items-center gap-3 bg-background px-4 transition-[width,height] ease-linear">
          <SidebarTrigger
            variant="outline"
            className="-ml-1 max-md:scale-125"
          />
          <Separator
            orientation="vertical"
            className="h-6 data-vertical:h-6 data-vertical:self-auto"
          />
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden sm:block">
                  <BreadcrumbLink href="/dashboard">Axion ERP</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden sm:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage className="truncate font-semibold">
                    {title}
                  </BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
            <div className="ml-auto flex items-center gap-1 sm:gap-2">
              {headerActions}
              <ThemeSwitch />
              <ProfileDropdown
                user={{
                  name: session.name,
                  email: session.email,
                  avatar: "",
                }}
              />
            </div>
          </div>
        </header>
        {children}
      </SidebarInset>
    </SidebarProvider>
  );
}
