"use client";

import Cookies from "js-cookie";
import { LogOutIcon, MenuIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

// auth_token tidak ada di sini karena httpOnly: hanya route handler
// /api/auth/logout yang bisa menghapusnya.
const SESSION_KEYS = [
  "session_active",
  "user_email",
  "user_name",
  "user_permissions",
  "user_app",
];

export function PosTopbar({ user }: { user: { name: string; email: string } }) {
  const handleLogout = async () => {
    for (const key of SESSION_KEYS) {
      Cookies.remove(key, { path: "/" });
    }

    try {
      await fetch("/api/auth/logout", { method: "POST" });
    } catch (err) {
      console.error("Gagal logout:", err);
    }

    window.location.replace("/");
  };

  return (
    <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-b bg-background px-4 4xl:h-16 4xl:px-8">
      <div className="flex items-center gap-2 font-semibold 4xl:gap-3 4xl:text-lg">
        <span className="grid size-10 place-items-center rounded-lg bg-primary text-sm font-bold text-primary-foreground md:size-8 4xl:size-10 4xl:text-base">
          A
        </span>
        Axion POS
      </div>
      <div className="flex items-center gap-3 4xl:gap-4">
        <span className="hidden text-sm text-muted-foreground md:inline 4xl:text-base">
          {user.name}
        </span>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleLogout}
          className="hidden cursor-pointer md:inline-flex 4xl:h-10 4xl:px-4 4xl:text-base"
        >
          <LogOutIcon className="size-4 4xl:size-5" />
          Keluar
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                className="size-12 cursor-pointer md:hidden"
                aria-label="Buka menu pengguna"
              />
            }
          >
            <MenuIcon className="size-5" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel className="truncate text-sm font-medium text-foreground">
              {user.name}
            </DropdownMenuLabel>
            <p className="truncate px-1.5 pb-2 text-xs text-muted-foreground">
              {user.email}
            </p>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={handleLogout}
              className="cursor-pointer text-destructive focus:text-destructive"
            >
              <LogOutIcon />
              Keluar
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
