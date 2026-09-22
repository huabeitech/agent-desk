import { request } from "@/lib/api/client"

export type PublicConfig = {
  language: string
  companyName?: string
  companyLogoUrl?: string
  companyFaviconUrl?: string
  passwordLoginEnabled?: boolean
  wxworkEnabled: boolean
  oidcEnabled: boolean
}

let publicConfigPromise: Promise<PublicConfig> | null = null

export function fetchPublicConfig() {
  publicConfigPromise ??= request<PublicConfig>("/api/config", {
    skipAuth: true,
  }).catch((error) => {
    publicConfigPromise = null
    throw error
  })
  return publicConfigPromise
}
