"use client"

import { useEffect, useMemo, useState } from "react"
import { zodResolver } from "@hookform/resolvers/zod"
import { Controller, Resolver, useForm } from "react-hook-form"
import { z } from "zod/v4"

import {
  type AdminUser,
  type UpdateAdminUserPayload,
  fetchUserDetail,
} from "@/lib/api/admin"
import { useI18n } from "@/i18n/provider"
import { ImageInput } from "@/components/image-input"
import { Button } from "@/components/ui/button"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import {
  Field,
  FieldContent,
  FieldError,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"

type UserEditDrawerProps = {
  open: boolean
  saving: boolean
  itemId: number | null
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: UpdateAdminUserPayload) => Promise<void>
}

const emptyForm: EditForm = {
  nickname: "",
  avatar: "",
  mobile: "",
  email: "",
}

type EditForm = {
  nickname: string
  avatar: string
  mobile: string
  email: string
}

function toNullableString(value: string) {
  const output = value.trim()
  return output ? output : null
}

function buildForm(item: AdminUser | null): EditForm {
  if (!item) {
    return emptyForm
  }

  return {
    nickname: item.nickname || "",
    avatar: item.avatarAssetId || item.avatar || "",
    mobile: item.mobile || "",
    email: item.email || "",
  }
}

function buildPayload(userId: number, form: EditForm): UpdateAdminUserPayload {
  return {
    id: userId,
    nickname: form.nickname.trim(),
    avatar: form.avatar.trim(),
    mobile: toNullableString(form.mobile),
    email: toNullableString(form.email),
    remark: "",
  }
}

export function EditDrawer({
  open,
  saving,
  itemId,
  onOpenChange,
  onSubmit,
}: UserEditDrawerProps) {
  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      {open ? (
        <UserEditDrawerBody
          key={itemId ? `edit-${itemId}` : "edit"}
          itemId={itemId}
          saving={saving}
          onOpenChange={onOpenChange}
          onSubmit={onSubmit}
        />
      ) : null}
    </Drawer>
  )
}

type UserEditDrawerBodyProps = {
  saving: boolean
  itemId: number | null
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: UpdateAdminUserPayload) => Promise<void>
}

function UserEditDrawerBody({
  saving,
  itemId,
  onOpenChange,
  onSubmit,
}: UserEditDrawerBodyProps) {
  const t = useI18n()
  const [loading, setLoading] = useState(false)
  const [item, setItem] = useState<AdminUser | null>(null)
  const editFormSchema = useMemo(
    () =>
      z.object({
        nickname: z.string().trim().min(1, t("user.nicknameRequired")),
        avatar: z.string().trim(),
        mobile: z
          .string()
          .trim()
          .refine(
            (value) => value.length === 0 || /^[0-9+\-\s]{6,20}$/.test(value),
            t("user.mobileInvalid")
          ),
        email: z
          .string()
          .trim()
          .refine(
            (value) =>
              value.length === 0 || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value),
            t("user.emailInvalid")
          ),
      }),
    [t]
  )
  const editFormResolver = useMemo(
    () => zodResolver(editFormSchema as never) as Resolver<EditForm>,
    [editFormSchema]
  )
  const form = useForm<EditForm>({
    resolver: editFormResolver,
    defaultValues: emptyForm,
  })
  const {
    control,
    handleSubmit,
    reset,
    register,
    formState: { errors },
  } = form

  useEffect(() => {
    async function loadDetail() {
      if (!itemId) {
        setItem(null)
        reset(emptyForm)
        return
      }
      setLoading(true)
      try {
        const data = await fetchUserDetail(itemId)
        setItem(data)
        reset(buildForm(data))
      } catch (error) {
        console.error("Failed to load user:", error)
      } finally {
        setLoading(false)
      }
    }
    void loadDetail()
  }, [itemId, reset])

  async function onFormSubmit(values: EditForm) {
    if (!itemId) {
      return
    }

    await onSubmit(buildPayload(itemId, values))
  }

  return (
    <DrawerContent className="min-w-2xl overflow-hidden">
      <DrawerHeader className="shrink-0">
        <DrawerTitle>{t("user.editTitle")}</DrawerTitle>
        <DrawerDescription>
          {t("user.currentUser", { username: item?.username || "-" })}
        </DrawerDescription>
      </DrawerHeader>
      {loading ? (
        <div className="flex min-h-0 flex-1 items-center justify-center py-12">
          <div className="text-muted-foreground">{t("user.loading")}</div>
        </div>
      ) : (
        <form
          className="flex min-h-0 flex-1 flex-col"
          onSubmit={handleSubmit(onFormSubmit)}
        >
          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 pb-4">
            <Field data-invalid={!!errors.nickname}>
              <FieldLabel htmlFor="user-nickname">{t("user.nickname")}</FieldLabel>
              <FieldContent>
                <Input
                  id="user-nickname"
                  placeholder={t("user.nicknamePlaceholder")}
                  aria-invalid={!!errors.nickname}
                  {...register("nickname")}
                />
                <FieldError errors={[errors.nickname]} />
              </FieldContent>
            </Field>
            <Field data-invalid={!!errors.avatar}>
              <FieldLabel>{t("user.avatar")}</FieldLabel>
              <FieldContent>
                <Controller
                  control={control}
                  name="avatar"
                  render={({ field }) => (
                    <ImageInput
                      value={field.value}
                      previewValue={item?.avatar}
                      onChange={field.onChange}
                      disabled={saving || loading}
                      prefix="avatar"
                      placeholder={t("user.avatarPlaceholder")}
                      className="size-16 rounded-full"
                    />
                  )}
                />
                <FieldError errors={[errors.avatar]} />
              </FieldContent>
            </Field>
            <Field data-invalid={!!errors.mobile}>
              <FieldLabel htmlFor="user-mobile">{t("user.mobile")}</FieldLabel>
              <FieldContent>
                <Input
                  id="user-mobile"
                  placeholder={t("user.mobilePlaceholder")}
                  aria-invalid={!!errors.mobile}
                  {...register("mobile")}
                />
                <FieldError errors={[errors.mobile]} />
              </FieldContent>
            </Field>
            <Field data-invalid={!!errors.email}>
              <FieldLabel htmlFor="user-email">{t("user.email")}</FieldLabel>
              <FieldContent>
                <Input
                  id="user-email"
                  placeholder={t("user.emailPlaceholder")}
                  aria-invalid={!!errors.email}
                  {...register("email")}
                />
                <FieldError errors={[errors.email]} />
              </FieldContent>
            </Field>
          </div>
          <DrawerFooter className="shrink-0 border-t bg-background">
            <Button type="submit" disabled={saving || loading}>
              {saving ? t("user.saving") : t("user.saveEdit")}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={saving}
            >
              {t("user.cancel")}
            </Button>
          </DrawerFooter>
        </form>
      )}
    </DrawerContent>
  );
}
