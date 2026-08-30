"use client"

import { useCallback } from "react"

import { SupportEmptyState } from "@/app/(support)/support/_components/support-ui"
import { PostCard } from "@/app/(support)/support/community/posts/_components/post-ui"
import { LoadMore } from "@/components/load-more"
import { useI18n } from "@/i18n/provider"
import { fetchPosts, type Post } from "@/lib/api/support-community"

type CommunityPostListProps = {
  emptyText: string
  limit?: number
  query?: Record<string, string | number | undefined>
  resetKey: string
}

export function CommunityPostList({
  emptyText,
  limit = 20,
  query,
  resetKey,
}: CommunityPostListProps) {
  const t = useI18n()
  const loadPosts = useCallback(({ cursor }: { cursor: string; force: boolean }) => {
    return fetchPosts({
      ...query,
      cursor,
      limit,
    })
  }, [limit, query])

  return (
    <LoadMore<Post>
      resetKey={resetKey}
      initialHasMore
      initialLoad
      labels={{
        loadMore: t("supportPublic.actions.loadMore"),
        noMore: t("supportPublic.actions.noMore"),
        loading: t("supportPublic.loading.posts"),
        error: t("supportPublic.empty.postsFailed"),
      }}
      loadPage={loadPosts}
      renderItems={(items) => (
        <ul className="divide-y divide-border">
          {items.map((item) => <PostCard key={item.id} item={item} />)}
        </ul>
      )}
      renderEmpty={() => <SupportEmptyState text={emptyText} />}
    />
  )
}
