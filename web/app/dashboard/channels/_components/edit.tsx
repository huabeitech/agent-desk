"use client"

import { useEffect, useMemo, useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { Controller, Resolver, useForm, useWatch } from "react-hook-form"
import { z } from "zod/v4"
import { CopyIcon, ExternalLinkIcon } from "lucide-react"
import { toast } from "sonner"

import { getWidgetDemoPath } from "@/components/support-chat/demo-navigation"
import { OptionCombobox } from "@/components/option-combobox"
import { ProjectDialog } from "@/components/project-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldContent,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  type AIAgent,
  type AdminChannel,
  type CreateAdminChannelPayload,
  type WxWorkKFAccount,
  fetchAIAgentsAll,
  fetchChannel,
  fetchWxWorkKFAccounts,
  resetChannelUserTokenSecret,
} from "@/lib/api/admin"
import { useI18n } from "@/i18n/provider"

type ChannelFormDialogProps = {
  open: boolean
  saving: boolean
  itemId: number | null
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: CreateAdminChannelPayload) => Promise<void>
}

type Translate = (key: string, values?: Record<string, string | number>) => string

type WebChannelConfig = {
  title?: string
  subtitle?: string
  themeColor?: string
  position?: "left" | "right"
  width?: string
  userTokenSecret?: string
}

type WechatMPChannelConfig = {
  title?: string
  subtitle?: string
  themeColor?: string
  userTokenSecret?: string
}

function getDefaultWebChannelConfig(t: Translate): Required<WebChannelConfig> {
  return {
    title: t("channel.defaultTitleWeb"),
    subtitle: t("channel.defaultSubtitle"),
    themeColor: "#2563eb",
    position: "right",
    width: "380px",
    userTokenSecret: "",
  }
}

function createSchema(t: Translate) {
  return z
    .object({
      channelType: z.enum(["web", "wechat_mp", "wxwork_kf"], t("channel.typeRequired")),
      aiAgentId: z.string().trim().regex(/^\d+$/, t("channel.agentRequired")),
      name: z.string().trim().min(1, t("channel.nameRequired")),
      openKfId: z.string().trim(),
      widgetTitle: z.string().trim(),
      widgetSubtitle: z.string().trim(),
      widgetThemeColor: z.string().trim(),
      widgetPosition: z.enum(["left", "right"]),
      widgetWidth: z.string().trim(),
      userTokenSecret: z.string().trim(),
      remark: z.string().trim(),
    })
    .superRefine((values, ctx) => {
      if (values.channelType === "wxwork_kf" && !values.openKfId.trim()) {
        ctx.addIssue({
          code: "custom",
          path: ["openKfId"],
          message: t("channel.wxworkAccountRequired"),
        })
      }
    })
}

type EditForm = {
  channelType: "web" | "wechat_mp" | "wxwork_kf"
  aiAgentId: string
  name: string
  openKfId: string
  widgetTitle: string
  widgetSubtitle: string
  widgetThemeColor: string
  widgetPosition: "left" | "right"
  widgetWidth: string
  userTokenSecret: string
  remark: string
}

function createEmptyForm(t: Translate): EditForm {
  const defaultWebChannelConfig = getDefaultWebChannelConfig(t)
  return {
    channelType: "web",
    aiAgentId: "",
    name: "",
    openKfId: "",
    widgetTitle: defaultWebChannelConfig.title,
    widgetSubtitle: defaultWebChannelConfig.subtitle,
    widgetThemeColor: defaultWebChannelConfig.themeColor,
    widgetPosition: defaultWebChannelConfig.position,
    widgetWidth: defaultWebChannelConfig.width,
    userTokenSecret: "",
    remark: "",
  }
}

function parseOpenKfId(configJson: string): string {
  if (!configJson.trim()) {
    return ""
  }
  try {
    const parsed = JSON.parse(configJson) as { openKfId?: string }
    return typeof parsed.openKfId === "string" ? parsed.openKfId.trim() : ""
  } catch {
    return ""
  }
}

function parseWebChannelConfig(configJson: string, t: Translate): Required<WebChannelConfig> {
  const defaultWebChannelConfig = getDefaultWebChannelConfig(t)
  if (!configJson.trim()) {
    return defaultWebChannelConfig
  }
  try {
    const parsed = JSON.parse(configJson) as WebChannelConfig
    const position = parsed.position === "left" ? "left" : "right"
    return {
      title: parsed.title?.trim() || defaultWebChannelConfig.title,
      subtitle: parsed.subtitle?.trim() ?? defaultWebChannelConfig.subtitle,
      themeColor:
        parsed.themeColor?.trim() || defaultWebChannelConfig.themeColor,
      position,
      width: parsed.width?.trim() || defaultWebChannelConfig.width,
      userTokenSecret: parsed.userTokenSecret?.trim() || "",
    }
  } catch {
    return defaultWebChannelConfig
  }
}

function parseWechatMPChannelConfig(configJson: string, t: Translate): Required<WechatMPChannelConfig> {
  const defaultWebChannelConfig = getDefaultWebChannelConfig(t)
  const fallback = {
    title: t("channel.defaultTitleWechat"),
    subtitle: defaultWebChannelConfig.subtitle,
    themeColor: defaultWebChannelConfig.themeColor,
    userTokenSecret: "",
  }
  if (!configJson.trim()) {
    return fallback
  }
  try {
    const parsed = JSON.parse(configJson) as WechatMPChannelConfig
    return {
      title: parsed.title?.trim() || fallback.title,
      subtitle: parsed.subtitle?.trim() ?? fallback.subtitle,
      themeColor:
        parsed.themeColor?.trim() || defaultWebChannelConfig.themeColor,
      userTokenSecret: parsed.userTokenSecret?.trim() || "",
    }
  } catch {
    return fallback
  }
}

function buildForm(item: AdminChannel | null, t: Translate): EditForm {
  if (!item) {
    return createEmptyForm(t)
  }
  const isWechatMP = item.channelType === "wechat_mp"
  const webConfig = parseWebChannelConfig(item.configJson, t)
  const wechatConfig = isWechatMP
    ? parseWechatMPChannelConfig(item.configJson, t)
    : null
  return {
    channelType:
      item.channelType === "wxwork_kf"
        ? "wxwork_kf"
        : item.channelType === "wechat_mp"
          ? "wechat_mp"
          : "web",
    aiAgentId: item.aiAgentId > 0 ? String(item.aiAgentId) : "",
    name: item.name,
    openKfId: parseOpenKfId(item.configJson),
    widgetTitle: wechatConfig?.title ?? webConfig.title,
    widgetSubtitle: wechatConfig?.subtitle ?? webConfig.subtitle,
    widgetThemeColor: wechatConfig?.themeColor ?? webConfig.themeColor,
    widgetPosition: webConfig.position,
    widgetWidth: webConfig.width,
    userTokenSecret: wechatConfig?.userTokenSecret ?? webConfig.userTokenSecret,
    remark: item.remark || "",
  }
}

function buildPayload(form: EditForm, status: number, t: Translate): CreateAdminChannelPayload {
  const channelType = form.channelType
  const defaultWebChannelConfig = getDefaultWebChannelConfig(t)
  const webLikeConfig = {
    title:
      form.widgetTitle.trim() ||
      (channelType === "wechat_mp" ? t("channel.defaultTitleWechat") : defaultWebChannelConfig.title),
    subtitle: form.widgetSubtitle.trim(),
    themeColor:
      form.widgetThemeColor.trim() || defaultWebChannelConfig.themeColor,
    userTokenSecret: form.userTokenSecret.trim(),
  }
  const configJson =
    channelType === "wxwork_kf"
      ? JSON.stringify({ openKfId: form.openKfId.trim() })
      : channelType === "wechat_mp"
        ? JSON.stringify(webLikeConfig)
        : JSON.stringify({
            ...webLikeConfig,
            position: form.widgetPosition || defaultWebChannelConfig.position,
            width: form.widgetWidth.trim() || defaultWebChannelConfig.width,
            userTokenSecret: form.userTokenSecret.trim(),
          })
  return {
    channelType,
    aiAgentId: Number(form.aiAgentId),
    name: form.name.trim(),
    configJson,
    status,
    remark: form.remark.trim(),
  }
}

function isAgentWorkflowPublished(agent: AIAgent | undefined) {
  return Boolean(agent?.workflowPublished ?? (agent?.workflowVersionId ?? 0) > 0)
}

type ChannelFormBodyProps = Omit<ChannelFormDialogProps, "open">

export function EditDialog({
  open,
  saving,
  itemId,
  onOpenChange,
  onSubmit,
}: ChannelFormDialogProps) {
  if (!open) {
    return null
  }

  return (
    <ChannelFormBody
      key={itemId ? `edit-${itemId}` : "create"}
      itemId={itemId}
      saving={saving}
      onOpenChange={onOpenChange}
      onSubmit={onSubmit}
    />
  )
}

function ChannelFormBody({
  saving,
  itemId,
  onOpenChange,
  onSubmit,
}: ChannelFormBodyProps) {
  const t = useI18n()
  const formId = "channel-edit-form"
  const emptyForm = useMemo(() => createEmptyForm(t), [t])
  const schema = useMemo(() => createSchema(t), [t])
  const resolver = useMemo(
    () =>
      zodResolver(schema as never) as Resolver<
        z.input<typeof schema>,
        undefined,
        z.output<typeof schema>
      >,
    [schema],
  )
  const [loading, setLoading] = useState(false)
  const [aiAgents, setAIAgents] = useState<AIAgent[]>([])
  const [wxWorkKFAccounts, setWxWorkKFAccounts] = useState<WxWorkKFAccount[]>([])
  const [wxWorkKFAccountsLoading, setWxWorkKFAccountsLoading] = useState(false)
  const [wxWorkKFAccountsError, setWxWorkKFAccountsError] = useState("")
  const [channelDetail, setChannelDetail] = useState<AdminChannel | null>(null)
  const [currentStatus, setCurrentStatus] = useState(0)
  const form = useForm<
    z.input<typeof schema>,
    undefined,
    z.output<typeof schema>
  >({
    resolver,
    defaultValues: emptyForm,
  })
  const {
    control,
    handleSubmit,
    register,
    reset,
    setValue,
    formState: { errors },
  } = form
  const channelType = useWatch({ control, name: "channelType" })
  const aiAgentId = useWatch({ control, name: "aiAgentId" })
  const openKfId = useWatch({ control, name: "openKfId" })
  const userTokenSecret = useWatch({ control, name: "userTokenSecret" })

  useEffect(() => {
    async function loadAIAgents() {
      try {
        const data = await fetchAIAgentsAll({ status: 1 })
        setAIAgents(data)
      } catch (error) {
        console.error("Failed to load AI agents:", error)
      }
    }
    void loadAIAgents()
  }, [])

  useEffect(() => {
    async function loadDetail() {
      if (!itemId) {
        setCurrentStatus(0)
        setChannelDetail(null)
        reset(emptyForm)
        return
      }
      setLoading(true)
      try {
        const data = await fetchChannel(itemId)
        setChannelDetail(data)
        setCurrentStatus(data.status)
        reset(buildForm(data, t))
      } catch (error) {
        console.error("Failed to load channel:", error)
      } finally {
        setLoading(false)
      }
    }
    void loadDetail()
  }, [emptyForm, itemId, reset, t])

  useEffect(() => {
    if (
      channelType !== "wxwork_kf" ||
      wxWorkKFAccounts.length > 0 ||
      wxWorkKFAccountsLoading ||
      wxWorkKFAccountsError
    ) {
      return
    }
    async function loadWxWorkKFAccounts() {
      setWxWorkKFAccountsLoading(true)
      setWxWorkKFAccountsError("")
      try {
        const data = await fetchWxWorkKFAccounts()
        setWxWorkKFAccounts(data)
      } catch (error) {
        console.error("Failed to load WeCom KF accounts:", error)
        setWxWorkKFAccountsError(
          error instanceof Error ? error.message : t("channel.loadWxworkAccountsFailed")
        )
      } finally {
        setWxWorkKFAccountsLoading(false)
      }
    }
    void loadWxWorkKFAccounts()
  }, [
    channelType,
    wxWorkKFAccounts.length,
    wxWorkKFAccountsError,
    wxWorkKFAccountsLoading,
    t,
  ])

  const selectedAIAgent = aiAgents.find((item) => String(item.id) === aiAgentId)
  const availableAIAgents = aiAgents.filter(
    (item) => isAgentWorkflowPublished(item) || String(item.id) === aiAgentId
  )
  const aiAgentOptions = availableAIAgents.map((item) => ({
    value: String(item.id),
    label: isAgentWorkflowPublished(item)
      ? `${item.name} · 当前生效 #${item.workflowVersionId}`
      : `${item.name} · 未发布`,
  }))
  const wxWorkKFAccountOptions = wxWorkKFAccounts.map((item) => ({
    value: item.openKfId,
    label: item.name ? `${item.name} (${item.openKfId})` : item.openKfId,
  }))
  const channelTypeOptions = [
    { value: "web", label: t("channel.typeWeb") },
    { value: "wechat_mp", label: t("channel.typeWechatMp") },
    { value: "wxwork_kf", label: t("channel.typeWxworkKf") },
  ] as const
  const widgetPositionOptions = [
    { value: "right", label: t("channel.positionRight") },
    { value: "left", label: t("channel.positionLeft") },
  ] as const
  if (
    channelType === "wxwork_kf" &&
    openKfId &&
    !wxWorkKFAccountOptions.some((item) => item.value === openKfId)
  ) {
    wxWorkKFAccountOptions.unshift({
      value: openKfId,
      label: openKfId,
    })
  }

  async function onFormSubmit(values: EditForm) {
    const selected = aiAgents.find((item) => String(item.id) === values.aiAgentId)
    if (!isAgentWorkflowPublished(selected)) {
      toast.error("该 Agent 尚未发布流程，不能绑定渠道")
      return
    }
    await onSubmit(buildPayload(values, currentStatus, t))
  }

  async function handleResetUserTokenSecret() {
    if (!itemId) {
      return
    }
    if (!window.confirm(t("channel.resetSecretConfirm"))) {
      return
    }
    try {
      const result = await resetChannelUserTokenSecret(itemId)
      setValue("userTokenSecret", result.userTokenSecret, {
        shouldDirty: true,
      })
      if (channelDetail) {
        const parsed = JSON.parse(channelDetail.configJson || "{}") as Record<string, unknown>
        parsed.userTokenSecret = result.userTokenSecret
        setChannelDetail({
          ...channelDetail,
          configJson: JSON.stringify(parsed),
        })
      }
      toast.success(t("channel.resetSecretSuccess"))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("channel.resetSecretFailed"))
    }
  }

  async function copyUserTokenSecret() {
    if (!userTokenSecret) {
      return
    }
    try {
      await navigator.clipboard.writeText(userTokenSecret)
      toast.success(t("channel.copySecretSuccess"))
    } catch {
      toast.error(t("channel.copyFailed"))
    }
  }

  return (
    <ProjectDialog
      open={true}
      onOpenChange={onOpenChange}
      title={itemId ? t("channel.editTitle") : t("channel.createTitle")}
      size="lg"
      allowFullscreen
      footer={
        <>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("channel.cancel")}
          </Button>
          <Button type="submit" form={formId} disabled={saving || loading}>
            {saving ? t("channel.saving") : t("channel.save")}
          </Button>
        </>
      }
    >
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <div className="text-muted-foreground">{t("channel.loadingDetail")}</div>
        </div>
      ) : (
        <form id={formId} onSubmit={handleSubmit(onFormSubmit)} className="space-y-5">
          <div className="grid grid-cols-1 gap-4">
            <Field data-invalid={!!errors.name}>
              <FieldLabel htmlFor="channel-name">{t("channel.name")}</FieldLabel>
              <FieldContent>
                <Input id="channel-name" {...register("name")} />
                <FieldError errors={[errors.name]} />
              </FieldContent>
            </Field>

            <Field data-invalid={!!errors.aiAgentId}>
              <FieldLabel>{t("channel.columnAgent")}</FieldLabel>
              <FieldContent>
                <Controller
                  control={control}
                  name="aiAgentId"
                  render={({ field }) => (
                    <OptionCombobox
                      value={field.value}
                      options={aiAgentOptions}
                      placeholder={t("channel.agentRequired")}
                      searchPlaceholder={t("channel.searchAiAgent")}
                      emptyText={t("channel.emptyAiAgent")}
                      onChange={field.onChange}
                    />
                  )}
                />
                {selectedAIAgent && !isAgentWorkflowPublished(selectedAIAgent) ? (
                  <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
                    该 Agent 尚未发布流程，AI 不会自动回复。请先在 Agent 配置中发布流程版本。
                  </div>
                ) : null}
                {selectedAIAgent && isAgentWorkflowPublished(selectedAIAgent) ? (
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Badge variant="secondary">
                      {selectedAIAgent.workflowStateText || "已发布"}
                    </Badge>
                    <span>当前生效版本 #{selectedAIAgent.workflowVersionId}</span>
                  </div>
                ) : null}
                <FieldError errors={[errors.aiAgentId]} />
              </FieldContent>
            </Field>

            <Field data-invalid={!!errors.channelType}>
              <FieldLabel>{t("channel.channelType")}</FieldLabel>
              <FieldContent>
                <Controller
                  control={control}
                  name="channelType"
                  render={({ field }) => (
                    <OptionCombobox
                      value={field.value}
                      options={[...channelTypeOptions]}
                      placeholder={t("channel.selectChannelType")}
                      searchPlaceholder={t("channel.searchChannelType")}
                      emptyText={t("channel.emptyChannelType")}
                      onChange={field.onChange}
                    />
                  )}
                />
                <FieldError errors={[errors.channelType]} />
              </FieldContent>
            </Field>
          </div>

          <div className="space-y-4 rounded-md border p-4">
            <div>
              <div className="text-sm font-medium">{t("channel.configTitle")}</div>
              <div className="text-xs text-muted-foreground">
                {channelType === "wxwork_kf"
                  ? t("channel.configWxworkDescription")
                  : channelType === "wechat_mp"
                    ? t("channel.configWechatDescription")
                    : t("channel.configWebDescription")}
              </div>
            </div>

            {channelType === "wxwork_kf" ? (
              <Field data-invalid={!!errors.openKfId}>
                <FieldLabel>{t("channel.wxworkAccount")}</FieldLabel>
                <FieldContent>
                  <Controller
                    control={control}
                    name="openKfId"
                    render={({ field }) => (
                      <OptionCombobox
                        value={field.value}
                        options={wxWorkKFAccountOptions}
                        placeholder={
                          wxWorkKFAccountsLoading ? t("channel.loadingWxworkAccount") : t("channel.selectWxworkAccount")
                        }
                        searchPlaceholder={t("channel.searchWxworkAccount")}
                        emptyText={
                          wxWorkKFAccountsError || t("channel.emptyWxworkAccount")
                        }
                        disabled={wxWorkKFAccountsLoading}
                        onChange={field.onChange}
                      />
                    )}
                  />
                  <FieldError errors={[errors.openKfId]} />
                </FieldContent>
              </Field>
            ) : null}

            {channelType === "web" || channelType === "wechat_mp" ? (
              <>
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field data-invalid={!!errors.widgetTitle}>
                    <FieldLabel htmlFor="channel-widget-title">{t("channel.widgetTitle")}</FieldLabel>
                    <FieldContent>
                      <Input id="channel-widget-title" {...register("widgetTitle")} />
                      <FieldError errors={[errors.widgetTitle]} />
                    </FieldContent>
                  </Field>

                  <Field data-invalid={!!errors.widgetSubtitle}>
                    <FieldLabel htmlFor="channel-widget-subtitle">{t("channel.widgetSubtitle")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-widget-subtitle"
                        {...register("widgetSubtitle")}
                      />
                      <FieldError errors={[errors.widgetSubtitle]} />
                    </FieldContent>
                  </Field>

                  <Field data-invalid={!!errors.widgetThemeColor}>
                    <FieldLabel htmlFor="channel-widget-theme-color">{t("channel.themeColor")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-widget-theme-color"
                        placeholder="#2563eb"
                        {...register("widgetThemeColor")}
                      />
                      <FieldError errors={[errors.widgetThemeColor]} />
                    </FieldContent>
                  </Field>

                  {channelType === "web" ? (
                    <>
                      <Field data-invalid={!!errors.widgetPosition}>
                        <FieldLabel>{t("channel.mountPosition")}</FieldLabel>
                        <FieldContent>
                          <Controller
                            control={control}
                            name="widgetPosition"
                            render={({ field }) => (
                              <OptionCombobox
                                value={field.value}
                                options={[...widgetPositionOptions]}
                                placeholder={t("channel.selectMountPosition")}
                                searchPlaceholder={t("channel.searchMountPosition")}
                                emptyText={t("channel.emptyMountPosition")}
                                onChange={field.onChange}
                              />
                            )}
                          />
                          <FieldError errors={[errors.widgetPosition]} />
                        </FieldContent>
                      </Field>

                      <Field data-invalid={!!errors.widgetWidth}>
                        <FieldLabel htmlFor="channel-widget-width">{t("channel.widgetWidth")}</FieldLabel>
                        <FieldContent>
                          <Input
                            id="channel-widget-width"
                            placeholder="380px"
                            {...register("widgetWidth")}
                          />
                          <FieldError errors={[errors.widgetWidth]} />
                        </FieldContent>
                      </Field>
                    </>
                  ) : null}
                </div>
                <div className="space-y-3 rounded-md border p-3">
                  <div>
                    <div className="text-sm font-medium">{t("channel.userJwtSecret")}</div>
                    <div className="text-xs text-muted-foreground">
                      {t("channel.userJwtSecretDescription")}
                    </div>
                  </div>
                  {!itemId ? (
                    <div className="rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground">
                      {t("channel.secretAfterSave")}
                    </div>
                  ) : (
                    <Field data-invalid={!!errors.userTokenSecret}>
                      <FieldLabel htmlFor="channel-user-token-secret">Secret</FieldLabel>
                      <FieldContent>
                        <div className="flex flex-col gap-2 sm:flex-row">
                          <Input
                            id="channel-user-token-secret"
                            readOnly
                            className="font-mono text-xs"
                            {...register("userTokenSecret")}
                          />
                          <div className="flex gap-2">
                            <Button
                              type="button"
                              variant="outline"
                              onClick={copyUserTokenSecret}
                              disabled={!userTokenSecret}
                            >
                              <CopyIcon className="size-4" />
                              {t("channel.copy")}
                            </Button>
                            <Button
                              type="button"
                              variant="outline"
                              onClick={() => void handleResetUserTokenSecret()}
                            >
                              {t("channel.reset")}
                            </Button>
                          </div>
                        </div>
                        <FieldError errors={[errors.userTokenSecret]} />
                      </FieldContent>
                    </Field>
                  )}
                </div>
                {channelType === "wechat_mp" ? (
                  <WechatMPAccessGuide channelId={channelDetail?.channelId || ""} />
                ) : (
                  <WebAccessGuide channelId={channelDetail?.channelId || ""} />
                )}
              </>
            ) : null}
          </div>

          <Field data-invalid={!!errors.remark}>
            <FieldLabel htmlFor="channel-remark">{t("channel.remark")}</FieldLabel>
            <FieldContent>
              <Textarea id="channel-remark" rows={3} {...register("remark")} />
              <FieldError errors={[errors.remark]} />
            </FieldContent>
          </Field>
        </form>
      )}
    </ProjectDialog>
  )
}

function WebAccessGuide({ channelId }: { channelId: string }) {
  const t = useI18n()
  const [origin] = useState(() =>
    typeof window === "undefined" ? "" : window.location.origin
  )

  const accessUrl = useMemo(() => {
    if (!origin || !channelId) {
      return ""
    }
    const url = new URL("/support/chat/", origin)
    url.searchParams.set("channelId", channelId)
    return url.toString()
  }, [channelId, origin])

  const testUrl = useMemo(() => {
    if (!origin || !channelId) {
      return ""
    }
    const url = new URL(getWidgetDemoPath(), origin)
    url.searchParams.set("channelId", channelId)
    return url.toString()
  }, [channelId, origin])

  const snippet = useMemo(() => {
    if (!origin || !channelId) {
      return ""
    }
    return `<script>
  window.AgentDeskConfig = {
    channelId: "${channelId}"
  };
</script>
<script async src="${origin}/sdk/agent-desk-sdk.min.js"></script>`
  }, [channelId, origin])

  async function copyText(text: string, successMessage: string) {
    if (!text) {
      return
    }
    try {
      await navigator.clipboard.writeText(text)
      toast.success(successMessage)
    } catch {
      toast.error(t("channel.copyFailed"))
    }
  }

  return (
    <div className="space-y-4 border-t pt-4">
      <div>
        <div className="text-sm font-medium">{t("channel.webAccessInfo")}</div>
        <div className="text-xs text-muted-foreground">
          {channelId
            ? t("channel.webAccessReady")
            : t("channel.webAccessPending")}
        </div>
      </div>

      {!channelId ? (
        <div className="rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground">
          {t("channel.newChannelPending")}
        </div>
      ) : (
        <div className="space-y-4">
          <div className="space-y-2">
            <div className="text-xs font-medium text-muted-foreground">{t("channel.directAccessUrl")}</div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Input readOnly value={accessUrl} className="font-mono text-xs" />
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  title={t("channel.copyLink")}
                  onClick={() => copyText(accessUrl, t("channel.copiedAccessLink"))}
                >
                  <CopyIcon className="size-4" />
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  title={t("channel.openLink")}
                  onClick={() => window.open(accessUrl, "_blank", "noopener,noreferrer")}
                >
                  <ExternalLinkIcon className="size-4" />
                </Button>
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <div className="text-xs font-medium text-muted-foreground">
                {t("channel.embeddedSnippet")}
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => copyText(snippet, t("channel.copiedSnippet"))}
              >
                <CopyIcon className="size-4" />
                {t("channel.copyCode")}
              </Button>
            </div>
            <pre className="max-h-48 overflow-auto rounded-md bg-muted p-3 text-xs leading-5">
              <code>{snippet}</code>
            </pre>
          </div>

          <div className="flex flex-col gap-2 rounded-md bg-muted px-3 py-3 text-xs text-muted-foreground">
            <div className="font-medium text-foreground">{t("channel.accessGuide")}</div>
            <div>{t("channel.webGuide1")}</div>
            <div>{t("channel.webGuide2")}</div>
            <div>{t("channel.webGuide3")}</div>
            <div>{t("channel.webGuide4")}</div>
            <div className="pt-1">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => window.open(testUrl, "_blank", "noopener,noreferrer")}
              >
                <ExternalLinkIcon className="size-4" />
                {t("channel.openTestPage")}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function WechatMPAccessGuide({ channelId }: { channelId: string }) {
  const t = useI18n()
  const [origin] = useState(() =>
    typeof window === "undefined" ? "" : window.location.origin
  )

  const menuUrl = useMemo(() => {
    if (!origin || !channelId) {
      return ""
    }
    const url = new URL("/support/chat/", origin)
    url.searchParams.set("channelId", channelId)
    return url.toString()
  }, [channelId, origin])

  async function copyText(text: string) {
    if (!text) {
      return
    }
    try {
      await navigator.clipboard.writeText(text)
      toast.success(t("channel.copiedWechatMenuUrl"))
    } catch {
      toast.error(t("channel.copyFailed"))
    }
  }

  return (
    <div className="space-y-4 border-t pt-4">
      <div>
        <div className="text-sm font-medium">{t("channel.wechatAccessInfo")}</div>
        <div className="text-xs text-muted-foreground">
          {channelId
            ? t("channel.wechatAccessReady")
            : t("channel.wechatAccessPending")}
        </div>
      </div>

      {!channelId ? (
        <div className="rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground">
          {t("channel.newChannelPending")}
        </div>
      ) : (
        <div className="space-y-4">
          <div className="space-y-2">
            <div className="text-xs font-medium text-muted-foreground">
              {t("channel.wechatMenuUrl")}
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Input readOnly value={menuUrl} className="font-mono text-xs" />
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  title={t("channel.copyLink")}
                  onClick={() => copyText(menuUrl)}
                >
                  <CopyIcon className="size-4" />
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  title={t("channel.openLink")}
                  onClick={() => window.open(menuUrl, "_blank", "noopener,noreferrer")}
                >
                  <ExternalLinkIcon className="size-4" />
                </Button>
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-2 rounded-md bg-muted px-3 py-3 text-xs text-muted-foreground">
            <div className="font-medium text-foreground">{t("channel.accessGuide")}</div>
            <div>{t("channel.webGuide1")}</div>
            <div>{t("channel.wechatGuide2")}</div>
            <div>{t("channel.wechatGuide3")}</div>
          </div>
        </div>
      )}
    </div>
  )
}
