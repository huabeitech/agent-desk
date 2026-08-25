import { request } from "@/lib/api/client"

export type SupportNavigationMenuItem = {
  id: string
  title: string
  url: string
  openInNewWindow: boolean
  visible: boolean
  sortNo: number
  children?: SupportNavigationMenuItem[]
}

export type PublicSupportConfig = {
  navigationMenu: SupportNavigationMenuItem[]
  aiCustomerService: {
    enabled: boolean
    channelId: string
  }
}

export type SupportAICustomerServiceUserToken = {
  userToken: string
  expiresAt: string
}

export function fetchSupportConfig() {
  return request<PublicSupportConfig>("/api/support/config", { skipAuth: true })
}

export function fetchSupportAICustomerServiceUserToken() {
  return request<SupportAICustomerServiceUserToken>("/api/support/ai-customer-service/user-token")
}
