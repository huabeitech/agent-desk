export type AuthUser = {
  id: number
  username: string
  nickname: string
  avatar: string
  avatarAssetId?: string
  email?: string
  userType: string
  status: number
  roles: string[]
}

export type AuthSession = {
  accessToken: string
  expiresAt?: string
  user: AuthUser
  permissions: string[]
  roles: string[]
}

const SESSION_STORAGE_KEY = "agent-desk-session"
export const AUTH_SESSION_EXPIRED_EVENT = "agent-desk-auth-expired"
export const AUTH_SESSION_CHANGED_EVENT = "agent-desk-auth-changed"

function hasWindow() {
  return typeof window !== "undefined"
}

export function readSession(): AuthSession | null {
  if (!hasWindow()) {
    return null
  }

  const raw = window.localStorage.getItem(SESSION_STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    return JSON.parse(raw) as AuthSession
  } catch {
    window.localStorage.removeItem(SESSION_STORAGE_KEY)
    return null
  }
}

export function writeSession(session: AuthSession) {
  if (!hasWindow()) {
    return
  }
  window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(session))
  window.dispatchEvent(new Event(AUTH_SESSION_CHANGED_EVENT))
}

export function clearSession() {
  if (!hasWindow()) {
    return
  }
  window.localStorage.removeItem(SESSION_STORAGE_KEY)
  window.dispatchEvent(new Event(AUTH_SESSION_CHANGED_EVENT))
}

export function expireSession() {
  if (!hasWindow()) {
    return
  }
  clearSession()
  window.dispatchEvent(new Event(AUTH_SESSION_EXPIRED_EVENT))
}
