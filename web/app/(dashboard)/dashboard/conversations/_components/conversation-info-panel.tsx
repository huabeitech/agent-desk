"use client";
import {
  AlertTriangleIcon,
  Building2Icon,
  Link2Icon,
  MailIcon,
  PencilIcon,
  PhoneIcon,
  TimerIcon,
  UserRoundIcon,
  WorkflowIcon,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";

import { type CustomerFormSavePayload } from "@/components/customer-form";
import { CustomerFormDialog } from "@/components/customer-form-dialog";
import { CustomerLinkOrCreateDialog } from "@/components/customer-link-or-create-dialog";
import { JsonTreeViewer } from "@/components/json-tree-viewer";
import { ProjectDialog } from "@/components/project-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Field,
  FieldContent,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { AgentConversation } from "@/lib/api/agent";
import {
  fetchAIWorkflowRun,
  fetchAIWorkflowRuns,
  type AIWorkflowNodeRun,
  type AIWorkflowRun,
  type TagTree,
  fetchTagsAll,
} from "@/lib/api/admin";
import { updateCompany, type AdminCompany } from "@/lib/api/company";
import { fetchTickets, type TicketItem } from "@/lib/api/ticket";
import {
  fetchCustomer,
  saveCustomerProfile,
  type AdminCustomer,
} from "@/lib/api/customer";
import {
  fetchCustomerContacts,
  type AdminCustomerContact,
} from "@/lib/api/customer-contact";
import {
  ContactType,
  Gender,
} from "@/lib/generated/enums";
import { useAgentConversationsStore } from "@/lib/stores/agent-conversations";
import { cn, formatDateTime } from "@/lib/utils";
import { useI18n } from "@/i18n/provider";
import {
  ConversationTagBadges,
  ConversationTagPicker,
} from "./conversation-tag-picker";
import { TicketStatusBadge } from "../../tickets/_components/ticket-status-badge";

function contactTypeLabel(
  contactType: ContactType | string,
  t: (key: string, values?: Record<string, string | number>) => string
) {
  switch (contactType) {
    case ContactType.Mobile:
      return t("conversation.contactMobile");
    case ContactType.Email:
      return t("conversation.contactEmail");
    case ContactType.Other:
      return t("conversation.contactOther");
    default:
      return String(contactType);
  }
}

function ContactTypeIcon({ contactType }: { contactType: ContactType | string }) {
  const cls = "size-3.5 shrink-0 text-muted-foreground";
  switch (contactType) {
    case ContactType.Mobile:
      return <PhoneIcon className={cls} aria-hidden />;
    case ContactType.Email:
      return <MailIcon className={cls} aria-hidden />;
    default:
      return <Link2Icon className={cls} aria-hidden />;
  }
}

function DetailRow({
  label,
  value,
  valueClassName,
}: {
  label: string;
  value: string;
  valueClassName?: string;
}) {
  const empty = !value.trim();
  return (
    <div className="flex gap-2.5 text-sm leading-snug">
      <span className="w-17 shrink-0 pt-px text-xs text-muted-foreground">{label}</span>
      <span
        className={cn(
          "min-w-0 flex-1 break-all text-foreground",
          empty && "text-muted-foreground",
          valueClassName,
        )}
      >
        {empty ? "—" : value}
      </span>
    </div>
  );
}

function SectionHeading({
  children,
  action,
}: {
  children: React.ReactNode;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-2">
      <h3 className="text-xs font-medium text-muted-foreground">{children}</h3>
      {action}
    </div>
  );
}

function UnlinkedCustomerEmpty({ conversation }: { conversation: AgentConversation }) {
  const t = useI18n();
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const loadConversations = useAgentConversationsStore((s) => s.loadConversations);

  return (
    <div className="space-y-6 pt-2">
      <div className="flex flex-col items-center justify-center rounded-xl bg-muted/35 px-4 py-8 text-center">
        <UserRoundIcon className="mb-2 size-10 text-muted-foreground" aria-hidden />
        <p className="text-sm font-medium text-foreground">
          {t("conversation.unlinkedCustomerTitle")}
        </p>
        <p className="mt-1 max-w-xs text-xs leading-relaxed text-muted-foreground">
          {t("conversation.unlinkedCustomerDescription")}
        </p>
        <Button
          type="button"
          className="mt-4 gap-2"
          onClick={() => setLinkDialogOpen(true)}
        >
          <Link2Icon className="size-4" />
          {t("conversation.linkOrCreateCustomer")}
        </Button>
      </div>
      <CustomerLinkOrCreateDialog
        open={linkDialogOpen}
        onOpenChange={setLinkDialogOpen}
        conversationId={conversation.id}
        onSuccess={() => void loadConversations()}
      />
    </div>
  );
}

function MissingCustomerEmpty({ conversation }: { conversation: AgentConversation }) {
  const t = useI18n();
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const loadConversations = useAgentConversationsStore((s) => s.loadConversations);

  return (
    <div className="space-y-6 pt-2">
      <div className="flex flex-col items-center justify-center rounded-xl bg-muted/35 px-4 py-8 text-center">
        <UserRoundIcon className="mb-2 size-10 text-muted-foreground" aria-hidden />
        <p className="text-sm font-medium text-foreground">
          {t("conversation.missingCustomerTitle")}
        </p>
        <p className="mt-1 max-w-xs text-xs leading-relaxed text-muted-foreground">
          {t("conversation.missingCustomerDescription")}
        </p>
        <Button
          type="button"
          className="mt-4 gap-2"
          onClick={() => setLinkDialogOpen(true)}
        >
          <Link2Icon className="size-4" />
          {t("conversation.relinkOrCreateCustomer")}
        </Button>
      </div>
      <div className="space-y-2">
        <SectionHeading>{t("conversation.conversationOwner")}</SectionHeading>
        <div className="space-y-2">
          <DetailRow label={t("conversation.channelId")} value={conversation.channelId ? `${conversation.channelId}` : "-"} />
          <DetailRow label={t("conversation.customerId")} value={conversation.customerId ? `${conversation.customerId}` : "-"} />
        </div>
      </div>
      <CustomerLinkOrCreateDialog
        open={linkDialogOpen}
        onOpenChange={setLinkDialogOpen}
        conversationId={conversation.id}
        onSuccess={() => void loadConversations()}
      />
    </div>
  );
}

type ConversationInfoPanelProps = {
  conversation: AgentConversation | null;
  className?: string;
  variant?: "default" | "embedded";
};

export function ConversationInfoPanel({
  conversation,
  className,
  variant = "default",
}: ConversationInfoPanelProps) {
  const t = useI18n();
  const embedded = variant === "embedded";

  return (
    <div
      className={cn(
        "flex h-full min-h-0 flex-col overflow-hidden",
        embedded
          ? "bg-card text-card-foreground"
          : "border-border/80 bg-card text-card-foreground",
        className,
      )}
    >
      <div className="flex h-12.5 shrink-0 items-center border-b border-border/80 bg-card px-3">
        <h2 className="text-sm font-medium text-foreground">
          {t("conversation.conversationInfo")}
        </h2>
      </div>

      <div
        className={cn(
          "min-h-0 flex-1 overflow-y-auto px-3 pb-4",
          embedded && "pb-[max(1rem,env(safe-area-inset-bottom))] pt-1",
        )}
      >
        {!conversation ? (
          <p className="pt-4 text-sm text-muted-foreground">
            {embedded
              ? t("conversation.selectConversationForInfo")
              : t("conversation.selectSidebarConversationForInfo")}
          </p>
        ) : (
          <div className="space-y-4 py-3">
            <section>
              <DetailRow
                label={t("conversation.conversationId")}
                value={`${conversation.id}`}
                valueClassName="font-mono text-xs"
              />
            </section>
            <CustomerBody conversation={conversation} />
            <WorkflowRunsSection conversation={conversation} />
          </div>
        )}
      </div>
    </div>
  );
}

function ConversationTagSection({
  conversation,
}: {
  conversation: AgentConversation;
}) {
  const t = useI18n();
  const setConversationTags = useAgentConversationsStore(
    (state) => state.setConversationTags,
  );
  const [availableTags, setAvailableTags] = useState<TagTree[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadTags() {
      setLoading(true);
      try {
        const data = await fetchTagsAll();
        if (!cancelled) {
          setAvailableTags(Array.isArray(data) ? data : []);
        }
      } catch (error) {
        if (!cancelled) {
          toast.error(error instanceof Error ? error.message : t("conversation.loadTagsFailed"));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadTags();

    return () => {
      cancelled = true;
    };
  }, [t]);

  return (
    <section className="space-y-2 border-t pt-2">
      <SectionHeading
        action={
          <ConversationTagPicker
            conversation={conversation}
            availableTags={availableTags}
            loading={loading}
            onTagsChange={(tags) => {
              setConversationTags(conversation.id, tags);
            }}
          />
        }
      >
        {t("conversation.conversationTags")}
      </SectionHeading>
      <ConversationTagBadges
        tags={conversation.tags}
        availableTags={availableTags}
      />
      {!conversation.tags || conversation.tags.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("conversation.noConversationTags")}</p>
      ) : null}
    </section>
  );
}

function WorkflowRunsSection({ conversation }: { conversation: AgentConversation }) {
  const t = useI18n();
  const [runs, setRuns] = useState<AIWorkflowRun[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [activeRun, setActiveRun] = useState<AIWorkflowRun | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadRuns() {
      setLoading(true);
      try {
        const data = await fetchAIWorkflowRuns({
          conversationId: conversation.id,
          page: 1,
          limit: 5,
        });
        if (!cancelled) {
          setRuns(Array.isArray(data.results) ? data.results : []);
        }
      } catch (error) {
        if (!cancelled) {
          toast.error(error instanceof Error ? error.message : t("workflowRun.loadLogsFailed"));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadRuns();
    return () => {
      cancelled = true;
    };
  }, [conversation.id, t]);

  async function openDetail(runId: number) {
    setDetailOpen(true);
    setDetailLoading(true);
    try {
      const data = await fetchAIWorkflowRun(runId);
      setActiveRun(data);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("workflowRun.loadDetailFailed"));
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  }

  return (
    <section className="space-y-2 border-t pt-2">
      <SectionHeading>{t("workflowRun.recordsTitle")}</SectionHeading>
      {loading ? (
        <p className="text-sm text-muted-foreground">{t("workflowRun.loadingLogs")}</p>
      ) : runs.length > 0 ? (
        <div className="space-y-2">
          {runs.map((run) => (
            <button
              key={run.id}
              type="button"
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-left transition-colors hover:bg-muted/40"
              onClick={() => void openDetail(run.id)}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex min-w-0 items-center gap-2">
                    <WorkflowIcon className="size-3.5 shrink-0 text-muted-foreground" />
                    <span className="truncate text-sm font-medium text-foreground">
                      Run #{run.id}
                    </span>
                    <WorkflowRunStatusBadge statusName={run.statusName} />
                  </div>
                  <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                    <span>Workflow #{run.workflowVersionId || run.workflowId || "-"}</span>
                    <span>Message #{run.messageId || "-"}</span>
                  </div>
                </div>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {run.startedAt ? formatDateTime(run.startedAt) : "—"}
                </span>
              </div>
              {run.errorMessage ? (
                <div className="mt-2 flex items-start gap-1.5 text-xs text-destructive">
                  <AlertTriangleIcon className="mt-0.5 size-3.5 shrink-0" />
                  <span className="line-clamp-2 break-all">{run.errorMessage}</span>
                </div>
              ) : null}
            </button>
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">{t("workflowRun.emptyLogs")}</p>
      )}
      <WorkflowRunDetailDialog
        open={detailOpen}
        loading={detailLoading}
        run={activeRun}
        onOpenChange={(open) => {
          setDetailOpen(open);
          if (!open) {
            setActiveRun(null);
          }
        }}
      />
    </section>
  );
}

function WorkflowRunDetailDialog({
  open,
  loading,
  run,
  onOpenChange,
}: {
  open: boolean;
  loading: boolean;
  run: AIWorkflowRun | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useI18n();

  return (
    <ProjectDialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        <span className="flex items-center gap-2">
          <WorkflowIcon className="size-4" />
          {t("workflowRun.detailTitle")}
        </span>
      }
      description={run ? `Run #${run.id}` : t("workflowRun.description")}
      size="xl"
      allowFullscreen
      footer={
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          {t("common.close")}
        </Button>
      }
    >
      {loading ? (
        <div className="px-6 py-10 text-sm text-muted-foreground">{t("workflowRun.loadingDetail")}</div>
      ) : run ? (
        <div className="space-y-4 px-6 pb-6">
          <div className="grid gap-2 rounded-lg border bg-muted/20 p-3 text-sm md:grid-cols-2">
            <DetailRow label={t("workflowRun.labelWorkflow")} value={`#${run.workflowId} / v${run.workflowVersionId}`} />
            <DetailRow label={t("workflowRun.labelConversation")} value={`#${run.conversationId}`} />
            <DetailRow label={t("workflowRun.labelMessage")} value={`#${run.messageId}`} />
            <DetailRow label={t("workflowRun.labelAgent")} value={`#${run.aiAgentId}`} />
            <DetailRow label={t("workflowRun.labelStatus")} value={run.statusName || String(run.status)} />
            <DetailRow label={t("workflowRun.labelStartedAt")} value={run.startedAt ? formatDateTime(run.startedAt) : ""} />
            <DetailRow label={t("workflowRun.labelEndedAt")} value={run.endedAt ? formatDateTime(run.endedAt) : ""} />
            <DetailRow label={t("workflowRun.labelInterruptNode")} value={run.interruptNodeId || ""} />
          </div>
          {run.errorMessage ? (
            <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {run.errorMessage}
            </div>
          ) : null}
          <div className="space-y-3">
            {(run.nodes ?? []).map((node) => (
              <WorkflowNodeRunBlock key={node.id} node={node} />
            ))}
            {!run.nodes || run.nodes.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("workflowRun.emptyNodes")}</p>
            ) : null}
          </div>
        </div>
      ) : (
        <div className="px-6 py-10 text-sm text-muted-foreground">{t("workflowRun.emptyDetail")}</div>
      )}
    </ProjectDialog>
  );
}

function WorkflowNodeRunBlock({ node }: { node: AIWorkflowNodeRun }) {
  const t = useI18n();
  const inputValue = safeParseJSON(node.inputPreview);
  const outputValue = safeParseJSON(node.outputPreview);

  return (
    <div className="rounded-lg border bg-background p-3">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex min-w-0 items-center gap-2">
            <span className="truncate text-sm font-medium text-foreground">
              {node.nodeId || `Node #${node.id}`}
            </span>
            <WorkflowRunStatusBadge statusName={node.statusName} />
          </div>
          <div className="mt-1 text-xs text-muted-foreground">{node.nodeType || "unknown"}</div>
        </div>
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <TimerIcon className="size-3.5" />
          {node.durationMs} ms
        </div>
      </div>
      {node.errorMessage ? (
        <div className="mt-3 rounded-md bg-destructive/5 px-3 py-2 text-xs text-destructive">
          {node.errorMessage}
        </div>
      ) : null}
      <div className="mt-3 grid gap-3 lg:grid-cols-2">
        <PreviewBlock title={t("workflowRun.input")} raw={node.inputPreview} value={inputValue} />
        <PreviewBlock title={t("workflowRun.output")} raw={node.outputPreview} value={outputValue} />
      </div>
    </div>
  );
}

function PreviewBlock({
  title,
  raw,
  value,
}: {
  title: string;
  raw: string;
  value: unknown;
}) {
  return (
    <div className="min-w-0">
      <div className="mb-1 text-xs font-medium text-muted-foreground">{title}</div>
      {value !== null ? (
        <JsonTreeViewer value={value} collapsed={2} />
      ) : raw.trim() ? (
        <pre className="max-h-72 overflow-auto rounded-md border bg-muted/20 p-3 text-xs whitespace-pre-wrap break-all">
          {raw}
        </pre>
      ) : (
        <div className="rounded-md border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
          —
        </div>
      )}
    </div>
  );
}

function WorkflowRunStatusBadge({ statusName }: { statusName: string }) {
  const normalized = statusName.trim();
  const variant =
    normalized === "failed"
      ? "destructive"
      : normalized === "interrupted"
        ? "outline"
        : "secondary";
  return (
    <Badge variant={variant} className="h-5 px-1.5 text-[11px]">
      {normalized || "unknown"}
    </Badge>
  );
}

function safeParseJSON(raw: string): unknown | null {
  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }
  try {
    return JSON.parse(trimmed);
  } catch {
    return null;
  }
}

function CustomerBody({ conversation }: { conversation: AgentConversation }) {
  const customerId = conversation.customerId ?? 0;

  if (customerId <= 0) {
    return (
      <div className="space-y-4">
        <UnlinkedCustomerEmpty conversation={conversation} />
        <ConversationTagSection conversation={conversation} />
      </div>
    );
  }

  return <CustomerLinkedBody conversation={conversation} customerId={customerId} />;
}

type CustomerLinkedBodyProps = {
  conversation: AgentConversation;
  customerId: number;
};

function CustomerLinkedBody({ conversation, customerId }: CustomerLinkedBodyProps) {
  const t = useI18n();
  const [loading, setLoading] = useState(true);
  const [customer, setCustomer] = useState<AdminCustomer | null>(null);
  const [contacts, setContacts] = useState<AdminCustomerContact[]>([]);

  const [customerEditOpen, setCustomerEditOpen] = useState(false);
  const [customerEditSaving, setCustomerEditSaving] = useState(false);
  const [companyEditOpen, setCompanyEditOpen] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const c = await fetchCustomer(customerId);
      setCustomer(c);
      if (!c) {
        setContacts([]);
        return;
      }
      const list = await fetchCustomerContacts(customerId);
      setContacts(Array.isArray(list) ? list : []);
    } catch (e) {
      const msg = e instanceof Error ? e.message : t("conversation.loadCustomerFailed");
      toast.error(msg);
      setCustomer(null);
      setContacts([]);
    } finally {
      setLoading(false);
    }
  }, [customerId, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const isProfileEmpty =
    customer &&
    !customer.name.trim() &&
    !customer.primaryMobile.trim() &&
    !customer.primaryEmail.trim() &&
    customer.companyId === 0 &&
    !customer.remark.trim();

  if (loading && !customer) {
    return (
      <p className="pt-4 text-sm text-muted-foreground">{t("conversation.loadingCustomer")}</p>
    );
  }

  if (!customer) {
    return (
      <div className="space-y-4">
        <MissingCustomerEmpty conversation={conversation} />
        <ConversationTagSection conversation={conversation} />
      </div>
    );
  }

  const displayName = customer.name.trim() || t("conversation.unnamedCustomer");
  const company = customer.company ?? null;
  const genderLabel =
    customer.gender === Gender.Male
      ? t("conversation.genderMale")
      : customer.gender === Gender.Female
        ? t("conversation.genderFemale")
      : null;

  return (
    <div className="space-y-4">
      {isProfileEmpty ? (
        <div className="rounded-lg bg-amber-500/10 px-3 py-2.5 text-xs leading-relaxed text-amber-950 dark:text-amber-100">
          {t("conversation.customerProfileEmpty")}
        </div>
      ) : null}

      <section className="space-y-2">
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-1 items-start gap-2 text-sm">
            <UserRoundIcon
              className="mt-0.5 size-4 shrink-0 text-muted-foreground"
              aria-hidden
            />
            <div className="min-w-0 flex-1 space-y-0.5">
              <p className="line-clamp-2 leading-snug text-foreground">
                <span className="font-medium">{displayName}</span>
                {genderLabel ? (
                  <span className="font-normal text-muted-foreground">
                    {" "}
                    · {genderLabel}
                  </span>
                ) : null}
              </p>
            </div>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 shrink-0 gap-1 px-2 text-xs"
            onClick={() => setCustomerEditOpen(true)}
          >
            <PencilIcon className="size-3.5" />
            {t("conversation.edit")}
          </Button>
        </div>

        <div className="space-y-2">
          <DetailRow
            label={t("conversation.lastActive")}
            value={
              customer.lastActiveAt ? formatDateTime(customer.lastActiveAt) : ""
            }
          />
          <DetailRow
            label={t("conversation.remark")}
            value={customer.remark.trim() ? customer.remark : ""}
            valueClassName="whitespace-pre-wrap"
          />
          <DetailRow
            label={t("conversation.createdAt")}
            value={formatDateTime(customer.createdAt)}
            valueClassName="whitespace-pre-wrap"
          />
          <DetailRow
            label={t("conversation.updatedAt")}
            value={formatDateTime(customer.updatedAt)}
            valueClassName="whitespace-pre-wrap"
          />
        </div>
      </section>

      <section className="space-y-2">
        {contacts.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("conversation.noContacts")}</p>
        ) : (
          <ul className="space-y-3">
            {contacts.map((row) => {
              const tags: string[] = [];
              if (row.isPrimary) {
                tags.push(t("conversation.primaryContactBadge"));
              }
              if (row.isVerified) {
                tags.push(t("conversation.verifiedContactBadge"));
              }
              return (
                <li key={row.id} className="text-sm">
                  <div className="flex items-center gap-2">
                    <ContactTypeIcon contactType={row.contactType} />
                    <div className="min-w-0 flex-1">
                      <p className="break-all font-medium leading-snug text-foreground">
                        {row.contactValue}
                        <span className="ml-2 text-xs font-normal text-muted-foreground">
                          {contactTypeLabel(row.contactType, t)}
                        </span>
                        {tags.length > 0 ? (
                          <span className="ml-2 text-xs text-muted-foreground">
                            {tags.join(" · ")}
                          </span>
                        ) : null}
                      </p>
                      {row.remark ? (
                        <p className="mt-1 line-clamp-3 break-all text-xs leading-relaxed text-muted-foreground">
                          {row.remark}
                        </p>
                      ) : null}
                    </div>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      {customer.companyId > 0 ? (
        <section className="border-t pt-2">
          {company ? (
            <div className="space-y-2">
              <div className="flex items-start justify-between gap-2">
                <div className="flex min-w-0 flex-1 items-start gap-2 text-sm">
                  <Building2Icon
                    className="mt-0.5 size-4 shrink-0 text-muted-foreground"
                    aria-hidden
                  />
                  <div className="min-w-0 flex-1 space-y-0.5">
                    <p className="line-clamp-2 font-medium leading-snug text-foreground">
                      {company.name}
                    </p>
                    {company.code ? (
                      <p className="font-mono text-xs text-muted-foreground">
                        {company.code}
                      </p>
                    ) : null}
                  </div>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-7 shrink-0 gap-1 px-2 text-xs"
                  onClick={() => setCompanyEditOpen(true)}
                >
                  <PencilIcon className="size-3.5" />
                  {t("conversation.edit")}
                </Button>
              </div>
              <div className="space-y-2 pt-1">
                <DetailRow
                  label={t("conversation.createdAt")}
                  value={formatDateTime(company.createdAt)}
                />
                <DetailRow
                  label={t("conversation.updatedAt")}
                  value={formatDateTime(company.updatedAt)}
                />
              </div>
              <DetailRow
                label={t("conversation.remark")}
                value={company.remark.trim() ? company.remark : ""}
                valueClassName="whitespace-pre-wrap"
              />
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              {t("conversation.companyUnavailable")}
            </p>
          )}
        </section>
      ) : null}

      <RelatedTicketsSection conversation={conversation} />

      <ConversationTagSection conversation={conversation} />

      <CustomerFormDialog
        open={customerEditOpen}
        onOpenChange={setCustomerEditOpen}
        saving={customerEditSaving}
        itemId={customer.id}
        onSave={async (payload: CustomerFormSavePayload) => {
          if (customerEditSaving) {
            return;
          }
          setCustomerEditSaving(true);
          try {
            await saveCustomerProfile({ ...payload, id: customer.id });
            toast.success(t("conversation.saved"));
            void load();
            setCustomerEditOpen(false);
          } catch (e) {
            toast.error(e instanceof Error ? e.message : t("conversation.saveFailed"));
          } finally {
            setCustomerEditSaving(false);
          }
        }}
      />
      {company ? (
        <CompanyEditDialog
          open={companyEditOpen}
          onOpenChange={setCompanyEditOpen}
          company={company}
          onSaved={() => {
            void load();
          }}
        />
      ) : null}
    </div>
  );
}

function RelatedTicketsSection({ conversation }: { conversation: AgentConversation }) {
  const t = useI18n();
  const [tickets, setTickets] = useState<TicketItem[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function loadTickets() {
      setLoading(true);
      try {
        const data = await fetchTickets({
          conversationId: conversation.id,
          page: 1,
          limit: 5,
        });
        if (!cancelled) {
          setTickets(Array.isArray(data.results) ? data.results : []);
        }
      } catch (error) {
        if (!cancelled) {
          toast.error(error instanceof Error ? error.message : t("conversation.loadTicketsFailed"));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }
    void loadTickets();
    return () => {
      cancelled = true;
    };
  }, [conversation.id, t]);

  return (
    <section className="space-y-2 border-t pt-2">
      <SectionHeading>{t("conversation.relatedTickets")}</SectionHeading>
      {loading ? (
        <p className="text-sm text-muted-foreground">{t("conversation.loadingTickets")}</p>
      ) : tickets.length > 0 ? (
        <div className="space-y-2">
          {tickets.map((ticket) => (
            <div
              key={ticket.id}
              className="rounded-lg border border-border bg-background px-3 py-2"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium text-foreground">
                    {ticket.title}
                  </div>
                  <div className="mt-0.5 text-xs text-muted-foreground">
                    {ticket.ticketNo}
                  </div>
                </div>
                <TicketStatusBadge status={ticket.status} />
              </div>
              <div className="mt-2 flex items-center justify-between gap-3">
                <span className="text-xs text-muted-foreground">
                  {ticket.updatedAt ? formatDateTime(ticket.updatedAt) : "—"}
                </span>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">{t("conversation.noRelatedTickets")}</p>
      )}
    </section>
  );
}

type CompanyEditDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  company: AdminCompany;
  onSaved: () => void;
};

function CompanyEditDialog({
  open,
  onOpenChange,
  company,
  onSaved,
}: CompanyEditDialogProps) {
  const t = useI18n();
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [remark, setRemark] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }
    setName(company.name);
    setCode(company.code);
    setRemark(company.remark);
  }, [open, company]);

  const handleSubmit = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      toast.error(t("conversation.companyNameRequired"));
      return;
    }
    setSaving(true);
    try {
      await updateCompany({
        id: company.id,
        name: trimmedName,
        code: code.trim(),
        remark: remark.trim(),
      });
      toast.success(t("conversation.saved"));
      onSaved();
      onOpenChange(false);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("conversation.saveFailed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" showCloseButton>
        <DialogHeader>
          <DialogTitle>{t("conversation.editCompany")}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-4 py-1">
          <Field orientation="vertical">
            <FieldLabel htmlFor="co-name">{t("conversation.companyName")}</FieldLabel>
            <FieldContent>
              <Input id="co-name" value={name} onChange={(e) => setName(e.target.value)} />
            </FieldContent>
          </Field>
          <Field orientation="vertical">
            <FieldLabel htmlFor="co-code">{t("conversation.companyCode")}</FieldLabel>
            <FieldContent>
              <Input id="co-code" value={code} onChange={(e) => setCode(e.target.value)} />
            </FieldContent>
          </Field>
          <Field orientation="vertical">
            <FieldLabel htmlFor="co-remark">{t("conversation.remark")}</FieldLabel>
            <FieldContent>
              <Textarea
                id="co-remark"
                value={remark}
                onChange={(e) => setRemark(e.target.value)}
                rows={3}
              />
            </FieldContent>
          </Field>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("conversation.cancel")}
          </Button>
          <Button type="button" disabled={saving} onClick={() => void handleSubmit()}>
            {saving ? t("conversation.saving") : t("conversation.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
