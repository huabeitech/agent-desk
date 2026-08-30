"use client"

import Link from "next/link"
import { useRouter } from "next/navigation"
import { useEffect, useState } from "react"
import { toast } from "sonner"

import { useSupportAuth } from "@/app/(support)/support/_components/support-auth-provider"
import { SupportPageContent, SupportPageShell } from "@/app/(support)/support/_components/support-page-shell"
import { SupportFormField } from "@/app/(support)/support/_components/support-ui"
import { ImageInput } from "@/components/image-input"
import { Button, buttonVariants } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useI18n } from "@/i18n/provider"
import { updateProfile, uploadProfileAvatar } from "@/lib/api/auth"
import { cn } from "@/lib/utils"

export function SupportProfileEditPage() {
  const t = useI18n()
  const router = useRouter()
  const { ready, session, refreshSession } = useSupportAuth()
  const [nickname, setNickname] = useState("")
  const [avatar, setAvatar] = useState("")
  const [email, setEmail] = useState("")
  const [error, setError] = useState("")
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (ready && !session) {
      router.replace("/support/login?next=/support/profile/edit")
    }
  }, [ready, router, session])

  useEffect(() => {
    if (!session) return
    setNickname(session.user.nickname || session.user.username)
    setAvatar(session.user.avatarAssetId || session.user.avatar || "")
    setEmail(session.user.email || "")
  }, [session])

  const submit = async () => {
    if (submitting) return
    const nextNickname = nickname.trim()
    if (!nextNickname) {
      setError(t("supportPublic.profile.nicknameRequired"))
      return
    }
    setSubmitting(true)
    setError("")
    try {
      await updateProfile({
        avatar: avatar.trim(),
        email: email.trim() || undefined,
        nickname: nextNickname,
      })
      await refreshSession()
      toast.success(t("supportPublic.toast.profileUpdated"))
      router.replace("/support/profile")
    } catch (err) {
      setError((err as Error).message || t("api.requestFailed"))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <SupportPageShell section="community">
      <SupportPageContent className="py-6 sm:py-8">
        <div className="mx-auto max-w-2xl rounded-md bg-card">
          <div className="border-b px-5 py-4 sm:px-6">
            <h1 className="text-lg font-semibold">{t("supportPublic.profile.editTitle")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t("supportPublic.profile.editDescription")}</p>
          </div>
          <div className="grid gap-5 p-5 sm:p-6">
            <SupportFormField label={t("supportPublic.profile.avatar")}>
              <ImageInput value={avatar} previewValue={session?.user.avatar} onChange={setAvatar} upload={uploadProfileAvatar} placeholder={t("supportPublic.profile.avatarPlaceholder")} />
            </SupportFormField>
            <SupportFormField label={t("supportPublic.profile.nickname")}>
              <Input value={nickname} onChange={(event) => setNickname(event.target.value)} placeholder={t("supportPublic.profile.nicknamePlaceholder")} />
            </SupportFormField>
            <SupportFormField label={t("supportPublic.profile.email")}>
              <Input value={email} onChange={(event) => setEmail(event.target.value)} placeholder={t("supportPublic.profile.emailPlaceholder")} />
            </SupportFormField>
            {error ? <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p> : null}
          </div>
          <div className="flex flex-col-reverse gap-2 border-t px-5 py-4 sm:flex-row sm:justify-end sm:px-6">
            <Link className={cn(buttonVariants({ variant: "outline" }), "sm:w-auto")} href="/support/profile">
              {t("supportPublic.actions.cancel")}
            </Link>
            <Button disabled={submitting} onClick={() => void submit()}>
              {submitting ? t("supportPublic.actions.processing") : t("supportPublic.actions.save")}
            </Button>
          </div>
        </div>
      </SupportPageContent>
    </SupportPageShell>
  )
}
