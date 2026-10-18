"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { ChevronRightIcon } from "lucide-react"
import { motion } from "framer-motion"
import { useEffect, useState } from "react"
import { cn } from "@/lib/utils"

type NavSubItem = {
  title: string
  url: string
  icon?: React.ComponentType<{ className?: string }>
}

type NavItem = {
  title: string
  url: string
  icon?: React.ComponentType<{ className?: string }>
  isActive?: boolean
  permission?: string
  items?: NavSubItem[]
}

// NavCollapsible mengelola sendiri status buka/tutup grup menu. State-nya
// controlled supaya Base UI tidak memperingatkan perubahan `defaultOpen`
// (uncontrolled) saat rute berpindah-pindah.
function NavCollapsible({
  item,
  isCurrentActive,
  pathname,
}: {
  item: NavItem
  isCurrentActive: boolean
  pathname: string
}) {
  const Icon = item.icon
  const [open, setOpen] = useState(isCurrentActive)

  useEffect(() => {
    setOpen(isCurrentActive)
  }, [isCurrentActive])

  return (
    <Collapsible
      open={open}
      onOpenChange={(nextOpen) => setOpen(nextOpen)}
      className="group/collapsible"
      render={<SidebarMenuItem />}
    >
      <CollapsibleTrigger
        render={<SidebarMenuButton tooltip={item.title} isActive={isCurrentActive} />}
      >
        {Icon && <Icon className="h-4 w-4" />}
        <span>{item.title}</span>
        <ChevronRightIcon className="ml-auto transition-transform duration-200 group-data-open/collapsible:rotate-90" />
      </CollapsibleTrigger>

      <CollapsibleContent
        render={
          <motion.div
            initial={false}
            animate={{
              height: "auto",
              opacity: 1,
              transition: { height: { duration: 0.25, ease: "easeOut" }, opacity: { duration: 0.2 } }
            }}
            exit={{
              height: 0,
              opacity: 0,
              transition: { height: { duration: 0.2, ease: "easeIn" }, opacity: { duration: 0.15 } }
            }}
            style={{ overflow: "hidden" }}
          />
        }
      >
        <SidebarMenuSub>
          {item.items?.map((subItem) => {
            const isSubActive = pathname === subItem.url
            const SubIcon = subItem.icon
            return (
              <SidebarMenuSubItem key={subItem.title}>
                <SidebarMenuSubButton
                  isActive={isSubActive}
                  render={<Link href={subItem.url} />}
                >
                  {SubIcon && <SubIcon className="h-4 w-4 shrink-0" />}
                  <span>{subItem.title}</span>
                </SidebarMenuSubButton>
              </SidebarMenuSubItem>
            )
          })}
        </SidebarMenuSub>
      </CollapsibleContent>
    </Collapsible>
  )
}

export function NavMain({
  items,
}: {
  items: NavItem[]
}) {
  const pathname = usePathname()
  const { state, isMobile } = useSidebar()
  const isCollapsed = state === "collapsed" && !isMobile

  return (
    <SidebarGroup>
      <SidebarGroupLabel>Platform</SidebarGroupLabel>
      <SidebarMenu>
        {items.map((item) => {
          const Icon = item.icon
          const hasSubItems = Boolean(item.items && item.items.length > 0)
          const isCurrentActive =
            item.isActive ??
            (item.url === pathname || Boolean(item.items?.some((sub) => sub.url === pathname)))

          // Jika tidak ada child items, render sebagai link biasa
          if (!hasSubItems) {
            return (
              <SidebarMenuItem key={item.title}>
                <SidebarMenuButton
                  tooltip={item.title}
                  isActive={isCurrentActive}
                  render={<Link href={item.url} />}
                >
                  {Icon && <Icon className="h-4 w-4" />}
                  <span>{item.title}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            )
          }

          // Ketika sidebar tertutup (collapsed icon mode):
          // Render sebagai DropdownMenu flyout ke samping kanan
          if (isCollapsed) {
            return (
              <SidebarMenuItem key={item.title}>
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <SidebarMenuButton
                        isActive={isCurrentActive}
                        className="aria-expanded:bg-sidebar-accent aria-expanded:text-sidebar-accent-foreground"
                      />
                    }
                  >
                    {Icon && <Icon className="h-4 w-4" />}
                    <span>{item.title}</span>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent
                    side="right"
                    align="start"
                    sideOffset={8}
                    className="min-w-48 rounded-lg shadow-lg border border-sidebar-border bg-sidebar text-sidebar-foreground p-1 z-50"
                  >
                    <DropdownMenuGroup>
                      <DropdownMenuLabel className="px-2 py-1.5 font-semibold text-xs text-sidebar-foreground/70">
                        {item.title}
                      </DropdownMenuLabel>
                    </DropdownMenuGroup>
                    <DropdownMenuSeparator className="bg-sidebar-border" />
                    <DropdownMenuGroup>
                      {item.items?.map((subItem) => {
                        const isSubActive = pathname === subItem.url
                        const SubIcon = subItem.icon
                        return (
                          <DropdownMenuItem
                            key={subItem.title}
                            render={<Link href={subItem.url} />}
                            className={cn(
                              "cursor-pointer flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-sidebar-accent hover:text-sidebar-accent-foreground transition-colors",
                              isSubActive && "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
                            )}
                          >
                            {SubIcon && <SubIcon className="h-4 w-4 shrink-0" />}
                            <span>{subItem.title}</span>
                          </DropdownMenuItem>
                        )
                      })}
                    </DropdownMenuGroup>
                  </DropdownMenuContent>
                </DropdownMenu>
              </SidebarMenuItem>
            )
          }

          // Ketika sidebar terbuka (expanded) atau di mobile:
          // Gunakan Collapsible accordion standar
          return (
            <NavCollapsible
              key={item.title}
              item={item}
              isCurrentActive={isCurrentActive}
              pathname={pathname}
            />
          )
        })}
      </SidebarMenu>
    </SidebarGroup>
  )
}

