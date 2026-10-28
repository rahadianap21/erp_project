"use client";

import * as React from "react";
import { NavMain, type NavGroup } from "@/components/nav-main";
import { NavUser } from "@/components/nav-user";
import { TeamSwitcher } from "@/components/layout/team-switcher";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
import {
  LayoutDashboardIcon,
  Settings2Icon,
  UsersIcon,
  ShieldCheckIcon,
  BarChart3Icon,
  UserSquare,
  FolderTree,
  Package,
  FileText,
  PackageCheck,
  BookOpenCheck,
  BookOpenText,
  ListTree,
  ShoppingCartIcon,
  ReceiptTextIcon,
  Users2Icon,
  CreditCardIcon,
  TicketPercentIcon,
} from "lucide-react";

const NAV_GROUPS: NavGroup[] = [
  {
    label: "Umum",
    items: [
      {
        title: "Dashboard",
        url: "/dashboard",
        icon: LayoutDashboardIcon,
      },
    ],
  },
  {
    label: "Master Data",
    items: [
      {
        title: "Suppliers",
        url: "/suppliers",
        icon: UserSquare,
        permission: "suppliers:read",
      },
      {
        title: "Product Categories",
        url: "/product-categories",
        icon: FolderTree,
        permission: "categories:read",
      },
      {
        title: "Products",
        url: "/products",
        icon: Package,
        permission: "products:read",
      },
      {
        title: "Members",
        url: "/members",
        icon: Users2Icon,
        permission: "members:read",
      },
      {
        title: "Payment Methods",
        url: "/payment-methods",
        icon: CreditCardIcon,
        permission: "payment-methods:read",
      },
      {
        title: "Vouchers",
        url: "/vouchers",
        icon: TicketPercentIcon,
        permission: "vouchers:read",
      },
    ],
  },
  {
    label: "Transaksi",
    items: [
      {
        title: "Purchase Orders",
        url: "/purchase-orders",
        icon: FileText,
        permission: "po:read",
      },
      {
        title: "Goods Receipt Notes",
        url: "/goods-receipt-notes",
        icon: PackageCheck,
        permission: "grn:read",
      },
      {
        title: "Sales",
        url: "/sales",
        icon: ReceiptTextIcon,
        permission: "sales:read",
      },
      {
        title: "POS Terminal",
        url: "/pos",
        icon: ShoppingCartIcon,
        permission: "pos:read",
      },
    ],
  },
  {
    label: "Akuntansi",
    items: [
      {
        title: "Chart of Accounts",
        url: "/accounting/chart-of-accounts",
        icon: BookOpenCheck,
        permission: "accounting:read",
      },
      {
        title: "General Journal",
        url: "/accounting/journals",
        icon: BookOpenText,
        permission: "accounting:read",
      },
      {
        title: "Laporan",
        url: "/accounting/reports",
        icon: BarChart3Icon,
        permission: "accounting:read",
      },
      {
        title: "Akun Otomatis",
        url: "/accounting/account-settings",
        icon: ListTree,
        permission: "accounting:read",
      },
    ],
  },
  {
    label: "Administrasi",
    items: [
      {
        title: "Users",
        url: "/users",
        icon: UsersIcon,
        permission: "users:read",
      },
      {
        title: "Roles & Permissions",
        url: "/roles",
        icon: ShieldCheckIcon,
        permission: "roles:read",
      },
      {
        title: "Pengaturan Pajak",
        url: "/settings",
        icon: Settings2Icon,
        permission: "settings:read",
      },
    ],
  },
];

export interface AppSidebarProps extends React.ComponentProps<typeof Sidebar> {
  initialPermissions?: string[];
  initialUser?: {
    name: string;
    email: string;
    avatar?: string;
  };
}

export function AppSidebar({
  initialPermissions,
  initialUser,
  ...props
}: AppSidebarProps) {
  const permissions = React.useMemo(
    () => initialPermissions ?? [],
    [initialPermissions]
  );
  const user = {
    name: initialUser?.name || "Axion User",
    email: initialUser?.email || "user@axion.erp",
    avatar: initialUser?.avatar || "",
  };

  const filteredGroups = React.useMemo(() => {
    return NAV_GROUPS.map((group) => ({
      ...group,
      items: group.items.filter(
        (item) => !item.permission || permissions.includes(item.permission)
      ),
    })).filter((group) => group.items.length > 0);
  }, [permissions]);

  return (
    <Sidebar collapsible="icon" variant="inset" {...props}>
      <SidebarHeader>
        <TeamSwitcher />
      </SidebarHeader>
      <SidebarContent>
        <NavMain groups={filteredGroups} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
