"use client"

import { useEffect, useMemo, useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { Controller, Resolver, useForm, useWatch } from "react-hook-form"
import { z } from "zod/v4"
import { CopyIcon, ExternalLinkIcon, RotateCcwIcon } from "lucide-react"
import { toast } from "sonner"

import { getWidgetDemoPath } from "@/components/support-chat/demo-navigation"
import { OptionCombobox } from "@/components/option-combobox"
import { ProjectDialog } from "@/components/project-dialog"
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
	rollbackChannelAIAgentRollout,
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

type TelegramChannelConfig = {
  botToken?: string
  botUsername?: string
  webhookSecret?: string
}

type ZaloOAChannelConfig = {
  appId?: string
  oaId?: string
  secretKey?: string
  accessToken?: string
  refreshToken?: string
  webhookSecret?: string
}

type EmailChannelConfig = {
  emailAddress?: string
  senderName?: string
  provider?: string
  apiKey?: string
  smtpHost?: string
  smtpPort?: number
  smtpUser?: string
  smtpPassword?: string
  webhookSecret?: string
}

type DiscordChannelConfig = {
  guildId?: string
  guildName?: string
  botToken?: string
  webhookSecret?: string
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
      channelType: z.enum(["web", "wechat_mp", "wxwork_kf", "telegram", "zalo_oa", "email", "discord"], t("channel.typeRequired")),
      aiAgentId: z.string().trim().regex(/^\d+$/, t("channel.agentRequired")),
		aiAgentRolloutPercent: z.coerce.number().int().min(1).max(100),
      name: z.string().trim().min(1, t("channel.nameRequired")),
      openKfId: z.string().trim(),
      botToken: z.string().trim(),
      botUsername: z.string().trim(),
      webhookSecret: z.string().trim(),
      zaloAppId: z.string().trim(),
      zaloOaId: z.string().trim(),
      zaloAccessToken: z.string().trim(),
      zaloSecretKey: z.string().trim(),
      emailAddress: z.string().trim(),
      senderName: z.string().trim(),
      emailProvider: z.string().trim(),
      emailApiKey: z.string().trim(),
      smtpHost: z.string().trim(),
      smtpPort: z.coerce.number().int().optional(),
      smtpUser: z.string().trim(),
      smtpPassword: z.string().trim(),
      discordGuildId: z.string().trim(),
      discordGuildName: z.string().trim(),
      discordBotToken: z.string().trim(),
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
      if (values.channelType === "email" && !values.emailAddress.trim()) {
        ctx.addIssue({
          code: "custom",
          path: ["emailAddress"],
          message: "Email address is required (e.g. help@crove.com)",
        })
      }
      if (values.channelType === "telegram" && !values.botToken.trim()) {
        ctx.addIssue({
          code: "custom",
          path: ["botToken"],
          message: "Telegram Bot Token is required",
        })
      }
      if (values.channelType === "zalo_oa" && !values.zaloAccessToken.trim()) {
        ctx.addIssue({
          code: "custom",
          path: ["zaloAccessToken"],
          message: "Zalo OA Access Token is required",
        })
      }
    })
}

type EditForm = {
  channelType: "web" | "wechat_mp" | "wxwork_kf" | "telegram" | "zalo_oa" | "email" | "discord"
  aiAgentId: string
	aiAgentRolloutPercent: number
  name: string
  openKfId: string
  botToken: string
  botUsername: string
  webhookSecret: string
  zaloAppId: string
  zaloOaId: string
  zaloAccessToken: string
  zaloSecretKey: string
  emailAddress: string
  senderName: string
  emailProvider: string
  emailApiKey: string
  smtpHost: string
  smtpPort?: number
  smtpUser: string
  smtpPassword: string
  discordGuildId: string
  discordGuildName: string
  discordBotToken: string
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
		aiAgentRolloutPercent: 100,
    name: "",
    openKfId: "",
    botToken: "",
    botUsername: "",
    webhookSecret: "",
    zaloAppId: "",
    zaloOaId: "",
    zaloAccessToken: "",
    zaloSecretKey: "",
    emailAddress: "help@crove.com",
    senderName: "Crove Desk Support",
    emailProvider: "brevo",
    emailApiKey: "",
    smtpHost: "",
    smtpPort: 587,
    smtpUser: "",
    smtpPassword: "",
    discordGuildId: "",
    discordGuildName: "",
    discordBotToken: "",
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

function parseTelegramChannelConfig(configJson: string): TelegramChannelConfig {
  if (!configJson.trim()) return {}
  try {
    const parsed = JSON.parse(configJson) as TelegramChannelConfig
    return {
      botToken: parsed.botToken?.trim() || "",
      botUsername: parsed.botUsername?.trim() || "",
      webhookSecret: parsed.webhookSecret?.trim() || "",
    }
  } catch {
    return {}
  }
}

function parseZaloOAChannelConfig(configJson: string): ZaloOAChannelConfig {
  if (!configJson.trim()) return {}
  try {
    const parsed = JSON.parse(configJson) as ZaloOAChannelConfig
    return {
      appId: parsed.appId?.trim() || "",
      oaId: parsed.oaId?.trim() || "",
      secretKey: parsed.secretKey?.trim() || "",
      accessToken: parsed.accessToken?.trim() || "",
      refreshToken: parsed.refreshToken?.trim() || "",
      webhookSecret: parsed.webhookSecret?.trim() || "",
    }
  } catch {
    return {}
  }
}

function parseEmailChannelConfig(configJson: string): EmailChannelConfig {
  if (!configJson.trim()) return {}
  try {
    const parsed = JSON.parse(configJson) as EmailChannelConfig
    return {
      emailAddress: parsed.emailAddress?.trim() || "",
      senderName: parsed.senderName?.trim() || "",
      provider: parsed.provider?.trim() || "brevo",
      apiKey: parsed.apiKey?.trim() || "",
      smtpHost: parsed.smtpHost?.trim() || "",
      smtpPort: parsed.smtpPort || 587,
      smtpUser: parsed.smtpUser?.trim() || "",
      smtpPassword: parsed.smtpPassword?.trim() || "",
      webhookSecret: parsed.webhookSecret?.trim() || "",
    }
  } catch {
    return {}
  }
}

function parseDiscordChannelConfig(configJson: string): DiscordChannelConfig {
  if (!configJson.trim()) return {}
  try {
    const parsed = JSON.parse(configJson) as DiscordChannelConfig
    return {
      guildId: parsed.guildId?.trim() || "",
      guildName: parsed.guildName?.trim() || "",
      botToken: parsed.botToken?.trim() || "",
      webhookSecret: parsed.webhookSecret?.trim() || "",
    }
  } catch {
    return {}
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
  const isTelegram = item.channelType === "telegram"
  const isZaloOA = item.channelType === "zalo_oa"
  const isEmail = item.channelType === "email"
  const isDiscord = item.channelType === "discord"
  const webConfig = parseWebChannelConfig(item.configJson, t)
  const wechatConfig = isWechatMP
    ? parseWechatMPChannelConfig(item.configJson, t)
    : null
  const telegramConfig = isTelegram
    ? parseTelegramChannelConfig(item.configJson)
    : null
  const zaloConfig = isZaloOA
    ? parseZaloOAChannelConfig(item.configJson)
    : null
  const emailConfig = isEmail
    ? parseEmailChannelConfig(item.configJson)
    : null
  const discordConfig = isDiscord
    ? parseDiscordChannelConfig(item.configJson)
    : null
  return {
    channelType:
      item.channelType === "wxwork_kf"
        ? "wxwork_kf"
        : item.channelType === "telegram"
          ? "telegram"
          : item.channelType === "zalo_oa"
            ? "zalo_oa"
            : item.channelType === "email"
              ? "email"
              : item.channelType === "discord"
                ? "discord"
                : item.channelType === "wechat_mp"
                ? "wechat_mp"
                : "web",
    aiAgentId: item.aiAgentId > 0 ? String(item.aiAgentId) : "",
		aiAgentRolloutPercent: item.aiAgentRolloutPercent || 100,
    name: item.name,
    openKfId: parseOpenKfId(item.configJson),
    botToken: telegramConfig?.botToken ?? "",
    botUsername: telegramConfig?.botUsername ?? "",
    webhookSecret:
      telegramConfig?.webhookSecret ??
      zaloConfig?.webhookSecret ??
      emailConfig?.webhookSecret ??
      discordConfig?.webhookSecret ??
      "",
    zaloAppId: zaloConfig?.appId ?? "",
    zaloOaId: zaloConfig?.oaId ?? "",
    zaloAccessToken: zaloConfig?.accessToken ?? "",
    zaloSecretKey: zaloConfig?.secretKey ?? "",
    emailAddress: emailConfig?.emailAddress || "help@crove.com",
    senderName: emailConfig?.senderName || "Crove Desk Support",
    emailProvider: emailConfig?.provider || "brevo",
    emailApiKey: emailConfig?.apiKey || "",
    smtpHost: emailConfig?.smtpHost || "",
    smtpPort: emailConfig?.smtpPort || 587,
    smtpUser: emailConfig?.smtpUser || "",
    smtpPassword: emailConfig?.smtpPassword || "",
    discordGuildId: discordConfig?.guildId ?? "",
    discordGuildName: discordConfig?.guildName ?? "",
    discordBotToken: discordConfig?.botToken ?? "",
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
      : channelType === "email"
        ? JSON.stringify({
            emailAddress: form.emailAddress.trim(),
            senderName: form.senderName.trim(),
            provider: form.emailProvider.trim(),
            apiKey: form.emailApiKey.trim(),
            smtpHost: form.smtpHost.trim(),
            smtpPort: form.smtpPort || 587,
            smtpUser: form.smtpUser.trim(),
            smtpPassword: form.smtpPassword.trim(),
            webhookSecret: form.webhookSecret.trim(),
          })
        : channelType === "telegram"
          ? JSON.stringify({
              botToken: form.botToken.trim(),
              botUsername: form.botUsername.trim(),
              webhookSecret: form.webhookSecret.trim(),
            })
          : channelType === "zalo_oa"
            ? JSON.stringify({
                appId: form.zaloAppId.trim(),
                oaId: form.zaloOaId.trim(),
                accessToken: form.zaloAccessToken.trim(),
                secretKey: form.zaloSecretKey.trim(),
                webhookSecret: form.webhookSecret.trim(),
              })

          : channelType === "discord"
            ? JSON.stringify({
                guildId: form.discordGuildId.trim(),
                guildName: form.discordGuildName.trim(),
                botToken: form.discordBotToken.trim(),
                webhookSecret: form.webhookSecret.trim(),
              })
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
		aiAgentRolloutPercent: form.aiAgentRolloutPercent,
    name: form.name.trim(),
    configJson,
    status,
    remark: form.remark.trim(),
  }
}

function isAgentChannelBindable(agent: AIAgent | undefined) {
  return Boolean(agent && agent.publishedRevisionId > 0)
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
	const [rollingBackRollout, setRollingBackRollout] = useState(false)
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
  const emailProvider = useWatch({ control, name: "emailProvider" })
  const emailAddressValue = useWatch({ control, name: "emailAddress" })
  const nameValue = useWatch({ control, name: "name" })
	const previousRolloutPercent = channelDetail?.previousAiAgentRolloutPercent ?? 0

  const forwardingAddressPreview = useMemo(() => {
    const raw = (emailAddressValue || "").trim().toLowerCase()
    if (raw.endsWith(".crove.io") || raw.endsWith(".on.crove.email") || raw.endsWith(".crove-mail.com")) {
      return raw
    }
    const cleanName = (nameValue || "").trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "") || "org"
    return `help@${cleanName}.crove.io`
  }, [emailAddressValue, nameValue])

	async function rollbackRolloutPercent() {
		if (!channelDetail || previousRolloutPercent < 1) return
		setRollingBackRollout(true)
		try {
			await rollbackChannelAIAgentRollout(channelDetail.id)
			setValue("aiAgentRolloutPercent", previousRolloutPercent)
			setChannelDetail({
				...channelDetail,
				aiAgentRolloutPercent: previousRolloutPercent,
				previousAiAgentRolloutPercent: channelDetail.aiAgentRolloutPercent,
			})
			toast.success(t("aiAgent.channelRolloutRestored"))
		} catch (error) {
			toast.error(error instanceof Error ? error.message : t("aiAgent.channelRolloutRestoreFailed"))
		} finally {
			setRollingBackRollout(false)
		}
	}

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
  const aiAgentOptions = aiAgents.map((item) => ({
    value: String(item.id),
    label: isAgentChannelBindable(item) ? item.name : `${item.name} · ${t("aiAgent.agentNotPublishedShort")}`,
    disabled: !isAgentChannelBindable(item),
  }))
  const wxWorkKFAccountOptions = wxWorkKFAccounts.map((item) => ({
    value: item.openKfId,
    label: item.name ? `${item.name} (${item.openKfId})` : item.openKfId,
  }))
  const channelTypeOptions = [
    { value: "web", label: t("channel.typeWeb") },
    { value: "email", label: t("channel.typeEmail") },
    { value: "telegram", label: t("channel.typeTelegram") },
    { value: "zalo_oa", label: t("channel.typeZaloOa") },
    { value: "discord", label: t("channel.typeDiscord") },
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
    if (!isAgentChannelBindable(selected)) {
      toast.error(t("aiAgent.agentNotPublishedWarning"))
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
      size="xl"
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
                {selectedAIAgent && !isAgentChannelBindable(selectedAIAgent) ? (
                  <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
                    {t("aiAgent.agentNotPublishedWarning")}
                  </div>
                ) : null}
                <FieldError errors={[errors.aiAgentId]} />
              </FieldContent>
            </Field>

            <Field data-invalid={!!errors.aiAgentRolloutPercent}>
              <FieldLabel htmlFor="channel-ai-agent-rollout">{t("aiAgent.channelRolloutTitle")}</FieldLabel>
              <FieldContent>
                <div className="flex items-center gap-2">
                  <Input id="channel-ai-agent-rollout" type="number" min={1} max={100} step={1} {...register("aiAgentRolloutPercent")} />
                  {previousRolloutPercent > 0 ? (
                    <Button type="button" variant="outline" size="sm" disabled={saving || rollingBackRollout} onClick={rollbackRolloutPercent}>
                      <RotateCcwIcon />
                      {t("aiAgent.channelRolloutRestoreBtn", { percent: String(previousRolloutPercent) })}
                    </Button>
                  ) : null}
                </div>
                <FieldError errors={[errors.aiAgentRolloutPercent]} />
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
                {channelType === "email"
                  ? t("channel.configEmailDescription")
                  : channelType === "wxwork_kf"
                    ? t("channel.configWxworkDescription")
                    : channelType === "wechat_mp"
                      ? t("channel.configWechatDescription")
                      : t("channel.configWebDescription")}
              </div>
            </div>

            {channelType === "email" ? (
              <div className="space-y-4">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field data-invalid={!!errors.emailAddress}>
                    <FieldLabel htmlFor="channel-email-address">{t("channel.emailAddress")} *</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-email-address"
                        type="email"
                        placeholder="help@crove.com"
                        {...register("emailAddress")}
                      />
                      <FieldError errors={[errors.emailAddress]} />
                    </FieldContent>
                  </Field>

                  <Field data-invalid={!!errors.senderName}>
                    <FieldLabel htmlFor="channel-sender-name">{t("channel.senderName")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-sender-name"
                        placeholder="Crove Desk Support"
                        {...register("senderName")}
                      />
                      <FieldError errors={[errors.senderName]} />
                    </FieldContent>
                  </Field>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field data-invalid={!!errors.emailProvider}>
                    <FieldLabel htmlFor="channel-email-provider">{t("channel.emailProvider")}</FieldLabel>
                    <FieldContent>
                      <select
                        id="channel-email-provider"
                        className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                        {...register("emailProvider")}
                      >
                        <option value="default">{t("channel.emailProviderDefault")}</option>
                        <option value="smtp">{t("channel.emailProviderSmtp")}</option>
                        <option value="brevo">{t("channel.emailProviderBrevo")}</option>
                        <option value="sendgrid">{t("channel.emailProviderSendGrid")}</option>
                        <option value="resend">{t("channel.emailProviderResend")}</option>
                        <option value="postmark">{t("channel.emailProviderPostmark")}</option>
                        <option value="mailgun">{t("channel.emailProviderMailgun")}</option>
                      </select>
                      <FieldError errors={[errors.emailProvider]} />
                    </FieldContent>
                  </Field>

                  <Field data-invalid={!!errors.webhookSecret}>
                    <FieldLabel htmlFor="channel-email-webhook-secret">{t("channel.webhookSecret")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-email-webhook-secret"
                        placeholder="Secret for Inbound Webhook"
                        {...register("webhookSecret")}
                      />
                      <FieldError errors={[errors.webhookSecret]} />
                    </FieldContent>
                  </Field>
                </div>

                {emailProvider === "brevo" || emailProvider === "sendgrid" || emailProvider === "resend" || emailProvider === "postmark" || emailProvider === "mailgun" ? (
                  <Field data-invalid={!!errors.emailApiKey}>
                    <FieldLabel htmlFor="channel-email-apikey">{t("channel.emailApiKey")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-email-apikey"
                        type="password"
                        placeholder="API Key / Server Token"
                        {...register("emailApiKey")}
                      />
                      <FieldError errors={[errors.emailApiKey]} />
                    </FieldContent>
                  </Field>
                ) : emailProvider === "smtp" ? (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                      <div className="sm:col-span-2">
                        <Field data-invalid={!!errors.smtpHost}>
                          <FieldLabel htmlFor="channel-smtp-host">SMTP Host</FieldLabel>
                          <FieldContent>
                            <Input
                              id="channel-smtp-host"
                              placeholder="smtp.example.com"
                              {...register("smtpHost")}
                            />
                            <FieldError errors={[errors.smtpHost]} />
                          </FieldContent>
                        </Field>
                      </div>
                      <Field data-invalid={!!errors.smtpPort}>
                        <FieldLabel htmlFor="channel-smtp-port">SMTP Port</FieldLabel>
                        <FieldContent>
                          <Input
                            id="channel-smtp-port"
                            type="number"
                            placeholder="587"
                            {...register("smtpPort")}
                          />
                          <FieldError errors={[errors.smtpPort]} />
                        </FieldContent>
                      </Field>
                    </div>
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                      <Field data-invalid={!!errors.smtpUser}>
                        <FieldLabel htmlFor="channel-smtp-user">SMTP Username</FieldLabel>
                        <FieldContent>
                          <Input
                            id="channel-smtp-user"
                            placeholder="user@example.com"
                            {...register("smtpUser")}
                          />
                          <FieldError errors={[errors.smtpUser]} />
                        </FieldContent>
                      </Field>
                      <Field data-invalid={!!errors.smtpPassword}>
                        <FieldLabel htmlFor="channel-smtp-password">SMTP Password</FieldLabel>
                        <FieldContent>
                          <Input
                            id="channel-smtp-password"
                            type="password"
                            placeholder="••••••••"
                            {...register("smtpPassword")}
                          />
                          <FieldError errors={[errors.smtpPassword]} />
                        </FieldContent>
                      </Field>
                    </div>
                  </div>
                ) : null}

                <div className="rounded-md border border-primary/20 bg-primary/5 p-3.5 text-xs text-muted-foreground space-y-2.5">
                  <div className="font-medium text-sm text-foreground">{t("channel.emailAutoConnectTitle")}</div>
                  <div className="leading-relaxed">{t("channel.emailAutoConnectDescription")}</div>
                  <div className="flex flex-col gap-1 pt-1">
                    <span className="font-medium text-foreground">{t("channel.forwardingAddressLabel")}</span>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 rounded bg-background px-2.5 py-1.5 font-mono text-[12px] font-semibold text-primary border">
                        {forwardingAddressPreview}
                      </code>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={async () => {
                          try {
                            await navigator.clipboard.writeText(forwardingAddressPreview)
                            toast.success(t("channel.copySecretSuccess"))
                          } catch {
                            toast.error(t("channel.copyFailed"))
                          }
                        }}
                      >
                        <CopyIcon className="size-3.5 mr-1" />
                        {t("channel.copy")}
                      </Button>
                    </div>
                  </div>
                  <div className="font-mono text-[11px] text-muted-foreground pt-0.5">
                    {t("channel.inboundWebhookUrl")}: /api/third/email/webhook
                  </div>
                </div>
              </div>
            ) : null}

            {channelType === "zalo_oa" ? (
              <div className="space-y-4">
                <Field data-invalid={!!errors.zaloAccessToken}>
                  <FieldLabel htmlFor="channel-zalo-token">{t("channel.zaloAccessToken")} *</FieldLabel>
                  <FieldContent>
                    <Input
                      id="channel-zalo-token"
                      type="password"
                      placeholder="Enter Zalo OA Access Token"
                      {...register("zaloAccessToken")}
                    />
                    <FieldError errors={[errors.zaloAccessToken]} />
                  </FieldContent>
                </Field>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field data-invalid={!!errors.zaloOaId}>
                    <FieldLabel htmlFor="channel-zalo-oaid">{t("channel.zaloOaId")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-zalo-oaid"
                        placeholder="e.g. 1234567890"
                        {...register("zaloOaId")}
                      />
                      <FieldError errors={[errors.zaloOaId]} />
                    </FieldContent>
                  </Field>

                  <Field data-invalid={!!errors.zaloAppId}>
                    <FieldLabel htmlFor="channel-zalo-appid">{t("channel.zaloAppId")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-zalo-appid"
                        placeholder="e.g. 9876543210"
                        {...register("zaloAppId")}
                      />
                      <FieldError errors={[errors.zaloAppId]} />
                    </FieldContent>
                  </Field>
                </div>

                <div className="rounded-md border border-primary/20 bg-primary/5 p-3 text-xs text-muted-foreground">
                  <div className="font-medium text-foreground">{t("channel.zaloAutoConnectTitle")}</div>
                  <div className="mt-1">{t("channel.zaloAutoConnectDescription")}</div>
                </div>
              </div>
            ) : null}

            {channelType === "discord" ? (
              <div className="space-y-4">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field data-invalid={!!errors.discordGuildId}>
                    <FieldLabel htmlFor="channel-discord-guildid">{t("channel.discordGuildId")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-discord-guildid"
                        placeholder="e.g. 1234567890123456789"
                        {...register("discordGuildId")}
                      />
                      <FieldError errors={[errors.discordGuildId]} />
                    </FieldContent>
                  </Field>

                  <Field data-invalid={!!errors.discordGuildName}>
                    <FieldLabel htmlFor="channel-discord-guildname">{t("channel.discordGuildName")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-discord-guildname"
                        placeholder="e.g. Support Server"
                        {...register("discordGuildName")}
                      />
                      <FieldError errors={[errors.discordGuildName]} />
                    </FieldContent>
                  </Field>
                </div>

                <Field data-invalid={!!errors.discordBotToken}>
                  <FieldLabel htmlFor="channel-discord-bottoken">{t("channel.discordBotToken")}</FieldLabel>
                  <FieldContent>
                    <Input
                      id="channel-discord-bottoken"
                      type="password"
                      placeholder="Optional per-channel bot token"
                      {...register("discordBotToken")}
                    />
                    <FieldError errors={[errors.discordBotToken]} />
                  </FieldContent>
                </Field>

                <div className="rounded-md border border-primary/20 bg-primary/5 p-3 text-xs text-muted-foreground">
                  <div className="font-medium text-foreground">{t("channel.discordSetupTitle")}</div>
                  <div className="mt-1">{t("channel.discordSetupDescription")}</div>
                  <div className="mt-2 font-mono text-[11px]">
                    {t("channel.inboundWebhookUrl")}: /api/third/discord/webhook
                  </div>
                </div>
              </div>
            ) : null}

            {channelType === "telegram" ? (
              <div className="space-y-4">
                <Field data-invalid={!!errors.botToken}>
                  <FieldLabel htmlFor="channel-telegram-token">{t("channel.botToken")} *</FieldLabel>
                  <FieldContent>
                    <Input
                      id="channel-telegram-token"
                      type="password"
                      placeholder="123456789:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
                      {...register("botToken")}
                    />
                    <FieldError errors={[errors.botToken]} />
                  </FieldContent>
                </Field>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field data-invalid={!!errors.botUsername}>
                    <FieldLabel htmlFor="channel-telegram-username">{t("channel.botUsername")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-telegram-username"
                        placeholder="e.g. CroveDeskBot"
                        {...register("botUsername")}
                      />
                      <FieldError errors={[errors.botUsername]} />
                    </FieldContent>
                  </Field>

                  <Field data-invalid={!!errors.webhookSecret}>
                    <FieldLabel htmlFor="channel-telegram-secret">{t("channel.webhookSecret")}</FieldLabel>
                    <FieldContent>
                      <Input
                        id="channel-telegram-secret"
                        placeholder="Auto-generated secret"
                        {...register("webhookSecret")}
                      />
                      <FieldError errors={[errors.webhookSecret]} />
                    </FieldContent>
                  </Field>
                </div>

                <div className="rounded-md border border-primary/20 bg-primary/5 p-3 text-xs text-muted-foreground">
                  <div className="font-medium text-foreground">{t("channel.telegramAutoConnectTitle")}</div>
                  <div className="mt-1">{t("channel.telegramAutoConnectDescription")}</div>
                </div>
              </div>
            ) : null}

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
  const [origin, setOrigin] = useState("")

  useEffect(() => {
    setOrigin(window.location.origin)
  }, [])

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
  const [origin, setOrigin] = useState("")

  useEffect(() => {
    setOrigin(window.location.origin)
  }, [])

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
