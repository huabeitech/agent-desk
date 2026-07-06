"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import { toast } from "sonner"

import {
  buildDashboardCrudQuery,
  normalizeDashboardCrudPageResult,
  type DashboardCrudPageResult,
  type DashboardCrudFilterStateConfig,
  type DashboardCrudQueryFilter,
  type DashboardCrudQueryValue,
} from "@/components/dashboard/crud/dashboard-crud-utils"
import { useDashboardCrudFilters } from "@/components/dashboard/crud/use-dashboard-crud-filters"

export type DashboardPagedListFilter = DashboardCrudQueryFilter &
  DashboardCrudFilterStateConfig

export type DashboardPagedListOptions<TItem> = {
  filters: DashboardPagedListFilter[]
  fetchList: (
    query: Record<string, DashboardCrudQueryValue>
  ) => Promise<DashboardCrudPageResult<TItem>>
  pageSize?: number
  loadFailed: string
  enabled?: boolean
  reloadKey?: string | number | null
}

export function useDashboardPagedList<TItem>({
  filters,
  fetchList,
  pageSize = 20,
  loadFailed,
  enabled = true,
  reloadKey,
}: DashboardPagedListOptions<TItem>) {
  const fetchListRef = useRef(fetchList)
  fetchListRef.current = fetchList
  const filtersKey = filters
    .map(
      (filter) =>
        `${filter.name}:${String(filter.defaultValue)}:${String(filter.allValue)}:${filter.trim ? "1" : "0"}:${filter.valueType ?? ""}`
    )
    .join("|")
  const {
    draftFilters,
    appliedFilters,
    setDraftFilter,
    applyFilter,
    applyFilters,
    resetFilters,
  } =
    useDashboardCrudFilters(filters)
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(pageSize)
  const [loading, setLoading] = useState(enabled)
  const [result, setResult] = useState<DashboardCrudPageResult<TItem>>({
    results: [],
    page: { page: 1, limit: pageSize, total: 0 },
  })

  const loadData = useCallback(async () => {
    if (!enabled) {
      setLoading(false)
      setResult({ results: [], page: { page: 1, limit, total: 0 } })
      return
    }

    setLoading(true)
    try {
      const data = await fetchListRef.current(
        buildDashboardCrudQuery({
          values: appliedFilters,
          filters,
          page,
          limit,
        })
      )
      setResult(normalizeDashboardCrudPageResult(data, page, limit))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : loadFailed)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [appliedFilters, enabled, filtersKey, limit, loadFailed, page, reloadKey])

  useEffect(() => {
    void loadData()
  }, [loadData])

  function applyDraftFilters() {
    applyFilters()
    setPage(1)
  }

  function applyDraftFilter(name: string, value: string | number | undefined) {
    applyFilter(name, value)
    setPage(1)
  }

  function handlePageChange(nextPage: number) {
    if (nextPage < 1 || nextPage === page) return
    setPage(nextPage)
  }

  function handleLimitChange(nextLimit: number) {
    if (nextLimit <= 0 || nextLimit === limit) return
    setLimit(nextLimit)
    setPage(1)
  }

  function resetDraftFilters() {
    resetFilters()
    setPage(1)
  }

  return {
    draftFilters,
    appliedFilters,
    setDraftFilter,
    applyFilter: applyDraftFilter,
    applyFilters: applyDraftFilters,
    page,
    setPage,
    limit,
    setLimit,
    loading,
    result,
    setResult,
    loadData,
    handlePageChange,
    handleLimitChange,
    resetFilters: resetDraftFilters,
  }
}
