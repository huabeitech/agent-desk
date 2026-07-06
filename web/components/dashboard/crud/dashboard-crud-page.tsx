"use client"

import type { ReactNode } from "react"
import { useCallback, useEffect, useRef, useState } from "react"
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core"
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable"
import { CSS } from "@dnd-kit/utilities"
import {
  GripVerticalIcon,
  MoreHorizontalIcon,
  PlusIcon,
  RefreshCwIcon,
  SearchIcon,
  Trash2Icon,
} from "lucide-react"
import { toast } from "sonner"

import { useConfirm, type ConfirmOptions } from "@/components/confirm-provider"
import {
  DashboardPage,
  DashboardTableShell,
  DashboardTableStateRow,
  DashboardToolbar,
} from "@/components/dashboard-page"
import { ListPagination } from "@/components/list-pagination"
import { OptionCombobox } from "@/components/option-combobox"
import { useDashboardPagedList } from "@/components/dashboard/list"
import { Button } from "@/components/ui/button"
import { ButtonGroup } from "@/components/ui/button-group"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  isDashboardCrudActionDisabled,
  isDashboardCrudActionVisible,
  type DashboardCrudActionRule,
  type DashboardCrudFormField,
  type DashboardCrudPageResult,
  type DashboardCrudQueryFilter,
  type DashboardCrudQueryValue,
} from "./dashboard-crud-utils"
import { DashboardCrudFormDialog } from "./dashboard-crud-form-dialog"

export type DashboardCrudFilter<TValue extends string | number = string> =
  DashboardCrudQueryFilter & {
    label: string
    placeholder?: string
    defaultValue: TValue
    type?: "text" | "select"
    className?: string
    options?: ReadonlyArray<{ value: string; label: string }>
  }

export type DashboardCrudColumn<TItem> = {
  key: string
  label: ReactNode
  className?: string
  render: (item: TItem, context: DashboardCrudRowActionContext<TItem>) => ReactNode
}

export type DashboardCrudDialogProps<TItem, TPayload> = {
  open: boolean
  saving: boolean
  item: TItem | null
  itemId: number | null
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: TPayload) => Promise<void>
}

export type DashboardCrudRowActionContext<TItem> = {
  item: TItem
  itemId: number
  actionLoading: boolean
  actionLoadingId: number | null
  reload: () => Promise<void>
  setActionLoadingId: (id: number | null) => void
}

export type DashboardCrudRowAction<TItem> = DashboardCrudActionRule<TItem> & {
  key: string
  label: ReactNode | ((item: TItem) => ReactNode)
  icon?: ReactNode | ((item: TItem) => ReactNode)
  variant?: "default" | "destructive"
  confirm?: ConfirmOptions | ((item: TItem) => ConfirmOptions)
  run: (context: DashboardCrudRowActionContext<TItem>) => Promise<void> | void
}

export type DashboardCrudActionState = {
  onRefresh: () => void
  onCreate: () => void
  loading: boolean
}

export type DashboardCrudToolbarActionContext = {
  reload: () => Promise<void>
  loading: boolean
}

export type DashboardCrudPageProps<TItem, TPayload> = {
  filters: DashboardCrudFilter[]
  columns: DashboardCrudColumn<TItem>[]
  fetchList: (
    query: Record<string, DashboardCrudQueryValue>
  ) => Promise<DashboardCrudPageResult<TItem>>
  renderEditDialog?: (props: DashboardCrudDialogProps<TItem, TPayload>) => ReactNode
  form?: {
    fields: DashboardCrudFormField<TItem>[]
    fetchDetail?: (id: number) => Promise<TItem>
    transformSubmitValues?: (
      values: Record<string, string | number | boolean | string[] | number[]>,
      context: { mode: "create" | "edit"; item: TItem | null }
    ) => TPayload
    labels: {
      createTitle: string
      editTitle: string
      create: string
      save: string
      saving: string
      cancel: string
      loadingDetail: string
      required: string
      invalidNumber: string
      minValue: (min: number) => string
      maxValue: (max: number) => string
    }
  }
  getItemId: (item: TItem) => number
  createItem: (payload: TPayload) => Promise<unknown>
  updateItem: (item: TItem, payload: TPayload) => Promise<unknown>
  onCreateItem?: () => void
  canEdit?: (item: TItem) => boolean
  onEditItem?: (item: TItem) => void
  deleteItem?: (item: TItem) => Promise<unknown>
  canDelete?: (item: TItem) => boolean
  deleteConfirm?: false | ConfirmOptions | ((item: TItem) => ConfirmOptions)
  rowActions?: DashboardCrudRowAction<TItem>[]
  renderToolbarActions?: (context: DashboardCrudToolbarActionContext) => ReactNode
  renderRowActions?: (context: DashboardCrudRowActionContext<TItem>) => ReactNode
  sort?: {
    enabled?: boolean
    disabled?: boolean
    onReorder: (items: TItem[]) => Promise<unknown>
    successMessage?: string
    errorMessage: string
    handleLabel: string
  }
  pageSize?: number
  reloadKey?: string | number | null
  layout?: "page" | "fragment"
  showToolbar?: boolean
  showToolbarActions?: boolean
  tableShellClassName?: string
  onActionStateChange?: (state: DashboardCrudActionState) => void
  labels: {
    refresh: string
    create: string
    query: string
    loading: string
    empty: string
    actions: string
    edit: string
    delete: string
    processing: string
    moreActions: (item: TItem) => string
    loadFailed: string
    saveFailed: string
    deleteFailed: string
    created: (item: TPayload) => string
    updated: (item: TItem, payload: TPayload) => string
    deleted?: (item: TItem) => string
  }
}

export function DashboardCrudPage<TItem, TPayload>({
  filters,
  columns,
  fetchList,
  renderEditDialog,
  form,
  getItemId,
  createItem,
  updateItem,
  onCreateItem,
  canEdit,
  onEditItem,
  deleteItem,
  canDelete,
  deleteConfirm,
  rowActions = [],
  renderToolbarActions,
  renderRowActions,
  sort,
  pageSize = 20,
  reloadKey,
  layout = "page",
  showToolbar = true,
  showToolbarActions = true,
  tableShellClassName,
  onActionStateChange,
  labels,
}: DashboardCrudPageProps<TItem, TPayload>) {
  const confirm = useConfirm()
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 6,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )
  const list = useDashboardPagedList<TItem>({
    filters,
    fetchList,
    pageSize,
    reloadKey,
    loadFailed: labels.loadFailed,
  })
  const draftFilters = list.draftFilters
  const setDraftFilter = list.setDraftFilter
  const loading = list.loading
  const result = list.result
  const loadData = list.loadData
  const [saving, setSaving] = useState(false)
  const [actionLoadingId, setActionLoadingId] = useState<number | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<TItem | null>(null)

  function handleApplyFilters() {
    list.applyFilters()
  }

  function handleFilterKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key !== "Enter") return
    event.preventDefault()
    handleApplyFilters()
  }

  const openCreateDialog = useCallback(() => {
    if (onCreateItem) {
      onCreateItem()
      return
    }
    setEditingItem(null)
    setDialogOpen(true)
  }, [onCreateItem])
  const loadDataRef = useRef(loadData)
  const openCreateDialogRef = useRef(openCreateDialog)
  loadDataRef.current = loadData
  openCreateDialogRef.current = openCreateDialog
  const actionRefresh = useCallback(() => void loadDataRef.current(), [])
  const actionCreate = useCallback(() => openCreateDialogRef.current(), [])

  function openEditDialog(item: TItem) {
    setEditingItem(item)
    setDialogOpen(true)
  }

  useEffect(() => {
    onActionStateChange?.({
      onRefresh: actionRefresh,
      onCreate: actionCreate,
      loading,
    })
  }, [actionCreate, actionRefresh, loading, onActionStateChange])

  function handleDialogOpenChange(open: boolean) {
    if (saving) return
    if (!open) setEditingItem(null)
    setDialogOpen(open)
  }

  async function handleSubmit(payload: TPayload) {
    if (saving) return
    setSaving(true)
    try {
      if (editingItem) {
        await updateItem(editingItem, payload)
        toast.success(labels.updated(editingItem, payload))
      } else {
        await createItem(payload)
        toast.success(labels.created(payload))
      }
      setDialogOpen(false)
      setEditingItem(null)
      await loadData()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : labels.saveFailed)
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(item: TItem) {
    if (!deleteItem) return
    if (deleteConfirm !== false) {
      const confirmed = await confirm(
        typeof deleteConfirm === "function"
          ? deleteConfirm(item)
          : {
              confirmText: labels.delete,
              variant: "destructive",
              ...deleteConfirm,
            }
      )
      if (!confirmed) return
    }
    const id = getItemId(item)
    setActionLoadingId(id)
    try {
      await deleteItem(item)
      toast.success(labels.deleted?.(item) ?? labels.delete)
      await loadData()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : labels.deleteFailed)
    } finally {
      setActionLoadingId(null)
    }
  }

  async function runRowAction(
    action: DashboardCrudRowAction<TItem>,
    item: TItem,
    actionLoading: boolean
  ) {
    if (actionLoading || isDashboardCrudActionDisabled(action, item)) return

    if (action.confirm) {
      const confirmed = await confirm(
        typeof action.confirm === "function" ? action.confirm(item) : action.confirm
      )
      if (!confirmed) return
    }

    const id = getItemId(item)
    setActionLoadingId(id)
    try {
      await action.run({
        item,
        itemId: id,
        actionLoading: true,
        actionLoadingId: id,
        reload: loadData,
        setActionLoadingId,
      })
    } finally {
      setActionLoadingId(null)
    }
  }

  async function handleSortEnd(event: DragEndEvent) {
    if (!sort?.enabled || sort.disabled) return

    const { active, over } = event
    if (!over || active.id === over.id) return

    const oldIndex = result.results.findIndex(
      (item) => String(getItemId(item)) === String(active.id)
    )
    const newIndex = result.results.findIndex(
      (item) => String(getItemId(item)) === String(over.id)
    )
    if (oldIndex < 0 || newIndex < 0) return

    const previousResults = result.results
    const nextResults = arrayMove(result.results, oldIndex, newIndex)
    list.setResult((current) => ({ ...current, results: nextResults }))

    try {
      await sort.onReorder(nextResults)
      if (sort.successMessage) {
        toast.success(sort.successMessage)
      }
    } catch (error) {
      list.setResult((current) => ({ ...current, results: previousResults }))
      toast.error(error instanceof Error ? error.message : sort.errorMessage)
    }
  }

  const sortable = Boolean(sort?.enabled)
  const colSpan = columns.length + 1 + (sortable ? 1 : 0)

  function renderTableRow(
    item: TItem,
    options: { sortable?: boolean; sortDisabled?: boolean } = {}
  ) {
    const id = getItemId(item)
    const actionLoading = actionLoadingId === id
    const rowCells = (
      <>
        {columns.map((column) => (
          <TableCell key={column.key} className={column.className}>
            {column.render(item, {
              item,
              itemId: id,
              actionLoading,
              actionLoadingId,
              reload: loadData,
              setActionLoadingId,
            })}
          </TableCell>
        ))}
        <TableCell className="text-right">
          <ButtonGroup className="ml-auto">
            <Button
              variant="outline"
              size="sm"
              disabled={canEdit ? !canEdit(item) : false}
              onClick={() => {
                if (onEditItem) {
                  onEditItem(item)
                  return
                }
                openEditDialog(item)
              }}
            >
              {labels.edit}
            </Button>
            {renderRowActions || rowActions.length > 0 || deleteItem ? (
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={<Button variant="outline" size="icon-sm" />}
                  aria-label={labels.moreActions(item)}
                >
                  <MoreHorizontalIcon />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-40 min-w-40">
                  {rowActions
                    .filter((action) => isDashboardCrudActionVisible(action, item))
                    .map((action) => {
                      const icon =
                        typeof action.icon === "function"
                          ? action.icon(item)
                          : action.icon
                      const label =
                        typeof action.label === "function"
                          ? action.label(item)
                          : action.label
                      return (
                        <DropdownMenuItem
                          key={action.key}
                          disabled={
                            actionLoading ||
                            isDashboardCrudActionDisabled(action, item)
                          }
                          className={
                            action.variant === "destructive"
                              ? "text-destructive focus:text-destructive"
                              : undefined
                          }
                          onClick={() => void runRowAction(action, item, actionLoading)}
                        >
                          {icon}
                          {actionLoading ? labels.processing : label}
                        </DropdownMenuItem>
                      )
                    })}
                  {renderRowActions?.({
                    item,
                    itemId: id,
                    actionLoading,
                    actionLoadingId,
                    reload: loadData,
                    setActionLoadingId,
                  })}
                  {deleteItem ? (
                    <DropdownMenuItem
                      onClick={() => void handleDelete(item)}
                      disabled={canDelete ? !canDelete(item) : false}
                      className="text-destructive focus:text-destructive"
                    >
                      <Trash2Icon />
                      {actionLoading ? labels.processing : labels.delete}
                    </DropdownMenuItem>
                  ) : null}
                </DropdownMenuContent>
              </DropdownMenu>
            ) : null}
          </ButtonGroup>
        </TableCell>
      </>
    )

    if (options.sortable) {
      return (
        <DashboardCrudSortableRow
          key={id}
          id={String(id)}
          disabled={loading || options.sortDisabled}
          handleLabel={sort?.handleLabel ?? labels.actions}
        >
          {rowCells}
        </DashboardCrudSortableRow>
      )
    }

    return <TableRow key={id}>{rowCells}</TableRow>
  }

  function renderStateRow() {
    if (!loading && result.results.length > 0) return null
    return (
      <DashboardTableStateRow
        colSpan={colSpan}
        loading={loading}
        loadingText={labels.loading}
        emptyText={labels.empty}
      />
    )
  }

  const tableElement = (
    <Table>
      <TableHeader className="bg-muted/40">
        <TableRow>
          {sortable ? <TableHead className="w-10" /> : null}
          {columns.map((column) => (
            <TableHead key={column.key} className={column.className}>
              {column.label}
            </TableHead>
          ))}
          <TableHead className="w-[92px] text-right">
            {labels.actions}
          </TableHead>
        </TableRow>
      </TableHeader>
      {sortable ? (
        <SortableContext
          items={result.results.map((item) => String(getItemId(item)))}
          strategy={verticalListSortingStrategy}
        >
          <TableBody>
            {result.results.map((item) =>
              renderTableRow(item, {
                sortable: true,
                sortDisabled: sort?.disabled,
              })
            )}
            {renderStateRow()}
          </TableBody>
        </SortableContext>
      ) : (
        <TableBody>
          {result.results.map((item) => renderTableRow(item))}
          {renderStateRow()}
        </TableBody>
      )}
    </Table>
  )

  const table = sortable ? (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragEnd={(event) => void handleSortEnd(event)}
    >
      {tableElement}
    </DndContext>
  ) : (
    tableElement
  )

  const content = (
    <>
      {showToolbar ? (
        <DashboardToolbar
          actions={
            showToolbarActions ? (
              <>
                <Button variant="outline" onClick={() => void loadData()} disabled={loading}>
                  <RefreshCwIcon className={loading ? "animate-spin" : undefined} />
                  {labels.refresh}
                </Button>
                <Button onClick={openCreateDialog}>
                  <PlusIcon />
                  {labels.create}
                </Button>
                {renderToolbarActions?.({
                  reload: loadData,
                  loading,
                })}
              </>
            ) : null
          }
        >
          {filters.map((filter) => {
            const value = draftFilters[filter.name]
            if (filter.type === "select") {
              return (
                <div key={filter.name} className={filter.className ?? "w-full sm:w-40"}>
                  <OptionCombobox
                    value={String(value ?? "")}
                    onChange={(nextValue) => setDraftFilter(filter.name, nextValue)}
                    placeholder={filter.placeholder ?? filter.label}
                    options={[...(filter.options ?? [])]}
                  />
                </div>
              )
            }

            return (
              <div key={filter.name} className={filter.className ?? "w-full sm:w-64"}>
                <Input
                  value={String(value ?? "")}
                  onChange={(event) =>
                    setDraftFilter(filter.name, event.target.value)
                  }
                  onKeyDown={handleFilterKeyDown}
                  placeholder={filter.placeholder ?? filter.label}
                />
              </div>
            )
          })}
          <Button variant="outline" onClick={handleApplyFilters} disabled={loading}>
            <SearchIcon />
            {labels.query}
          </Button>
        </DashboardToolbar>
      ) : null}

      <DashboardTableShell
        className={tableShellClassName}
        pagination={
          <ListPagination
            page={result.page.page}
            total={result.page.total}
            limit={list.limit}
            loading={loading}
            onPageChange={list.handlePageChange}
            onLimitChange={list.handleLimitChange}
          />
        }
      >
        {table}
      </DashboardTableShell>
    </>
  )

  const dialog = form ? (
    <DashboardCrudFormDialog
      open={dialogOpen}
      saving={saving}
      item={editingItem}
      itemId={editingItem ? getItemId(editingItem) : null}
      fields={form.fields}
      fetchDetail={form.fetchDetail}
      transformSubmitValues={form.transformSubmitValues}
      labels={form.labels}
      onOpenChange={handleDialogOpenChange}
      onSubmit={handleSubmit}
    />
  ) : (
    renderEditDialog?.({
      open: dialogOpen,
      saving,
      item: editingItem,
      itemId: editingItem ? getItemId(editingItem) : null,
      onOpenChange: handleDialogOpenChange,
      onSubmit: handleSubmit,
    })
  )

  if (layout === "fragment") {
    return (
      <>
        {content}
        {dialog}
      </>
    )
  }

  return (
    <>
      <DashboardPage>{content}</DashboardPage>
      {dialog}
    </>
  )
}

function DashboardCrudSortableRow({
  id,
  disabled,
  handleLabel,
  children,
}: {
  id: string
  disabled?: boolean
  handleLabel: string
  children: ReactNode
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id, disabled })

  return (
    <TableRow
      ref={setNodeRef}
      style={{
        transform: CSS.Transform.toString(transform),
        transition,
      }}
      className={isDragging ? "relative z-10 bg-muted/60" : undefined}
    >
      <TableCell className="w-10">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          disabled={disabled}
          aria-label={handleLabel}
          {...attributes}
          {...listeners}
        >
          <GripVerticalIcon className="text-muted-foreground" />
        </Button>
      </TableCell>
      {children}
    </TableRow>
  )
}
