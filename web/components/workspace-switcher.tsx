"use client"

import {
  Building2Icon,
  CheckIcon,
  ChevronsUpDownIcon,
  LayoutDashboardIcon,
  PlusIcon,
  SettingsIcon,
  WrenchIcon,
} from "lucide-react"
import Link from "next/link"
import { useEffect, useState, type ReactElement } from "react"
import { toast } from "sonner"

import { CreateOrganizationDialog, ManageOrganizationDialog } from "@/components/organization-dialogs"
import { useI18n } from "@/i18n/provider"
import { fetchPublicConfig, type PublicConfig } from "@/lib/api/config"
import { listMyOrganizations, switchOrganization, type OrganizationItem } from "@/lib/api/organization"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

export type WorkspaceKey = "dashboard" | "workbench"

export type WorkspaceOption = {
  key: WorkspaceKey
  href: string
  labelKey: string
  icon: typeof LayoutDashboardIcon
}

export const workspaceOptions: WorkspaceOption[] = [
  {
    key: "dashboard",
    href: "/dashboard",
    labelKey: "workspace.dashboard",
    icon: LayoutDashboardIcon,
  },
  {
    key: "workbench",
    href: "/workbench",
    labelKey: "workspace.workbench",
    icon: WrenchIcon,
  },
]

type WorkspaceSwitcherProps = {
  currentWorkspace: WorkspaceKey
  variant?: "sidebar" | "header" | "rail"
  className?: string
  trigger?: ReactElement
}

export function WorkspaceSwitcher({
  currentWorkspace,
  variant = "header",
  className,
  trigger,
}: WorkspaceSwitcherProps) {
  const t = useI18n()
  const [orgs, setOrgs] = useState<OrganizationItem[]>([])
  const [activeOrgId, setActiveOrgId] = useState<number | null>(null)
  const [switching, setSwitching] = useState(false)
  const [publicConfig, setPublicConfig] = useState<PublicConfig | null>(null)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [manageDialogOpen, setManageDialogOpen] = useState(false)

  const loadOrgs = () => {
    listMyOrganizations()
      .then((res) => {
        if (res?.organizations) {
          setOrgs(res.organizations)
          setActiveOrgId(res.currentOrganizationId || res.organizations[0]?.id || null)
        }
      })
      .catch(() => {})
  }

  useEffect(() => {
    let mounted = true
    loadOrgs()
    fetchPublicConfig()
      .then((cfg) => {
        if (mounted) setPublicConfig(cfg)
      })
      .catch(() => {})
    return () => {
      mounted = false
    }
  }, [])

  const currentOption =
    workspaceOptions.find((item) => item.key === currentWorkspace) ?? workspaceOptions[0]
  const currentOrg = orgs.find((o) => o.id === activeOrgId) || orgs[0]

  const brandName = publicConfig?.companyName || t("app.brand")
  const brandLogo = publicConfig?.companyLogoUrl || "/images/logo.svg"

  const handleSwitchOrg = async (orgId: number) => {
    if (orgId === activeOrgId || switching) return
    setSwitching(true)
    try {
      await switchOrganization(orgId)
      setActiveOrgId(orgId)
      toast.success("Switched organization successfully")
      window.location.href = "/dashboard"
    } catch {
      toast.error("Failed to switch organization")
    } finally {
      setSwitching(false)
    }
  }

  const switchIndicatorClassName =
    "absolute bottom-0.5 right-0.5 size-2.5 rounded-full bg-sidebar text-sidebar-foreground/70"

  const triggerClassName = cn(
    "gap-2 text-left",
    variant === "header" &&
      "h-9 rounded-md border border-border/70 bg-background px-2.5 shadow-xs hover:bg-muted",
    variant === "sidebar" &&
      "relative data-[slot=sidebar-menu-button]:p-1.5! group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:p-0! group-data-[collapsible=icon]:data-[slot=sidebar-menu-button]:p-0!",
    variant === "rail" &&
      "relative size-8 rounded-md border-0 bg-transparent p-0 shadow-none hover:bg-sidebar-accent",
    className
  )

  const triggerContent =
    variant === "rail" ? (
      <>
        <img
          src={brandLogo}
          alt={brandName}
          width="32"
          height="32"
          className="size-7 shrink-0 object-contain"
        />
        <span className="sr-only">
          {currentOrg?.name || brandName} - {t(currentOption.labelKey)}
        </span>
        <ChevronsUpDownIcon className={switchIndicatorClassName} />
      </>
    ) : (
      <>
        <img
          src={brandLogo}
          alt={brandName}
          width="32"
          height="32"
          className="size-7 shrink-0 object-contain"
        />
        <div className="grid min-w-0 flex-1 text-left leading-tight">
          <span className="truncate text-sm font-semibold">{currentOrg?.name || brandName}</span>
          <span className="truncate text-xs text-muted-foreground">
            {t(currentOption.labelKey)} {currentOrg?.role ? `• ${currentOrg.role}` : ""}
          </span>
        </div>
        <ChevronsUpDownIcon className="ml-auto size-4 shrink-0 text-muted-foreground group-data-[collapsible=icon]:hidden" />
        {variant === "sidebar" ? (
          <ChevronsUpDownIcon
            className={cn(switchIndicatorClassName, "hidden group-data-[collapsible=icon]:block")}
          />
        ) : null}
      </>
    )

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            trigger ?? <Button variant="ghost" className={triggerClassName} />
          }
        >
          {triggerContent}
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="start"
          side={variant === "sidebar" || variant === "rail" ? "right" : "bottom"}
          sideOffset={8}
          className="w-64 min-w-64"
        >
          {orgs.length > 0 ? (
            <>
              <DropdownMenuGroup>
                <DropdownMenuLabel className="flex items-center justify-between text-xs text-muted-foreground font-medium">
                  <span className="flex items-center gap-1.5">
                    <Building2Icon className="size-3.5" />
                    Organizations / Workspaces
                  </span>
                  {currentOrg ? (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation()
                        setManageDialogOpen(true)
                      }}
                      className="hover:text-foreground inline-flex items-center gap-1 p-0.5 rounded"
                      title="Manage organization members"
                    >
                      <SettingsIcon className="size-3.5" />
                    </button>
                  ) : null}
                </DropdownMenuLabel>
                {orgs.map((org) => {
                  const isActive = org.id === activeOrgId
                  return (
                    <DropdownMenuItem
                      key={org.id}
                      className="cursor-pointer gap-2"
                      onClick={() => handleSwitchOrg(org.id)}
                    >
                      <div className="flex flex-1 flex-col min-w-0">
                        <span className="truncate font-medium text-sm">{org.name}</span>
                        <span className="truncate text-xs text-muted-foreground">
                          {org.role || "Member"} {org.plan ? `• ${org.plan}` : ""}
                        </span>
                      </div>
                      {isActive ? <CheckIcon className="size-4 text-primary shrink-0" /> : null}
                    </DropdownMenuItem>
                  )
                })}
                <DropdownMenuItem
                  className="cursor-pointer gap-2 text-primary font-medium mt-1"
                  onClick={() => setCreateDialogOpen(true)}
                >
                  <PlusIcon className="size-4" />
                  <span>Create Organization</span>
                </DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
            </>
          ) : (
            <>
              <DropdownMenuGroup>
                <DropdownMenuItem
                  className="cursor-pointer gap-2 text-primary font-medium"
                  onClick={() => setCreateDialogOpen(true)}
                >
                  <PlusIcon className="size-4" />
                  <span>Create Organization</span>
                </DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
            </>
          )}

          <DropdownMenuGroup>
            <DropdownMenuLabel>{t("workspace.switchWorkspace")}</DropdownMenuLabel>
            {workspaceOptions.map((item) => (
              <DropdownMenuItem
                key={item.key}
                render={<Link href={item.href} />}
                className="cursor-pointer gap-2"
              >
                <item.icon className="size-4 text-muted-foreground" />
                <span className="flex-1 truncate">{t(item.labelKey)}</span>
                {item.key === currentWorkspace ? (
                  <CheckIcon className="size-4 text-primary" />
                ) : null}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      <CreateOrganizationDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onCreated={(newOrg) => {
          loadOrgs()
          setActiveOrgId(newOrg.id)
          window.location.href = "/dashboard"
        }}
      />

      <ManageOrganizationDialog
        open={manageDialogOpen}
        onOpenChange={setManageDialogOpen}
        organization={currentOrg || null}
        onUpdated={() => {
          loadOrgs()
        }}
      />
    </>
  )
}
