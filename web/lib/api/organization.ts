import { request } from "@/lib/api/client"

export type OrganizationItem = {
  id: number
  code: string
  name: string
  logo: string
  plan: string
  status: number
  role?: string
  isActive: boolean
  createdAt: string
}

export type OrganizationMemberItem = {
  id: number
  userId: number
  username: string
  nickname: string
  email: string
  avatar: string
  role: string
  status: number
  createdAt: string
}

export type UserOrganizationListResponse = {
  currentOrganizationId: number
  organizations: OrganizationItem[]
}

export async function listMyOrganizations() {
  return request<UserOrganizationListResponse>("/api/dashboard/organization/my_list", {
    method: "GET",
  })
}

export async function createOrganization(data: { name: string; code?: string; logo?: string }) {
  return request<OrganizationItem>("/api/dashboard/organization/create", {
    method: "POST",
    body: JSON.stringify(data),
  })
}

export async function switchOrganization(organizationId: number) {
  return request<OrganizationItem>("/api/dashboard/organization/switch", {
    method: "POST",
    body: JSON.stringify({ organizationId }),
  })
}

export async function getOrganizationMembers() {
  return request<OrganizationMemberItem[]>("/api/dashboard/organization/members", {
    method: "GET",
  })
}

export async function addOrganizationMember(data: { emailOrUsername: string; role?: string }) {
  return request<OrganizationMemberItem>("/api/dashboard/organization/add_member", {
    method: "POST",
    body: JSON.stringify(data),
  })
}

export async function removeOrganizationMember(userId: number) {
  return request<{ success: boolean }>("/api/dashboard/organization/remove_member", {
    method: "POST",
    body: JSON.stringify({ userId }),
  })
}

export async function updateOrganization(data: { name: string; logo?: string }) {
  return request<OrganizationItem>("/api/dashboard/organization/update", {
    method: "POST",
    body: JSON.stringify(data),
  })
}
