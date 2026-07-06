#!/usr/bin/env node

const baseUrl = (process.env.AGENT_DESK_BASE_URL || "http://127.0.0.1:8084").replace(/\/$/, "")
const adminUsername = process.env.AGENT_DESK_ADMIN_USERNAME || "admin"
const adminPassword = process.env.AGENT_DESK_ADMIN_PASSWORD || "ChangeMe123!"

const statusOk = 0
const knowledgeBaseId = Number(process.env.MUSE_KNOWLEDGE_BASE_ID || 1)

const products = [
  {
    name: "慕斯T10释压枕",
    category: "枕头",
    priceMin: 800,
    priceMax: 1800,
    sellingPoints: "分区承托颈肩、慢回弹释压，适合和床垫一起做睡眠方案搭配。",
    suitablePeople: "落枕、颈肩紧、侧睡较多、想改善枕头高度的人群。",
    unsuitablePeople: "明确喜欢很低很薄枕或对慢回弹材质敏感的客户。",
    scenarios: "主卧、老人房、床垫搭配升级、颈肩睡眠咨询。",
    specs: "高度需到店试枕确认；成人常规高度和侧睡高度可现场对比。",
    industryAttributes: "睡感：慢回弹释压；关注点：颈肩承托、枕高匹配、侧睡/仰睡差异。",
    priority: 68,
    status: statusOk,
    remark: "Codex realistic muse seed: pillow",
  },
  {
    name: "慕斯床架与排骨架检测服务",
    category: "售后服务",
    priceMin: 0,
    priceMax: 0,
    sellingPoints: "针对床垫异响、翻身响、床架松动等问题，记录订单后由顾问/售后确认检测方式。",
    suitablePeople: "已购床垫出现异响、担心质量问题、需要售后排查的客户。",
    unsuitablePeople: "要求在线直接判定质量责任、直接承诺退换赔付的客户。",
    scenarios: "售后异响、床架排查、订单售后、投诉安抚。",
    specs: "需留下购买时间、订单信息、产品型号、异响位置、联系方式；处理结论以售后检测和订单条款为准。",
    industryAttributes: "禁用口径：不先推责给床架，不承诺马上上门、彻底解决、无条件退换；先安抚并登记转人工。",
    priority: 92,
    status: statusOk,
    remark: "Codex realistic muse seed: after-sales diagnostic service",
  },
]

const promotions = [
  {
    name: "徐汇门店周末试躺预约",
    promotionType: "预约服务",
    description: "面向周末到徐汇门店试躺的客户，记录姓名、手机号、到店时间、人数、预算、关注产品，由门店顾问确认安排。",
    applicableProducts: "慕斯脊护支撑款、慕斯云感舒睡款、慕斯智能电动床、慕斯T10释压枕",
    startAt: relativeDate(-7),
    endAt: relativeDate(45),
    discountRule: "最终成交价、叠加优惠、库存和配送时效均以门店顾问确认为准。",
    storeBenefit: "到店可做软硬度对比、枕高体验、电动床升降演示和睡眠需求沟通。",
    appointmentBenefit: "预约信息记录后由顾问确认时段；活动权益和礼品数量以门店当天确认为准。",
    scriptSuggestion: "客户说周六、徐汇、两个人、预算或电动床时，先复述已知信息，再说已记录待顾问确认，不说预约成功或已预留。",
    priority: 96,
    status: statusOk,
    remark: "Codex realistic muse seed: Xuhui appointment",
  },
  {
    name: "售后异响快速登记",
    promotionType: "售后服务",
    description: "面向已购床垫异响、床架响、翻身响、投诉风险客户，优先登记联系方式并转人工顾问。",
    applicableProducts: "慕斯床垫、慕斯床架与排骨架检测服务",
    startAt: relativeDate(-7),
    endAt: relativeDate(60),
    discountRule: "不承诺退款、退货、赔付、上门时间或质量责任；处理结论以订单条款和售后检测为准。",
    storeBenefit: "顾问会收集订单、型号、购买时间、异响位置、视频或现场检测需求。",
    appointmentBenefit: "留下手机号后记录售后诉求并转人工确认，不重复索要联系方式。",
    scriptSuggestion: "客户生气或说投诉时先道歉、承认影响休息、记录电话和诉求；不要说通常不是床垫问题。",
    priority: 98,
    status: statusOk,
    remark: "Codex realistic muse seed: after-sales",
  },
]

const faqs = [
  faq("价格贵不贵怎么回答", [
    "客户问慕斯是不是比别人贵时，先承认这是正常顾虑，不要上来反驳。",
    "推荐话术：慕斯不是走最低价路线，主要价值在材料、承托结构、睡感体验和门店服务。若客户重视性价比，可先看云感舒睡款8000-13000元；若更重视支撑承托，可看脊护支撑款12000-18000元。",
    "最后只问一个关键问题：偏软包裹还是偏硬支撑。不要立即索要电话，除非客户明确要报价、库存或到店。",
  ], ["你们是不是很贵", "比别家贵在哪", "床垫价格为什么这么高", "慕斯性价比怎么样"]),
  faq("用户追问到底能给什么方案", [
    "客户问“你到底能给我什么方案”时，必须先给方案，不要只反问。",
    "如果客户只明确1.8米和价格关注，可给两个方向：方案一，云感舒睡款，8000-13000元，偏柔和包裹，适合日常舒适睡眠；方案二，脊护支撑款，12000-18000元，偏支撑承托，适合关注腰背支撑或喜欢偏硬睡感的人群。",
    "同时说明库存、活动权益、最终成交价需要门店顾问确认。最后只问：您更倾向偏软还是偏硬？",
    "禁止把客户没说过的症状写成事实，例如客户没说腰疼，就不要说“您腰疼/您腰背不适”。",
  ], ["到底什么方案", "直接给我方案", "别绕了给方案", "现在能给我什么"]),
  faq("1.8米床垫预算推荐", [
    "1.8米床垫按预算推荐：8000-13000元优先云感舒睡款，柔和包裹、释压舒适；12000-18000元优先脊护支撑款，分区承托、偏硬支撑。",
    "预算约15000元时，如果客户未说明腰背问题，不要默认客户腰疼；可以说这个预算可覆盖脊护支撑款的主力配置，也可以对比云感舒睡款。",
    "回答后追问一个问题：偏软还是偏硬，或者是给自己、长辈还是孩子用。",
  ], ["1.8米多少钱", "一万五预算买什么床垫", "主卧床垫推荐", "预算15000"]),
  faq("腰疼和护脊边界", [
    "客户提到腰疼、腰酸、早上僵、久坐腰累时，可以推荐关注支撑感和分区承托，但必须强调床垫不能替代医疗诊断或治疗，也不能保证治好疼痛。",
    "自然话术：腰不舒服确实影响睡眠，床垫可以从支撑和贴合上帮您改善睡姿受力，但如果持续疼痛建议先咨询医生。门店可以重点试脊护支撑款，看腰部是否贴合、有无悬空。",
    "不要说很多客户治好了、一定改善、保证有效。",
  ], ["腰疼床垫有用吗", "能不能治好腰疼", "护脊是不是噱头", "腰酸买硬床垫吗"]),
  faq("软硬怎么选", [
    "软硬选择不要说越硬越好。判断顺序：睡姿、体重、腰背感受、原床垫问题、是否侧睡。",
    "侧睡多、喜欢包裹感、体重较轻：可试云感舒睡款。仰睡多、体重较大、关注支撑：可试脊护支撑款。",
    "真实导购说法：先躺10-15分钟，看肩臀是否压迫、腰部是否悬空、翻身是否费力。",
  ], ["床垫越硬越好吗", "软床垫会不会塌", "我不知道软硬怎么选", "侧睡选什么"]),
  faq("老人电动床咨询", [
    "老人起夜多、起身困难、床上阅读休息，可介绍慕斯智能电动床：头脚升降、阅读观影模式，价格16000-28000元，建议搭配适配床垫。",
    "安全问题回答要稳：具体防夹、防护结构、遥控器功能以门店实物和顾问演示为准；建议带老人到店体验升降速度、按键清晰度和床垫适配。",
    "不要说绝对不会夹到人、老人一定一学就会。可以说正常使用需按说明操作，门店会演示。",
  ], ["老人起夜电动床", "电动床安全吗", "抬背会不会夹到人", "老人会不会操作"]),
  faq("老人电动床两万预算方案", [
    "客户预算两万以内，想看1.8米电动床加床垫时，回答：两万预算可以先看智能电动床基础组合方向，但具体能否包含1.8米床垫、活动权益和配送安装，需要门店顾问按规格确认。",
    "推荐路径：先体验电动床升降功能，再试一张适配的脊护支撑方向床垫；如果预算紧，就优先确认电动床尺寸和核心功能，再看床垫配置。",
    "不要说完全OK、肯定够、已经预留。",
  ], ["两万电动床加床垫够吗", "1.8米电动床预算", "老人电动床组合推荐", "电动床加床垫多少钱"]),
  faq("预约留资闭环话术", [
    "客户留下姓名、电话、到店时间、门店、人数、预算、意向产品后，先复述已知信息。",
    "标准话术：好的李先生，我已记录：电话13900001234，周六下午两点，徐汇店，两位到店，预算两万，重点体验老人电动床。接下来会转给门店顾问确认具体时段、库存/活动和体验安排。",
    "缺什么只问缺什么。不要重复问已说过的尺寸、人数、预算、产品。不要说预约成功、已预留、周六见。",
  ], ["我姓李电话周六到店", "预约试躺怎么确认", "留电话后会联系吗", "周六两点徐汇店两个人"]),
  faq("顾问联系时效", [
    "客户问顾问什么时候联系时，不能承诺24小时、当天、最晚次日上午，除非门店配置明确。",
    "回答：我已记录您的联系方式和到店意向，会转给门店顾问确认；具体联系时间以门店工作安排为准。您也可以补充方便接听的时间段，我一起备注。",
  ], ["留电话后多久联系", "顾问什么时候打电话", "会有人联系我吗", "多久回电"]),
  faq("库存现货边界", [
    "库存是实时信息，聊天里不得直接承诺现货、有货、可当天送。",
    "回答：这款有对应规格，但今天是否有现货、最快配送和安装时间，需要门店顾问按规格实时确认。可以留下联系方式和目标尺寸，我帮您记录给顾问。",
  ], ["今天有现货吗", "能不能马上送", "1.8米有没有货", "最快什么时候送"]),
  faq("退换试睡边界", [
    "客户问不合适能不能退、有没有试睡时，不要直接承诺30天试睡、无条件退换。",
    "回答：部分活动或指定订单可能会有体验/换购权益，但退换条件需要以购买合同、订单条款和门店售后政策为准。我可以帮您把顾虑记录给顾问，到店前先确认清楚。",
  ], ["不合适能退吗", "有没有30天试睡", "能保证退吗", "睡着不舒服怎么办"]),
  faq("售后异响投诉", [
    "客户说床垫异响、翻身咯吱响、生气、投诉时，先安抚，不要先替产品排除责任。",
    "标准话术：真的抱歉影响您休息了。异响原因需要售后结合订单、产品型号、床架/排骨架和现场情况检查确认，我先帮您记录售后诉求并转人工顾问。您留下的电话我已记录，还可以补充购买时间、型号和异响位置。",
    "禁止说通常不是床垫问题、马上上门、彻底解决、一定退换或赔付。",
  ], ["床垫咯吱响怎么办", "我要投诉售后", "电话是让人工联系我", "床垫异响质量问题"]),
  faq("人工转接和投诉升级", [
    "客户明确说人工、真人、投诉、平台投诉、12315、差评时，应优先转人工或提示已记录人工诉求。",
    "如果客户已留下手机号，要确认已记录，不要再说“你可以留下手机号”。",
    "表达方式：我已记录您的人工/投诉诉求和联系方式，会转给门店顾问或售后继续确认；处理结论以订单和售后检测为准。",
  ], ["转人工", "我要真人", "没人处理我投诉", "电话给你了"]),
  faq("闲聊和不耐烦", [
    "客户问你是谁：先回答身份，我是慕小眠，慕斯寝具在线睡眠顾问，可以帮您挑床垫、电动床、预约试躺，也能把价格、库存、售后问题转给门店顾问。",
    "客户问会不会写诗：可以轻松接一句，不要生硬拒绝，然后拉回睡眠顾问职责。",
    "客户说怎么没反应：先道歉并说我在，然后继续处理上一条需求，不要重复长篇介绍。",
  ], ["你是谁", "你会写诗吗", "你听得懂吗", "怎么没反应"]),
  faq("竞品对比和怕被忽悠", [
    "客户说怕被忽悠、是不是噱头、和别家有什么区别时，不要攻击竞品。",
    "回答重点：建议用试躺指标判断，不听概念。看腰部是否悬空、肩臀是否压迫、翻身是否费力、边缘支撑是否稳定、起身是否轻松。",
    "可以说慕斯门店会让您对比云感和脊护两种睡感，您自己身体感受最重要。",
  ], ["护脊是不是噱头", "怕被忽悠", "和喜临门哪个好", "怎么判断不是营销"]),
]

function faq(question, lines, similarQuestions) {
  return {
    question,
    answer: lines.join("\n"),
    similarQuestions,
    remark: "Codex realistic muse seed",
  }
}

function relativeDate(days) {
  const date = new Date()
  date.setDate(date.getDate() + days)
  return date.toISOString().slice(0, 10)
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
    throw new Error(`${options.method || "GET"} ${path} failed ${response.status}: ${text.slice(0, 800)}`)
  }
  return body.data ?? body
}

function authHeaders(token) {
  return { Authorization: `Bearer ${token}` }
}

function pickToken(login) {
  return login.token || login.accessToken || login.access_token || ""
}

async function login() {
  const ret = await request("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ username: adminUsername, password: adminPassword }),
  })
  const token = pickToken(ret)
  if (!token) throw new Error("login did not return token")
  return token
}

async function listAll(path, token, body = {}) {
  const data = await request(path, {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify({ page: 1, limit: 200, ...body }),
  })
  return data.results || []
}

async function upsertProduct(token, product) {
  const existing = (await listAll("/api/dashboard/product/list", token, { keyword: product.name }))
    .find((item) => item.name === product.name)
  const payload = { ...product, knowledgeBaseId }
  if (existing?.id) {
    await request("/api/dashboard/product/update", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify({ ...payload, id: existing.id }),
    })
    await request("/api/dashboard/product/reindex", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify({ id: existing.id }),
    })
    return { action: "updated", id: existing.id, name: product.name }
  }
  const created = await request("/api/dashboard/product/create", {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify(payload),
  })
  return { action: "created", id: created.id, name: product.name }
}

async function upsertPromotion(token, promotion) {
  const existing = (await listAll("/api/dashboard/promotion/list", token, { keyword: promotion.name }))
    .find((item) => item.name === promotion.name)
  const payload = { ...promotion, knowledgeBaseId }
  if (existing?.id) {
    await request("/api/dashboard/promotion/update", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify({ ...payload, id: existing.id }),
    })
    await request("/api/dashboard/promotion/reindex", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify({ id: existing.id }),
    })
    return { action: "updated", id: existing.id, name: promotion.name }
  }
  const created = await request("/api/dashboard/promotion/create", {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify(payload),
  })
  return { action: "created", id: created.id, name: promotion.name }
}

async function upsertFAQ(token, item) {
  const query = new URLSearchParams({
    knowledgeBaseId: String(knowledgeBaseId),
    question: item.question,
    limit: "200",
  })
  const list = await request(`/api/dashboard/knowledge-faq/list?${query.toString()}`, {
    headers: authHeaders(token),
  })
  const existing = (list.results || []).find((faqItem) => faqItem.question === item.question)
  const payload = { knowledgeBaseId, directoryId: 0, ...item }
  if (existing?.id) {
    await request("/api/dashboard/knowledge-faq/update", {
      method: "POST",
      headers: authHeaders(token),
      body: JSON.stringify({ ...payload, id: existing.id }),
    })
    return { action: "updated", id: existing.id, question: item.question }
  }
  const created = await request("/api/dashboard/knowledge-faq/create", {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify(payload),
  })
  return { action: "created", id: created.id, question: item.question }
}

async function main() {
  console.log(`Seeding realistic Muse data into ${baseUrl}`)
  const token = await login()
  await request("/api/dashboard/digital-store/apply_template", {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify({ templateCode: "muse_bedding" }),
  })
  await request("/api/dashboard/product/seed_muse", {
    method: "POST",
    headers: authHeaders(token),
    body: "{}",
  })
  await request("/api/dashboard/promotion/seed_muse", {
    method: "POST",
    headers: authHeaders(token),
    body: "{}",
  })

  const productResults = []
  for (const product of products) productResults.push(await upsertProduct(token, product))

  const promotionResults = []
  for (const promotion of promotions) promotionResults.push(await upsertPromotion(token, promotion))

  const faqResults = []
  for (const item of faqs) faqResults.push(await upsertFAQ(token, item))

  await request("/api/dashboard/digital-store/sync_knowledge", {
    method: "POST",
    headers: authHeaders(token),
    body: "{}",
  })
  const runtime = await request("/api/dashboard/digital-store/ensure_runtime", {
    method: "POST",
    headers: authHeaders(token),
    body: "{}",
  })

  console.log(JSON.stringify({
    products: productResults,
    promotions: promotionResults,
    faqs: {
      total: faqResults.length,
      created: faqResults.filter((item) => item.action === "created").length,
      updated: faqResults.filter((item) => item.action === "updated").length,
    },
    runtime: {
      ready: runtime.ready,
      webChannelCode: runtime.webChannelCode,
      missingSteps: runtime.missingSteps || [],
    },
  }, null, 2))
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
