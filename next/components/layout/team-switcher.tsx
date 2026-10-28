"use client";

import Link from "next/link";
import {
  BoxesIcon,
  ChevronsUpDownIcon,
  LandmarkIcon,
  StoreIcon,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

const teams = [
  {
    name: "Axion ERP",
    plan: "Enterprise Suite",
    url: "/dashboard",
    icon: BoxesIcon,
  },
  {
    name: "Axion POS",
    plan: "Point of Sale",
    url: "/pos",
    icon: StoreIcon,
  },
  {
    name: "Accounting",
    plan: "Finance & Ledger",
    url: "/accounting",
    icon: LandmarkIcon,
  },
];

export function TeamSwitcher() {
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                size="lg"
                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              />
            }
          >
            <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
              <BoxesIcon className="size-4" />
            </div>
            <div className="grid flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
              <span className="truncate font-semibold">Axion ERP</span>
              <span className="truncate text-xs text-muted-foreground">
                Enterprise Suite
              </span>
            </div>
            <ChevronsUpDownIcon className="ml-auto size-4 group-data-[collapsible=icon]:hidden" />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="min-w-56 rounded-lg"
            align="start"
            side="bottom"
            sideOffset={4}
          >
            <DropdownMenuLabel className="text-xs text-muted-foreground">
              Aplikasi
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              {teams.map((team) => {
                const Icon = team.icon;
                return (
                  <DropdownMenuItem
                    key={team.name}
                    render={<Link href={team.url} />}
                    className="cursor-pointer gap-3 p-2"
                  >
                    <div className="flex size-7 items-center justify-center rounded-md border bg-background">
                      <Icon className="size-4" />
                    </div>
                    <div className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-medium">{team.name}</span>
                      <span className="truncate text-xs text-muted-foreground">
                        {team.plan}
                      </span>
                    </div>
                  </DropdownMenuItem>
                );
              })}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
