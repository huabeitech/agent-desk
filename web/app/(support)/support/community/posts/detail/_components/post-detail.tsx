"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import Link from "next/link"
import { usePathname, useSearchParams } from "next/navigation"
import { ChevronDownIcon, EyeIcon, LoaderCircleIcon, MessageCircleMoreIcon, ThumbsUpIcon } from "lucide-react"
import { toast } from "sonner"

import { ContentEditor } from "@/components/content-editor"
import { Button } from "@/components/ui/button"
import { Breadcrumb, BreadcrumbItem, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb"
import { PublicArticleToc, hasArticleTocHeadings } from "@/app/(support)/support/_components/support-article-toc"
import { CommunityFrame } from "@/app/(support)/support/_components/community-frame"
import { PostArticleContent } from "@/app/(support)/support/_components/post-article-content"
import { ensureSupportLogin, useCommunityCategoryRoute } from "@/app/(support)/support/_components/support-community-route"
import { PostStatusBadge as PostStatusBadge } from "@/app/(support)/support/_components/support-ui"
import { EmptyState } from "@/components/empty-state"
import { CommentItem } from "@/app/(support)/support/community/posts/detail/_components/comment-item"
import { PostMetric } from "@/app/(support)/support/community/posts/_components/post-ui"
import { useI18n } from "@/i18n/provider"
import { createComment, fetchComments, fetchPost, postsHref, toggleReaction, type Comment, type Post } from "@/lib/api/support-community"
import { readSession } from "@/lib/auth"
import { formatDateTime, cn } from "@/lib/utils"
import type { ContentValue } from "@/components/content-editor"

type CommentSort = "default" | "latest" | "hot"

export function PostDetail() {
  const t = useI18n()
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const categoryRoute = useCommunityCategoryRoute()
  const postId = useMemo(() => {
    const queryId = Number(searchParams.get("id"))
    if (queryId > 0) {
      return queryId
    }
    const pathId = Number(pathname.split("/").filter(Boolean).at(-1))
    return pathId > 0 ? pathId : 0
  }, [pathname, searchParams])
  const [post, setPost] = useState<Post | null>(null)
  const [comments, setComments] = useState<Comment[]>([])
  const [content, setContent] = useState<ContentValue>({ mode: "html", raw: "" })
  const [commentSort, setCommentSort] = useState<CommentSort>("default")
  const [commentPage, setCommentPage] = useState({ page: 1, limit: 20, total: 0 })
  const [commentsLoading, setCommentsLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const postArticleId = post ? `support-post-${post.id}` : ""
  const postToc = post && hasArticleTocHeadings(post.content, post.contentType)
    ? <PublicArticleToc articleId={postArticleId} content={post.content} contentType={post.contentType} stickyOffset="content" />
    : null
  const currentUserId = readSession()?.user.id ?? 0

  const loadComments = useCallback((page = 1, append = false) => {
    if (postId > 0) {
      setCommentsLoading(true)
      void fetchComments({ postId, sort: commentSort, page, limit: commentPage.limit })
        .then((result) => {
          setComments((current) => append ? [...current, ...result.results] : result.results)
          setCommentPage(result.page)
        })
        .finally(() => setCommentsLoading(false))
    }
  }, [commentPage.limit, commentSort, postId])

  const reload = useCallback(() => {
    if (postId <= 0) return
    void Promise.all([
      fetchPost(postId),
      fetchComments({ postId, sort: commentSort, page: 1, limit: commentPage.limit }),
    ]).then(([detail, commentResult]) => {
      setPost(detail.post)
      setComments(commentResult.results)
      setCommentPage(commentResult.page)
    })
  }, [commentPage.limit, commentSort, postId])

  useEffect(() => {
    if (postId <= 0) return
    void fetchPost(postId).then((detail) => {
      setPost(detail.post)
    })
  }, [postId])

  useEffect(() => {
    loadComments(1)
  }, [loadComments])

  const submitComment = async () => {
    if (!post || submitting) return
    setSubmitting(true)
    try {
      await ensureSupportLogin()
      await createComment({ postId: post.id, contentType: content.mode, content: content.raw })
      setContent({ mode: "html", raw: "" })
      toast.success(t("supportPublic.toast.commentCreated"))
      reload()
    } finally {
      setSubmitting(false)
    }
  }
  const hasMoreComments = comments.length < commentPage.total

  return (
    <CommunityFrame active={post?.categoryId ?? "all"} categoryRoute={categoryRoute} toc={postToc} contentCard={false}>
      {post ? (
        <article className="flex w-full max-w-6xl flex-col gap-6">
          <section className="rounded-md bg-card px-4 py-4 sm:px-6 sm:py-6 lg:px-8 2xl:px-8">
            <Breadcrumb className="text-xs">
              <BreadcrumbList className="gap-y-1">
                <BreadcrumbItem>
                  <Link href={postsHref()} className="transition-colors hover:text-foreground">{t("supportPublic.posts.title")}</Link>
                </BreadcrumbItem>
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                  <BreadcrumbPage>{post.categoryName || t("supportPublic.common.uncategorized")}</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>

            <header className="mt-6">
              <h1 className="text-balance text-3xl font-semibold tracking-tight sm:text-4xl">{post.title}</h1>
            </header>

            <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
              <PostStatusBadge status={post.status} />
              <span>{post.categoryName || t("supportPublic.common.uncategorized")}</span>
              <span className="text-muted-foreground/40">/</span>
              <span>{t("supportPublic.posts.createdBy", { name: post.user.displayName || t("supportPublic.common.user") })}</span>
              <span className="text-muted-foreground/40">/</span>
              <span>{t("supportPublic.posts.updatedAt", { date: formatDateTime(post.updatedAt || post.createdAt) })}</span>
            </div>

            <div className="mt-8">
              <PostArticleContent id={postArticleId} content={post.content} contentType={post.contentType} />
            </div>

            <div className="mt-8 flex flex-wrap items-center gap-2">
              <Button variant="secondary" size="sm" className="rounded-md" onClick={() => void ensureSupportLogin().then(() => toggleReaction({ targetType: "post", targetId: post.id })).then(reload)}>
                <ThumbsUpIcon /> {post.reactionCount}
              </Button>
              <PostMetric icon={<MessageCircleMoreIcon className="size-3.5" />} value={post.commentCount} label={t("supportPublic.posts.comments")} />
              <PostMetric icon={<EyeIcon className="size-3.5" />} value={post.viewCount} label={t("supportPublic.posts.views")} />
            </div>
          </section>

          <section className="rounded-md bg-card px-4 py-6 sm:px-6 sm:py-8 lg:px-8" aria-label={t("supportPublic.posts.comments")}>
            <div className="mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 className="text-lg font-semibold tracking-tight">{t("supportPublic.posts.comments")}</h2>
                <div className="mt-1 text-sm text-muted-foreground">{t("supportPublic.comment.count", { count: commentPage.total || post.commentCount })}</div>
              </div>
              <div className="inline-flex w-fit gap-1 rounded-md bg-muted p-0.5">
                {(["default", "latest", "hot"] as const).map((sort) => (
                  <button
                    key={sort}
                    type="button"
                    className={cn(
                      "h-7 rounded-md px-2.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                      commentSort === sort ? "bg-background text-foreground shadow-xs" : "text-muted-foreground hover:text-foreground"
                    )}
                    aria-pressed={commentSort === sort}
                    onClick={() => setCommentSort(sort)}
                  >
                    {t(`supportPublic.comment.sort.${sort}`)}
                  </button>
                ))}
              </div>
            </div>
            {comments.length ? (
              <div className="divide-y divide-border/70">
                {comments.map((comment) => (
                  <CommentItem key={comment.id} comment={comment} post={post} currentUserId={currentUserId} onChanged={reload} />
                ))}
              </div>
            ) : commentsLoading ? (
              <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
                <LoaderCircleIcon className="size-4 animate-spin" />
                {t("supportPublic.loading.comments")}
              </div>
            ) : <EmptyState text={t("supportPublic.empty.noComments")} />}
            {hasMoreComments ? (
              <div className="mt-4 flex justify-center">
                <Button variant="secondary" size="sm" className="rounded-md" disabled={commentsLoading} onClick={() => loadComments(commentPage.page + 1, true)}>
                  {commentsLoading ? <LoaderCircleIcon className="animate-spin" /> : <ChevronDownIcon />}
                  {commentsLoading ? t("supportPublic.loading.comments") : t("supportPublic.actions.loadMore")}
                </Button>
              </div>
            ) : null}
            <section className="mt-6 border-t border-border/70 pt-5" aria-labelledby="support-comment-editor-title">
              <h2 id="support-comment-editor-title" className="text-base font-semibold">{t("supportPublic.comment.title")}</h2>
              <div className="mt-3 min-w-0">
                <ContentEditor
                  value={content}
                  onChange={setContent}
                  placeholder={t("supportPublic.comment.placeholder")}
                  disabled={submitting}
                  allowedModes={["html", "markdown"]}
                  height={260}
                  className="min-w-0"
                />
              </div>
              <div className="mt-3 flex justify-end">
                <Button disabled={submitting || !content.raw.trim()} onClick={() => void submitComment()}>
                  {submitting ? t("supportPublic.actions.publishing") : t("supportPublic.actions.publishComment")}
                </Button>
              </div>
            </section>
          </section>
        </article>
      ) : (
        <div className="grid min-h-[60svh] place-items-center">
          <EmptyState text={t("supportPublic.loading.post")} />
        </div>
      )}
    </CommunityFrame>
  )
}
