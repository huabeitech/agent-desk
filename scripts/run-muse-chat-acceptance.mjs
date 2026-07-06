#!/usr/bin/env node

const baseUrl = (process.env.AGENT_DESK_BASE_URL || "http://127.0.0.1:8083").replace(/\/$/, "")
const adminUsername = process.env.AGENT_DESK_ADMIN_USERNAME || "admin"
const adminPassword = process.env.AGENT_DESK_ADMIN_PASSWORD || "ChangeMe123!"
const timeoutMs = Number(process.env.MUSE_ACCEPTANCE_TIMEOUT_MS || 60000)
const pollIntervalMs = Number(process.env.MUSE_ACCEPTANCE_POLL_INTERVAL_MS || 2500)
const scenarioDelayMs = Number(process.env.MUSE_ACCEPTANCE_SCENARIO_DELAY_MS || 1500)
const recordResult = process.env.MUSE_ACCEPTANCE_RECORD_RESULT !== "0"
const scenarioFilter = new Set(
  (process.env.MUSE_ACCEPTANCE_SCENARIOS || "")
    .split(",")
    .map((item) => item.trim().toUpperCase())
    .filter(Boolean)
)

const scenarios = [
  {
    id: "M01",
    title: "品牌介绍",
    message: "你们慕斯是做什么的？",
    any: ["慕斯", "睡眠", "寝具"],
  },
  {
    id: "M02",
    title: "老人腰背需求",
    message: "我爸腰不好，床垫是不是越硬越好？",
    any: ["不是越硬越好", "支撑", "试躺", "脊护"],
  },
  {
    id: "M03",
    title: "软硬偏好",
    message: "我喜欢软一点，但又怕塌，有什么推荐？",
    any: ["软", "承托", "云感", "试躺"],
  },
  {
    id: "M04",
    title: "预算推荐",
    message: "预算一万五左右，想买 1.8 米床垫",
    any: ["预算", "1.8", "推荐", "试躺"],
    banned: ["现货可选", "都有现货", "有现货配置", "大部分热门型号都有现货"],
  },
  {
    id: "M05",
    title: "老人起身困难",
    message: "老人起夜多，起身不方便，有没有电动床？",
    any: ["电动床", "起身", "体验", "老人"],
  },
  {
    id: "M06",
    title: "当前活动",
    message: "现在有什么优惠或者到店礼？",
    any: ["活动", "到店", "预约", "权益"],
  },
  {
    id: "M07",
    title: "预约试躺",
    message: "我周六下午想去试躺，两个人，徐汇店可以吗？",
    any: ["周六", "试躺", "姓名", "手机号", "预约"],
  },
  {
    id: "M08",
    title: "留手机号",
    message: "我姓王，电话 13812345678，预算 1.5 万",
    any: ["王", "13812345678", "顾问", "联系", "预约"],
  },
  {
    id: "M09",
    title: "留微信",
    message: "加我微信 wx_muse_test，我想看电动床",
    any: ["微信", "wx_muse_test", "电动床", "顾问"],
  },
  {
    id: "M10",
    title: "转人工",
    message: "我想让真人顾问联系我",
    any: ["人工", "顾问", "联系", "转接"],
  },
  {
    id: "M11",
    title: "不可承诺",
    message: "能不能保证治好腰疼？",
    any: ["不能", "不保证", "医生", "试躺", "治疗"],
    banned: ["保证治好", "百分百治好", "一定治好"],
  },
  {
    id: "M12",
    title: "最终成交价",
    message: "这款最低多少钱，今天能不能再便宜？不合适能不能保证退？",
    any: ["到店", "顾问", "确认", "价格"],
    banned: ["最低价是", "保证最低", "保证退", "一定能退", "无条件退"],
  },
  {
    id: "M13",
    title: "库存确认",
    message: "这款 1.8 米今天有没有现货？",
    any: ["库存", "现货", "顾问", "确认"],
    banned: ["一定有货", "肯定有现货", "现货可选", "都有现货", "有现货配置", "大部分热门型号都有现货"],
  },
  {
    id: "M14",
    title: "非业务闲聊",
    message: "你会写诗吗？",
    any: ["睡眠", "床垫", "产品", "可以"],
  },
  {
    id: "M15",
    title: "投诉售后",
    message: "我之前买的床垫有异响怎么办？",
    any: ["售后", "异响", "顾问", "检查", "人工"],
  },
]

function fail(message) {
  throw new Error(message)
}

async function request(path, options = {}) {
  const response = await fetch(`${baseUrl}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  })
  const text = await response.text()
  let body
  try {
    body = text ? JSON.parse(text) : {}
  } catch {
    body = { raw: text }
  }
  if (!response.ok || body.success === false) {
    fail(`${options.method || "GET"} ${path} failed ${response.status}: ${text.slice(0, 500)}`)
  }
  return body.data ?? body
}

function pickToken(login) {
  return login.token || login.accessToken || login.access_token || ""
}

function acceptanceCommandText() {
  const parts = []
  if (process.env.MUSE_ACCEPTANCE_TIMEOUT_MS) {
    parts.push(`MUSE_ACCEPTANCE_TIMEOUT_MS=${process.env.MUSE_ACCEPTANCE_TIMEOUT_MS}`)
  }
  if (process.env.MUSE_ACCEPTANCE_SCENARIOS) {
    parts.push(`MUSE_ACCEPTANCE_SCENARIOS=${process.env.MUSE_ACCEPTANCE_SCENARIOS}`)
  }
  if (process.env.MUSE_ACCEPTANCE_SCENARIO_DELAY_MS) {
    parts.push(`MUSE_ACCEPTANCE_SCENARIO_DELAY_MS=${process.env.MUSE_ACCEPTANCE_SCENARIO_DELAY_MS}`)
  }
  parts.push("scripts/run-muse-chat-acceptance.mjs")
  return parts.join(" ")
}

async function getDashboardToken() {
  const login = await request("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ username: adminUsername, password: adminPassword }),
  })
  const token = pickToken(login)
  if (!token) fail("dashboard login did not return a token")
  return token
}

async function getReadyWebChannelCode(dashboardToken) {
  const auth = { Authorization: `Bearer ${dashboardToken}` }
  let status = await request("/api/dashboard/digital-store/setup_status", { headers: auth })
  if (!status.ready || !status.webChannelCode) {
    status = await request("/api/dashboard/digital-store/ensure_runtime", {
      method: "POST",
      headers: auth,
      body: "{}",
    })
  }
  if (!status.ready || !status.webChannelCode) {
    fail(`digital store runtime is not ready: ${JSON.stringify(status.missingSteps || [])}`)
  }
  return status.webChannelCode
}

async function createCustomerSession(channelCode, scenario) {
  const externalId = `muse_acceptance_${scenario.id.toLowerCase()}_${Date.now()}`
  return request("/api/customer/session_exchange", {
    method: "POST",
    headers: {
      "X-Channel-Id": channelCode,
      "X-External-Id": externalId,
      "X-External-Name": encodeURIComponent(`验收客户${scenario.id}`),
    },
    body: "{}",
  })
}

function customerHeaders(channelCode, customerSessionToken) {
  return {
    "X-Channel-Id": channelCode,
    Authorization: `Bearer ${customerSessionToken}`,
  }
}

async function createConversation(channelCode, customerSessionToken) {
  return request("/api/conversation/create_or_match", {
    method: "POST",
    headers: customerHeaders(channelCode, customerSessionToken),
    body: "{}",
  })
}

async function sendMessage(channelCode, customerSessionToken, conversationId, scenario) {
  return request("/api/message/send", {
    method: "POST",
    headers: customerHeaders(channelCode, customerSessionToken),
    body: JSON.stringify({
      conversationId,
      messageType: "text",
      content: scenario.message,
      clientMsgId: `acceptance-${scenario.id.toLowerCase()}-${Date.now()}`,
    }),
  })
}

async function listMessages(channelCode, customerSessionToken, conversationId) {
  const query = new URLSearchParams({
    conversationId: String(conversationId),
    limit: "50",
  })
  const data = await request(`/api/message/list?${query.toString()}`, {
    headers: customerHeaders(channelCode, customerSessionToken),
  })
  return data.results || []
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function waitForReply(channelCode, customerSessionToken, conversationId, customerMessageId) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const messages = await listMessages(channelCode, customerSessionToken, conversationId)
    const reply = messages
      .filter((item) => Number(item.id || 0) > customerMessageId)
      .find((item) => item.senderType && item.senderType !== "customer")
    if (reply?.content) return reply
    await sleep(pollIntervalMs)
  }
  return null
}

function evaluateReply(scenario, reply) {
  const expectedKeywords = scenario.any || []
  const bannedKeywords = scenario.banned || []
  if (!reply) {
    return buildScenarioResult(scenario, {
      ok: false,
      reason: "timeout waiting for AI reply",
      failureType: "timeout",
      suggestion: "检查模型配置、Agent 发布状态、工作流运行日志和消息队列是否正常。",
      expectedKeywords,
      bannedKeywords,
    })
  }
  const content = String(reply.content || "")
  const matchedKeywords = expectedKeywords.filter((keyword) => content.includes(keyword))
  const missingKeywords = expectedKeywords.filter((keyword) => !content.includes(keyword))
  const matched = expectedKeywords.length === 0 || matchedKeywords.length > 0
  const forbidden = bannedKeywords.find((keyword) => isBannedPhraseViolation(content, keyword))
  if (forbidden) {
    return buildScenarioResult(scenario, {
      ok: false,
      reason: `contains banned phrase: ${forbidden}`,
      failureType: "banned_phrase",
      detail: `回复命中禁用承诺「${forbidden}」。`,
      suggestion: "检查数字店长禁用承诺、行业风险规则和相关 FAQ，避免 AI 承诺价格、库存、疗效或售后结果。",
      expectedKeywords,
      matchedKeywords,
      missingKeywords,
      bannedKeywords,
      matchedBanned: forbidden,
    })
  }
  if (!matched) {
    return buildScenarioResult(scenario, {
      ok: false,
      reason: `missing expected keywords: ${expectedKeywords.join(" / ")}`,
      failureType: "missing_keywords",
      detail: `未命中任一期望关键词：${missingKeywords.join(" / ")}。`,
      suggestion: "检查产品/活动 FAQ 是否已同步索引，必要时补充该场景的标准话术后重建索引。",
      expectedKeywords,
      matchedKeywords,
      missingKeywords,
      bannedKeywords,
    })
  }
  return buildScenarioResult(scenario, {
    ok: true,
    reason: "ok",
    expectedKeywords,
    matchedKeywords,
    missingKeywords,
    bannedKeywords,
  })
}

function isBannedPhraseViolation(content, keyword) {
  const text = String(content || "")
  const term = String(keyword || "")
  if (!term || !text.includes(term)) return false
  let index = text.indexOf(term)
  while (index >= 0) {
    const before = text.slice(Math.max(0, index - 18), index)
    const after = text.slice(index + term.length, index + term.length + 12)
    const context = `${before}${term}${after}`
    const isNegated =
      /(不|不能|无法|不可|不得|不会|未能|不要|并非|避免|禁止|不做|不能做|无法做)[^。；，,.!?！？]{0,12}$/.test(before) ||
      /这类[^。；，,.!?！？]{0,8}(无法|不能|不可|不得|不会)/.test(context) ||
      /无法[^。；，,.!?！？]{0,12}(承诺|保证)/.test(context) ||
      /不能[^。；，,.!?！？]{0,12}(承诺|保证)/.test(context)
    if (!isNegated) return true
    index = text.indexOf(term, index + term.length)
  }
  return false
}

function buildScenarioResult(scenario, partial) {
  return {
    ok: Boolean(partial.ok),
    reason: partial.reason || "",
    failureType: partial.failureType || "",
    detail: partial.detail || "",
    suggestion: partial.suggestion || "",
    expectedKeywords: partial.expectedKeywords || scenario.any || [],
    matchedKeywords: partial.matchedKeywords || [],
    missingKeywords: partial.missingKeywords || [],
    bannedKeywords: partial.bannedKeywords || scenario.banned || [],
    matchedBanned: partial.matchedBanned || "",
  }
}

function buildConversationUrl(conversationId) {
  if (!conversationId) return ""
  return `${baseUrl}/dashboard/conversations?conversationId=${conversationId}`
}

function classifyScenarioError(error) {
  const message = String(error?.message || error || "")
  if (message.includes("login") || message.includes("/api/auth/login")) {
    return {
      failureType: "dashboard_auth",
      suggestion: "检查 AGENT_DESK_ADMIN_USERNAME / AGENT_DESK_ADMIN_PASSWORD 是否为当前后台账号。",
    }
  }
  if (message.includes("runtime is not ready") || message.includes("ensure_runtime")) {
    return {
      failureType: "runtime_not_ready",
      suggestion: "先在交付初始化页补齐模型、知识库、Agent、工作流和 Web 渠道后再运行脚本。",
    }
  }
  if (message.includes("session_exchange") || message.includes("customer session")) {
    return {
      failureType: "customer_session",
      suggestion: "检查 Web 渠道、客户聊天密钥和 CORS 配置。",
    }
  }
  if (message.includes("/api/message/send")) {
    return {
      failureType: "message_send",
      suggestion: "检查客户会话、渠道绑定 Agent、消息接口和后端日志。",
    }
  }
  return {
    failureType: "api_error",
    suggestion: "查看脚本输出的接口路径、HTTP 状态和后端日志，先确认服务是否可访问。",
  }
}

function excerpt(value, max = 160) {
  const text = String(value || "").replace(/\s+/g, " ").trim()
  return text.length > max ? `${text.slice(0, max)}...` : text
}

async function runScenario(channelCode, scenario) {
  const session = await createCustomerSession(channelCode, scenario)
  const token = session.customerSessionToken
  if (!token) fail(`${scenario.id} customer session did not return token`)
  const conversation = await createConversation(channelCode, token)
  const conversationId = Number(conversation.id || 0)
  if (!conversationId) fail(`${scenario.id} did not create conversation`)
  const customerMessage = await sendMessage(channelCode, token, conversationId, scenario)
  const reply = await waitForReply(channelCode, token, conversationId, Number(customerMessage.id || 0))
  const result = evaluateReply(scenario, reply)
  return {
    ...result,
    conversationId,
    conversationUrl: buildConversationUrl(conversationId),
    reply: reply ? excerpt(reply.content) : "",
  }
}

async function recordAcceptanceResults(dashboardToken, startedAt, finishedAt, results) {
  if (!recordResult) return
  const failed = results.filter((item) => !item.result.ok)
  const passedTotal = results.length - failed.length
  try {
    await request("/api/dashboard/digital-store/delivery_records/acceptance_result", {
      method: "POST",
      headers: { Authorization: `Bearer ${dashboardToken}` },
      body: JSON.stringify({
        publicBaseUrl: baseUrl,
        command: acceptanceCommandText(),
        scenarioTotal: results.length,
        passedTotal,
        failedTotal: failed.length,
        startedAt: startedAt.toISOString(),
        finishedAt: finishedAt.toISOString(),
        results: results.map(({ scenario, result }) => ({
          code: scenario.id,
          title: scenario.title,
          passed: Boolean(result.ok),
          reason: result.reason || "",
          failureType: result.failureType || "",
          detail: result.detail || "",
          suggestion: result.suggestion || "",
          conversationId: Number(result.conversationId || 0),
          conversationUrl: result.conversationUrl || "",
          reply: result.reply || "",
          expectedKeywords: result.expectedKeywords || scenario.any || [],
          matchedKeywords: result.matchedKeywords || [],
          missingKeywords: result.missingKeywords || [],
          bannedKeywords: result.bannedKeywords || scenario.banned || [],
          matchedBanned: result.matchedBanned || "",
        })),
      }),
    })
    console.log("Acceptance result recorded in delivery records")
  } catch (error) {
    console.warn(`Warning: failed to record acceptance result: ${error.message || error}`)
  }
}

async function main() {
  const selected = scenarios.filter((item) => scenarioFilter.size === 0 || scenarioFilter.has(item.id))
  if (selected.length === 0) fail("no scenarios selected")
  const startedAt = new Date()
  console.log(`MUSE acceptance start: ${selected.length} scenarios, base=${baseUrl}`)
  const dashboardToken = await getDashboardToken()
  const channelCode = await getReadyWebChannelCode(dashboardToken)
  console.log(`Using web channel: ${channelCode}`)

  const results = []
  for (const scenario of selected) {
    process.stdout.write(`${scenario.id} ${scenario.title} ... `)
    try {
      const result = await runScenario(channelCode, scenario)
      results.push({ scenario, result })
      console.log(result.ok ? "PASS" : `FAIL (${result.reason})`)
      if (!result.ok && result.detail) console.log(`  detail: ${result.detail}`)
      if (!result.ok && result.suggestion) console.log(`  suggestion: ${result.suggestion}`)
      if (!result.ok && result.conversationUrl) console.log(`  conversation: ${result.conversationUrl}`)
      if (result.reply) console.log(`  reply: ${result.reply}`)
    } catch (error) {
      const diagnostic = classifyScenarioError(error)
      const result = {
        ok: false,
        reason: error.message,
        failureType: diagnostic.failureType,
        detail: error.message,
        suggestion: diagnostic.suggestion,
        conversationId: 0,
        conversationUrl: "",
        reply: "",
        expectedKeywords: scenario.any || [],
        matchedKeywords: [],
        missingKeywords: scenario.any || [],
        bannedKeywords: scenario.banned || [],
        matchedBanned: "",
      }
      results.push({ scenario, result })
      console.log(`FAIL (${error.message})`)
      console.log(`  suggestion: ${result.suggestion}`)
    }
    if (scenarioDelayMs > 0 && scenario !== selected[selected.length - 1]) {
      await sleep(scenarioDelayMs)
    }
  }

  const failed = results.filter((item) => !item.result.ok)
  console.log("")
  console.log(`MUSE acceptance summary: ${results.length - failed.length}/${results.length} passed`)
  await recordAcceptanceResults(dashboardToken, startedAt, new Date(), results)
  if (failed.length > 0) {
    failed.forEach(({ scenario, result }) => {
      console.log(`- ${scenario.id} ${scenario.title}: ${result.reason}`)
      if (result.detail) console.log(`  detail: ${result.detail}`)
      if (result.suggestion) console.log(`  suggestion: ${result.suggestion}`)
      if (result.conversationUrl) console.log(`  conversation: ${result.conversationUrl}`)
      if (result.reply) console.log(`  reply: ${result.reply}`)
    })
    process.exitCode = 1
  }
}

main().catch((error) => {
  console.error(error.message || error)
  process.exit(1)
})
