"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef } from "react";
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";

export type NavItem = {
  title: string;
  url: string;
  icon?: React.ComponentType<{ className?: string }>;
  isActive?: boolean;
  permission?: string;
  badge?: string;
};

export type NavGroup = {
  label: string;
  items: NavItem[];
};

function isItemActive(pathname: string, item: NavItem) {
  if (item.isActive !== undefined) return item.isActive;
  if (pathname === item.url) return true;
  return item.url !== "/dashboard" && pathname.startsWith(`${item.url}/`);
}

export function NavMain({ groups }: { groups: NavGroup[] }) {
  const pathname = usePathname();
  const { isMobile, setOpenMobile } = useSidebar();

  // Tutup sheet navigasi mobile hanya setelah berpindah halaman. Jangan
  // menutupnya saat NavMain pertama kali di-mount di dalam sheet, karena itu
  // membuat menu langsung hilang begitu tombol hamburger ditekan.
  const previousPathname = useRef(pathname);

  useEffect(() => {
    if (isMobile && previousPathname.current !== pathname) {
      setOpenMobile(false);
    }
    previousPathname.current = pathname;
  }, [pathname, isMobile, setOpenMobile]);

  return (
    <>
      {groups.map((group) => (
        <SidebarGroup key={group.label} className="py-0">
          <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
          <SidebarMenu>
            {group.items.map((item) => {
              const Icon = item.icon;
              const active = isItemActive(pathname, item);

              return (
                <SidebarMenuItem key={`${group.label}-${item.title}`}>
                  <SidebarMenuButton
                    tooltip={item.title}
                    isActive={active}
                    render={<Link href={item.url} />}
                  >
                    {Icon && <Icon />}
                    <span>{item.title}</span>
                  </SidebarMenuButton>
                  {item.badge && <SidebarMenuBadge>{item.badge}</SidebarMenuBadge>}
                </SidebarMenuItem>
              );
            })}
          </SidebarMenu>
        </SidebarGroup>
      ))}
    </>
  );
}
