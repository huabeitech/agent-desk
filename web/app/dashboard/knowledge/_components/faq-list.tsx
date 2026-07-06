"use client";

import {
  EyeIcon,
  FolderInputIcon,
  MoreHorizontalIcon,
  PencilIcon,
  PowerIcon,
  SearchIcon,
  Trash2Icon,
  WrenchIcon,
  XIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState, type KeyboardEvent } from "react";
import { toast } from "sonner";

import { useConfirm } from "@/components/confirm-provider";
import {
  useDashboardPagedList,
  type DashboardPagedListFilter,
} from "@/components/dashboard/list";
import { ListPagination } from "@/components/list-pagination";
import { OptionCombobox } from "@/components/option-combobox";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  batchDeleteKnowledgeFAQs,
  batchMoveKnowledgeFAQs,
  buildKnowledgeFAQIndex,
  createKnowledgeFAQ,
  deleteKnowledgeFAQ,
  fetchKnowledgeFAQ,
  fetchKnowledgeFAQs,
  updateKnowledgeFAQ,
  updateKnowledgeFAQStatus,
  type CreateKnowledgeFAQPayload,
  type KnowledgeFAQ,
} from "@/lib/api/admin";
import { KnowledgeDocumentIndexStatus, Status } from "@/lib/generated/enums";
import { useI18n } from "@/i18n/provider";
import { cn, formatDateTime } from "@/lib/utils";
import { FAQEditDialog } from "./faq-edit";
import { FAQImportDialog } from "./faq-import-dialog";
import { KnowledgeContentDetailDialog } from "./knowledge-content-detail";
import { KnowledgeBulkMoveDialog } from "./knowledge-bulk-move-dialog";
import { KnowledgeDirectoryPanel } from "./knowledge-directory-panel";
import { SelectionCheckbox } from "./selection-checkbox";

type FAQListProps = {
  knowledgeBaseId: number | null;
  focusedFaqId?: number | null;
  onActionStateChange?: (state: FAQListActionState) => void;
};

export type FAQListActionState = {
  onRefresh: () => void;
  onCreate: () => void;
  onImport: () => void;
  loading: boolean;
  importing: boolean;
};

type TFunction = (key: string, values?: Record<string, string | number>) => string;

function getIndexStatusOptions(t: TFunction) {
  return [
    { value: "all", label: t("knowledge.allIndexStatus") },
    { value: KnowledgeDocumentIndexStatus.Pending, label: t("knowledge.indexPending") },
    { value: KnowledgeDocumentIndexStatus.Indexed, label: t("knowledge.indexIndexed") },
    { value: KnowledgeDocumentIndexStatus.Failed, label: t("knowledge.indexFailed") },
  ];
}

function getFAQStatusOptions(t: TFunction) {
  return [
    { value: "all", label: t("knowledge.allStatus") },
    { value: String(Status.Disabled), label: t("knowledge.statusDisabled") },
    { value: String(Status.Ok), label: t("knowledge.statusOk") },
  ];
}

function getIndexStatusLabel(status: string, t: TFunction) {
  if (status === KnowledgeDocumentIndexStatus.Pending) return t("knowledge.indexPending");
  if (status === KnowledgeDocumentIndexStatus.Indexed) return t("knowledge.indexIndexed");
  if (status === KnowledgeDocumentIndexStatus.Failed) return t("knowledge.indexFailed");
  return status || "-";
}

function getIndexStatusBadgeVariant(status: string) {
  switch (status) {
    case KnowledgeDocumentIndexStatus.Indexed:
      return "secondary" as const;
    case KnowledgeDocumentIndexStatus.Failed:
      return "destructive" as const;
    default:
      return "outline" as const;
  }
}

function renderIndexStatusBadge(item: KnowledgeFAQ, t: TFunction) {
  const badge = (
    <Badge variant={getIndexStatusBadgeVariant(item.indexStatus)}>
      {getIndexStatusLabel(item.indexStatus, t)}
    </Badge>
  );

  if (
    item.indexStatus !== KnowledgeDocumentIndexStatus.Failed ||
    !item.indexError
  ) {
    return badge;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <span className="inline-flex">{badge}</span>
        </TooltipTrigger>
        <TooltipContent align="start" className="max-w-sm whitespace-normal">
          {item.indexError}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function getFAQStatusLabel(status: number, fallback: string, t: TFunction) {
  if (status === Status.Ok) return t("knowledge.statusOk");
  if (status === Status.Disabled) return t("knowledge.statusDisabled");
  return fallback || String(status);
}

function getFAQStatusBadgeVariant(status: number) {
  if (status === Status.Ok) return "default" as const;
  if (status === Status.Disabled) return "secondary" as const;
  return "outline" as const;
}

export function FAQList({
  knowledgeBaseId,
  focusedFaqId = null,
  onActionStateChange,
}: FAQListProps) {
  const t = useI18n();
  const confirm = useConfirm();
  const [saving, setSaving] = useState(false);
  const [moving, setMoving] = useState(false);
  const [bulkMoveOpen, setBulkMoveOpen] = useState(false);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [moveTargetIds, setMoveTargetIds] = useState<number[]>([]);
  const [contextMenuFAQId, setContextMenuFAQId] = useState<number | null>(null);
  const [detailItemId, setDetailItemId] = useState<number | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [importing, setImporting] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [selectedDirectoryId, setSelectedDirectoryId] = useState<number | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<KnowledgeFAQ | null>(null);
  const [actionLoadingMap, setActionLoadingMap] = useState<Record<number, { rebuildIndex: boolean; delete: boolean; status: boolean }>>({});
  const indexStatusOptions = useMemo(() => getIndexStatusOptions(t), [t]);
  const faqStatusOptions = useMemo(() => getFAQStatusOptions(t), [t]);
  const [lastFocusedFaqId, setLastFocusedFaqId] = useState<number | null>(null);

  useEffect(() => {
    setSelectedDirectoryId(null);
    setSelectedIds([]);
  }, [knowledgeBaseId]);

  const filters = useMemo<DashboardPagedListFilter[]>(() => [
    {
      name: "question",
      defaultValue: "",
      trim: true,
    },
    {
      name: "indexStatus",
      defaultValue: "all",
      allValue: "all",
    },
    {
      name: "status",
      defaultValue: "all",
      allValue: "all",
      valueType: "number",
    },
  ], []);

  const fetchList = useCallback(async (query: Record<string, string | number | undefined>) => {
    return fetchKnowledgeFAQs({
      knowledgeBaseId: knowledgeBaseId ?? 0,
      directoryId: selectedDirectoryId === null ? undefined : selectedDirectoryId,
      question: typeof query.question === "string" ? query.question : undefined,
      indexStatus: typeof query.indexStatus === "string" ? query.indexStatus : undefined,
      status: query.status === undefined ? undefined : query.status,
      page: typeof query.page === "number" ? query.page : Number(query.page ?? 1),
      limit: typeof query.limit === "number" ? query.limit : Number(query.limit ?? 20),
    });
  }, [knowledgeBaseId, selectedDirectoryId]);

  const {
    draftFilters,
    setDraftFilter,
    applyFilter,
    applyFilters,
    loading,
    result: faqs,
    loadData,
    handlePageChange,
    handleLimitChange,
  } = useDashboardPagedList<KnowledgeFAQ>({
    filters,
    fetchList,
    enabled: Boolean(knowledgeBaseId),
    reloadKey: `${knowledgeBaseId ?? 0}-${selectedDirectoryId ?? "all"}`,
    loadFailed: t("knowledge.loadFAQFailed"),
  });

  const currentPageIds = useMemo(
    () => faqs.results.map((item) => item.id),
    [faqs.results],
  );
  const currentPageSelectedCount = useMemo(
    () => currentPageIds.filter((id) => selectedIds.includes(id)).length,
    [currentPageIds, selectedIds],
  );
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageSelectedCount === currentPageIds.length;

  useEffect(() => {
    setSelectedIds((prev) => prev.filter((id) => currentPageIds.includes(id)));
  }, [currentPageIds]);

  const openCreateDialog = useCallback(() => {
    setEditingItem(null);
    setDialogOpen(true);
  }, []);

  useEffect(() => {
    onActionStateChange?.({
      onRefresh: () => void loadData(),
      onCreate: openCreateDialog,
      onImport: () => setImportDialogOpen(true),
      loading,
      importing,
    });
  }, [loadData, loading, importing, onActionStateChange, openCreateDialog]);

  function handleIndexStatusFilterChange(value: string | null) {
    applyFilter("indexStatus", value ?? "all");
  }

  function handleFAQStatusFilterChange(value: string | null) {
    applyFilter("status", value ?? "all");
  }

  function handleFilterKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key !== "Enter") {
      return;
    }
    event.preventDefault();
    applyFilters();
  }

  function toggleSelected(id: number, checked: boolean) {
    setSelectedIds((prev) => {
      if (checked) {
        return prev.includes(id) ? prev : [...prev, id];
      }
      return prev.filter((currentId) => currentId !== id);
    });
  }

  function toggleCurrentPage(checked: boolean) {
    setSelectedIds(checked ? currentPageIds : []);
  }

  function openMoveDialog(ids: number[]) {
    setMoveTargetIds(ids);
    setBulkMoveOpen(true);
  }

  function openEditDialog(item: KnowledgeFAQ) {
    setEditingItem(item);
    setDialogOpen(true);
  }

  useEffect(() => {
    if (!knowledgeBaseId || !focusedFaqId || lastFocusedFaqId === focusedFaqId) {
      return;
    }

    let cancelled = false;
    const targetFaqId = focusedFaqId;

    async function openFocusedFAQ() {
      try {
        const item = await fetchKnowledgeFAQ(targetFaqId);
        if (cancelled || item.knowledgeBaseId !== knowledgeBaseId) {
          return;
        }
        setLastFocusedFaqId(targetFaqId);
        setEditingItem(item);
        setDialogOpen(true);
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t("knowledge.loadFAQFailed"));
      }
    }

    void openFocusedFAQ();

    return () => {
      cancelled = true;
    };
  }, [focusedFaqId, knowledgeBaseId, lastFocusedFaqId, t]);

  function openDetailDialog(item: KnowledgeFAQ) {
    setDetailItemId(item.id);
    setDetailOpen(true);
  }

  function handleDialogOpenChange(open: boolean) {
    if (saving) {
      return;
    }
    if (!open) {
      setEditingItem(null);
    }
    setDialogOpen(open);
  }

  async function handleSubmit(payload: CreateKnowledgeFAQPayload) {
    if (saving) {
      return;
    }
    setSaving(true);
    try {
      if (editingItem) {
        await updateKnowledgeFAQ({ id: editingItem.id, ...payload });
        toast.success(t("knowledge.faqSaved"));
      } else {
        await createKnowledgeFAQ(payload);
        toast.success(t("knowledge.faqSaved"));
      }
      setDialogOpen(false);
      setEditingItem(null);
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.faqSaveFailed"));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(item: KnowledgeFAQ) {
    setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], delete: true } }));
    try {
      await deleteKnowledgeFAQ(item.id);
      toast.success(t("knowledge.faqDeleted"));
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.faqDeleteFailed"));
    } finally {
      setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], delete: false } }));
    }
  }

  async function handleBatchDelete() {
    if (selectedIds.length === 0) {
      return;
    }
    const confirmed = await confirm({
      title: t("knowledge.batchDelete"),
      description: t("knowledge.batchDeleteConfirm", { count: selectedIds.length }),
      confirmText: t("knowledge.delete"),
      variant: "destructive",
    });
    if (!confirmed) {
      return;
    }
    setSaving(true);
    try {
      await batchDeleteKnowledgeFAQs(selectedIds);
      toast.success(t("knowledge.batchDeleted", { count: selectedIds.length }));
      setSelectedIds([]);
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.batchDeleteFailed"));
    } finally {
      setSaving(false);
    }
  }

  async function handleBatchMove(directoryId: number) {
    const ids = moveTargetIds.length > 0 ? moveTargetIds : selectedIds;
    if (!knowledgeBaseId || ids.length === 0) {
      return;
    }
    setMoving(true);
    try {
      await batchMoveKnowledgeFAQs({
        knowledgeBaseId,
        directoryId,
        ids,
      });
      toast.success(t("knowledge.batchMoved", { count: ids.length }));
      setBulkMoveOpen(false);
      setMoveTargetIds([]);
      setSelectedIds([]);
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.batchMoveFailed"));
    } finally {
      setMoving(false);
    }
  }

  async function handleBuildIndex(item: KnowledgeFAQ) {
    setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], rebuildIndex: true } }));
    try {
      await buildKnowledgeFAQIndex(item.id);
      toast.success(t("knowledge.faqIndexRebuilt"));
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.faqIndexRebuildFailed"));
    } finally {
      setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], rebuildIndex: false } }));
    }
  }

  async function handleUpdateStatus(item: KnowledgeFAQ, status: Status) {
    setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], status: true } }));
    try {
      await updateKnowledgeFAQStatus({ id: item.id, status });
      toast.success(status === Status.Ok ? t("knowledge.faqEnabled") : t("knowledge.faqDisabled"));
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.faqStatusUpdateFailed"));
    } finally {
      setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], status: false } }));
    }
  }

  if (!knowledgeBaseId) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
        {t("knowledge.selectFAQBase")}
      </div>
    );
  }

  return (
    <>
      <div className="flex h-full min-h-0">
        <KnowledgeDirectoryPanel
          knowledgeBaseId={knowledgeBaseId}
          selectedDirectoryId={selectedDirectoryId}
          onSelectDirectory={setSelectedDirectoryId}
          onChanged={() => void loadData()}
        />
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex flex-col gap-2 border-b bg-background px-6 py-2">
            <div className="flex min-w-0 gap-2">
              <div className="relative min-w-0 flex-1">
                <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={String(draftFilters.question ?? "")}
                  onChange={(event) => setDraftFilter("question", event.target.value)}
                  onKeyDown={handleFilterKeyDown}
                  placeholder={t("knowledge.searchFAQ")}
                  className="h-8 pl-8 text-xs"
                />
              </div>
              <div className="w-32 shrink-0">
                <OptionCombobox
                  value={String(draftFilters.indexStatus ?? "all")}
                  onChange={handleIndexStatusFilterChange}
                  options={indexStatusOptions}
                  placeholder={t("knowledge.allIndexStatus")}
                  searchPlaceholder={t("knowledge.searchStatus")}
                  emptyText={t("knowledge.emptyStatus")}
                />
              </div>
              <div className="w-28 shrink-0">
                <OptionCombobox
                  value={String(draftFilters.status ?? "all")}
                  onChange={handleFAQStatusFilterChange}
                  options={faqStatusOptions}
                  placeholder={t("knowledge.allStatus")}
                  searchPlaceholder={t("knowledge.searchStatus")}
                  emptyText={t("knowledge.emptyStatus")}
                />
              </div>
              <Button
                variant="outline"
                size="sm"
                className="h-8 shrink-0"
                onClick={applyFilters}
                disabled={loading}
              >
                <SearchIcon className="size-3.5" />
                {t("knowledge.query")}
              </Button>
            </div>
          </div>
          <div className="relative min-h-0 flex-1">
            <ScrollArea className="h-full">
              <div className="p-2 space-y-0.5">
                {faqs.results.length > 0 ? (
                  <label className="mb-1 flex h-8 w-fit cursor-pointer items-center gap-2 rounded-md px-2 text-xs text-muted-foreground hover:bg-accent">
                    <Checkbox
                      checked={allCurrentPageSelected}
                      onCheckedChange={(checked) => toggleCurrentPage(Boolean(checked))}
                      aria-label={t("knowledge.selectCurrentPage")}
                    />
                    <span>{t("knowledge.selectCurrentPage")}</span>
                  </label>
                ) : null}
                {faqs.results.map((item) => (
                  <ContextMenu
                    key={item.id}
                    onOpenChange={(open) => setContextMenuFAQId(open ? item.id : null)}
                  >
                    <ContextMenuTrigger className="w-full">
                      <div
                        className={cn(
                          "flex items-center gap-3 bg-background p-2 transition-colors hover:bg-accent w-full cursor-pointer",
                          contextMenuFAQId === item.id && "bg-accent text-accent-foreground",
                        )}
                        onClick={() => openDetailDialog(item)}
                      >
                        <SelectionCheckbox
                          checked={selectedIds.includes(item.id)}
                          onCheckedChange={(checked) => toggleSelected(item.id, Boolean(checked))}
                          aria-label={t("knowledge.selectItem", { name: item.question })}
                        />
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <div className="truncate text-sm font-medium">{item.question}</div>
                            <Badge variant={getFAQStatusBadgeVariant(item.status)}>
                              {getFAQStatusLabel(item.status, item.statusName, t)}
                            </Badge>
                            {renderIndexStatusBadge(item, t)}
                          </div>
                          <div className="mt-1 line-clamp-1 text-xs text-muted-foreground">
                            {item.answer}
                          </div>
                        </div>
                        <div className="hidden shrink-0 text-xs text-muted-foreground sm:block">
                          {formatDateTime(item.updatedAt)}
                        </div>
                        <DropdownMenu>
                          <DropdownMenuTrigger
                            onClick={(event) => event.stopPropagation()}
                            render={<Button variant="ghost" size="icon" className="size-6" />}
                            aria-label={t("knowledge.moreActions", { name: item.question })}
                          >
                            <MoreHorizontalIcon className="size-3.5" />
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end" className="w-32 min-w-32">
                            <DropdownMenuItem onClick={() => openDetailDialog(item)}>
                              <EyeIcon className="mr-2 size-3.5" />
                              {t("knowledge.view")}
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => openEditDialog(item)}>
                              <PencilIcon className="mr-2 size-3.5" />
                              {t("knowledge.edit")}
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => openMoveDialog([item.id])}>
                              <FolderInputIcon className="mr-2 size-3.5" />
                              {t("knowledge.move")}
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => void handleBuildIndex(item)}>
                              <WrenchIcon className="mr-2 size-3.5" />
                              {actionLoadingMap[item.id]?.rebuildIndex ? t("knowledge.running") : t("knowledge.rebuildIndex")}
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => void handleUpdateStatus(item, item.status === Status.Ok ? Status.Disabled : Status.Ok)}
                              disabled={actionLoadingMap[item.id]?.status}
                            >
                              <PowerIcon className="mr-2 size-3.5" />
                              {actionLoadingMap[item.id]?.status
                                ? t("knowledge.running")
                                : item.status === Status.Ok
                                  ? t("knowledge.disable")
                                  : t("knowledge.enable")}
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => void handleDelete(item)}
                              className="text-destructive focus:text-destructive"
                            >
                              <Trash2Icon className="mr-2 size-3.5" />
                              {actionLoadingMap[item.id]?.delete ? t("knowledge.deleting") : t("knowledge.delete")}
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                    </ContextMenuTrigger>
                    <ContextMenuContent className="w-40">
                      <ContextMenuItem onClick={() => openDetailDialog(item)}>
                        <EyeIcon className="mr-2 size-3.5" />
                        {t("knowledge.view")}
                      </ContextMenuItem>
                      <ContextMenuItem onClick={() => openEditDialog(item)}>
                        <PencilIcon className="mr-2 size-3.5" />
                        {t("knowledge.edit")}
                      </ContextMenuItem>
                      <ContextMenuItem onClick={() => openMoveDialog([item.id])}>
                        <FolderInputIcon className="mr-2 size-3.5" />
                        {t("knowledge.move")}
                      </ContextMenuItem>
                      <ContextMenuItem onClick={() => void handleBuildIndex(item)} disabled={actionLoadingMap[item.id]?.rebuildIndex}>
                        <WrenchIcon className="mr-2 size-3.5" />
                        {actionLoadingMap[item.id]?.rebuildIndex ? t("knowledge.running") : t("knowledge.rebuildIndex")}
                      </ContextMenuItem>
                      <ContextMenuItem
                        onClick={() => void handleUpdateStatus(item, item.status === Status.Ok ? Status.Disabled : Status.Ok)}
                        disabled={actionLoadingMap[item.id]?.status}
                      >
                        <PowerIcon className="mr-2 size-3.5" />
                        {actionLoadingMap[item.id]?.status
                          ? t("knowledge.running")
                          : item.status === Status.Ok
                            ? t("knowledge.disable")
                            : t("knowledge.enable")}
                      </ContextMenuItem>
                      <ContextMenuItem
                        onClick={() => void handleDelete(item)}
                        variant="destructive"
                        disabled={actionLoadingMap[item.id]?.delete}
                      >
                        <Trash2Icon className="mr-2 size-3.5" />
                        {actionLoadingMap[item.id]?.delete ? t("knowledge.deleting") : t("knowledge.delete")}
                      </ContextMenuItem>
                    </ContextMenuContent>
                  </ContextMenu>
                ))}
                {!loading && faqs.results.length === 0 ? (
                  <div className="py-8 text-center text-sm text-muted-foreground">
                    {t("knowledge.emptyFAQ")}
                  </div>
                ) : null}
              </div>
            </ScrollArea>
            {selectedIds.length > 0 ? (
              <div className="pointer-events-none absolute inset-x-0 bottom-3 z-10 flex justify-center px-4">
                <div className="pointer-events-auto flex items-center gap-2 rounded-md border bg-popover px-3 py-2 text-xs shadow-lg">
                  <span className="text-muted-foreground">
                    {t("knowledge.selectedCount", { count: selectedIds.length })}
                  </span>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-7"
                    onClick={() => openMoveDialog(selectedIds)}
                    disabled={saving || moving}
                  >
                    <FolderInputIcon className="size-3.5" />
                    {t("knowledge.move")}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-7 text-destructive hover:text-destructive"
                    onClick={() => void handleBatchDelete()}
                    disabled={saving || moving}
                  >
                    <Trash2Icon className="size-3.5" />
                    {t("knowledge.delete")}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-7"
                    onClick={() => setSelectedIds([])}
                    disabled={saving || moving}
                    aria-label={t("knowledge.clearSelection")}
                  >
                    <XIcon className="size-3.5" />
                  </Button>
                </div>
              </div>
            ) : null}
          </div>
          <div className="border-t px-6 py-3">
            <ListPagination
              page={faqs.page.page}
              limit={faqs.page.limit}
              total={faqs.page.total}
              loading={loading}
              onPageChange={handlePageChange}
              onLimitChange={handleLimitChange}
            />
          </div>
        </div>
      </div>

      <FAQEditDialog
        open={dialogOpen}
        saving={saving}
        itemId={editingItem?.id ?? null}
        knowledgeBaseId={knowledgeBaseId}
        initialDirectoryId={selectedDirectoryId ?? 0}
        onOpenChange={handleDialogOpenChange}
        onSubmit={handleSubmit}
      />
      <KnowledgeContentDetailDialog
        open={detailOpen}
        type="faq"
        itemId={detailItemId}
        onOpenChange={(open) => {
          setDetailOpen(open);
          if (!open) {
            setDetailItemId(null);
          }
        }}
      />

      <FAQImportDialog
        open={importDialogOpen}
        knowledgeBaseId={knowledgeBaseId}
        importing={importing}
        onOpenChange={setImportDialogOpen}
        onImportingChange={setImporting}
        onImported={async () => {
          await loadData();
        }}
      />
      <KnowledgeBulkMoveDialog
        open={bulkMoveOpen}
        knowledgeBaseId={knowledgeBaseId}
        moving={moving}
        selectedCount={moveTargetIds.length || selectedIds.length}
        onOpenChange={(open) => {
          setBulkMoveOpen(open);
          if (!open) {
            setMoveTargetIds([]);
          }
        }}
        onSubmit={(directoryId) => void handleBatchMove(directoryId)}
      />
    </>
  );
}
