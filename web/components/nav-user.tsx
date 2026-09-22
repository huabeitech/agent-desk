"use client"

import { useRouter } from "next/navigation"
import { useState } from "react"

import { useAuth } from "@/components/auth-provider"
import { useI18n } from "@/i18n/provider"
import { ChangePasswordDialog } from "@/components/change-password-dialog"
import { LanguageToggle } from "@/components/language-toggle"
import { useNotifications } from "@/components/notification-provider"
import { PaletteToggle } from "@/components/palette-toggle"
import { ThemeToggle } from "@/components/theme-toggle"
import { Badge } from "@/components/ui/badge"
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@/components/ui/avatar"
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
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import {
  BellIcon,
  EllipsisVerticalIcon,
  KeyRoundIcon,
  LogOutIcon
} from "lucide-react"

export function NavUser({
  user,
}: {
  user: {
    name: string
    email: string
    avatar: string
  }
}) {
  const t = useI18n()
  const { signOut } = useAuth()
  const { unreadCount } = useNotifications()
  const { isMobile } = useSidebar()
  const router = useRouter()
  const [changePasswordOpen, setChangePasswordOpen] = useState(false)
  const fallback = user.name.slice(0, 1).toUpperCase() || "U"
  return (
    <>
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <SidebarMenuButton size="lg" className="aria-expanded:bg-muted" />
              }
            >
              <Avatar className="size-8 rounded-lg grayscale">
                <AvatarImage src={user.avatar} alt={user.name} />
                <AvatarFallback className="rounded-lg">{fallback}</AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{user.name}</span>
                <span className="truncate text-xs text-foreground/70">
                  {user.email}
                </span>
              </div>
              <EllipsisVerticalIcon className="ml-auto size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className="min-w-56"
              side={isMobile ? "bottom" : "right"}
              align="end"
              sideOffset={4}
            >
              <DropdownMenuGroup>
                <DropdownMenuLabel className="p-0 font-normal">
                  <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                    <Avatar className="size-8">
                      <AvatarImage src={user.avatar} alt={user.name} />
                      <AvatarFallback className="rounded-lg">{fallback}</AvatarFallback>
                    </Avatar>
                    <div className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-medium">{user.name}</span>
                      <span className="truncate text-xs text-muted-foreground">
                        {user.email}
                      </span>
                    </div>
                  </div>
                </DropdownMenuLabel>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuLabel className="px-2 py-1.5 text-xs font-normal text-muted-foreground">
                  {t("nav.preferences")}
                </DropdownMenuLabel>
                <div className="flex items-center gap-2 px-2 pb-2">
                  <LanguageToggle />
                  <PaletteToggle />
                  <ThemeToggle />
                </div>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem
                  onClick={() => {
                    router.push("/dashboard/notifications")
                  }}
                  className="gap-2"
                >
                  <BellIcon />
                  <span className="flex-1">{t("nav.notifications")}</span>
                  {unreadCount > 0 ? (
                    <Badge className="h-5 min-w-5 px-1.5">
                      {unreadCount > 99 ? "99+" : unreadCount}
                    </Badge>
                  ) : null}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => {
                    setChangePasswordOpen(true)
                  }}
                >
                  <KeyRoundIcon />
                  {t("nav.changePassword")}
                </DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => {
                  void signOut()
                }}
              >
                <LogOutIcon />
                {t("nav.signOut")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
      <ChangePasswordDialog
        open={changePasswordOpen}
        onOpenChange={setChangePasswordOpen}
        onSuccess={signOut}
      />
    </>
  )
}
