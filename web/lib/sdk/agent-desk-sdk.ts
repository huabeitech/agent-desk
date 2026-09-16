import type {
  AgentDeskConfig,
  AgentDeskWidget,
  SupportChatRuntimeConfig,
} from "./config-types"

type NormalizedAgentDeskConfig = AgentDeskConfig & {
  baseUrl: string
  channelId: string
  language: string
  position: "left" | "right"
  themeColor: string
  width: string
}

type WidgetState = {
  button: HTMLButtonElement | null
  frame: HTMLIFrameElement | null
  frameLoaded: boolean
  frameReady: boolean
  initSent: boolean
  isOpen: boolean
  isMaximized: boolean
  configLoading: boolean
  frameHideTimer: number | null
  frameDestroyTimer: number | null
  config: NormalizedAgentDeskConfig | null
  frameConfig: SupportChatRuntimeConfig | null
  frameUrl: URL | null
  animationDuration: number
  listenerBound?: boolean
  launcherPos: { x: number; y: number } | null
  launcherDragging: boolean
  launcherDragStart: { x: number; y: number; px: number; py: number } | null
  launcherMoved: boolean
  launcherDocked: boolean
  launcherDockSide: "left" | "right" | null
  launcherExpanded: boolean
  launcherHoverTimer: number | null
  launcherResizeHandler: (() => void) | null
  launcherContent: HTMLSpanElement | null
  dockLabel: HTMLSpanElement | null
}

type WidgetConfigResponse = {
  success?: boolean
  data?: Partial<Pick<
    AgentDeskConfig,
    "title" | "subtitle" | "themeColor" | "position" | "width"
  >>
}

type PublicConfigResponse = {
  success?: boolean
  data?: {
    language?: string
  }
}

function normalizeWidgetLanguage(language: string | undefined) {
  return String(language || "").toLowerCase().startsWith("en") ? "en-US" : "zh-CN"
}

function getDefaultWidgetTitle(config?: NormalizedAgentDeskConfig | null) {
  return normalizeWidgetLanguage(config?.language) === "en-US" ? "Support" : "\u5728\u7ebf\u5ba2\u670d"
}

function getLauncherText(config?: NormalizedAgentDeskConfig | null) {
  return normalizeWidgetLanguage(config?.language) === "en-US" ? "Support" : "\u5ba2\u670d"
}

type FrameMessage =
  | { type: "agent-desk:init"; payload: SupportChatRuntimeConfig }
  | { type: "agent-desk:open" }
  | { type: "agent-desk:minimize" }
  | { type: "agent-desk:maximized"; payload: { isMaximized: boolean } }

(function () {
  const DEFAULT_CONFIG: Pick<
    NormalizedAgentDeskConfig,
    "language" | "position" | "themeColor" | "width"
  > = {
    language: "zh-CN",
    position: "right",
    themeColor: "#0f6cbd",
    width: "380px",
  }

  // 启动器（悬浮按钮）尺寸与交互常量
  const LAUNCHER_SIZE = 64
  const LAUNCHER_MARGIN = 24
  // 移动距离超过该阈值视为拖拽而非点击
  const DRAG_THRESHOLD = 6
  // 距屏幕左/右边缘该距离内释放时自动吸附
  const EDGE_SNAP_DISTANCE = 80
  // 吸附后半隐藏状态可见宽度
  const DOCK_VISIBLE_PX = 16
  // 半隐藏状态在鼠标离开后延迟收回的时长
  const AUTO_COLLAPSE_DELAY = 1500
  // 拖拽后位置持久化 key（按 channelId 区分，避免不同渠道复用同一坐标）
  const LAUNCHER_POS_STORAGE_KEY_PREFIX = "agent-desk:launcher-pos:"

  const existingState = window.__CS_AI_AGENT_WIDGET_STATE__ as WidgetState | undefined
  const state: WidgetState =
    existingState || {
      button: null,
      frame: null,
      frameLoaded: false,
      frameReady: false,
      initSent: false,
      isOpen: false,
      isMaximized: false,
      configLoading: false,
      frameHideTimer: null,
      frameDestroyTimer: null,
      config: null,
      frameConfig: null,
      frameUrl: null,
      animationDuration: 260,
      launcherPos: null,
      launcherDragging: false,
      launcherDragStart: null,
      launcherMoved: false,
      launcherDocked: false,
      launcherDockSide: null,
      launcherExpanded: false,
      launcherHoverTimer: null,
      launcherResizeHandler: null,
      launcherContent: null,
      dockLabel: null,
    }
  if (!existingState) {
    window.__CS_AI_AGENT_WIDGET_STATE__ = state
  }

  function normalizeConfig(config?: AgentDeskConfig): NormalizedAgentDeskConfig {
    const merged: Record<string, unknown> = { ...DEFAULT_CONFIG, ...(config || {}) }
    merged.baseUrl = String(merged.baseUrl || window.location.origin).replace(/\/$/, "")
    if (merged.apiBaseUrl) {
      merged.apiBaseUrl = String(merged.apiBaseUrl).replace(/\/$/, "")
    } else {
      delete merged.apiBaseUrl
    }
    merged.channelId = String(merged.channelId || "")
    merged.language = normalizeWidgetLanguage(String(merged.language || "zh-CN"))
    if (merged.externalId) {
      merged.externalId = String(merged.externalId)
    }
    if (typeof merged.getUserToken !== "function") {
      delete merged.getUserToken
    }
    return merged as NormalizedAgentDeskConfig
  }

  function resolveWidgetBaseUrl(config: NormalizedAgentDeskConfig) {
    const currentScript = document.currentScript as HTMLScriptElement | null
    if (currentScript?.src) {
      return currentScript.src.replace(/\/sdk\/agent-desk-sdk\.min\.js(?:\?.*)?$/, "")
    }
    return String(config.widgetBaseUrl || config.baseUrl || window.location.origin).replace(/\/$/, "")
  }

  function createFrameUrl(config: NormalizedAgentDeskConfig, userToken: string) {
    const widgetBaseUrl = resolveWidgetBaseUrl(config)
    const frameUrl = new URL(`${widgetBaseUrl}/support/chat/`)
    frameUrl.searchParams.set("channelId", config.channelId)
    frameUrl.searchParams.set("baseUrl", config.baseUrl)
    if (config.apiBaseUrl) frameUrl.searchParams.set("apiBaseUrl", config.apiBaseUrl)
    if (config.externalId) frameUrl.searchParams.set("externalId", config.externalId)
    if (config.externalName) frameUrl.searchParams.set("externalName", config.externalName)
    if (userToken) frameUrl.searchParams.set("userToken", userToken)
    return frameUrl
  }

  function createFrameConfig(
    config: NormalizedAgentDeskConfig,
    userToken: string
  ): SupportChatRuntimeConfig {
    const { getUserToken: _getUserToken, ...payload } = config
    if (userToken) {
      return { ...payload, userToken }
    }
    return payload
  }

  function resolveUserToken() {
    const config = state.config
    if (typeof config?.getUserToken !== "function") {
      return Promise.resolve("")
    }
    try {
      return Promise.resolve(config.getUserToken()).then((token) =>
        String(token || "").trim()
      )
    } catch (error) {
      return Promise.reject(error)
    }
  }

  function prepareFrameUrl() {
    return resolveUserToken().then((userToken) => {
      if (!state.config) {
        throw new Error("channelId is required")
      }
      state.frameUrl = createFrameUrl(state.config, userToken)
      state.frameConfig = createFrameConfig(state.config, userToken)
      return state.frameUrl
    })
  }

  function mergeWidgetConfig(
    config: NormalizedAgentDeskConfig,
    remoteConfig?: WidgetConfigResponse["data"]
  ) {
    if (!remoteConfig) {
      return config
    }
    const merged: NormalizedAgentDeskConfig = { ...config }
    const remoteKeys = ["title", "subtitle", "themeColor", "position", "width"] as const
    remoteKeys.forEach((key) => {
      const value = remoteConfig[key]
      if (value !== undefined && value !== null) {
        ;(merged[key] as typeof value) = value
      }
    })
    return merged
  }

  function fetchWidgetConfig(config: NormalizedAgentDeskConfig) {
    const baseUrl = String(config.apiBaseUrl || config.baseUrl || "").replace(/\/$/, "")
    if (!baseUrl || !config.channelId || typeof fetch !== "function") {
      return Promise.resolve(config)
    }
    const url = `${baseUrl}/api/channel/config?channelId=${encodeURIComponent(config.channelId)}`
    return fetch(url, {
      method: "GET",
      cache: "no-store",
      headers: {
        "X-Channel-Id": config.channelId,
      },
    })
      .then((response) => response.json() as Promise<WidgetConfigResponse>)
      .then((payload) => {
        if (!payload || payload.success === false) {
          return config
        }
        return mergeWidgetConfig(config, payload.data || {})
      })
      .catch(() => config)
  }

  function fetchPublicConfig(config: NormalizedAgentDeskConfig) {
    const baseUrl = String(config.apiBaseUrl || config.baseUrl || "").replace(/\/$/, "")
    if (!baseUrl || typeof fetch !== "function") {
      return Promise.resolve(config)
    }
    return fetch(`${baseUrl}/api/config`, {
      method: "GET",
      cache: "no-store",
    })
      .then((response) => response.json() as Promise<PublicConfigResponse>)
      .then((payload) => {
        if (!payload || payload.success === false) {
          return config
        }
        return normalizeConfig({
          ...config,
          language: payload.data?.language || config.language,
        })
      })
      .catch(() => config)
  }

  function clearFrameTimers() {
    if (state.frameHideTimer) {
      window.clearTimeout(state.frameHideTimer)
      state.frameHideTimer = null
    }
    if (state.frameDestroyTimer) {
      window.clearTimeout(state.frameDestroyTimer)
      state.frameDestroyTimer = null
    }
  }

  function applyFrameLayout() {
    const frame = state.frame
    const config = state.config
    if (!frame || !config) {
      return
    }

    frame.style.position = "fixed"
    frame.style.border = "0"
    frame.style.overflow = "hidden"
    frame.style.background = "#fff"
    frame.style.zIndex = "2147483000"
    frame.style.boxShadow = "0 28px 80px rgba(15, 35, 65, 0.28)"
    frame.style.willChange = "top,right,bottom,left,width,height,opacity,transform,border-radius"
    frame.style.transition =
      "top 260ms cubic-bezier(0.22, 1, 0.36, 1), right 260ms cubic-bezier(0.22, 1, 0.36, 1), bottom 260ms cubic-bezier(0.22, 1, 0.36, 1), left 260ms cubic-bezier(0.22, 1, 0.36, 1), width 260ms cubic-bezier(0.22, 1, 0.36, 1), height 260ms cubic-bezier(0.22, 1, 0.36, 1), opacity 220ms ease, transform 260ms cubic-bezier(0.22, 1, 0.36, 1), border-radius 260ms cubic-bezier(0.22, 1, 0.36, 1), box-shadow 260ms ease"
    frame.style.transformOrigin =
      config.position === "left" ? "left bottom" : "right bottom"

    if (state.isMaximized) {
      frame.style.top = "max(12px, env(safe-area-inset-top))"
      frame.style.right = "max(12px, env(safe-area-inset-right))"
      frame.style.bottom = "max(12px, env(safe-area-inset-bottom))"
      frame.style.left = "max(12px, env(safe-area-inset-left))"
      frame.style.width =
        "calc(100vw - max(12px, env(safe-area-inset-left)) - max(12px, env(safe-area-inset-right)))"
      frame.style.maxWidth = "none"
      frame.style.height =
        "calc(100dvh - max(12px, env(safe-area-inset-top)) - max(12px, env(safe-area-inset-bottom)))"
      frame.style.borderRadius = "16px"
      return
    }

    frame.style.top = ""
    frame.style.bottom = "max(88px, calc(72px + env(safe-area-inset-bottom)))"
    frame.style.right = config.position === "left" ? "" : "max(12px, env(safe-area-inset-right))"
    frame.style.left = config.position === "left" ? "max(12px, env(safe-area-inset-left))" : ""
    frame.style.width = config.width || "380px"
    frame.style.maxWidth =
      "calc(100vw - max(12px, env(safe-area-inset-left)) - max(12px, env(safe-area-inset-right)))"
    frame.style.height =
      "min(760px, calc(100dvh - max(104px, calc(88px + env(safe-area-inset-bottom))) - max(12px, env(safe-area-inset-top))))"
    frame.style.borderRadius = "18px"
  }

  function postToFrame(message: FrameMessage) {
    if (!state.frame?.contentWindow || !state.frameUrl) {
      return
    }
    try {
      state.frame.contentWindow.postMessage(message, state.frameUrl.origin)
    } catch (error) {
      console.error("[agent-desk-widget] postMessage failed", error)
    }
  }

  function flushFrameState() {
    if (!state.frame || !state.frameLoaded || !state.frameReady || !state.config) {
      return
    }

    if (!state.initSent) {
      state.initSent = true
      postToFrame({
        type: "agent-desk:init",
        payload: state.frameConfig || createFrameConfig(state.config, ""),
      })
    }

    postToFrame({ type: state.isOpen ? "agent-desk:open" : "agent-desk:minimize" })
    postToFrame({
      type: "agent-desk:maximized",
      payload: { isMaximized: state.isMaximized },
    })
  }

  function syncFrameVisibility() {
    const frame = state.frame
    if (!frame) {
      return
    }
    clearFrameTimers()
    applyFrameLayout()
    frame.style.display = "block"

    if (state.isOpen) {
      frame.style.visibility = "visible"
      frame.style.pointerEvents = "auto"
      state.frameHideTimer = window.setTimeout(() => {
        if (!state.frame) {
          return
        }
        state.frame.style.opacity = "1"
        state.frame.style.transform = "translate3d(0, 0, 0) scale(1)"
      }, 16)
      flushFrameState()
      return
    }

    frame.style.pointerEvents = "none"
    frame.style.opacity = "0"
    frame.style.transform = state.isMaximized
      ? "translate3d(0, 10px, 0) scale(0.985)"
      : "translate3d(0, 16px, 0) scale(0.96)"
    state.frameHideTimer = window.setTimeout(() => {
      if (!state.frame || state.isOpen) {
        return
      }
      state.frame.style.visibility = "hidden"
    }, state.animationDuration)
    flushFrameState()
  }

  function destroyFrame() {
    if (!state.frame) {
      return
    }
    clearFrameTimers()
    state.frame.style.pointerEvents = "none"
    state.frame.style.opacity = "0"
    state.frame.style.transform = "translate3d(0, 18px, 0) scale(0.94)"
    state.frame.style.visibility = "hidden"
    state.frameDestroyTimer = window.setTimeout(() => {
      if (!state.frame) {
        return
      }
      if (state.frame.parentNode) {
        state.frame.parentNode.removeChild(state.frame)
      }
      state.frame = null
      state.frameLoaded = false
      state.frameReady = false
      state.initSent = false
      state.isOpen = false
      state.isMaximized = false
      clearFrameTimers()
    }, state.animationDuration)
  }

  function createFrame() {
    if (state.frame) {
      return state.frame
    }
    if (!state.frameUrl || !state.config) {
      return null
    }

    state.frame = document.createElement("iframe")
    state.frame.dataset.agentDeskWidget = "frame"
    state.frame.title = state.config.title || getDefaultWidgetTitle(state.config)
    state.frame.src = state.frameUrl.toString()
    applyFrameLayout()
    state.frame.style.display = "block"
    state.frame.style.visibility = "hidden"
    state.frame.style.pointerEvents = "none"
    state.frame.style.opacity = "0"
    state.frame.style.transform = "translate3d(0, 18px, 0) scale(0.96)"
    state.frame.addEventListener("load", () => {
      state.frameLoaded = true
      syncFrameVisibility()
    })

    document.body.appendChild(state.frame)
    return state.frame
  }

  function handleWindowMessage(event: MessageEvent) {
    if (!state.frame || event.source !== state.frame.contentWindow) {
      return
    }

    const data = (event.data || {}) as { type?: string }
    if (data.type === "agent-desk:ready") {
      state.frameReady = true
      flushFrameState()
      return
    }

    if (data.type === "agent-desk:request-minimize") {
      state.isOpen = false
      syncFrameVisibility()
      return
    }

    if (data.type === "agent-desk:request-close") {
      destroyFrame()
      return
    }

    if (data.type === "agent-desk:request-toggle-maximize") {
      state.isMaximized = !state.isMaximized
      syncFrameVisibility()
    }
  }

  function getViewportSize() {
    const w = typeof window.innerWidth === "number" ? window.innerWidth : 0
    const h = typeof window.innerHeight === "number" ? window.innerHeight : 0
    return { width: w, height: h }
  }

  // 持久化/读取悬浮按钮位置（localStorage）。
  // 不同 channelId 用不同 key，避免多渠道互相覆盖。
  function launcherPosStorageKey(): string | null {
    const cfg = state.config
    const id = cfg?.channelId
    if (!id) {
      return null
    }
    return `${LAUNCHER_POS_STORAGE_KEY_PREFIX}${id}`
  }

  type LauncherPersistedState = {
    x: number
    y: number
    docked: boolean
    dockSide: "left" | "right" | null
  }

  function saveLauncherState(
    pos: { x: number; y: number },
    docked: boolean,
    dockSide: "left" | "right" | null,
  ) {
    if (typeof window === "undefined") {
      return
    }
    try {
      const key = launcherPosStorageKey()
      if (!key) {
        return
      }
      const storage = window.localStorage
      if (!storage) {
        return
      }
      storage.setItem(
        key,
        JSON.stringify({
          x: pos.x,
          y: pos.y,
          docked: !!docked,
          dockSide: dockSide ?? null,
        }),
      )
    } catch {
      // 无痕模式或被禁用时静默忽略
    }
  }

  function loadLauncherState(): LauncherPersistedState | null {
    if (typeof window === "undefined") {
      return null
    }
    try {
      const key = launcherPosStorageKey()
      if (!key) {
        return null
      }
      const storage = window.localStorage
      if (!storage) {
        return null
      }
      const raw = storage.getItem(key)
      if (!raw) {
        return null
      }
      const parsed = JSON.parse(raw) as {
        x?: unknown
        y?: unknown
        docked?: unknown
        dockSide?: unknown
      }
      const x = Number(parsed.x)
      const y = Number(parsed.y)
      if (!Number.isFinite(x) || !Number.isFinite(y)) {
        return null
      }
      const docked = parsed.docked === true
      const dockSide =
        parsed.dockSide === "left" || parsed.dockSide === "right"
          ? parsed.dockSide
          : null
      return { x, y, docked, dockSide }
    } catch {
      return null
    }
  }

  // 校验坐标是否在当前视口可见区内（按钮整体在屏幕内）。
  function isLauncherPosVisible(pos: { x: number; y: number }) {
    const vp = getViewportSize()
    if (vp.width <= 0 || vp.height <= 0) {
      return false
    }
    return (
      pos.x >= 0 &&
      pos.y >= 0 &&
      pos.x + LAUNCHER_SIZE <= vp.width &&
      pos.y + LAUNCHER_SIZE <= vp.height
    )
  }

  // 初始化启动器位置：优先恢复上次记录的位置，否则按 config.position 默认放置。
  function initLauncherPosition() {
    if (state.launcherPos) {
      return
    }
    const vp = getViewportSize()
    // 视口尺寸尚未就绪（脚本在 head 中同步加载、body 未完成布局）时
    // 暂不写入坐标，避免落到 (0,0) 左上角；由 ensureLauncherPosition
    // 在首次布局/拖拽时补算。
    if (vp.width <= 0 || vp.height <= 0) {
      return
    }
    // 优先恢复上次记录的位置与停靠状态，但要校验仍在当前屏幕可见区内
    // （窗口尺寸变化/外接显示器变更可能导致旧坐标越界）。
    const saved = loadLauncherState()
    if (saved && isLauncherPosVisible(saved)) {
      state.launcherPos = { x: saved.x, y: saved.y }
      state.launcherDocked = saved.docked
      state.launcherDockSide = saved.docked ? saved.dockSide : null
      state.launcherExpanded = false
      return
    }
    const side = state.config?.position === "left" ? "left" : "right"
    state.launcherPos = {
      x:
        side === "left"
          ? LAUNCHER_MARGIN
          : Math.max(0, vp.width - LAUNCHER_SIZE - LAUNCHER_MARGIN),
      // 默认放屏幕右下角（position=left 则左下角）。
      y: Math.max(0, vp.height - LAUNCHER_SIZE - LAUNCHER_MARGIN),
    }
    state.launcherDocked = false
    state.launcherDockSide = null
    state.launcherExpanded = false
  }

  // 确保坐标已初始化：若此前因视口尺寸为 0 而跳过，则在可读视口时补算。
  function ensureLauncherPosition() {
    if (state.launcherPos) {
      return
    }
    initLauncherPosition()
  }

  function clampLauncherPos(pos: { x: number; y: number }) {
    const vp = getViewportSize()
    return {
      x: Math.max(0, Math.min(Math.max(0, vp.width - LAUNCHER_SIZE), pos.x)),
      y: Math.max(0, Math.min(Math.max(0, vp.height - LAUNCHER_SIZE), pos.y)),
    }
  }

  // 应用启动器布局：top/left 像素定位 + 停靠/展开状态下的 transform
  function applyLauncherLayout() {
    const button = state.button
    if (!button) {
      return
    }
    // 视口可读但尚未初始化坐标时，先补算再定位。
    ensureLauncherPosition()
    if (!state.launcherPos) {
      return
    }
    button.style.left = `${state.launcherPos.x}px`
    button.style.top = `${state.launcherPos.y}px`

    const isDockedCollapsed =
      state.launcherDocked &&
      !state.launcherExpanded &&
      !state.launcherDragging

    if (isDockedCollapsed) {
      const offset = LAUNCHER_SIZE - DOCK_VISIBLE_PX
      if (state.launcherDockSide === "right") {
        button.style.transform = `translate3d(${offset}px, 0, 0)`
      } else if (state.launcherDockSide === "left") {
        button.style.transform = `translate3d(${-offset}px, 0, 0)`
      } else {
        button.style.transform = "translate3d(0, 0, 0)"
      }
    } else {
      button.style.transform = "translate3d(0, 0, 0)"
    }

    // 切换停靠窄条文字与常规内容（图标+主文字）的显隐。
    if (state.dockLabel) {
      const showDockLabel = isDockedCollapsed
      state.dockLabel.style.display = showDockLabel ? "flex" : "none"
      // 外露 16px 在按钮的某一侧：右吸附时外露在按钮左端 16px，
      // 文字须靠左对齐；左吸附时外露在按钮右端 16px，文字须靠右对齐。
      // 否则文字居中落在按钮中段，会随按钮一起被 transform 移出视口。
      if (showDockLabel) {
        if (state.launcherDockSide === "right") {
          state.dockLabel.style.margin = "0 auto 0 0"
        } else if (state.launcherDockSide === "left") {
          state.dockLabel.style.margin = "0 0 0 auto"
        } else {
          state.dockLabel.style.margin = "0 auto"
        }
      }
    }
    if (state.launcherContent) {
      const showContent = !isDockedCollapsed
      state.launcherContent.style.display = showContent ? "flex" : "none"
    }
  }

  // 拖拽结束时判断是否吸附到屏幕左/右边缘
  function checkEdgeSnap() {
    if (!state.launcherPos) {
      return
    }
    const vp = getViewportSize()
    const pos = state.launcherPos
    const distLeft = pos.x
    const distRight = vp.width - (pos.x + LAUNCHER_SIZE)

    if (distRight <= EDGE_SNAP_DISTANCE && distRight <= distLeft) {
      state.launcherPos = {
        x: Math.max(0, vp.width - LAUNCHER_SIZE),
        y: pos.y,
      }
      state.launcherDocked = true
      state.launcherDockSide = "right"
      state.launcherExpanded = false
    } else if (distLeft <= EDGE_SNAP_DISTANCE) {
      state.launcherPos = { x: 0, y: pos.y }
      state.launcherDocked = true
      state.launcherDockSide = "left"
      state.launcherExpanded = false
    } else {
      state.launcherDocked = false
      state.launcherDockSide = null
      state.launcherExpanded = false
    }
    applyLauncherLayout()
    // 持久化最终位置与停靠状态，下次启动时恢复。
    if (state.launcherPos) {
      saveLauncherState(
        state.launcherPos,
        state.launcherDocked,
        state.launcherDockSide,
      )
    }
  }

  function attachLauncherInteractions(button: HTMLButtonElement) {
    button.addEventListener("pointerdown", (event: PointerEvent) => {
      if (event.button !== 0 && event.pointerType === "mouse") {
        return
      }
      // 拖拽起手前确保坐标已初始化，避免视口尺寸为 0 时跳变。
      ensureLauncherPosition()
      if (!state.launcherPos) {
        return
      }
      try {
        button.setPointerCapture(event.pointerId)
      } catch {
        // 部分环境（如测试沙盒）不支持 setPointerCapture，忽略即可
      }
      state.launcherDragging = true
      state.launcherMoved = false
      state.launcherDragStart = {
        x: event.clientX,
        y: event.clientY,
        px: state.launcherPos.x,
        py: state.launcherPos.y,
      }
      // 拖拽期间禁用过渡，避免跟随指针时出现延迟
      button.style.transition = "none"
    })

    button.addEventListener("pointermove", (event: PointerEvent) => {
      if (!state.launcherDragging || !state.launcherDragStart || !state.launcherPos) {
        return
      }
      const dx = event.clientX - state.launcherDragStart.x
      const dy = event.clientY - state.launcherDragStart.y
      if (!state.launcherMoved && Math.hypot(dx, dy) > DRAG_THRESHOLD) {
        state.launcherMoved = true
        // 拖拽开始时退出半隐藏状态以便用户看清按钮
        if (state.launcherDocked && !state.launcherExpanded) {
          state.launcherExpanded = true
          applyLauncherLayout()
        }
      }
      if (state.launcherMoved) {
        state.launcherPos = clampLauncherPos({
          x: state.launcherDragStart.px + dx,
          y: state.launcherDragStart.py + dy,
        })
        applyLauncherLayout()
      }
    })

    const endDrag = (event: PointerEvent) => {
      if (!state.launcherDragging) {
        return
      }
      state.launcherDragging = false
      state.launcherDragStart = null
      button.style.transition = ""
      try {
        button.releasePointerCapture(event.pointerId)
      } catch {
        // 同上，忽略不支持的环境
      }
      if (state.launcherMoved) {
        // 拖拽结束，做边缘吸附与半隐藏
        checkEdgeSnap()
      }
    }
    button.addEventListener("pointerup", endDrag)
    button.addEventListener("pointercancel", endDrag)

    button.addEventListener("pointerenter", () => {
      if (!state.launcherDocked || state.launcherDragging) {
        return
      }
      if (state.launcherHoverTimer) {
        window.clearTimeout(state.launcherHoverTimer)
        state.launcherHoverTimer = null
      }
      if (!state.launcherExpanded) {
        state.launcherExpanded = true
        applyLauncherLayout()
      }
    })

    button.addEventListener("pointerleave", () => {
      if (!state.launcherDocked || !state.launcherExpanded || state.launcherDragging) {
        return
      }
      if (state.launcherHoverTimer) {
        window.clearTimeout(state.launcherHoverTimer)
      }
      state.launcherHoverTimer = window.setTimeout(() => {
        state.launcherExpanded = false
        state.launcherHoverTimer = null
        applyLauncherLayout()
      }, AUTO_COLLAPSE_DELAY)
    })
  }

  function handleLauncherResize() {
    ensureLauncherPosition()
    if (!state.launcherPos) {
      return
    }
    state.launcherPos = clampLauncherPos(state.launcherPos)
    if (state.launcherDocked) {
      const vp = getViewportSize()
      if (state.launcherDockSide === "right") {
        state.launcherPos = {
          x: Math.max(0, vp.width - LAUNCHER_SIZE),
          y: state.launcherPos.y,
        }
      } else if (state.launcherDockSide === "left") {
        state.launcherPos = { x: 0, y: state.launcherPos.y }
      }
    }
    applyLauncherLayout()
  }

  function createLauncher() {
    if (state.button) {
      return state.button
    }

    const config = state.config
    if (!config) {
      return null
    }
    const button = document.createElement("button")
    const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg")
    const iconPaths = [
      "M3 11a9 9 0 1 1 18 0",
      "M3 11h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z",
      "M21 11h-3a2 2 0 0 0-2 2v3a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2z",
      "M21 16v2a4 4 0 0 1-4 4h-5",
    ]
    const text = document.createElement("span")
    const content = document.createElement("span") // 常规态容器：图标 + 主文字
    const dockLabel = document.createElement("span") // 停靠半隐藏态外露窄条文字
    button.type = "button"
    button.dataset.agentDeskWidget = "launcher"
    button.setAttribute("aria-label", config.title || getDefaultWidgetTitle(config))
    icon.setAttribute("viewBox", "0 0 24 24")
    icon.setAttribute("fill", "none")
    icon.setAttribute("stroke", "currentColor")
    icon.setAttribute("stroke-width", "2")
    icon.setAttribute("stroke-linecap", "round")
    icon.setAttribute("stroke-linejoin", "round")
    icon.setAttribute("aria-hidden", "true")
    icon.style.width = "24px"
    icon.style.height = "24px"
    icon.style.flex = "0 0 auto"
    iconPaths.forEach((pathData) => {
      const path = document.createElementNS("http://www.w3.org/2000/svg", "path")
      path.setAttribute("d", pathData)
      icon.appendChild(path)
    })
    text.textContent = getLauncherText(config)
    text.style.display = "block"
    content.style.display = "flex"
    content.style.flexDirection = "column"
    content.style.alignItems = "center"
    content.style.justifyContent = "center"
    content.style.gap = "4px"
    content.style.width = "100%"
    content.style.height = "100%"
    content.appendChild(icon)
    content.appendChild(text)

    // 停靠半隐藏态外露窄条：竖排显示“客服/Support”。
    // 外露宽度 DOCK_VISIBLE_PX(16px)，竖排文字字号 12px，每字一行，刚好落入窄条。
    dockLabel.textContent = getLauncherText(config)
    dockLabel.style.display = "none"
    dockLabel.style.flexDirection = "column"
    dockLabel.style.alignItems = "center"
    dockLabel.style.justifyContent = "center"
    dockLabel.style.width = `${DOCK_VISIBLE_PX}px`
    dockLabel.style.height = "100%"
    dockLabel.style.margin = "0 auto"
    dockLabel.style.padding = "0"
    dockLabel.style.fontSize = "12px"
    dockLabel.style.lineHeight = "1"
    dockLabel.style.fontWeight = "600"
    dockLabel.style.letterSpacing = "1px"
    dockLabel.style.color = "#fff"
    dockLabel.style.whiteSpace = "nowrap"
    dockLabel.style.writingMode = "vertical-rl"
    dockLabel.style.textOrientation = "upright"

    button.style.position = "fixed"
    button.style.top = "0px"
    button.style.left = "0px"
    button.style.right = ""
    button.style.bottom = ""
    button.style.zIndex = "2147483000"
    button.style.display = "inline-flex"
    button.style.flexDirection = "column"
    button.style.alignItems = "center"
    button.style.justifyContent = "center"
    button.style.gap = "0"
    button.style.width = `${LAUNCHER_SIZE}px`
    button.style.height = `${LAUNCHER_SIZE}px`
    button.style.border = "0"
    button.style.borderRadius = "999px"
    button.style.padding = "0"
    button.style.background = config.themeColor || "#0f6cbd"
    button.style.color = "#fff"
    button.style.font = "600 13px/1 sans-serif"
    button.style.boxShadow = "0 18px 40px rgba(15, 35, 65, 0.24)"
    button.style.cursor = "pointer"
    button.style.transition =
      "transform 240ms cubic-bezier(0.22, 1, 0.36, 1), box-shadow 240ms ease, background 240ms ease"
    // 触摸拖拽时禁用默认行为（如文本选区、长按菜单）
    button.style.touchAction = "none"
    button.appendChild(content)
    button.appendChild(dockLabel)
    state.launcherContent = content
    state.dockLabel = dockLabel

    initLauncherPosition()
    applyLauncherLayout()
    attachLauncherInteractions(button)

    button.addEventListener("click", () => {
      // 拖拽刚结束，吞掉本次 click，避免误触打开窗口
      if (state.launcherMoved) {
        state.launcherMoved = false
        return
      }
      if (state.isOpen) {
        state.isOpen = false
        syncFrameVisibility()
        return
      }

      void openWidget()
    })

    if (!state.launcherResizeHandler) {
      state.launcherResizeHandler = handleLauncherResize
      window.addEventListener("resize", state.launcherResizeHandler)
      window.addEventListener("orientationchange", state.launcherResizeHandler)
    }

    document.body.appendChild(button)
    state.button = button

    // 兜底：脚本可能在 body 完成布局前就执行（如 head 中同步加载），
    // 导致首次 initLauncherPosition 因视口尺寸为 0 而跳过。这里在下一帧
    // 以及 window load 后各补算一次，确保坐标落在右边缘中部而非左上角。
    const scheduleRelayout = (handler: () => void) => {
      if (typeof window.requestAnimationFrame === "function") {
        window.requestAnimationFrame(() => {
          try {
            handler()
          } catch {
            // 忽略
          }
        })
      } else {
        window.setTimeout(handler, 0)
      }
    }
    scheduleRelayout(() => {
      ensureLauncherPosition()
      applyLauncherLayout()
    })
    if (document.readyState !== "complete") {
      const onLoad = () => {
        window.removeEventListener("load", onLoad)
        ensureLauncherPosition()
        applyLauncherLayout()
      }
      window.addEventListener("load", onLoad)
    }

    return button
  }

  function mount(config?: AgentDeskConfig) {
    const rawConfig = config || window.AgentDeskConfig || { channelId: "" }
    state.config = normalizeConfig(rawConfig)
    const widgetBaseUrl = resolveWidgetBaseUrl(state.config)
    if (!rawConfig.baseUrl) {
      state.config.baseUrl = widgetBaseUrl
    }
    if (!state.config.channelId) {
      console.error("[agent-desk-widget] channelId is required")
      return
    }

    state.configLoading = true
    fetchPublicConfig(state.config)
      .then((nextConfig) => fetchWidgetConfig(nextConfig))
      .then((nextConfig) => {
        state.configLoading = false
        state.config = normalizeConfig(nextConfig)
        if (state.button?.parentNode) {
          state.button.parentNode.removeChild(state.button)
          state.button = null
        }
        createLauncher()
      })
  }

  function destroy() {
    clearFrameTimers()
    if (state.launcherHoverTimer) {
      window.clearTimeout(state.launcherHoverTimer)
      state.launcherHoverTimer = null
    }
    if (state.launcherResizeHandler) {
      window.removeEventListener("resize", state.launcherResizeHandler)
      window.removeEventListener("orientationchange", state.launcherResizeHandler)
      state.launcherResizeHandler = null
    }
    if (state.frame?.parentNode) {
      state.frame.parentNode.removeChild(state.frame)
    }
    if (state.button?.parentNode) {
      state.button.parentNode.removeChild(state.button)
    }
    state.button = null
    state.frame = null
    state.frameLoaded = false
    state.frameReady = false
    state.initSent = false
    state.isOpen = false
    state.isMaximized = false
    state.configLoading = false
    state.frameConfig = null
    state.frameUrl = null
    state.launcherPos = null
    state.launcherDragging = false
    state.launcherDragStart = null
    state.launcherMoved = false
    state.launcherDocked = false
    state.launcherDockSide = null
    state.launcherExpanded = false
    state.launcherContent = null
    state.dockLabel = null
  }

  function openWidget() {
    return prepareFrameUrl()
      .then(() => {
        if (!state.frame) {
          createFrame()
        }
        if (!state.frame) {
          return
        }
        state.isOpen = true
        syncFrameVisibility()
      })
      .catch((error) => {
        console.error("[agent-desk-widget] open failed", error)
      })
  }

  window.AgentDeskWidget = {
    mount,
    destroy,
    open: () => openWidget(),
    close: () => {
      state.isOpen = false
      syncFrameVisibility()
    },
    getChatUrl: () => {
      if (!state.config) {
        mount(window.AgentDeskConfig || { channelId: "" })
      }
      if (!state.config?.channelId) {
        return Promise.reject(new Error("channelId is required"))
      }
      return prepareFrameUrl().then((frameUrl) => frameUrl.toString())
    },
  } satisfies AgentDeskWidget

  if (!state.listenerBound) {
    window.addEventListener("message", handleWindowMessage)
    state.listenerBound = true
  }

  if (window.AgentDeskConfig) {
    mount(window.AgentDeskConfig)
  }
})()
