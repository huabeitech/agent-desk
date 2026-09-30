"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { SaveIcon } from "lucide-react"
import { toast } from "sonner"

import { DashboardPage } from "@/components/dashboard-page"
import { OptionCombobox } from "@/components/option-combobox"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { useI18n } from "@/i18n/provider"
import { fetchSystemConfig, saveSystemConfig } from "@/lib/api/admin"

const LOG_LEVELS = ["debug", "info", "warn", "error"] as const

export default function DashboardSettingsPage() {
  const t = useI18n()
  const [logLevel, setLogLevel] = useState<string>("")
  const [savedLevel, setSavedLevel] = useState<string>("")
  const [idleTimeout, setIdleTimeout] = useState<string>("")
  const [savedIdleTimeout, setSavedIdleTimeout] = useState<string>("")
  const [reminderMessage, setReminderMessage] = useState<string>("")
  const [savedReminderMessage, setSavedReminderMessage] = useState<string>("")
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const levelOptions = useMemo(() => LOG_LEVELS.map((level) => ({
    value: level,
    label: t(`systemConfig.level${level.charAt(0).toUpperCase()}${level.slice(1)}`),
  })), [t])

  const loadConfig = useCallback(async () => {
    try {
      setLoading(true)
      const config = await fetchSystemConfig()
      setLogLevel(config.logLevel)
      setSavedLevel(config.logLevel)
      setIdleTimeout(String(config.conversationIdleTimeout))
      setSavedIdleTimeout(String(config.conversationIdleTimeout))
      setReminderMessage(config.conversationIdleReminderMessage)
      setSavedReminderMessage(config.conversationIdleReminderMessage)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("systemConfig.loadFailed"))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadConfig()
  }, [loadConfig])

  const dirty = logLevel !== savedLevel || idleTimeout !== savedIdleTimeout || reminderMessage !== savedReminderMessage

  useEffect(() => {
    if (!dirty) {
      return
    }
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ""
    }
    window.addEventListener("beforeunload", handleBeforeUnload)
    return () => window.removeEventListener("beforeunload", handleBeforeUnload)
  }, [dirty])

  async function handleSave() {
    const timeoutNum = Number(idleTimeout)
    if (!idleTimeout || Number.isNaN(timeoutNum) || timeoutNum < 5 || timeoutNum > 1440) {
      toast.error(t("systemConfig.idleTimeoutInvalid"))
      return
    }
    const trimmedReminder = reminderMessage.trim()
    if (!trimmedReminder) {
      toast.error(t("systemConfig.idleReminderInvalid"))
      return
    }
    try {
      setSaving(true)
      const config = await saveSystemConfig(logLevel, timeoutNum, trimmedReminder)
      setLogLevel(config.logLevel)
      setSavedLevel(config.logLevel)
      setIdleTimeout(String(config.conversationIdleTimeout))
      setSavedIdleTimeout(String(config.conversationIdleTimeout))
      setReminderMessage(config.conversationIdleReminderMessage)
      setSavedReminderMessage(config.conversationIdleReminderMessage)
      toast.success(t("systemConfig.saved"))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("systemConfig.saveFailed"))
    } finally {
      setSaving(false)
    }
  }

  return (
    <DashboardPage>
      <div className="min-w-0 space-y-1">
        <h1 className="text-lg font-semibold tracking-tight">{t("systemConfig.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("systemConfig.description")}</p>
      </div>

      <div className="grid max-w-2xl gap-5 rounded-md border bg-card p-4">
        <div className="grid gap-2">
          <Label htmlFor="system-log-level">{t("systemConfig.logLevel")}</Label>
          <OptionCombobox
            value={logLevel}
            onChange={setLogLevel}
            options={levelOptions}
            placeholder={loading ? t("systemConfig.loading") : t("systemConfig.selectLogLevel")}
            emptyText={t("systemConfig.loadFailed")}
            disabled={loading || saving}
            triggerClassName="rounded-md"
          />
          <p className="text-xs text-muted-foreground">{t("systemConfig.logLevelHelp")}</p>
        </div>

        <div className="grid gap-2">
          <Label htmlFor="system-idle-timeout">{t("systemConfig.conversationIdleTimeout")}</Label>
          <div className="flex items-center gap-2">
            <Input
              id="system-idle-timeout"
              type="number"
              min={5}
              max={1440}
              value={idleTimeout}
              onChange={(e) => setIdleTimeout(e.target.value)}
              disabled={loading || saving}
              className="w-28 rounded-md"
            />
            <span className="text-sm text-muted-foreground">{t("systemConfig.idleTimeoutUnit")}</span>
          </div>
          <p className="text-xs text-muted-foreground">{t("systemConfig.conversationIdleTimeoutHelp")}</p>
        </div>

        <div className="grid gap-2">
          <Label htmlFor="system-idle-reminder">{t("systemConfig.conversationIdleReminderMessage")}</Label>
          <Textarea
            id="system-idle-reminder"
            value={reminderMessage}
            onChange={(e) => setReminderMessage(e.target.value)}
            disabled={loading || saving}
            className="rounded-md"
            rows={2}
          />
          <p className="text-xs text-muted-foreground">{t("systemConfig.conversationIdleReminderMessageHelp")}</p>
        </div>

        <div className="flex justify-end">
          <Button type="button" onClick={() => void handleSave()} disabled={loading || saving || !dirty}>
            <SaveIcon />
            {saving ? t("systemConfig.saving") : t("systemConfig.save")}
          </Button>
        </div>
      </div>
    </DashboardPage>
  )
}
