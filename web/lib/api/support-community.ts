import { type PageResult } from "@/lib/api/admin"
import { request } from "@/lib/api/client"
import type { PageData } from "@/lib/api/types"

export type Category = {
  id: number
  name: string
  slug: string
  description: string
  parentId?: number
  sortNo: number
  status: number
}

export type SimpleUserInfo = {
  id: number
  username: string
  nickname: string
  displayName: string
  avatar: string
  userType: string
}

export type Post = {
  id: number
  categoryId: number
  categoryName: string
  user: SimpleUserInfo
  title: string
  contentType: string
  content: string
  tags: string[]
  status: string
  acceptedCommentId: number
  commentCount: number
  reactionCount: number
  viewCount: number
  createdAt: string
  updatedAt: string
}

export type PostListItem = Omit<Post, "contentType" | "content"> & {
  summary: string
}

export type Comment = {
  id: number
  postId: number
  parentId: number
  authorType: string
  user: SimpleUserInfo
  contentType: string
  content: string
  status: string
  reactionCount: number
  replyCount: number
  reportCount: number
  isAccepted: boolean
  replies: Comment[]
  createdAt: string
  updatedAt: string
}

export type PostDetail = {
  comments: Comment[]
  post: Post
}

export type ReactionTargetType = "post" | "comment"
export type ReactionType = "like"

export function categoryHref(slug: string) {
  return `/support/community/categories/${encodeURIComponent(slug)}`
}

export function newPostHref() {
  return "/support/community/posts/new"
}

export function postHref(id: number) {
  return `/support/community/posts/${id}`
}

export function postsHref() {
  return "/support/community/posts"
}

export function fetchCategories() {
  return request<Category[]>("/api/support/community/categories/list", { skipAuth: true })
}

export function fetchPosts(query?: Record<string, string | number | undefined>) {
  return request<PageData<PostListItem>>(`/api/support/community/posts/list${toQueryString(query)}`, {
    skipAuth: true,
  })
}

export function fetchPost(id: number) {
  return request<PostDetail>(`/api/support/community/posts/${id}`, { skipAuth: true })
}

export function createPost(payload: {
  categoryId: number
  content: string
  contentType: string
  tags: string[]
  title: string
}) {
  return request<Post>("/api/support/community/posts/create", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function fetchComments(query: {
  limit?: number
  page?: number
  parentId?: number
  postId: number
  sort?: string
}) {
  return request<PageResult<Comment>>(`/api/support/community/comments/list${toQueryString(query)}`, {
    skipAuth: true,
  })
}

export function createComment(payload: {
  content: string
  contentType: string
  parentId?: number
  postId: number
}) {
  return request<Comment>("/api/support/community/comments/create", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function updateComment(payload: { content: string; contentType: string; id: number }) {
  return request<void>("/api/support/community/comments/update", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export function deleteComment(id: number) {
  return request<void>("/api/support/community/comments/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  })
}

export function reportComment(id: number, reason = "") {
  return request<void>("/api/support/community/comments/report", {
    method: "POST",
    body: JSON.stringify({ id, reason }),
  })
}

export function acceptComment(postId: number, commentId: number) {
  return request<void>("/api/support/community/posts/accept_comment", {
    method: "POST",
    body: JSON.stringify({ postId, commentId }),
  })
}

export function toggleReaction(payload: { reactionType?: ReactionType; targetId: number; targetType: ReactionTargetType }) {
  return request<void>("/api/support/community/reactions/toggle", {
    method: "POST",
    body: JSON.stringify({ reactionType: payload.reactionType || "like", targetId: payload.targetId, targetType: payload.targetType }),
  })
}

function toQueryString(query?: Record<string, string | number | undefined>) {
  if (!query) {
    return ""
  }
  const params = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined && value !== "" && value !== "all") {
      params.set(key, String(value))
    }
  })
  const raw = params.toString()
  return raw ? `?${raw}` : ""
}
