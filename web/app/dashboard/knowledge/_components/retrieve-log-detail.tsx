"use client"

import { useRouter } from "next/navigation"
import { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"

import {
  createKnowledgeFeedback,
  createKnowledgeFAQDraftFromRetrieveLog,
  fetchKnowledgeRetrieveLog,
  type KnowledgeRetrieveHit,
  type KnowledgeRetrieveLogDetail,
} from "@/lib/api/admin"
import { KnowledgeFeedbackType } from "@/lib/generated/enums"
import { formatDateTime } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"
import { useI18n } from "@/i18n/provider"
import {
  getKnowledgeAnswerStatusLabel,
  getKnowledgeChunkProviderLabel,
  getKnowledgeRetrieveChannelLabel,
  getKnowledgeRetrieveSceneLabel,
} from "@/lib/knowledge-i18n"

type RetrieveLogDetailDrawerProps = {
  open: boolean
  retrieveLogId: number | null
  onOpenChange: (open: boolean) => void
}

function safeParseJSON(value: string) {
  if (!value) {
    return null
  }
  try {
    return JSON.parse(value) as Record<string, unknown>
  } catch {
    return null
  }
}

type TFunction = (key: string, values?: Record<string, string | number>) => string

const FEEDBACK_TYPES = [
  KnowledgeFeedbackType.Like,
  KnowledgeFeedbackType.Dislike,
  KnowledgeFeedbackType.NotHelpful,
  KnowledgeFeedbackType.WrongCitation,
  KnowledgeFeedbackType.Other,
]

function CitationList({ hits, t }: { hits: KnowledgeRetrieveHit[]; t: TFunction }) {
  const citations = hits.filter((item) => item.isCitation)
  if (citations.length === 0) {
    return <div className="text-sm text-muted-foreground">{t("knowledge.noCitations")}</div>
  }
  return (
    <div className="space-y-3">
      {citations.map((item) => (
        <div key={item.id} className="rounded-lg border p-3">
          <div className="flex items-center gap-2">
            <span className="font-medium">{getHitSourceLabel(item, t)}</span>
            <Badge variant="outline">Chunk #{item.chunkNo}</Badge>
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {item.sectionPath || item.title || t("knowledge.unrecordedSection")}
          </div>
          <div className="mt-2 text-sm leading-6 whitespace-pre-wrap text-foreground/90">
            {item.snippet || "-"}
          </div>
        </div>
      ))}
    </div>
  )
}

export function RetrieveLogDetailDrawer({
  open,
  retrieveLogId,
  onOpenChange,
}: RetrieveLogDetailDrawerProps) {
  const t = useI18n()
  const router = useRouter()
  const [loading, setLoading] = useState(false)
  const [detail, setDetail] = useState<KnowledgeRetrieveLogDetail | null>(null)
  const [feedbackType, setFeedbackType] = useState<KnowledgeFeedbackType>(KnowledgeFeedbackType.Like)
  const [feedbackReason, setFeedbackReason] = useState("")
  const [feedbackSaving, setFeedbackSaving] = useState(false)
  const [faqDraftSaving, setFaqDraftSaving] = useState(false)

  const loadDetail = useCallback(async () => {
    if (!retrieveLogId) {
      setDetail(null)
      return
    }
    setLoading(true)
    try {
      const data = await fetchKnowledgeRetrieveLog(retrieveLogId)
      setDetail(data)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.loadRetrieveLogDetailFailed"))
    } finally {
      setLoading(false)
    }
  }, [retrieveLogId, t])

  useEffect(() => {
    if (open && retrieveLogId) {
      void loadDetail()
    } else if (!open) {
      setFeedbackType(KnowledgeFeedbackType.Like)
      setFeedbackReason("")
    }
  }, [open, retrieveLogId, loadDetail])

  const traceData = useMemo(() => safeParseJSON(detail?.log.traceData ?? ""), [detail?.log.traceData])

  const handleSubmitFeedback = useCallback(async () => {
    if (!detail) {
      return
    }
    setFeedbackSaving(true)
    try {
      await createKnowledgeFeedback({
        retrieveLogId: detail.log.id,
        feedbackType,
        feedbackReason: feedbackReason.trim(),
      })
      toast.success(t("knowledge.feedbackSaved"))
      setFeedbackReason("")
      await loadDetail()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.feedbackSaveFailed"))
    } finally {
      setFeedbackSaving(false)
    }
  }, [detail, feedbackReason, feedbackType, loadDetail, t])

  const handleCreateFAQDraft = useCallback(async () => {
    if (!detail) {
      return
    }
    setFaqDraftSaving(true)
    try {
      const faq = await createKnowledgeFAQDraftFromRetrieveLog({
        retrieveLogId: detail.log.id,
        remark: t("knowledge.faqDraftRemark"),
      })
      const faqHref = `/dashboard/knowledge?tab=documents&knowledgeBaseId=${faq.knowledgeBaseId}&faqId=${faq.id}`
      toast.success(t("knowledge.faqDraftCreated"), {
        action: {
          label: t("knowledge.edit"),
          onClick: () => router.push(faqHref),
        },
      })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("knowledge.faqDraftCreateFailed"))
    } finally {
      setFaqDraftSaving(false)
    }
  }, [detail, router, t])

  if (!open) {
    return null
  }

  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="max-w-3xl">
        <DrawerHeader>
          <DrawerTitle>{t("knowledge.retrieveLogDetail")}</DrawerTitle>
          <DrawerDescription>
            {detail?.log.question || (loading ? t("knowledge.loading") : t("knowledge.retrieveLogNotFound"))}
          </DrawerDescription>
        </DrawerHeader>
        <ScrollArea className="h-[calc(100vh-6rem)] px-4 pb-6">
          {!detail ? (
            <div className="py-6 text-sm text-muted-foreground">
              {loading ? t("knowledge.loadingDetail") : t("knowledge.emptyDetail")}
            </div>
          ) : (
            <div className="space-y-6 pb-6">
              <section className="space-y-3">
                <h3 className="text-sm font-semibold">{t("knowledge.requestInfo")}</h3>
                <div className="grid gap-3 rounded-lg border p-4 md:grid-cols-2">
                  <div>
                    <div className="text-xs text-muted-foreground">{t("knowledge.title")}</div>
                    <div className="mt-1 text-sm">{detail.log.knowledgeBaseName || `#${detail.log.knowledgeBaseId}`}</div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">{t("knowledge.createdAt")}</div>
                    <div className="mt-1 text-sm">{formatDateTime(detail.log.createdAt)}</div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">{t("knowledge.channelScene")}</div>
                    <div className="mt-1 text-sm">
                      {getKnowledgeRetrieveChannelLabel(detail.log.channel, detail.log.channelName, t)} /{" "}
                      {getKnowledgeRetrieveSceneLabel(detail.log.scene, detail.log.sceneName, t)}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">Request ID</div>
                    <div className="mt-1 break-all font-mono text-xs">{detail.log.requestId || "-"}</div>
                  </div>
                  <div className="md:col-span-2">
                    <div className="text-xs text-muted-foreground">{t("knowledge.originalQuestion")}</div>
                    <div className="mt-1 text-sm leading-6 whitespace-pre-wrap">{detail.log.question || "-"}</div>
                  </div>
                  <div className="md:col-span-2">
                    <div className="text-xs text-muted-foreground">{t("knowledge.rewriteQuestion")}</div>
                    <div className="mt-1 text-sm leading-6 whitespace-pre-wrap">{detail.log.rewriteQuestion || "-"}</div>
                  </div>
                  <div className="md:col-span-2">
                    <div className="text-xs text-muted-foreground">{t("knowledge.answerContent")}</div>
                    <div className="mt-1 text-sm leading-6 whitespace-pre-wrap">{detail.log.answer || "-"}</div>
                  </div>
                </div>
              </section>

              <section className="space-y-3">
                <h3 className="text-sm font-semibold">{t("knowledge.retrieveStrategy")}</h3>
                <div className="grid gap-3 rounded-lg border p-4 md:grid-cols-3">
                  <Metric
                    label="Chunk Provider"
                    value={detail.log.chunkProvider ? getKnowledgeChunkProviderLabel(detail.log.chunkProvider, t) : "-"}
                    mono
                  />
                  <Metric label="Target Tokens" value={detail.log.chunkTargetTokens} />
                  <Metric label="Max Tokens" value={detail.log.chunkMaxTokens} />
                  <Metric label="Overlap Tokens" value={detail.log.chunkOverlapTokens} />
                  <Metric label="Rerank" value={detail.log.rerankEnabled ? t("knowledge.rerankEnabled") : t("knowledge.rerankDisabled")} />
                  <Metric label="Rerank Limit" value={detail.log.rerankLimit} />
                </div>
              </section>

              <section className="space-y-3">
                <h3 className="text-sm font-semibold">{t("knowledge.resultSummary")}</h3>
                <div className="grid gap-3 rounded-lg border p-4 md:grid-cols-4">
                  <Metric
                    label={t("knowledge.answerStatus")}
                    value={getKnowledgeAnswerStatusLabel(detail.log.answerStatus, detail.log.answerStatusName, t)}
                  />
                  <Metric label={t("knowledge.hitCount")} value={detail.log.hitCount} />
                  <Metric label={t("knowledge.citations")} value={detail.log.citationCount} />
                  <Metric label={t("knowledge.contextChunks")} value={detail.log.usedChunkCount} />
                  <Metric label="Top Score" value={detail.log.topScore.toFixed(4)} mono />
                  <Metric label={t("knowledge.retrieveMs")} value={`${detail.log.retrieveMs} ms`} />
                  <Metric label={t("knowledge.generateMs")} value={`${detail.log.generateMs} ms`} />
                  <Metric label={t("knowledge.totalMs")} value={`${detail.log.latencyMs} ms`} />
                  <Metric label="Prompt Tokens" value={detail.log.promptTokens} />
                  <Metric label="Completion Tokens" value={detail.log.completionTokens} />
                  <Metric label={t("knowledge.model")} value={detail.log.modelName || "-"} mono />
                  <Metric label={t("knowledge.sessionId")} value={detail.log.sessionId || "-"} mono />
                </div>
              </section>

              <section className="space-y-3">
                <h3 className="text-sm font-semibold">{t("knowledge.feedbackTitle")}</h3>
                <div className="space-y-3 rounded-lg border p-4">
                  <div className="flex flex-wrap gap-2">
                    {FEEDBACK_TYPES.map((item) => (
                      <Button
                        key={item}
                        type="button"
                        variant={feedbackType === item ? "default" : "outline"}
                        size="sm"
                        onClick={() => setFeedbackType(item)}
                      >
                        {getFeedbackTypeLabel(item, t)}
                      </Button>
                    ))}
                  </div>
                  <Textarea
                    rows={3}
                    value={feedbackReason}
                    maxLength={500}
                    placeholder={t("knowledge.feedbackReasonPlaceholder")}
                    onChange={(event) => setFeedbackReason(event.target.value)}
                  />
                  <div className="flex items-center justify-between gap-3">
                    <div className="text-xs text-muted-foreground">
                      {t("knowledge.feedbackHistoryCount", { count: detail.feedbacks?.length ?? 0 })}
                    </div>
                    <div className="flex flex-wrap justify-end gap-2">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void handleCreateFAQDraft()}
                        disabled={faqDraftSaving}
                      >
                        {faqDraftSaving ? t("knowledge.faqDraftCreating") : t("knowledge.faqDraftCreate")}
                      </Button>
                      <Button type="button" size="sm" onClick={() => void handleSubmitFeedback()} disabled={feedbackSaving}>
                        {feedbackSaving ? t("knowledge.feedbackSubmitting") : t("knowledge.feedbackSubmit")}
                      </Button>
                    </div>
                  </div>
                  <Separator />
                  {(detail.feedbacks?.length ?? 0) === 0 ? (
                    <div className="text-sm text-muted-foreground">{t("knowledge.noFeedback")}</div>
                  ) : (
                    <div className="space-y-2">
                      {detail.feedbacks.map((item) => (
                        <div key={item.id} className="rounded-md bg-muted/30 p-3">
                          <div className="flex flex-wrap items-center gap-2">
                            <Badge variant={item.feedbackType === KnowledgeFeedbackType.Like ? "default" : "secondary"}>
                              {getFeedbackTypeLabel(item.feedbackType, t, item.feedbackTypeName)}
                            </Badge>
                            <span className="text-xs text-muted-foreground">{formatDateTime(item.createdAt)}</span>
                          </div>
                          <div className="mt-2 text-sm leading-6 whitespace-pre-wrap">
                            {item.feedbackReason || item.remark || t("knowledge.noFeedbackReason")}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </section>

              <section className="space-y-3">
                <h3 className="text-sm font-semibold">{t("knowledge.citationSources")}</h3>
                <CitationList hits={detail.hits} t={t} />
              </section>

              <section className="space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-semibold">{t("knowledge.hitDetails")}</h3>
                  <div className="text-xs text-muted-foreground">{t("knowledge.itemsCount", { count: detail.hits.length })}</div>
                </div>
                <div className="space-y-3">
                  {detail.hits.map((item) => (
                    <div key={item.id} className="rounded-lg border p-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant="outline">#{item.rankNo}</Badge>
                        <span className="font-medium">{getHitSourceLabel(item, t)}</span>
                        <Badge variant={item.usedInAnswer ? "default" : "secondary"}>
                          {item.usedInAnswer ? t("knowledge.usedInContext") : t("knowledge.notUsedInContext")}
                        </Badge>
                        {item.isCitation ? <Badge>{t("knowledge.citations")}</Badge> : null}
                      </div>
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                        <span>{t("knowledge.section", { section: item.sectionPath || item.title || "-" })}</span>
                        <span>Chunk #{item.chunkNo}</span>
                        <span>Provider：{item.provider || "-"}</span>
                        <span>Score：{item.score.toFixed(4)}</span>
                        <span>Rerank：{item.rerankScore ? item.rerankScore.toFixed(4) : "-"}</span>
                      </div>
                      <Separator className="my-3" />
                      <div className="text-sm leading-6 whitespace-pre-wrap text-foreground/90">
                        {item.snippet || "-"}
                      </div>
                    </div>
                  ))}
                </div>
              </section>

              <section className="space-y-3">
                <h3 className="text-sm font-semibold">TraceData</h3>
                <div className="rounded-lg border bg-muted/20 p-4">
                  <pre className="overflow-x-auto text-xs leading-6 text-muted-foreground">
                    {traceData ? JSON.stringify(traceData, null, 2) : detail.log.traceData || "-"}
                  </pre>
                </div>
              </section>
            </div>
          )}
        </ScrollArea>
      </DrawerContent>
    </Drawer>
  )
}

function getHitSourceLabel(item: KnowledgeRetrieveHit, t: TFunction) {
  if (item.faqQuestion) {
    return item.faqQuestion
  }
  if (item.documentTitle) {
    return item.documentTitle
  }
  if (item.faqId > 0) {
    return `FAQ #${item.faqId}`
  }
  return `${t("knowledge.document")} #${item.documentId}`
}

function getFeedbackTypeLabel(feedbackType: number, t: TFunction, fallback = "") {
  switch (feedbackType) {
    case KnowledgeFeedbackType.Like:
      return t("knowledge.feedbackLike")
    case KnowledgeFeedbackType.Dislike:
      return t("knowledge.feedbackDislike")
    case KnowledgeFeedbackType.NotHelpful:
      return t("knowledge.feedbackNotHelpful")
    case KnowledgeFeedbackType.WrongCitation:
      return t("knowledge.feedbackWrongCitation")
    case KnowledgeFeedbackType.Other:
      return t("knowledge.feedbackOther")
    default:
      return fallback || String(feedbackType)
  }
}

function Metric({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string | number
  mono?: boolean
}) {
  return (
    <div>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`mt-1 text-sm ${mono ? "font-mono" : ""}`}>{String(value || value === 0 ? value : "-")}</div>
    </div>
  )
}
