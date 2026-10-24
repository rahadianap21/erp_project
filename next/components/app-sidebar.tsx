"use client";

import * as React from "react";
import { NavMain } from "@/components/nav-main";
import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarRail,
} from "@/components/ui/sidebar";
import {
  LayoutDashboardIcon,
  Settings2Icon,
  UsersIcon,
  ShieldCheckIcon,
  BarChart3Icon,
  Database,
  UserSquare,
  FolderTree,
  Package,
  FileText,
  PackageCheck,
  Landmark,
  BookOpenCheck,
  BookOpenText,
  ShoppingCartIcon,
  ReceiptTextIcon,
  Users2Icon,
  CreditCardIcon,
  TicketPercentIcon,
} from "lucide-react";
interface SubMenuItem {
  title: string;
  url: string;
  icon?: React.ComponentType<{ className?: string }>;
  permission?: string;
}

interface MenuItem {
  title: string;
  url: string;
  icon?: React.ComponentType<{ className?: string }>;
  isActive?: boolean;
  permission?: string;
  items: SubMenuItem[];
}

const ALL_MENU_ITEMS: MenuItem[] = [
  {
    title: "Dashboard",
    url: "/dashboard",
    icon: LayoutDashboardIcon,
    items: [{ title: "Overview", url: "/dashboard", icon: BarChart3Icon }],
  },
  {
    title: "Master Data",
    url: "/suppliers",
    icon: Database,
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
    title: "Transaction",
    url: "/purchase-orders",
    icon: FileText,
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
    title: "Accounting",
    url: "/accounting",
    icon: Landmark,
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
    ],
  },
  {
    title: "Reporting",
    url: "/accounting/reports",
    icon: BarChart3Icon,
    permission: "accounting:read",
    items: [],
  },
  {
    title: "Users Management",
    url: "/users",
    icon: Settings2Icon,
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

  const filteredNavMain = React.useMemo(() => {
    return ALL_MENU_ITEMS.map((parent) => {
      const allowedItems = parent.items.filter((subItem) => {
        if (!subItem.permission) return true;
        return permissions.includes(subItem.permission);
      });

      return {
        ...parent,
        items: allowedItems,
      };
    }).filter((parent) => {
      // Izin pada item induk menyaring seluruh grup bila ditetapkan.
      if (parent.permission && !permissions.includes(parent.permission))
        return false;
      // Grup dengan sub-menu tetap tampil (sub-menu sudah disaring di atas).
      if (parent.items.length > 0) return true;
      // Tautan tunggal hanya tampil bila punya izin yang sudah lolos di atas.
      return Boolean(parent.permission);
    });
  }, [permissions]);

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarContent>
        <NavMain items={filteredNavMain} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
