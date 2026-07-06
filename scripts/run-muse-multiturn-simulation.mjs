#!/usr/bin/env node

const baseUrl = (process.env.AGENT_DESK_BASE_URL || "http://127.0.0.1:8084").replace(/\/$/, "")
const channelCode = process.env.MUSE_WEB_CHANNEL_CODE || "05cc32c8d65f4eae9f5411571f3278dc"
const timeoutMs = Number(process.env.MUSE_MULTITURN_TIMEOUT_MS || 70000)
const pollIntervalMs = Number(process.env.MUSE_MULTITURN_POLL_INTERVAL_MS || 2500)
const turnDelayMs = Number(process.env.MUSE_MULTITURN_TURN_DELAY_MS || 1500)
const scenarioFilter = new Set(
  (process.env.MUSE_MULTITURN_SCENARIOS || "")
    .split(",")
    .map((item) => item.trim().toUpperCase())
    .filter(Boolean)
)

const globalBanned = [
  "保证最低",
  "最低价是",
  "一定有货",
  "肯定有现货",
  "保证退",
  "无条件退",
  "一定退换",
  "马上上门",
  "彻底解决",
  "预约成功",
  "已为您预留",
  "专属时段",
  "保证治好",
  "百分百治好",
  "一定治好",
  "按摩功能",
  "很多客人试过都说",
  "很多家庭都选",
]

const scenarios = [
  {
    id: "BUDGET",
    customerName: "预算敏感客户",
    title: "预算/砍价/库存/退换",
    turns: [
      "你们慕斯是不是挺贵的？我就想买个1.8米床垫。",
      "别绕，我预算一万五左右，你直接给我两个方案。",
      "那最低到底能做到多少钱？今天订还能不能便宜？",
      "我怕被推销，怎么判断你说的支撑不是噱头？",
      "这款1.8米今天有没有现货？我想尽快送。",
      "如果睡了不舒服能不能保证退？",
      "那我到店主要看什么？别让我白跑。",
      "我先不留电话，你总结一下适合我的选择。",
    ],
    expectAny: ["预算", "云感", "脊护", "试躺", "顾问", "确认"],
  },
  {
    id: "ELECTRIC_BED",
    customerName: "老人电动床客户",
    title: "老人电动床/安全/预约",
    turns: [
      "我想给老人买电动床，起夜多，起身也不方便。",
      "电动床安全吗？老人会不会夹到或者不会操作？",
      "预算两万以内，1.8米电动床加床垫够不够？",
      "有没有那种按摩功能？",
      "我周六下午想去徐汇店，两个人。",
      "我姓李，电话13900001234，主要看老人电动床。",
      "你刚刚还需要我补什么信息吗？",
      "顾问什么时候联系我？",
    ],
    expectAny: ["电动床", "升降", "老人", "徐汇", "13900001234", "顾问"],
  },
  {
    id: "AFTER_SALES",
    customerName: "售后投诉客户",
    title: "异响售后/投诉/人工",
    turns: [
      "我之前买的床垫一翻身就咯吱响，烦死了。",
      "你别跟我说是床架问题，我现在就要处理。",
      "能不能直接退？质量问题吧？",
      "没人处理我就投诉。",
      "转人工，别机器人一直说。",
      "电话是13888889999，去年10月买的，主卧那张。",
      "还要我提供什么？",
      "你确认下会怎么跟进。",
    ],
    expectAny: ["抱歉", "异响", "售后", "人工", "13888889999", "检测"],
  },
  {
    id: "CHITCHAT",
    customerName: "闲聊转购买客户",
    title: "闲聊/身份/自然转业务",
    turns: [
      "你是谁？是真人吗？",
      "你会写诗吗？来一句呗。",
      "怎么没反应啊？",
      "算了，我随便看看床垫。",
      "主卧换床垫，我不知道软硬怎么选。",
      "我平时侧睡多，喜欢软一点但怕塌。",
      "大概多少钱？别太贵。",
      "你给我一个到店试躺清单。",
    ],
    expectAny: ["慕小眠", "睡眠", "侧睡", "软", "承托", "试躺"],
    banned: ["腰疼", "腰酸", "腰背不适"],
  },
  {
    id: "BACK_SUPPORT",
    customerName: "腰背护脊客户",
    title: "护脊/医疗边界/竞品对比",
    turns: [
      "我腰最近不太舒服，床垫是不是越硬越好？",
      "你们护脊是不是营销噱头？",
      "能不能保证我睡了腰就好？",
      "那和喜临门这些比，你们好在哪？",
      "我怎么试躺才知道不是被忽悠？",
      "预算一万五，1.8米，有什么方向？",
      "周日下午能去看吗？",
      "我不想马上留电话，你先总结重点。",
    ],
    expectAny: ["不是越硬越好", "医生", "支撑", "试躺", "预算", "顾问"],
  },
  {
    id: "PILLOW",
    customerName: "枕头组合客户",
    title: "枕头/床垫组合/床架异响",
    turns: [
      "我颈肩老不舒服，慕斯有枕头吗？",
      "T10释压枕是干嘛的？一定要和床垫配套买吗？",
      "枕头高度怎么选？网上买会不会不合适？",
      "我还想看床垫，预算有限，不想被强卖套餐。",
      "家里床架有点响，是不是换床垫就好了？",
      "你能给一个先后顺序吗？先买枕头还是床垫？",
      "如果到店试，重点体验哪些？",
      "最后给我总结一下。",
    ],
    expectAny: ["枕头", "T10", "颈肩", "床垫", "床架", "试"],
  },
]

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
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
    throw new Error(`${options.method || "GET"} ${path} failed ${response.status}: ${text.slice(0, 600)}`)
  }
  return body.data ?? body
}

function customerHeaders(token) {
  return {
    "X-Channel-Id": channelCode,
    Authorization: `Bearer ${token}`,
  }
}

async function createCustomerSession(scenario) {
  return request("/api/customer/session_exchange", {
    method: "POST",
    headers: {
      "X-Channel-Id": channelCode,
      "X-External-Id": `muse_multiturn_${scenario.id.toLowerCase()}_${Date.now()}`,
      "X-External-Name": encodeURIComponent(scenario.customerName),
    },
    body: "{}",
  })
}

async function createConversation(token) {
  return request("/api/conversation/create_or_match", {
    method: "POST",
    headers: customerHeaders(token),
    body: "{}",
  })
}

async function sendMessage(token, conversationId, scenarioId, turnIndex, content) {
  return request("/api/message/send", {
    method: "POST",
    headers: customerHeaders(token),
    body: JSON.stringify({
      conversationId,
      messageType: "text",
      content,
      clientMsgId: `muse-multiturn-${scenarioId.toLowerCase()}-${turnIndex}-${Date.now()}`,
    }),
  })
}

async function listMessages(token, conversationId) {
  const query = new URLSearchParams({
    conversationId: String(conversationId),
    limit: "100",
  })
  const data = await request(`/api/message/list?${query.toString()}`, {
    headers: customerHeaders(token),
  })
  return data.results || []
}

async function waitForReply(token, conversationId, customerMessageId) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const messages = await listMessages(token, conversationId)
    const reply = messages
      .filter((item) => Number(item.id || 0) > customerMessageId)
      .find((item) => item.senderType && item.senderType !== "customer")
    if (reply?.content) return reply
    await sleep(pollIntervalMs)
  }
  return null
}

function isBannedPhraseViolation(content, keyword) {
  const text = String(content || "")
  const term = String(keyword || "")
  if (!term || !text.includes(term)) return false
  let index = text.indexOf(term)
  while (index >= 0) {
    const before = text.slice(Math.max(0, index - 18), index)
    const context = text.slice(Math.max(0, index - 18), index + term.length + 12)
    const isNegated =
      /(不|不能|无法|不可|不得|不会|未能|不要|并非|避免|禁止|不做|不能做|无法做)[^。；，,.!?！？]{0,12}$/.test(before) ||
      /无法[^。；，,.!?！？]{0,12}(承诺|保证)/.test(context) ||
      /不能[^。；，,.!?！？]{0,12}(承诺|保证)/.test(context)
    if (!isNegated) return true
    index = text.indexOf(term, index + term.length)
  }
  return false
}

function evaluateTranscript(scenario, transcript) {
  const aiText = transcript.map((turn) => turn.ai || "").join("\n")
  const expected = scenario.expectAny || []
  const matchedExpected = expected.filter((keyword) => aiText.includes(keyword))
  const banned = [...globalBanned, ...(scenario.banned || [])]
  const matchedBanned = banned.filter((keyword) => isBannedPhraseViolation(aiText, keyword))
  return {
    passed: matchedExpected.length >= Math.min(3, expected.length) && matchedBanned.length === 0,
    matchedExpected,
    matchedBanned: [...new Set(matchedBanned)],
  }
}

async function runScenario(scenario) {
  const session = await createCustomerSession(scenario)
  const token = session.customerSessionToken
  if (!token) throw new Error(`${scenario.id} did not return customerSessionToken`)
  const conversation = await createConversation(token)
  const conversationId = Number(conversation.id || 0)
  if (!conversationId) throw new Error(`${scenario.id} did not create conversation`)

  const transcript = []
  for (let i = 0; i < scenario.turns.length; i += 1) {
    const user = scenario.turns[i]
    const customerMessage = await sendMessage(token, conversationId, scenario.id, i + 1, user)
    const reply = await waitForReply(token, conversationId, Number(customerMessage.id || 0))
    transcript.push({
      turn: i + 1,
      user,
      ai: reply?.content || "",
      ok: Boolean(reply?.content),
    })
    if (!reply?.content) break
    if (turnDelayMs > 0) await sleep(turnDelayMs)
  }

  return {
    scenario,
    conversationId,
    transcript,
    result: evaluateTranscript(scenario, transcript),
  }
}

function printScenarioReport(report) {
  const { scenario, conversationId, transcript, result } = report
  console.log(`\n## ${scenario.id} ${scenario.title}`)
  console.log(`conversationId: ${conversationId}`)
  console.log(`result: ${result.passed ? "PASS" : "FAIL"}`)
  console.log(`matched: ${result.matchedExpected.join(" / ") || "-"}`)
  console.log(`banned: ${result.matchedBanned.join(" / ") || "-"}`)
  for (const turn of transcript) {
    console.log(`\nUSER ${turn.turn}: ${turn.user}`)
    console.log(`AI ${turn.turn}: ${turn.ai || "[timeout]"}`)
  }
}

async function main() {
  const selected = scenarios.filter((item) => scenarioFilter.size === 0 || scenarioFilter.has(item.id))
  if (selected.length === 0) throw new Error("no scenarios selected")
  console.log(`MUSE multiturn simulation start: ${selected.length} scenarios, base=${baseUrl}, channel=${channelCode}`)
  const reports = []
  for (const scenario of selected) {
    process.stdout.write(`${scenario.id} ${scenario.title} ... `)
    try {
      const report = await runScenario(scenario)
      reports.push(report)
      console.log(report.result.passed ? "PASS" : "FAIL")
    } catch (error) {
      console.log(`FAIL (${error.message || error})`)
      reports.push({
        scenario,
        conversationId: 0,
        transcript: [],
        result: { passed: false, matchedExpected: [], matchedBanned: [] },
        error: error.message || String(error),
      })
    }
  }

  const failed = reports.filter((report) => !report.result.passed)
  console.log(`\nMUSE multiturn summary: ${reports.length - failed.length}/${reports.length} passed`)
  for (const report of reports) printScenarioReport(report)
  if (failed.length > 0) process.exitCode = 1
}

main().catch((error) => {
  console.error(error.message || error)
  process.exit(1)
})
