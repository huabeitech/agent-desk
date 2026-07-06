import { request } from "@/lib/api/client"

export type PublicConfig = {
  language: string
  wxworkEnabled: boolean
  oidcEnabled: boolean
}

export function fetchPublicConfig() {
  return request<PublicConfig>("/api/config", {
    skipAuth: true,
  })
}
