"use client";

import {
  FileTextIcon,
  FolderInputIcon,
  EyeIcon,
  MoreHorizontalIcon,
  PencilIcon,
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
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import { OptionCombobox } from "@/components/option-combobox";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  batchBuildKnowledgeDocumentIndex,
  batchDeleteKnowledgeDocuments,
  batchMoveKnowledgeDocuments,
  buildKnowledgeDocumentIndex,
  createKnowledgeDocument,
  deleteKnowledgeDocument,
  fetchKnowledgeDocuments,
  updateKnowledgeDocument,
  type CreateKnowledgeDocumentPayload,
  type KnowledgeDocumentListItem,
} from "@/lib/api/admin";
import { useI18n } from "@/i18n/provider";
import {
  KnowledgeDocumentIndexStatus,
  Status,
} from "@/lib/generated/enums";
import { cn, formatDateTime } from "@/lib/utils";
import { DocumentEditDialog } from "./document-edit";
import { KnowledgeContentDetailDialog } from "./knowledge-content-detail";
import { KnowledgeBulkMoveDialog } from "./knowledge-bulk-move-dialog";
import { KnowledgeDirectoryPanel } from "./knowledge-directory-panel";
import { SelectionCheckbox } from "./selection-checkbox";

type DocumentListProps = {
  knowledgeBaseId: number | null;
  onActionStateChange?: (state: DocumentListActionState) => void;
};

export type DocumentListActionState = {
  onRefresh: () => void;
  onChangeViewMode: (mode: "list" | "grid") => void;
  onCreate: () => void;
  viewMode: "list" | "grid";
  loading: boolean;
};

type TFunction = (key: string, values?: Record<string, string | number>) => string;

// 卡片整块绑定了 onClick 打开"查看"，点击内部的菜单触发器、复选框、菜单项时
// 不能再冒泡触发查看，否则会出现"编辑"与"查看"两个窗口同时弹出。
function isInteractiveClickTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  return Boolean(
    target.closest(
      '[data-slot="dropdown-menu-trigger"],' +
        '[data-slot="context-menu-trigger"],' +
        '[data-slot="checkbox"],' +
        '[data-slot="dropdown-menu-item"],' +
        '[data-slot="context-menu-item"]',
    ),
  );
}

function getStatusOptions(t: TFunction) {
  return [
    { value: "all", label: t("knowledge.allStatus") },
    { value: String(Status.Ok), label: t("knowledge.statusOk") },
    { value: String(Status.Disabled), label: t("knowledge.statusDisabled") },
    { value: String(Status.Deleted), label: t("knowledge.statusDeleted") },
  ];
}

function getIndexStatusOptions(t: TFunction) {
  return [
    { value: "all", label: t("knowledge.allIndexStatus") },
    { value: KnowledgeDocumentIndexStatus.Pending, label: t("knowledge.indexPending") },
    { value: KnowledgeDocumentIndexStatus.Indexed, label: t("knowledge.indexIndexed") },
    { value: KnowledgeDocumentIndexStatus.Failed, label: t("knowledge.indexFailed") },
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

function renderIndexStatusBadge(item: KnowledgeDocumentListItem, t: TFunction) {
  const badge = (
    <Badge variant={getIndexStatusBadgeVariant(item.indexStatus)}>
      {getIndexStatusLabel(item.indexStatus, t)}
    </Badge>
  )

  if (
    item.indexStatus !== KnowledgeDocumentIndexStatus.Failed ||
    !item.indexError
  ) {
    return badge
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
  )
}

const VIEW_MODE_STORAGE_KEY = "knowledge-document-view-mode";

export function DocumentList({ knowledgeBaseId, onActionStateChange }: DocumentListProps) {
  const t = useI18n();
  const confirm = useConfirm();
  const [saving, setSaving] = useState(false);
  const [moving, setMoving] = useState(false);
  const [batchBuilding, setBatchBuilding] = useState(false);
  const [bulkMoveOpen, setBulkMoveOpen] = useState(false);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [moveTargetIds, setMoveTargetIds] = useState<number[]>([]);
  const [contextMenuDocumentId, setContextMenuDocumentId] = useState<number | null>(null);
  const [detailItemId, setDetailItemId] = useState<number | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [selectedDirectoryId, setSelectedDirectoryId] = useState<number | null>(null);
  const [actionLoadingMap, setActionLoadingMap] = useState<Record<number, { rebuildIndex: boolean; delete: boolean }>>({});
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<KnowledgeDocumentListItem | null>(
    null,
  );
  const [viewMode, setViewMode] = useState<"list" | "grid">(() => {
    if (typeof window === "undefined") return "grid";
    const saved = localStorage.getItem(VIEW_MODE_STORAGE_KEY);
    return saved === "list" || saved === "grid" ? saved : "grid";
  });
  const statusOptions = useMemo(() => getStatusOptions(t), [t]);
  const indexStatusOptions = useMemo(() => getIndexStatusOptions(t), [t]);

  useEffect(() => {
    setSelectedDirectoryId(null);
    setSelectedIds([]);
  }, [knowledgeBaseId]);

  const filters = useMemo<DashboardPagedListFilter[]>(() => [
    {
      name: "title",
      defaultValue: "",
      trim: true,
    },
    {
      name: "status",
      defaultValue: "all",
      allValue: "all",
    },
    {
      name: "indexStatus",
      defaultValue: "all",
      allValue: "all",
    },
  ], []);

  const fetchList = useCallback(async (query: Record<string, string | number | undefined>) => {
    return fetchKnowledgeDocuments({
      title: typeof query.title === "string" ? query.title : undefined,
      status: typeof query.status === "string" ? query.status : undefined,
      indexStatus: typeof query.indexStatus === "string" ? query.indexStatus : undefined,
      knowledgeBaseId: knowledgeBaseId ?? 0,
      directoryId: selectedDirectoryId === null ? undefined : selectedDirectoryId,
      page: typeof query.page === "number" ? query.page : Number(query.page ?? 1),
      limit: typeof query.limit === "number" ? query.limit : Number(query.limit ?? 20),
    });
  }, [knowledgeBaseId, selectedDirectoryId]);

  const {
    draftFilters,
    setDraftFilter,
    applyFilter,
    applyFilters: applyDraftFilters,
    loading,
    result: documents,
    loadData,
    handlePageChange,
    handleLimitChange,
  } = useDashboardPagedList<KnowledgeDocumentListItem>({
    filters,
    fetchList,
    enabled: Boolean(knowledgeBaseId),
    reloadKey: `${knowledgeBaseId ?? 0}-${selectedDirectoryId ?? "all"}`,
    loadFailed: t("knowledge.loadDocumentsFailed"),
  });

  const currentPageIds = useMemo(
    () => documents.results.map((item) => item.id),
    [documents.results],
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

  useEffect(() => {
    localStorage.setItem(VIEW_MODE_STORAGE_KEY, viewMode);
  }, [viewMode]);

  function handleStatusFilterChange(value: string | null) {
    applyFilter("status", value ?? "all");
  }

  function handleIndexStatusFilterChange(value: string | null) {
    applyFilter("indexStatus", value ?? "all");
  }

  function applyFilters() {
    applyDraftFilters();
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

  const openCreateDialog = useCallback(() => {
    setEditingItem(null);
    setDialogOpen(true);
  }, []);

  useEffect(() => {
    if (!onActionStateChange) {
      return;
    }

    onActionStateChange({
      onRefresh: () => void loadData(),
      onChangeViewMode: setViewMode,
      onCreate: openCreateDialog,
      viewMode,
      loading,
    });
  }, [onActionStateChange, loadData, openCreateDialog, viewMode, loading]);

  function openEditDialog(item: KnowledgeDocumentListItem) {
    // 互斥：打开编辑前关闭查看，避免两个窗口同时存在
    setDetailOpen(false);
    setDetailItemId(null);
    setEditingItem(item);
    setDialogOpen(true);
  }

  function openDetailDialog(item: KnowledgeDocumentListItem) {
    // 互斥：打开查看前关闭编辑
    setDialogOpen(false);
    setEditingItem(null);
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

  async function handleSubmit(payload: CreateKnowledgeDocumentPayload) {
    if (saving) {
      return;
    }

    setSaving(true);
    try {
      if (editingItem) {
        await updateKnowledgeDocument({
          id: editingItem.id,
          ...payload,
        });
        toast.success(t("knowledge.documentUpdated", { title: editingItem.title }));
      } else {
        await createKnowledgeDocument(payload);
        toast.success(t("knowledge.documentCreated", { title: payload.title }));
      }
      setDialogOpen(false);
      setEditingItem(null);
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.documentSaveFailed"));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(item: KnowledgeDocumentListItem) {
    setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], delete: true } }));
    try {
      await deleteKnowledgeDocument(item.id);
      toast.success(t("knowledge.documentDeleted", { title: item.title }));
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.documentDeleteFailed"));
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
      await batchDeleteKnowledgeDocuments(selectedIds);
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
      await batchMoveKnowledgeDocuments({
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

  async function handleBuildIndex(item: KnowledgeDocumentListItem) {
    setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], rebuildIndex: true } }));
    try {
      await buildKnowledgeDocumentIndex(item.id);
      toast.success(t("knowledge.documentIndexRebuilt", { title: item.title }));
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.documentIndexRebuildFailed"));
    } finally {
      setActionLoadingMap((prev) => ({ ...prev, [item.id]: { ...prev[item.id], rebuildIndex: false } }));
    }
  }

  async function handleBatchBuildIndex() {
    if (selectedIds.length === 0 || batchBuilding) {
      return;
    }
    const ids = selectedIds;
    setBatchBuilding(true);
    try {
      await batchBuildKnowledgeDocumentIndex(ids);
      toast.success(t("knowledge.batchRebuilt", { count: ids.length }));
      await loadData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.batchRebuildFailed"));
    } finally {
      setBatchBuilding(false);
    }
  }

  if (!knowledgeBaseId) {
    return (
      <div className="flex h-full flex-col items-center justify-center text-muted-foreground">
        <FileTextIcon className="mb-2 size-12 opacity-50" />
        <p>{t("knowledge.selectKnowledgeBaseForDocuments")}</p>
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
                value={String(draftFilters.title ?? "")}
                onChange={(event) => setDraftFilter("title", event.target.value)}
                onKeyDown={handleFilterKeyDown}
                placeholder={t("knowledge.searchDocumentTitle")}
                className="h-8 pl-8 text-xs"
              />
            </div>
            <div className="w-28 shrink-0">
              <OptionCombobox
                value={String(draftFilters.status ?? "all")}
                onChange={handleStatusFilterChange}
                options={statusOptions}
                placeholder={t("knowledge.allStatus")}
                searchPlaceholder={t("knowledge.searchStatus")}
                emptyText={t("knowledge.emptyStatus")}
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
            <div className={viewMode === "grid" ? "p-2 space-y-1" : "p-2 space-y-0.5"}>
            {documents.results.length > 0 ? (
              <label className="mb-1 flex h-8 w-fit cursor-pointer items-center gap-2 rounded-md px-2 text-xs text-muted-foreground hover:bg-accent">
                <Checkbox
                  checked={allCurrentPageSelected}
                  onCheckedChange={(checked) => toggleCurrentPage(Boolean(checked))}
                  aria-label={t("knowledge.selectCurrentPage")}
                />
                <span>{t("knowledge.selectCurrentPage")}</span>
              </label>
            ) : null}
            {documents.results.map((item) => (
              viewMode === "grid" ? (
                <ContextMenu
                  key={item.id}
                  onOpenChange={(open) => setContextMenuDocumentId(open ? item.id : null)}
                >
                  <ContextMenuTrigger className="w-full">
                    <div
                      className={cn(
                        "bg-background p-3 transition-colors hover:bg-accent w-full cursor-pointer",
                        contextMenuDocumentId === item.id && "bg-accent text-accent-foreground",
                      )}
                      onClick={(event) => {
                        if (isInteractiveClickTarget(event.target)) {
                          return;
                        }
                        openDetailDialog(item);
                      }}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex min-w-0 flex-1 items-start gap-2">
                          <SelectionCheckbox
                            checked={selectedIds.includes(item.id)}
                            onCheckedChange={(checked) => toggleSelected(item.id, Boolean(checked))}
                            aria-label={t("knowledge.selectItem", { name: item.title })}
                            className="mt-0.5"
                          />
                          {/* <FileTextIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" /> */}
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <div className="text-sm font-medium">{item.title}</div>
                              {renderIndexStatusBadge(item, t)}
                            </div>
                            <div className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                              <span>{item.createUserName || "-"}</span>
                              <span>{formatDateTime(item.createdAt)}</span>
                              <span className={cn(item.indexStatus === KnowledgeDocumentIndexStatus.Failed && "text-destructive")}>
                                {item.indexStatus === KnowledgeDocumentIndexStatus.Indexed
                                  ? t("knowledge.indexedAt", { time: formatDateTime(item.indexedAt) })
                                  : getIndexStatusLabel(item.indexStatus, t)}
                              </span>
                            </div>
                          </div>
                        </div>
                        <DropdownMenu>
                          <DropdownMenuTrigger
                            onClick={(event) => event.stopPropagation()}
                            render={
                              <Button
                                variant="ghost"
                                size="icon"
                                className="size-6"
                              />
                            }
                            aria-label={t("knowledge.moreActions", { name: item.title })}
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
                              onClick={() => void handleDelete(item)}
                              className="text-destructive focus:text-destructive"
                            >
                              <Trash2Icon className="mr-2 size-3.5" />
                              {actionLoadingMap[item.id]?.delete ? t("knowledge.deleting") : t("knowledge.delete")}
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
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
                      onClick={() => void handleDelete(item)}
                      variant="destructive"
                      disabled={actionLoadingMap[item.id]?.delete}
                    >
                      <Trash2Icon className="mr-2 size-3.5" />
                      {actionLoadingMap[item.id]?.delete ? t("knowledge.deleting") : t("knowledge.delete")}
                    </ContextMenuItem>
                  </ContextMenuContent>
                </ContextMenu>
              ) : (
                <ContextMenu
                  key={item.id}
                  onOpenChange={(open) => setContextMenuDocumentId(open ? item.id : null)}
                >
                  <ContextMenuTrigger className="w-full">
                    <div
                      className={cn(
                        "flex items-center gap-3 bg-background p-2 transition-colors hover:bg-accent w-full cursor-pointer",
                        contextMenuDocumentId === item.id && "bg-accent text-accent-foreground",
                      )}
                      onClick={(event) => {
                        if (isInteractiveClickTarget(event.target)) {
                          return;
                        }
                        openDetailDialog(item);
                      }}
                    >
                      <SelectionCheckbox
                        checked={selectedIds.includes(item.id)}
                        onCheckedChange={(checked) => toggleSelected(item.id, Boolean(checked))}
                        aria-label={t("knowledge.selectItem", { name: item.title })}
                      />
                      {/* <FileTextIcon className="size-4 shrink-0 text-muted-foreground" /> */}
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <div className="truncate text-sm font-medium">{item.title}</div>
                          {renderIndexStatusBadge(item, t)}
                        </div>
                        <div className="mt-1 text-xs text-muted-foreground">
                          {item.indexStatus === KnowledgeDocumentIndexStatus.Indexed
                            ? t("knowledge.indexTime", { time: formatDateTime(item.indexedAt) })
                            : item.indexError || getIndexStatusLabel(item.indexStatus, t)}
                        </div>
                      </div>
                      <div className="shrink-0 text-xs text-muted-foreground">
                        {formatDateTime(item.createdAt)}
                      </div>
                      <DropdownMenu>
                        <DropdownMenuTrigger
                          onClick={(event) => event.stopPropagation()}
                          render={
                            <Button
                              variant="ghost"
                              size="icon"
                              className="size-6"
                            />
                          }
                          aria-label={t("knowledge.moreActions", { name: item.title })}
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
                      onClick={() => void handleDelete(item)}
                      variant="destructive"
                      disabled={actionLoadingMap[item.id]?.delete}
                    >
                      <Trash2Icon className="mr-2 size-3.5" />
                      {actionLoadingMap[item.id]?.delete ? t("knowledge.deleting") : t("knowledge.delete")}
                    </ContextMenuItem>
                  </ContextMenuContent>
                </ContextMenu>
              )
            ))}
            {!loading && documents.results.length === 0 ? (
              <div className="py-8 text-center text-sm text-muted-foreground">
                {t("knowledge.emptyDocuments")}
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
                  disabled={saving || moving || batchBuilding}
                >
                  <FolderInputIcon className="size-3.5" />
                  {t("knowledge.move")}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7"
                  onClick={() => void handleBatchBuildIndex()}
                  disabled={saving || moving || batchBuilding}
                >
                  <WrenchIcon className={cn("size-3.5", batchBuilding && "animate-spin")} />
                  {batchBuilding ? t("knowledge.running") : t("knowledge.batchRebuildIndex")}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7 text-destructive hover:text-destructive"
                  onClick={() => void handleBatchDelete()}
                  disabled={saving || moving || batchBuilding}
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
                  disabled={saving || moving || batchBuilding}
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
            page={documents.page.page}
            limit={documents.page.limit}
            total={documents.page.total}
            loading={loading}
            onPageChange={handlePageChange}
            onLimitChange={handleLimitChange}
          />
        </div>
        </div>
      </div>
      <DocumentEditDialog
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
        type="document"
        itemId={detailItemId}
        onOpenChange={(open) => {
          setDetailOpen(open);
          if (!open) {
            setDetailItemId(null);
          }
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
