"use client"

import { useEffect, useState } from "react"
import { RefreshCwIcon, SaveIcon, SparklesIcon } from "lucide-react"
import { toast } from "sonner"

import { DashboardPage, DashboardToolbar } from "@/components/dashboard-page"
import { Button } from "@/components/ui/button"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  fetchDigitalStoreProfile,
  saveDigitalStoreProfile,
  seedMuseDigitalStoreProfile,
  syncDigitalStoreKnowledge,
  type DigitalStoreProfile,
} from "@/lib/api/digital-store"

type FormState = Omit<
  DigitalStoreProfile,
  "knowledgeFAQId" | "templateCode" | "templateVersion" | "templateAppliedAt" | "updatedAt"
>

const emptyForm: FormState = {
  brandName: "",
  industry: "",
  storeName: "",
  storeAddress: "",
  businessHours: "",
  contactPhone: "",
  serviceWechat: "",
  enterpriseWebhookUrl: "",
  aiManagerName: "",
  aiPersona: "",
  replyStyle: "",
  forbiddenClaims: "",
  handoffPolicy: "",
  appointmentPolicy: "",
  knowledgeBaseId: 0,
  initialized: false,
}

function toForm(profile: DigitalStoreProfile): FormState {
  return {
    ...emptyForm,
    ...profile,
    knowledgeBaseId: Number(profile.knowledgeBaseId || 0),
    initialized: Boolean(profile.initialized),
  }
}

export default function DashboardDigitalStorePage() {
  const [form, setForm] = useState<FormState>(emptyForm)
  const [profile, setProfile] = useState<DigitalStoreProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  async function loadProfile() {
    setLoading(true)
    try {
      const next = await fetchDigitalStoreProfile()
      setProfile(next)
      setForm(toForm(next))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载店长配置失败")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadProfile()
  }, [])

  function patchForm<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((current) => ({ ...current, [key]: value }))
  }

  async function handleSave() {
    if (saving) return
    setSaving(true)
    try {
      const next = await saveDigitalStoreProfile({
        ...form,
        knowledgeBaseId: Number(form.knowledgeBaseId || 0),
        initialized: true,
      })
      setProfile(next)
      setForm(toForm(next))
      toast.success("店长配置已保存，并同步到知识库")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "保存店长配置失败")
    } finally {
      setSaving(false)
    }
  }

  async function handleSeedMuse() {
    if (saving) return
    setSaving(true)
    try {
      const next = await seedMuseDigitalStoreProfile()
      setProfile(next)
      setForm(toForm(next))
      toast.success("已导入慕斯寝具样板配置")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "导入样板失败")
    } finally {
      setSaving(false)
    }
  }

  async function handleSyncKnowledge() {
    if (saving) return
    setSaving(true)
    try {
      const next = await syncDigitalStoreKnowledge()
      setProfile(next)
      setForm(toForm(next))
      toast.success("店长配置已同步到知识库")
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "同步知识库失败")
    } finally {
      setSaving(false)
    }
  }

  return (
    <DashboardPage>
      <DashboardToolbar
        actions={
          <>
            <Button variant="outline" onClick={handleSeedMuse} disabled={saving}>
              <SparklesIcon />
              导入慕斯样板
            </Button>
            <Button variant="outline" onClick={handleSyncKnowledge} disabled={saving || !profile?.initialized}>
              <RefreshCwIcon />
              同步知识库
            </Button>
            <Button onClick={handleSave} disabled={saving || loading}>
              <SaveIcon />
              {saving ? "保存中" : "保存"}
            </Button>
          </>
        }
      >
        <div>
          <h1 className="text-lg font-semibold">AI数字店长配置</h1>
          <p className="text-sm text-muted-foreground">
            {profile?.knowledgeFAQId
              ? `已同步为 FAQ #${profile.knowledgeFAQId}`
              : "配置品牌、门店、人设、预约和转人工规则"}
          </p>
        </div>
      </DashboardToolbar>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
        <div className="rounded-md border border-border/70 bg-card p-4">
          <div className="grid gap-4 md:grid-cols-2">
            <TextField label="品牌名称" value={form.brandName} onChange={(value) => patchForm("brandName", value)} placeholder="慕斯寝具" />
            <TextField label="行业类型" value={form.industry} onChange={(value) => patchForm("industry", value)} placeholder="家居寝具" />
            <TextField label="门店名称" value={form.storeName} onChange={(value) => patchForm("storeName", value)} placeholder="城市旗舰店" />
            <TextField label="营业时间" value={form.businessHours} onChange={(value) => patchForm("businessHours", value)} placeholder="周一至周日 10:00-21:00" />
            <TextField label="门店地址" value={form.storeAddress} onChange={(value) => patchForm("storeAddress", value)} placeholder="门店详细地址" className="md:col-span-2" />
            <TextField label="联系电话" value={form.contactPhone} onChange={(value) => patchForm("contactPhone", value)} placeholder="400 或门店电话" />
            <TextField label="客服微信" value={form.serviceWechat} onChange={(value) => patchForm("serviceWechat", value)} placeholder="微信号" />
            <TextField label="企业微信 Webhook" value={form.enterpriseWebhookUrl} onChange={(value) => patchForm("enterpriseWebhookUrl", value)} placeholder="后续用于通知顾问" className="md:col-span-2" />
            <TextField label="FAQ知识库ID" value={String(form.knowledgeBaseId || "")} onChange={(value) => patchForm("knowledgeBaseId", Number(value || 0))} placeholder="留空自动使用第一个 FAQ 知识库" type="number" />
            <TextField label="AI店长名称" value={form.aiManagerName} onChange={(value) => patchForm("aiManagerName", value)} placeholder="慕小眠" />
          </div>
        </div>

        <div className="rounded-md border border-border/70 bg-card p-4">
          <div className="space-y-4">
            <TextareaField label="AI人设" value={form.aiPersona} onChange={(value) => patchForm("aiPersona", value)} rows={4} />
            <TextareaField label="回复风格" value={form.replyStyle} onChange={(value) => patchForm("replyStyle", value)} rows={4} />
            <TextareaField label="预约规则" value={form.appointmentPolicy} onChange={(value) => patchForm("appointmentPolicy", value)} rows={4} />
            <TextareaField label="转人工规则" value={form.handoffPolicy} onChange={(value) => patchForm("handoffPolicy", value)} rows={4} />
            <TextareaField label="禁止承诺内容" value={form.forbiddenClaims} onChange={(value) => patchForm("forbiddenClaims", value)} rows={4} />
          </div>
        </div>
      </div>
    </DashboardPage>
  )
}

function TextField({
  label,
  value,
  onChange,
  placeholder,
  type = "text",
  className,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  type?: string
  className?: string
}) {
  return (
    <Field className={className}>
      <FieldLabel>{label}</FieldLabel>
      <FieldContent>
        <Input
          type={type}
          value={value}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
      </FieldContent>
    </Field>
  )
}

function TextareaField({
  label,
  value,
  onChange,
  rows,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  rows: number
}) {
  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <FieldContent>
        <Textarea
          value={value}
          rows={rows}
          onChange={(event) => onChange(event.target.value)}
        />
      </FieldContent>
    </Field>
  )
}
