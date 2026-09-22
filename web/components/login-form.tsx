"use client"

import Image from "next/image"
import Link from "next/link"
import { useRouter, useSearchParams } from "next/navigation"
import { startTransition, useEffect, useState } from "react"
import { toast } from "sonner"

import { useAuth } from "@/components/auth-provider"
import { loginWithPassword } from "@/lib/api/auth"
import { fetchPublicConfig, type PublicConfig } from "@/lib/api/config"
import { useI18n } from "@/i18n/provider"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { KeyRoundIcon, Loader2Icon, TriangleAlertIcon } from "lucide-react"

function detectWxWorkEnvironment() {
  if (typeof navigator === "undefined") {
    return false
  }
  const userAgent = navigator.userAgent.toLowerCase()
  return userAgent.includes("wxwork")
}

export function LoginForm({
  className,
  ...props
}: React.ComponentProps<"div">) {
  const t = useI18n()
  const router = useRouter()
  const searchParams = useSearchParams()
  const { session } = useAuth()
  const [isPending, setIsPending] = useState(false)
  const [isWxWorkEnv, setIsWxWorkEnv] = useState(false)
  const [publicConfig, setPublicConfig] = useState<PublicConfig | null>(null)
  const [publicConfigError, setPublicConfigError] = useState<string | null>(null)
  const nextPath = searchParams.get("next")
  const wxworkError = searchParams.get("wxworkError")
  const oidcError = searchParams.get("oidcError")
  const redirectPath =
    nextPath && nextPath.startsWith("/") ? nextPath : "/dashboard"
  const enabledProviderCount =
    Number(publicConfig?.wxworkEnabled) + Number(publicConfig?.oidcEnabled)
  const isPasswordLoginEnabled = publicConfig?.passwordLoginEnabled !== false

  useEffect(() => {
    if (session) {
      router.replace(redirectPath)
    }
  }, [redirectPath, router, session])

  useEffect(() => {
    if (wxworkError) {
      toast.error(wxworkError)
    }
  }, [wxworkError])

  useEffect(() => {
    if (oidcError) {
      toast.error(oidcError)
    }
  }, [oidcError])

  useEffect(() => {
    setIsWxWorkEnv(detectWxWorkEnvironment())
  }, [])

  useEffect(() => {
    let cancelled = false

    void fetchPublicConfig()
      .then((options) => {
        if (!cancelled) {
          setPublicConfig(options)
          setPublicConfigError(null)
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setPublicConfig(null)
          setPublicConfigError(error instanceof Error ? error.message : "")
        }
      })

    return () => {
      cancelled = true
    }
  }, [])

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!isPasswordLoginEnabled) {
      return
    }
    const formData = new FormData(event.currentTarget)
    const username = formData.get("username")?.toString().trim() ?? ""
    const password = formData.get("password")?.toString() ?? ""

    setIsPending(true)

    try {
      await loginWithPassword({ username, password })
      toast.success(t("auth.loginSuccess"))
      startTransition(() => {
        router.push(redirectPath)
      })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("auth.loginFailed"))
    } finally {
      setIsPending(false)
    }
  }

  if (publicConfigError) {
    return (
      <div className={cn("flex flex-col gap-6", className)} {...props}>
        <Card className="overflow-hidden p-0">
          <CardContent className="flex min-h-80 flex-col items-center justify-center gap-3 p-8 text-center">
            <TriangleAlertIcon className="size-8 text-destructive" />
            <div className="space-y-1">
              <h1 className="text-lg font-semibold">{t("auth.optionsLoadFailed")}</h1>
              <p className="text-sm text-muted-foreground">
                {publicConfigError || t("api.requestFailed")}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!publicConfig) {
    return (
      <div className={cn("flex flex-col gap-6", className)} {...props}>
        <Card className="overflow-hidden p-0">
          <CardContent className="flex min-h-80 flex-col items-center justify-center gap-3 p-8 text-center">
            <Loader2Icon className="size-7 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t("auth.loadingOptions")}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div
      className={cn("flex flex-col gap-6", className)}
      {...props}
    >
      <Card className="overflow-hidden p-0">
        <CardContent className="grid p-0 md:grid-cols-2">
          <form className="p-6 md:p-8" onSubmit={handleSubmit}>
            <FieldGroup>
              <div className="flex flex-col items-center gap-2 text-center">
                <div className="flex items-center gap-2.5 font-semibold tracking-tight text-lg mb-1">
                  <Image
                    src={publicConfig?.companyLogoUrl || "/images/logo.svg"}
                    alt={publicConfig?.companyName || t("app.brand")}
                    width={36}
                    height={36}
                    className="size-9 object-contain"
                    priority
                    unoptimized={Boolean(publicConfig?.companyLogoUrl?.startsWith("http"))}
                  />
                  <span>{publicConfig?.companyName ? (publicConfig.companyName.toLowerCase().includes("desk") ? publicConfig.companyName : `${publicConfig.companyName} Desk`) : t("app.brand")}</span>
                </div>
                <h1 className="text-2xl font-bold">{t("auth.welcome")}</h1>
                <p className="text-balance text-muted-foreground">
                  {t("auth.loginDescription", { brand: publicConfig.companyName || t("app.brand") })}
                </p>
              </div>
              {isPasswordLoginEnabled ? (
                <>
                  <Field>
                    <FieldLabel htmlFor="username">{t("auth.username")}</FieldLabel>
                    <Input
                      id="username"
                      name="username"
                      placeholder={t("auth.usernamePlaceholder")}
                      autoComplete="username"
                      required
                    />
                  </Field>
                  <Field>
                    <div className="flex items-center">
                      <FieldLabel htmlFor="password">{t("auth.password")}</FieldLabel>
                      <a
                        href="#"
                        className="ml-auto text-sm underline-offset-2 hover:underline"
                      >
                        {t("auth.forgotPassword")}
                      </a>
                    </div>
                    <Input
                      id="password"
                      name="password"
                      type="password"
                      placeholder={t("auth.passwordPlaceholder")}
                      autoComplete="current-password"
                      required
                    />
                  </Field>
                  <Field>
                    <Button type="submit" disabled={isPending}>
                      {isPending ? t("auth.signingIn") : t("auth.signIn")}
                    </Button>
                  </Field>
                </>
              ) : null}
              {enabledProviderCount > 0 ? (
                <>
                  {isPasswordLoginEnabled ? (
                    <FieldSeparator className="*:data-[slot=field-separator-content]:bg-card">
                      {t("auth.continueWith")}
                    </FieldSeparator>
                  ) : null}
                  <Field
                    className={cn(
                      "grid gap-4",
                      enabledProviderCount === 1 || !isPasswordLoginEnabled ? "grid-cols-1" : "grid-cols-2"
                    )}
                  >
                    {publicConfig.wxworkEnabled ? (
                      <Button
                        type="button"
                        variant="outline"
                        aria-label={t("auth.wxworkSignIn")}
                        onClick={() => {
                          const path = isWxWorkEnv
                            ? "/api/auth/wxwork_login"
                            : "/api/auth/wxwork_qr_login"
                          window.location.href = `${path}?next=${encodeURIComponent(redirectPath)}`
                        }}
                      >
                        <Image
                          src="/images/wxwork.svg"
                          alt=""
                          width={16}
                          height={16}
                          className="size-4 shrink-0"
                        />
                        <span>{t("auth.wxworkSignIn")}</span>
                      </Button>
                    ) : null}
                    {publicConfig.oidcEnabled ? (
                      <Button
                        type="button"
                        variant="outline"
                        aria-label={t("auth.oidcSignIn")}
                        onClick={() => {
                          window.location.href = `/api/auth/oidc_login?next=${encodeURIComponent(redirectPath)}`
                        }}
                      >
                        <KeyRoundIcon className="size-4 shrink-0" />
                        <span>{t("auth.oidcSignIn")}</span>
                      </Button>
                    ) : null}
                  </Field>
                </>
              ) : null}
              {/* <FieldDescription className="text-center">
                {t("auth.noAccount")} <a href="#">{t("auth.signUp")}</a>
              </FieldDescription> */}
            </FieldGroup>
          </form>
          <div className="relative hidden bg-muted md:block">
            {/* shadcn login-04 uses a plain img for the decorative side panel. */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/images/login-illustration.svg"
              alt={t("auth.imageAlt")}
              className="absolute inset-0 h-full w-full object-cover dark:brightness-[0.2] dark:grayscale"
            />
          </div>
        </CardContent>
      </Card>
      <FieldDescription className="px-6 text-center">
        {t("auth.termsPrefix")}{" "}
        <Link href="/legal/terms" target="_blank" rel="noreferrer">
          {t("auth.termsOfService")}
        </Link>{" "}
        {t("auth.termsJoiner")}{" "}
        <Link href="/legal/privacy" target="_blank" rel="noreferrer">
          {t("auth.privacyPolicy")}
        </Link>
        .
      </FieldDescription>
    </div>
  )
}
