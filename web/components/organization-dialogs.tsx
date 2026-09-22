"use client"

import { Building2Icon, Loader2Icon, PlusIcon, Trash2Icon, UserPlusIcon, UsersIcon } from "lucide-react"
import { useEffect, useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { useI18n } from "@/i18n/provider"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  addOrganizationMember,
  createOrganization,
  getOrganizationMembers,
  removeOrganizationMember,
  updateOrganization,
  type OrganizationItem,
  type OrganizationMemberItem,
} from "@/lib/api/organization"

export function CreateOrganizationDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (org: OrganizationItem) => void
}) {
  const t = useI18n()
  const [name, setName] = useState("")
  const [code, setCode] = useState("")
  const [isPending, setIsPending] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      toast.error(t("organization.nameRequired"))
      return
    }
    setIsPending(true)
    try {
      const org = await createOrganization({
        name: name.trim(),
        code: code.trim() || undefined,
      })
      toast.success(t("organization.created", { name: org.name }))
      setName("")
      setCode("")
      onOpenChange(false)
      onCreated(org)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("organization.createFailed"))
    } finally {
      setIsPending(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Building2Icon className="size-5 text-primary" />
              {t("organization.createTitle")}
            </DialogTitle>
            <DialogDescription>
              {t("organization.createDescription")}
            </DialogDescription>
          </DialogHeader>

          <FieldGroup className="py-4">
            <Field>
              <FieldLabel htmlFor="org-name">{t("organization.name")}</FieldLabel>
              <Input
                id="org-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("organization.namePlaceholder")}
                required
                autoFocus
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="org-code">{t("organization.code")}</FieldLabel>
              <Input
                id="org-code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder={t("organization.codePlaceholder")}
              />
            </Field>
          </FieldGroup>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isPending}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={isPending || !name.trim()}>
              {isPending ? (
                <>
                  <Loader2Icon className="mr-2 size-4 animate-spin" />
                  {t("organization.creating")}
                </>
              ) : (
                t("organization.create")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function ManageOrganizationDialog({
  open,
  onOpenChange,
  organization,
  onUpdated,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  organization: OrganizationItem | null
  onUpdated?: () => void
}) {
  const t = useI18n()
  const [members, setMembers] = useState<OrganizationMemberItem[]>([])
  const [loadingMembers, setLoadingMembers] = useState(false)
  const [emailOrUsername, setEmailOrUsername] = useState("")
  const [role, setRole] = useState("MEMBER")
  const [isAdding, setIsAdding] = useState(false)
  const [orgName, setOrgName] = useState("")
  const [isUpdatingName, setIsUpdatingName] = useState(false)

  useEffect(() => {
    if (open && organization) {
      setOrgName(organization.name)
      setLoadingMembers(true)
      getOrganizationMembers()
        .then((res) => {
          setMembers(res || [])
        })
        .catch(() => {
          setMembers([])
        })
        .finally(() => {
          setLoadingMembers(false)
        })
    }
  }, [open, organization])

  const handleUpdateName = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!orgName.trim() || orgName.trim() === organization?.name) return
    setIsUpdatingName(true)
    try {
      await updateOrganization({ name: orgName.trim() })
      toast.success(t("organization.updated"))
      onUpdated?.()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update name")
    } finally {
      setIsUpdatingName(false)
    }
  }

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!emailOrUsername.trim()) return
    setIsAdding(true)
    try {
      const added = await addOrganizationMember({
        emailOrUsername: emailOrUsername.trim(),
        role,
      })
      setMembers((prev) => {
        const filtered = prev.filter((m) => m.userId !== added.userId)
        return [...filtered, added]
      })
      setEmailOrUsername("")
      toast.success(t("organization.memberAdded"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add member")
    } finally {
      setIsAdding(false)
    }
  }

  const handleRemoveMember = async (userId: number, username: string) => {
    if (!confirm(t("organization.removeConfirm"))) return
    try {
      await removeOrganizationMember(userId)
      setMembers((prev) => prev.filter((m) => m.userId !== userId))
      toast.success(t("organization.memberRemoved"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to remove member")
    }
  }

  if (!organization) return null

  const isOwnerOrAdmin = organization.role === "OWNER" || organization.role === "ADMIN"

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Building2Icon className="size-5 text-primary" />
            {organization.name} — {t("organization.manageTitle")}
          </DialogTitle>
          <DialogDescription>
            {t("organization.general")}
          </DialogDescription>
        </DialogHeader>

        {isOwnerOrAdmin ? (
          <form onSubmit={handleUpdateName} className="space-y-3 pt-2">
            <Field>
              <FieldLabel>{t("organization.name")}</FieldLabel>
              <div className="flex gap-2">
                <Input
                  value={orgName}
                  onChange={(e) => setOrgName(e.target.value)}
                  placeholder={t("organization.namePlaceholder")}
                  required
                />
                <Button
                  type="submit"
                  size="sm"
                  variant="outline"
                  disabled={isUpdatingName || !orgName.trim() || orgName.trim() === organization.name}
                >
                  {isUpdatingName ? t("organization.saving") : t("organization.save")}
                </Button>
              </div>
            </Field>
          </form>
        ) : null}

        <div className="space-y-4 pt-4 border-t">
          <div className="flex items-center justify-between">
            <h4 className="text-sm font-semibold flex items-center gap-1.5">
              <UsersIcon className="size-4 text-muted-foreground" />
              {t("organization.members")} ({members.length})
            </h4>
          </div>

          {isOwnerOrAdmin ? (
            <form onSubmit={handleAddMember} className="flex gap-2">
              <Input
                value={emailOrUsername}
                onChange={(e) => setEmailOrUsername(e.target.value)}
                placeholder={t("organization.emailOrUsernamePlaceholder")}
                className="flex-1 text-xs"
                required
              />
              <select
                value={role}
                onChange={(e) => setRole(e.target.value)}
                className="rounded-md border border-input bg-background px-2 text-xs"
              >
                <option value="MEMBER">{t("organization.member")}</option>
                <option value="ADMIN">{t("organization.admin")}</option>
              </select>
              <Button type="submit" size="sm" disabled={isAdding || !emailOrUsername.trim()}>
                {isAdding ? <Loader2Icon className="size-4 animate-spin" /> : <UserPlusIcon className="size-4 mr-1" />}
                {t("organization.add")}
              </Button>
            </form>
          ) : null}

          {loadingMembers ? (
            <div className="flex justify-center py-6 text-muted-foreground">
              <Loader2Icon className="size-5 animate-spin" />
            </div>
          ) : (
            <div className="divide-y rounded-md border text-xs">
              {members.map((m) => (
                <div key={m.id} className="flex items-center justify-between p-2.5">
                  <div className="flex flex-col min-w-0 pr-2">
                    <span className="font-medium truncate">{m.nickname || m.username}</span>
                    <span className="text-muted-foreground truncate">{m.email || `@${m.username}`}</span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] font-semibold uppercase ${
                        m.role === "OWNER"
                          ? "bg-primary/10 text-primary"
                          : m.role === "ADMIN"
                          ? "bg-amber-500/10 text-amber-500"
                          : "bg-muted text-muted-foreground"
                      }`}
                    >
                      {m.role}
                    </span>
                    {isOwnerOrAdmin && m.role !== "OWNER" ? (
                      <Button
                        size="icon-xs"
                        variant="ghost"
                        className="text-destructive hover:bg-destructive/10"
                        onClick={() => handleRemoveMember(m.userId, m.nickname || m.username)}
                      >
                        <Trash2Icon className="size-3.5" />
                      </Button>
                    ) : null}
                  </div>
                </div>
              ))}
              {members.length === 0 ? (
                <div className="py-6 text-center text-muted-foreground">{t("common.emptyData")}</div>
              ) : null}
            </div>
          )}
        </div>

        <DialogFooter className="pt-4 border-t">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
