"use client";

import {
  ArrowRightLeftIcon,
  BotMessageSquareIcon,
  ChevronLeft,
  ChevronRight,
  ChevronsUpDown,
  CircleUserRoundIcon,
  CircleXIcon,
  ClipboardListIcon,
  FilePlus2Icon,
  Menu,
  MoreHorizontalIcon,
  X,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import type { PanelImperativeHandle } from "react-resizable-panels";

import { ConversationCloseDialog } from "@/components/conversation-actions/close-dialog";
import { ConversationTransferDialog } from "@/components/conversation-actions/transfer-dialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Sheet, SheetContent } from "@/components/ui/sheet";
import { useI18n } from "@/i18n/provider";
import {
  agentConversationFilterOptions,
  agentConversationSelectors,
  type AgentConversationFilterKey,
  useAgentConversationsStore,
} from "@/lib/stores/agent-conversations";
import { generateConversationFollowUpAdvice } from "@/lib/api/admin";
import { resumeAIConversation } from "@/lib/api/agent";
import { CreateTicketFromConversationDialog } from "../../tickets/_components/create-ticket-from-conversation-dialog";
import { ChatPanel } from "./chat-panel";
import { ConversationInfoPanel } from "./conversation-info-panel";
import { ConversationList } from "./conversation-list";

const workbenchIconButtonClassName =
  "size-8 text-muted-foreground hover:bg-muted hover:text-foreground";

function getCustomerOnlineClassName(online?: boolean) {
  return online
    ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/15 dark:text-emerald-300"
    : "border-border bg-muted text-muted-foreground";
}

function getCustomerOnlineDotClassName(online?: boolean) {
  return online ? "bg-emerald-500" : "bg-muted-foreground/70";
}

export function ConversationWorkbench() {
  const t = useI18n();
  const conversation = useAgentConversationsStore(
    agentConversationSelectors.selectedConversation,
  );
  const conversationFilter = useAgentConversationsStore(
    (state) => state.conversationFilter,
  );
  const setConversationFilter = useAgentConversationsStore(
    (state) => state.setConversationFilter,
  );
  const loadConversations = useAgentConversationsStore(
    (state) => state.loadConversations,
  );
  const loadMessages = useAgentConversationsStore(
    (state) => state.loadMessages,
  );
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [infoPanelCollapsed, setInfoPanelCollapsed] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [mobileCustomerSheetOpen, setMobileCustomerSheetOpen] = useState(false);
  const [transferOpen, setTransferOpen] = useState(false);
  const [closeOpen, setCloseOpen] = useState(false);
  const [createTicketOpen, setCreateTicketOpen] = useState(false);
  const [resumingAI, setResumingAI] = useState(false);
  const [generatingFollowUpAdvice, setGeneratingFollowUpAdvice] = useState(false);
  const sidebarPanelRef = useRef<PanelImperativeHandle | null>(null);
  const infoPanelRef = useRef<PanelImperativeHandle | null>(null);
  const filterContainerRef = useRef<HTMLDivElement | null>(null);
  const filterMeasureRef = useRef<HTMLDivElement | null>(null);
  const [showFilterDropdown, setShowFilterDropdown] = useState(false);

  useEffect(() => {
    const container = filterContainerRef.current;
    const measure = filterMeasureRef.current;
    if (!container || !measure) {
      return;
    }

    const updateFilterMode = () => {
      setShowFilterDropdown(measure.scrollWidth > container.clientWidth);
    };

    updateFilterMode();

    const observer = new ResizeObserver(() => {
      updateFilterMode();
    });

    observer.observe(container);
    observer.observe(measure);

    return () => {
      observer.disconnect();
    };
  }, []);

  const currentFilterOption =
    agentConversationFilterOptions.find((opt) => opt.value === conversationFilter) ??
    agentConversationFilterOptions[0];
  const getFilterLabel = (labelKey: string) => t(labelKey);
  const canResumeAI =
    Boolean(conversation) &&
    conversation?.serviceMode === 3 &&
    conversation.status !== 1 &&
    conversation.status !== 4;

  useEffect(() => {
    void loadConversations().catch((error) => {
      toast.error(error instanceof Error ? error.message : t("conversation.loadListFailed"));
    });
  }, [loadConversations, conversationFilter, t]);

  async function handleConversationChanged(conversationId: number) {
    await loadConversations();
    await loadMessages(conversationId, {
      forceLoading: false,
      reset: false,
    });
  }

  async function handleResumeAI() {
    if (!conversation || resumingAI || !canResumeAI) return;
    setResumingAI(true);
    try {
      await resumeAIConversation(conversation.id, t("conversation.resumeAIReason"));
      toast.success(t("conversation.resumeAISuccess"));
      await loadConversations();
      await loadMessages(conversation.id, { forceLoading: true, reset: true });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("conversation.resumeAIFailed"));
    } finally {
      setResumingAI(false);
    }
  }

  async function handleCopyFollowUpAdvice() {
    if (!conversation || generatingFollowUpAdvice) return;
    setGeneratingFollowUpAdvice(true);
    try {
      const advice = await generateConversationFollowUpAdvice(conversation.id);
      if (!advice.copyText) {
        toast.error("暂时没有可复制的跟进摘要");
        return;
      }
      await navigator.clipboard.writeText(advice.copyText);
      toast.success(advice.leadId ? "线索跟进摘要已复制" : "会话跟进摘要已复制");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "生成跟进摘要失败");
    } finally {
      setGeneratingFollowUpAdvice(false);
    }
  }

  const handleSidebarToggle = () => {
    const panel = sidebarPanelRef.current;
    if (!panel) {
      setSidebarCollapsed((current) => !current);
      return;
    }

    if (panel.isCollapsed()) {
      panel.expand();
      setSidebarCollapsed(false);
      return;
    }

    panel.collapse();
    setSidebarCollapsed(true);
  };

  const handleInfoPanelToggle = () => {
    const panel = infoPanelRef.current;
    if (!panel) {
      setInfoPanelCollapsed((current) => !current);
      return;
    }

    if (panel.isCollapsed()) {
      panel.expand();
      setInfoPanelCollapsed(false);
      return;
    }

    panel.collapse();
    setInfoPanelCollapsed(true);
  };

  const renderConversationSidebar = (opts?: { onListAfterSelect?: () => void }) => (
    <div className="flex h-full min-h-0 flex-1 flex-col bg-inherit">
      <div className="flex h-12.5 shrink-0 items-start justify-between gap-2 border-b border-border/80 bg-card px-2 py-2">
        <div ref={filterContainerRef} className="relative min-w-0 flex-1">
          {showFilterDropdown ? (
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="outline"
                    className="h-8.5 w-full min-w-0 justify-between gap-2 px-3 text-xs sm:text-sm"
                  />
                }
              >
                <span className="truncate">
                  {currentFilterOption
                    ? getFilterLabel(currentFilterOption.labelKey)
                    : t("conversation.filterPlaceholder")}
                </span>
                <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-44 min-w-44">
                <DropdownMenuRadioGroup
                  value={conversationFilter}
                  onValueChange={(value) =>
                    setConversationFilter(value as AgentConversationFilterKey)
                  }
                >
                  {agentConversationFilterOptions.map((opt) => (
                    <DropdownMenuRadioItem key={opt.value} value={opt.value}>
                      {getFilterLabel(opt.labelKey)}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <Tabs
              value={conversationFilter}
              onValueChange={(value) =>
                setConversationFilter(value as AgentConversationFilterKey)
              }
              className="min-w-0 flex-1 gap-0"
            >
              <TabsList
                className="w-full min-w-0 justify-start"
              >
                {agentConversationFilterOptions.map((opt) => (
                  <TabsTrigger
                    key={opt.value}
                    value={opt.value}
                    className="shrink-0 px-2.5 text-xs sm:text-sm"
                  >
                    {getFilterLabel(opt.labelKey)}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          )}
          <div
            ref={filterMeasureRef}
            className="pointer-events-none absolute whitespace-nowrap opacity-0"
            aria-hidden="true"
          >
            <div className="inline-flex">
              {agentConversationFilterOptions.map((opt) => (
                <span
                  key={opt.value}
                  className="shrink-0 px-2.5 text-xs sm:text-sm"
                >
                  {getFilterLabel(opt.labelKey)}
                </span>
              ))}
            </div>
          </div>
        </div>
        <Button
          variant="ghost"
          size="icon"
          className={`${workbenchIconButtonClassName} mt-0.5 shrink-0 lg:hidden`}
          onClick={() => setMobileMenuOpen(false)}
        >
          <X className="size-4" />
        </Button>
      </div>
      <ConversationList onAfterSelect={opts?.onListAfterSelect} />
    </div>
  );

  const workspaceContent = (
    <div className="flex h-full min-h-0 w-full flex-1 flex-col overflow-hidden bg-card text-card-foreground">
      <div className="flex h-12.5 shrink-0 items-center justify-between gap-3 border-b border-border/80 bg-card px-3 py-1">
        <div className="flex min-w-0 items-center gap-2 sm:gap-3">
          <Button
            variant="ghost"
            size="icon"
            className={`${workbenchIconButtonClassName} lg:hidden`}
            onClick={() => setMobileMenuOpen(true)}
          >
            <Menu className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className={`${workbenchIconButtonClassName} hidden lg:flex`}
            onClick={handleSidebarToggle}
          >
            {sidebarCollapsed ? (
              <ChevronRight className="size-4" />
            ) : (
              <ChevronLeft className="size-4" />
            )}
          </Button>
          {conversation ? (
            <>
              <Avatar className="size-8 shrink-0 lg:size-9">
                <AvatarImage src="" />
                <AvatarFallback className="bg-primary/10 text-sm text-primary">
                  {t("conversation.customerAvatar")}
                </AvatarFallback>
              </Avatar>
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <p className="min-w-0 truncate text-sm font-medium leading-tight">
                    {conversation.customerName ||
                      t("conversation.customerFallback", {
                        id: conversation.customerId || conversation.id,
                      })}
                  </p>
                  <span
                    className={`inline-flex shrink-0 items-center gap-1 rounded-md border px-1.5 py-0.5 text-[11px] leading-none ${getCustomerOnlineClassName(
                      conversation.customerOnline,
                    )}`}
                  >
                    <span
                      className={`size-1.5 rounded-full ${getCustomerOnlineDotClassName(
                        conversation.customerOnline,
                      )}`}
                    />
                    {conversation.customerOnline
                      ? t("conversation.customerOnline")
                      : t("conversation.customerOffline")}
                  </span>
                </div>
                <p className="mt-0.5 truncate text-xs text-muted-foreground">
                  <span>{t("conversation.channelNumber", { id: conversation.channelId || "-" })}</span>
                  {conversation.customerId ? (
                    <>
                      <span className="text-muted-foreground/60"> / </span>
                      <span>{t("conversation.linkedCustomer")}</span>
                    </>
                  ) : null}
                </p>
              </div>
            </>
          ) : (
            <div className="min-w-0">
              <p className="truncate font-medium text-[14px] leading-tight">
                {t("conversation.workbenchTitle")}
              </p>
              <p className="mt-0.5 truncate text-[14px] text-muted-foreground sm:text-[14px] lg:hidden">
                {t("conversation.openMenuSelectConversation")}
              </p>
              <p className="mt-0.5 hidden truncate text-[12px] text-muted-foreground lg:block">
                {t("conversation.selectConversationFromSidebar")}
              </p>
            </div>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-0.5 sm:gap-1">
          <Button
            variant="ghost"
            size="icon"
            className={`${workbenchIconButtonClassName} lg:hidden`}
            disabled={!conversation}
            aria-label={t("conversation.conversationInfo")}
            onClick={() => setMobileCustomerSheetOpen(true)}
          >
            <CircleUserRoundIcon className="size-4" />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon"
                  className={workbenchIconButtonClassName}
                  disabled={!conversation}
                />
              }
            >
              <MoreHorizontalIcon className="size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44 min-w-44">
              <DropdownMenuItem
                onClick={() => setCreateTicketOpen(true)}
                disabled={!conversation}
              >
                <FilePlus2Icon />
                {t("conversation.createTicket")}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => void handleCopyFollowUpAdvice()}
                disabled={!conversation || generatingFollowUpAdvice}
              >
                <ClipboardListIcon />
                {generatingFollowUpAdvice ? "生成中..." : "复制跟进摘要"}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => setTransferOpen(true)}
                disabled={!conversation || conversation.status !== 3}
              >
                <ArrowRightLeftIcon />
                {t("conversation.transferConversation")}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => void handleResumeAI()}
                disabled={!canResumeAI || resumingAI}
              >
                <BotMessageSquareIcon />
                {resumingAI ? t("conversation.resumingAI") : t("conversation.resumeAI")}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => setCloseOpen(true)}
                disabled={!conversation || conversation.status === 4}
              >
                <CircleXIcon />
                {t("conversation.closeConversation")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button
            variant="ghost"
            size="icon"
            className={`${workbenchIconButtonClassName} hidden lg:flex`}
            onClick={handleInfoPanelToggle}
            aria-label={
              infoPanelCollapsed
                ? t("conversation.expandConversationInfo")
                : t("conversation.collapseConversationInfo")
            }
          >
            {infoPanelCollapsed ? (
              <ChevronLeft className="size-4" />
            ) : (
              <ChevronRight className="size-4" />
            )}
          </Button>
        </div>
      </div>
      <div className="flex min-h-0 w-full flex-1 overflow-hidden">
        <ChatPanel />
      </div>
    </div>
  );

  return (
    <div className="flex h-[calc(100dvh-var(--header-height))] min-h-0 w-full min-w-0 flex-col overflow-hidden lg:h-full">
      {mobileMenuOpen && (
        <button
          type="button"
          aria-label={t("conversation.closeConversationList")}
          className="fixed top-12 right-0 bottom-0 left-0 z-30 bg-black/50 lg:hidden"
          onClick={() => setMobileMenuOpen(false)}
        />
      )}
      <div
        className={`fixed top-12 bottom-0 left-0 z-40 flex w-[min(22rem,calc(100vw-0.75rem))] max-w-[min(22rem,calc(100vw-0.75rem))] flex-col overflow-hidden border-r border-border/80 bg-card text-card-foreground shadow-xl transition-transform duration-300 ease-out will-change-transform touch-manipulation overscroll-contain supports-[padding:max(0px)]:pb-[env(safe-area-inset-bottom)] lg:hidden ${
          mobileMenuOpen ? "translate-x-0" : "-translate-x-full pointer-events-none"
        }`}
        aria-hidden={!mobileMenuOpen}
      >
        {renderConversationSidebar({
          onListAfterSelect: () => setMobileMenuOpen(false),
        })}
      </div>

      <div className="flex min-h-0 min-w-0 w-full flex-1 flex-col overflow-hidden lg:hidden">
        {workspaceContent}
      </div>
      <div className="hidden min-h-0 w-full flex-1 overflow-hidden lg:flex">
        <ResizablePanelGroup orientation="horizontal">
          <ResizablePanel
            panelRef={sidebarPanelRef}
            defaultSize="20%"
            minSize="10%"
            maxSize="40%"
            collapsedSize="0%"
            collapsible
            onResize={(panelSize: { asPercentage: number }) => {
              setSidebarCollapsed(panelSize.asPercentage <= 1);
            }}
            className="min-h-0 bg-card"
          >
            <div className="flex h-full min-h-0 flex-col overflow-hidden bg-card text-card-foreground">
              {renderConversationSidebar()}
            </div>
          </ResizablePanel>
          <ResizableHandle withHandle />
          <ResizablePanel defaultSize="50%" minSize="32%" className="min-h-0 bg-card">
            <div className="flex h-full min-h-0 flex-col overflow-hidden">
              {workspaceContent}
            </div>
          </ResizablePanel>
          <ResizableHandle withHandle />
          <ResizablePanel
            panelRef={infoPanelRef}
            defaultSize="500px"
            minSize="20%"
            maxSize="40%"
            collapsedSize="0%"
            collapsible
            onResize={(panelSize: { asPercentage: number }) => {
              setInfoPanelCollapsed(panelSize.asPercentage <= 1);
            }}
            className="min-h-0 bg-card"
          >
            <ConversationInfoPanel conversation={conversation} className="h-full" />
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
      <ConversationTransferDialog
        open={transferOpen}
        mode="transfer"
        conversationId={conversation?.id ?? null}
        onOpenChange={setTransferOpen}
        onSuccess={async () => {
          setTransferOpen(false);
          if (conversation?.id) {
            await handleConversationChanged(conversation.id);
          }
        }}
      />
      <ConversationCloseDialog
        open={closeOpen}
        conversationId={conversation?.id ?? null}
        onOpenChange={setCloseOpen}
        onSuccess={async () => {
          setCloseOpen(false);
          if (conversation?.id) {
            await handleConversationChanged(conversation.id);
          }
        }}
      />
      <CreateTicketFromConversationDialog
        open={createTicketOpen}
        onOpenChange={setCreateTicketOpen}
        conversation={
          conversation
            ? {
                id: conversation.id,
                customerName: conversation.customerName,
                customerId: conversation.customerId ?? 0,
                lastMessageSummary: conversation.lastMessageSummary,
                currentAssigneeId: conversation.currentAssigneeId,
              }
            : null
        }
        onSuccess={() => {
          setCreateTicketOpen(false);
        }}
      />

      <Sheet open={mobileCustomerSheetOpen} onOpenChange={setMobileCustomerSheetOpen}>
        <SheetContent
          side="right"
          className="flex w-full flex-col gap-0 border-l p-0 sm:max-w-md"
          showCloseButton
        >
          <ConversationInfoPanel
            conversation={conversation}
            variant="embedded"
            className="min-h-0 flex-1"
          />
        </SheetContent>
      </Sheet>
    </div>
  );
}
