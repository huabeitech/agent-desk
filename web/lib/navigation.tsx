import {
  BotMessageSquareIcon,
  BrainCircuitIcon,
  Building2Icon,
  BadgePercentIcon,
  CalendarClockIcon,
  ClipboardListIcon,
  FileTextIcon,
  GlobeIcon,
  KeyRoundIcon,
  LayoutDashboardIcon,
  MessageSquareCodeIcon,
  MessageSquareMoreIcon,
  PackageIcon,
  ShieldCheckIcon,
  SparklesIcon,
  TagsIcon,
  UserCogIcon,
  UsersIcon,
  WorkflowIcon,
} from "lucide-react";
import type { ReactNode } from "react";

/** Keep in sync with backend internal/pkg/constants/auth.go RoleCodeSuperAdmin. */
export const DASHBOARD_ROLE_SUPER_ADMIN = "super_admin";

export type DashboardNavMenuItem = {
  title: string;
  titleKey: string;
  url: string;
  icon: ReactNode;
};

export type DashboardNavItemConfig = Omit<DashboardNavMenuItem, "title"> & {
  /**
   * Keep in sync with backend Permission.Code. Missing value means any signed-in
   * admin can see the module.
   */
  requiredPermission?: string;
};

export type DashboardNavSectionConfig = {
  titleKey: string;
  icon: ReactNode;
  items: DashboardNavItemConfig[];
};

function navItemVisible(
  item: DashboardNavItemConfig,
  superAdmin: boolean,
  permissionSet: Set<string>,
): boolean {
  if (superAdmin) {
    return true;
  }
  if (!item.requiredPermission) {
    return true;
  }
  return permissionSet.has(item.requiredPermission);
}

export function filterDashboardNavForSession(
  permissions: readonly string[] | undefined,
  roles: readonly string[] | undefined,
): { titleKey: string; icon: ReactNode; items: DashboardNavMenuItem[] }[] {
  const superAdmin = roles?.includes(DASHBOARD_ROLE_SUPER_ADMIN) ?? false;
  const permissionSet = new Set(permissions ?? []);
  return dashboardNavSections
    .map((section) => ({
      titleKey: section.titleKey,
      icon: section.icon,
      items: section.items
        .filter((item) => navItemVisible(item, superAdmin, permissionSet))
        .map(({ titleKey, url, icon }) => ({ title: titleKey, titleKey, url, icon })),
    }))
    .filter((section) => section.items.length > 0);
}

export function filterDashboardSecondaryNavForSession(
  permissions: readonly string[] | undefined,
  roles: readonly string[] | undefined,
): DashboardNavMenuItem[] {
  const superAdmin = roles?.includes(DASHBOARD_ROLE_SUPER_ADMIN) ?? false;
  const permissionSet = new Set(permissions ?? []);
  return dashboardSecondaryNav
    .filter((item) => navItemVisible(item, superAdmin, permissionSet))
    .map(({ titleKey, url, icon }) => ({ title: titleKey, titleKey, url, icon }));
}

export const dashboardNavSections: DashboardNavSectionConfig[] = [
  // {
  //   title: "Overview",
  //   items: [
  //     {
  //       title: "Overview",
  //       url: "/",
  //       icon: <LayoutDashboardIcon />,
  //     },
  //   ],
  // },
  {
    titleKey: "nav.receptionCenter",
    icon: <BotMessageSquareIcon />,
    items: [
      {
        titleKey: "nav.overview",
        url: "/dashboard",
        icon: <LayoutDashboardIcon />,
      },
      {
        titleKey: "nav.conversations",
        url: "/dashboard/conversations",
        icon: <BotMessageSquareIcon />,
        requiredPermission: "conversation.view",
      },
      {
        titleKey: "nav.tickets",
        url: "/dashboard/tickets",
        icon: <FileTextIcon />,
        requiredPermission: "ticket.view",
      },
      {
        titleKey: "nav.conversationMonitor",
        url: "/dashboard/conversation-monitor",
        icon: <BotMessageSquareIcon />,
        requiredPermission: "conversation.view",
      },
      {
        titleKey: "nav.customers",
        url: "/dashboard/customers",
        icon: <UsersIcon />,
        requiredPermission: "customer.view",
      },
      {
        titleKey: "nav.salesLeads",
        url: "/dashboard/sales-leads",
        icon: <ClipboardListIcon />,
        requiredPermission: "salesLead.view",
      },
      {
        titleKey: "nav.storeSetup",
        url: "/dashboard/store-setup",
        icon: <SparklesIcon />,
        requiredPermission: "digitalStore.view",
      },
      {
        titleKey: "nav.operationManual",
        url: "/dashboard/help",
        icon: <FileTextIcon />,
        requiredPermission: "digitalStore.view",
      },
      {
        titleKey: "nav.digitalStore",
        url: "/dashboard/digital-store",
        icon: <BotMessageSquareIcon />,
        requiredPermission: "digitalStore.view",
      },
      {
        titleKey: "nav.products",
        url: "/dashboard/products",
        icon: <PackageIcon />,
        requiredPermission: "product.view",
      },
      {
        titleKey: "nav.promotions",
        url: "/dashboard/promotions",
        icon: <BadgePercentIcon />,
        requiredPermission: "promotion.view",
      },
      {
        titleKey: "nav.companies",
        url: "/dashboard/companies",
        icon: <Building2Icon />,
        requiredPermission: "company.view",
      },
    ],
  },
  {
    titleKey: "nav.agentConfig",
    icon: <UserCogIcon />,
    items: [
      {
        titleKey: "nav.tags",
        url: "/dashboard/tags",
        icon: <TagsIcon />,
        requiredPermission: "tag.view",
      },
      {
        titleKey: "nav.quickReplies",
        url: "/dashboard/quick-replies",
        icon: <MessageSquareMoreIcon />,
        requiredPermission: "quickReply.view",
      },
      {
        titleKey: "nav.agents",
        url: "/dashboard/agents",
        icon: <UserCogIcon />,
        requiredPermission: "agent.view",
      },
      {
        titleKey: "nav.agentTeamSchedules",
        url: "/dashboard/agent-team-schedules",
        icon: <CalendarClockIcon />,
        requiredPermission: "agentTeamSchedule.view",
      },
      {
        titleKey: "nav.channels",
        url: "/dashboard/channels",
        icon: <GlobeIcon />,
        requiredPermission: "channel.view",
      },
    ],
  },
  {
    titleKey: "nav.aiCapabilities",
    icon: <BrainCircuitIcon />,
    items: [
      {
        titleKey: "nav.knowledge",
        url: "/dashboard/knowledge",
        icon: <FileTextIcon />,
        requiredPermission: "knowledgeBase.view",
      },
      {
        titleKey: "nav.aiConfigs",
        url: "/dashboard/ai-configs",
        icon: <BrainCircuitIcon />,
        requiredPermission: "aiConfig.view",
      },
      {
        titleKey: "nav.aiAgents",
        url: "/dashboard/ai-agents",
        icon: <MessageSquareMoreIcon />,
        requiredPermission: "aiAgent.view",
      },
      {
        titleKey: "nav.skillDefinition",
        url: "/dashboard/skill-definition",
        icon: <MessageSquareCodeIcon />,
        requiredPermission: "skillDefinition.view",
      },
      {
        titleKey: "nav.mcp",
        url: "/dashboard/mcp",
        icon: <MessageSquareCodeIcon />,
        requiredPermission: "mcp.view",
      },
      {
        titleKey: "nav.workflowRuns",
        url: "/dashboard/ai-workflow-runs",
        icon: <WorkflowIcon />,
        requiredPermission: "aiAgent.view",
      },
    ],
  },
  {
    titleKey: "nav.system",
    icon: <ShieldCheckIcon />,
    items: [
      {
        titleKey: "nav.users",
        url: "/dashboard/users",
        icon: <UsersIcon />,
        requiredPermission: "user.view",
      },
      {
        titleKey: "nav.roles",
        url: "/dashboard/roles",
        icon: <ShieldCheckIcon />,
        requiredPermission: "role.view",
      },
      {
        titleKey: "nav.permissions",
        url: "/dashboard/permissions",
        icon: <KeyRoundIcon />,
        requiredPermission: "permission.view",
      },
    ],
  },
];

export const dashboardSecondaryNav: DashboardNavItemConfig[] = [
  // {
  //   title: "System Settings",
  //   url: "/settings",
  //   icon: <Settings2Icon />,
  // },
  // {
  //   title: "Help Center",
  //   url: "/help",
  //   icon: <LifeBuoyIcon />,
  // },
];

export const dashboardQuickActions = [
  {
    title: "View Conversations",
    icon: <BotMessageSquareIcon />,
  },
  {
    title: "Invite Members",
    icon: <UserCogIcon />,
  },
  {
    title: "Connect Bot",
    icon: <MessageSquareCodeIcon />,
  },
] as const;

export function getPageTitle(pathname: string): string {
  return getPageTitleKey(pathname);
}

export function getPageTitleKey(pathname: string): string {
  let matchedTitle = "nav.dashboardHome";
  let longestMatch = 0;

  for (const section of dashboardNavSections) {
    for (const item of section.items) {
      if (pathname === item.url || pathname.startsWith(item.url + "/")) {
        const matchLength = item.url.length;
        if (matchLength > longestMatch) {
          longestMatch = matchLength;
          matchedTitle = item.titleKey;
        }
      }
    }
  }

  for (const item of dashboardSecondaryNav) {
    if (pathname === item.url || pathname.startsWith(item.url + "/")) {
      const matchLength = item.url.length;
      if (matchLength > longestMatch) {
        longestMatch = matchLength;
        matchedTitle = item.titleKey;
      }
    }
  }

  return matchedTitle;
}
