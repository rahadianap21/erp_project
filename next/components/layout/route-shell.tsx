import { getSessionData } from "@/lib/session";
import { AppSidebar } from "@/components/app-sidebar";
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
// sticky + breadcrumb. Halaman cukup meneruskan judul (dan opsional aksi
// header) alih-alih menyalin komposisi ini.
interface RouteShellProps {
  /** Label breadcrumb terakhir, mis. "Users Management". */
  title: string;
  children: React.ReactNode;
  /** Konten opsional di ujung kanan header (mis. kotak pencarian). */
  headerActions?: React.ReactNode;
}

export async function RouteShell({
  title,
  children,
  headerActions,
}: RouteShellProps) {
  const { permissions, user } = await getSessionData();

  return (
    <SidebarProvider>
      <AppSidebar initialPermissions={permissions} initialUser={user} />
      <SidebarInset className="bg-muted/15 min-h-screen">
        <header className="sticky top-0 z-10 flex h-16 shrink-0 items-center justify-between border-b bg-background/95 px-4 backdrop-blur-md transition-[width,height] ease-linear">
          <div className="flex items-center gap-2">
            <SidebarTrigger className="-ml-1" />
            <Separator
              orientation="vertical"
              className="mr-2 data-vertical:h-4 data-vertical:self-auto"
            />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden sm:block">
                  <BreadcrumbLink href="/dashboard">Axion ERP</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden sm:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage className="font-semibold">
                    {title}
                  </BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
          {headerActions}
        </header>
        {children}
      </SidebarInset>
    </SidebarProvider>
  );
}
