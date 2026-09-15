"use client"

import { useState } from "react"
import {
  CheckCircle2Icon,
  FileSearchIcon,
  Loader2Icon,
  XCircleIcon,
} from "lucide-react"

import { ProjectDialog } from "@/components/project-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  testWxWorkKFReadMessages,
  type WxWorkKFMessageReadResult,
  type WxWorkKFMessageReadSample,
} from "@/lib/api/admin"
import { useI18n } from "@/i18n/provider"

type Translate = (key: string, values?: Record<string, string | number>) => string

const READ_TEST_STAGE_KEYS: Record<string, string> = {
  gettoken: "channel.readTestStageGettoken",
  syncmsg: "channel.readTestStageSyncmsg",
}

const READ_TEST_HINT_KEYS: Record<string, string> = {
  "-1": "channel.readTestHintMinus1",
  "40001": "channel.readTestHint40001",
  "40013": "channel.readTestHint40013",
  "40014": "channel.readTestHint40014",
  "40056": "channel.readTestHint40056",
  "42001": "channel.readTestHint42001",
  "45009": "channel.readTestHint45009",
  "48002": "channel.readTestHint48002",
  "48007": "channel.readTestHint48007",
  "60030": "channel.readTestHint60030",
  WXWORK_DISABLED: "channel.readTestHintWxworkDisabled",
  OPENKFID_MISSING: "channel.readTestHintOpenkfidMissing",
  INVALID_CHANNEL: "channel.readTestHintInvalidChannel",
  INVALID_CONFIG_JSON: "channel.readTestHintInvalidConfigJson",
  NETWORK_ERROR: "channel.readTestHintNetwork",
  BAD_RESPONSE: "channel.readTestHintBadResponse",
}

const READ_TEST_TYPE_KEYS: Record<string, string> = {
  text: "channel.readTestTypeText",
  image: "channel.readTestTypeImage",
  file: "channel.readTestTypeFile",
  voice: "channel.readTestTypeVoice",
  video: "channel.readTestTypeVideo",
  location: "channel.readTestTypeLocation",
  link: "channel.readTestTypeLink",
  business_card: "channel.readTestTypeBusinessCard",
  miniprogram: "channel.readTestTypeMiniProgram",
  event: "channel.readTestTypeEvent",
}

const READ_TEST_EVENT_KEYS: Record<string, string> = {
  enter_session: "channel.readTestEventEnterSession",
  session_status_change: "channel.readTestEventSessionStatusChange",
  servicer_status_change: "channel.readTestEventServicerStatusChange",
  msg_send_fail: "channel.readTestEventMsgSendFail",
}

function originLabel(t: Translate, origin: number): string {
  if (origin === 3) return t("channel.readTestOriginCustomer")
  if (origin === 4) return t("channel.readTestOriginEvent")
  if (origin === 5) return t("channel.readTestOriginServicer")
  return t("channel.readTestOriginUnknown", { origin })
}

function messageTypeLabel(t: Translate, msgType: string): string {
  return t(READ_TEST_TYPE_KEYS[msgType] ?? "channel.readTestTypeUnknown")
}

function eventTypeLabel(t: Translate, eventType: string): string {
  return READ_TEST_EVENT_KEYS[eventType] ? t(READ_TEST_EVENT_KEYS[eventType]) : eventType
}

function ReadTestSampleRow({
  t,
  sample,
}: {
  t: Translate
  sample: WxWorkKFMessageReadSample
}) {
  const isEvent = sample.msgType === "event"
  return (
    <div className="grid gap-1 px-1 py-2.5">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <span className="shrink-0 text-muted-foreground">{sample.sendTime || "—"}</span>
        <Badge variant={sample.origin === 4 ? "outline" : "secondary"} className="font-normal">
          {originLabel(t, sample.origin)}
        </Badge>
        <Badge variant={isEvent ? "destructive" : "outline"} className="font-normal">
          {isEvent && sample.eventType
            ? eventTypeLabel(t, sample.eventType)
            : messageTypeLabel(t, sample.msgType ?? "")}
        </Badge>
      </div>
      {!isEvent ? (
        <div className="break-words text-sm">{sample.textContent || "—"}</div>
      ) : null}
      <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-[11px] text-muted-foreground">
        {sample.externalUserId ? (
          <span className="min-w-0">
            {t("channel.readTestExternalUser")}
            <span className="ml-1 font-mono">{sample.externalUserId}</span>
          </span>
        ) : null}
        {sample.servicerUserId ? (
          <span className="min-w-0">
            {t("channel.readTestServicerUser")}
            <span className="ml-1 font-mono">{sample.servicerUserId}</span>
          </span>
        ) : null}
      </div>
    </div>
  )
}

export function WxWorkReadTestButton({ channelId }: { channelId: number | null }) {
  const t = useI18n()
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<WxWorkKFMessageReadResult | null>(null)

  async function runTest() {
    if (channelId === null) return
    setLoading(true)
    setResult(null)
    setOpen(true)
    try {
      setResult(await testWxWorkKFReadMessages(channelId))
    } catch (error) {
      setResult({
        success: false,
        errorCode: "NETWORK_ERROR",
        errorMessage: error instanceof Error ? error.message : String(error),
        totalScanned: 0,
        messageCount: 0,
        eventCount: 0,
        truncated: false,
        samples: [],
      })
    } finally {
      setLoading(false)
    }
  }

  const suggestion = result?.success
    ? ""
    : t(
        (result?.errorCode && READ_TEST_HINT_KEYS[result.errorCode]) ||
          "channel.readTestHintDefault",
      )

  return (
    <div className="space-y-1.5">
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="w-full sm:w-auto"
        disabled={channelId === null || loading}
        onClick={runTest}
      >
        {loading ? (
          <Loader2Icon className="size-4 animate-spin" />
        ) : (
          <FileSearchIcon className="size-4" />
        )}
        {loading ? t("channel.readTestRunning") : t("channel.readTestButton")}
      </Button>
      {channelId === null ? (
        <div className="text-xs text-muted-foreground">{t("channel.readTestUnsavedHint")}</div>
      ) : null}

      <ProjectDialog
        open={open}
        onOpenChange={setOpen}
        title={t("channel.readTestTitle")}
        size="lg"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={loading}
              onClick={runTest}
            >
              {t("channel.readTestRetry")}
            </Button>
            <Button type="button" size="sm" onClick={() => setOpen(false)}>
              {t("channel.readTestClose")}
            </Button>
          </div>
        }
      >
        {loading ? (
          <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
            <Loader2Icon className="size-4 animate-spin" />
            {t("channel.readTestRunning")}
          </div>
        ) : result ? (
          <div className="space-y-4">
            {result.success ? (
              <>
                <div className="flex items-start gap-2 rounded-md border border-primary/20 bg-primary/5 p-3">
                  <CheckCircle2Icon className="mt-0.5 size-4 shrink-0 text-primary" />
                  <div className="space-y-1 text-sm">
                    <div className="font-medium">{t("channel.readTestSuccess")}</div>
                    <div className="text-muted-foreground">
                      {t("channel.readTestSummary", {
                        total: result.totalScanned,
                        messages: result.messageCount,
                        events: result.eventCount,
                      })}
                    </div>
                    {result.earliestTime && result.latestTime ? (
                      <div className="text-xs text-muted-foreground">
                        {t("channel.readTestRange", {
                          earliest: result.earliestTime,
                          latest: result.latestTime,
                        })}
                      </div>
                    ) : null}
                  </div>
                </div>

                {result.truncated ? (
                  <div className="rounded-md border border-primary/20 bg-primary/5 p-3 text-xs text-muted-foreground">
                    {t("channel.readTestTruncated", { count: result.samples.length })}
                  </div>
                ) : null}

                {result.totalScanned === 0 ? (
                  <div className="rounded-md border p-4 text-sm text-muted-foreground">
                    {t("channel.readTestEmpty")}
                  </div>
                ) : (
                  <div className="max-h-80 overflow-y-auto rounded-md border divide-y">
                    {result.samples.map((sample) => (
                      <ReadTestSampleRow
                        key={`${sample.msgId}-${sample.sendTime}-${sample.eventType ?? sample.msgType}`}
                        t={t}
                        sample={sample}
                      />
                    ))}
                  </div>
                )}
              </>
            ) : (
              <div className="space-y-3">
                <div className="flex items-start gap-2 rounded-md border border-destructive/20 bg-destructive/5 p-3">
                  <XCircleIcon className="mt-0.5 size-4 shrink-0 text-destructive" />
                  <div className="space-y-1 text-sm">
                    <div className="font-medium text-destructive">
                      {t("channel.readTestFailed")}
                    </div>
                    <div>{suggestion}</div>
                  </div>
                </div>

                <div className="space-y-1.5 rounded-md border p-3 text-xs">
                  {result.stage ? (
                    <div className="flex gap-2">
                      <span className="w-20 shrink-0 text-muted-foreground">
                        {t("channel.readTestStage")}
                      </span>
                      <span>{t(READ_TEST_STAGE_KEYS[result.stage] ?? result.stage)}</span>
                    </div>
                  ) : null}
                  {result.errorCode ? (
                    <div className="flex gap-2">
                      <span className="w-20 shrink-0 text-muted-foreground">
                        {t("channel.readTestErrCode")}
                      </span>
                      <span className="font-mono">{result.errorCode}</span>
                    </div>
                  ) : null}
                  {result.errorMessage ? (
                    <div className="flex gap-2">
                      <span className="w-20 shrink-0 text-muted-foreground">
                        {t("channel.readTestErrMessage")}
                      </span>
                      <span className="min-w-0 break-words font-mono">
                        {result.errorMessage}
                      </span>
                    </div>
                  ) : null}
                </div>
              </div>
            )}
          </div>
        ) : null}
      </ProjectDialog>
    </div>
  )
}
