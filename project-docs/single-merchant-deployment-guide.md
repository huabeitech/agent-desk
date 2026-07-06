# AI 数字店长单商家部署交付手册

本文面向“每个商家独立部署一套”的交付方式。目标是让每次交付都有独立数据库、独立密钥、独立模型配置、独立知识库和独立聊天入口，避免多个商家之间共享敏感数据。

## 1. 交付原则

- 一个商家一套部署目录。
- 一个商家一个数据库或 SQLite 数据目录。
- 一个商家一套 `config.yaml` 和环境变量。
- 一个商家一个后台域名和客户聊天入口。
- 模型 API Key、数据库密码、会话密钥不得复用。
- 生产环境不使用默认管理员密码 `ChangeMe123!`。

## 2. 推荐目录

```bash
/opt/ai-store-manager/
  merchant-a/
    agent-desk/
    .env.production
    backups/
  merchant-b/
    agent-desk/
    .env.production
    backups/
```

`.env.production` 不提交到 Git，只保存在服务器安全位置。

## 3. 首次部署

### 3.1 准备代码和配置

```bash
git clone <repo-url> agent-desk
cd agent-desk
cp config/config.example.yaml config/config.yaml
cp .env.example .env.production
```

Docker 部署优先使用：

```bash
cp docker/agent-desk.yaml docker/agent-desk.production.yaml
```

如果是小型单店、追求最少依赖，可用 SQLite + LanceDB：

```bash
cp docker/agent-desk-sqlite-lancedb.yaml docker/agent-desk.production.yaml
```

### 3.2 生成商家独立密钥

先生成随机值，再写入 `.env.production`：

```bash
openssl rand -base64 24 # AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD
openssl rand -base64 32 # AGENT_DESK_CUSTOMERSESSION_SECRET
openssl rand -base64 24 # AGENT_DESK_MYSQL_PASSWORD
openssl rand -base64 24 # AGENT_DESK_MYSQL_ROOT_PASSWORD
```

如果使用 MySQL compose，应用的 DSN 会默认引用 `AGENT_DESK_MYSQL_PASSWORD`。如需外部 MySQL，显式设置：

```bash
AGENT_DESK_DB_DSN=user:password@tcp(mysql-host:3306)/merchant_db?charset=utf8mb4&parseTime=True&multiStatements=true&loc=Local
```

### 3.3 配置客服通知

交付时至少配置一种外部通知通道。最轻量的方式是 Webhook：

```bash
AGENT_DESK_NOTIFY_WEBHOOK_ENABLED=true
AGENT_DESK_NOTIFY_WEBHOOK_URL=https://example.com/webhook/merchant-a
AGENT_DESK_NOTIFY_WEBHOOK_FORMAT=generic
AGENT_DESK_NOTIFY_WEBHOOK_SECRET=<随机密钥>
```

`AGENT_DESK_NOTIFY_WEBHOOK_FORMAT` 可选：

- `generic`：发给自建服务、CRM、n8n。
- `wecom_robot` 或 `dingtalk`：发送文本机器人格式。
- `feishu`：发送飞书文本机器人格式。

配置后，高意向线索、预约线索、会话分配和转人工会推送到该 Webhook。

如需每天自动把老板经营日报推送到同一个 Webhook，开启：

```env
AGENT_DESK_NOTIFY_DAILYREPORT_ENABLED=true
AGENT_DESK_NOTIFY_DAILYREPORT_CRON="0 9 * * *"
AGENT_DESK_NOTIFY_DAILYREPORT_DATEOFFSETDAYS=0
AGENT_DESK_NOTIFY_DAILYREPORT_ALLOWDUPLICATE=false
```

`CRON` 使用标准 5 段表达式；`DATEOFFSETDAYS=0` 发送当天日报，`-1` 发送昨天日报。定时任务默认会记录最近一次成功发送的日报日期，同一天重复触发会跳过；只有需要压测或重放时才把 `ALLOWDUPLICATE` 设为 `true`。首页“今日经营复盘”也提供“发送日报”按钮，适合交付验收时立即测试。

后台首页还提供“经营趋势复盘”和“AI 质检待办”，可按今天、近 7 天、近 30 天查看咨询、留资、预约、到店、成交、转人工、AI 负反馈，以及热门产品、来源渠道、高频问题、未解决问题、顾问效率、无效原因和高频待处理问题。交付给商家时建议用近 7 天范围确认数据口径，再用近 30 天范围给老板演示长期运营价值。

“经营趋势复盘”支持一键复制周/月经营复盘 Markdown，内容包含核心指标、热门产品、来源渠道、高频问题、未解决问题、负反馈原因、顾问跟进和行动建议。交付时可让商家直接贴到飞书、企微或老板群，形成固定周报/月报动作。

看“线索转化漏斗”时，除了确认咨询、留资、预约、到店、成交数量，也要检查“无效原因 Top”和顾问表里的无效原因。交付验收可准备一条预算不匹配、联系不上或售后咨询被标为无效的样例线索，确认老板能看出无效线索来自渠道、价格、联系方式还是售后需求。

运营人员处理 AI 质检时，先看首页“高频待处理问题”，按出现次数、无答案、兜底、风控和负反馈构成决定优先级；可直接在问题卡片点击“生成 FAQ 草稿”，也可点击问题跳到对应检索日志。进入“知识库 - 检索日志”后，还可以点击“批量生成 FAQ 草稿”，系统会把当前知识库里的无答案、兜底、风控和负反馈问题生成待确认 FAQ 草稿。草稿默认停用，必须由运营人工检查答案后启用并重新索引。

检索日志还可按反馈状态筛选。交付验收时建议新增一条点踩或引用错误反馈，再在“反馈状态”选择“仅负反馈”，确认问题能被筛出并可继续生成 FAQ 草稿。

销售线索列表和详情会展示自动标签，例如高意向、已预约、准成交、售后风险、逾期跟进、有预算、高预算和来源渠道。详情页会继续展示每个标签的触发原因和建议动作；CRM/Webhook metadata 也会同步 `autoTags` 和 `autoTagDetails`。交付验收时建议准备几条不同阶段的样例线索，让商家顾问确认这些标签是否符合本行业跟进习惯，并确认外部表格或 CRM 能按标签分层。

销售线索列表、详情和导出 CSV 会展示归并方式、归并说明和归并时间。交付验收时建议用同手机号或同微信重复咨询一次，确认系统复用原线索，并能向顾问解释归并依据。

销售线索列表还会展示最近客户消息和会话摘要，顾问不用点进会话即可判断客户刚问了什么、AI 前面如何承接。验收时可用一条留资会话确认线索列表与详情都能看到最近客户消息。

会话工作台的更多菜单提供“复制跟进摘要”。如果该会话已形成销售线索，摘要会包含客户需求、联系方式、预算、产品、最近跟进和建议话术；如果尚未形成线索，系统会按会话摘要和最近对话生成补需求、补联系方式、建线索的兜底话术。交付验收时建议分别用一条已留资会话和一条未留资会话测试复制结果。

如商家需要把线索同步到 CRM、飞书表格或 n8n，在全局 `notify.webhook` 配好目标地址后，高意向、预约、准成交、已到店和已成交线索会自动发送 `sales_lead_crm_sync` 事件；销售线索列表也可点击“CRM”手动补同步单条线索。`metadata` 中包含客户姓名、电话、微信、预算、意向产品、预约、来源渠道、自动标签和后台链接，便于外部系统直接建表或建客户。

如要做 A/B 话术测试，让不同网页入口、二维码、开场白或预约引导版本传入不同 `sourceChannel`，例如 `opening_a`、`opening_b`、`reserve_v1`。后台首页“渠道来源统计”会先展示不同来源的线索占比、高意向、预约、到店、成交和无效率，方便老板判断官网、广告落地页、二维码和企微入口哪个更有效；“A/B 话术效果”会继续按这个标识对比线索数、高意向率、预约率、到店率、成交率、无效率、质量风险和主咨询产品，并结合周期 AI 负反馈率提醒是否适合继续放量。

交付时在“交付初始化”的“交付报告”区域点击“发送关键通知测试”，确认商家通知群、CRM Webhook 或自动化工具能收到高意向线索、预约线索、转人工、未分配线索和售后风险 5 类测试消息。页面会保留最近一次测试的成功/失败数量、逐项发送状态和接收端错误原因；若出现“发送失败”，先检查 Webhook 地址、格式、签名密钥和接收端日志。注意：“店长配置”中的 Webhook 字段只作为商家资料留存，实际发送以全局 `notify.webhook` / `AGENT_DESK_NOTIFY_WEBHOOK_*` 配置为准。

### 3.4 配置域名和 CORS

在生产配置中将 `server.cors.allowedOrigins` 改成真实域名，例如：

```yaml
server:
  cors:
    allowedOrigins:
      - https://admin.merchant.example.com
      - https://www.merchant.example.com
```

### 3.5 交付前检查

```bash
scripts/check-single-merchant-deploy.sh docker/agent-desk.production.yaml docker-compose.yml
```

默认会检查 `.env.production`。如环境文件放在其他位置，可显式指定：

```bash
AGENT_DESK_ENV_FILE=/opt/ai-store-manager/merchant-a/.env.production \
  scripts/check-single-merchant-deploy.sh docker/agent-desk.production.yaml docker-compose.yml
```

后台“交付初始化”的“交付报告”也会展示上线安全自检，覆盖客户聊天密钥、首次管理员密码环境变量、登录失败锁定、CORS 白名单、数据库、向量库和外部通知。页面中出现“阻断”时先处理配置，再交付上线；“提醒”项需要在交付备注中说明是否接受。

检查通过后再启动：

```bash
docker compose --env-file .env.production up -d --build
```

## 4. 后台初始化

1. 使用 `admin` 和 `AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD` 首次登录。
2. 在后台配置 AI 模型：
   - 聊天模型：配置 OpenAI-compatible 模型服务。
   - Embedding 模型：配置用于知识库向量化的 embedding 模型。
   - “交付初始化”的 AI 模型步骤必须同时显示聊天模型和 embedding 模型完成；只配置聊天模型时不建议上线。
3. 在“交付初始化”选择行业样板或商家样板；当前内置“慕斯寝具门店”“口腔门诊”“少儿英语培训”“金融顾问咨询”“家装装修门店”五个模板，应用前会预览将新建/更新的店长资料、产品/服务、活动权益和验收场景，确认后同步知识库。
   - 可先点击“导出 JSON”保存当前行业模板，作为后续复制到相近商家或沉淀新行业模板的底稿。
   - 新行业可复制导出的 JSON，修改 `template.code`、`template.industry`、`profile`、`products` 和 `promotions` 后，在“交付初始化”点击“导入 JSON 模板”；系统会先预览影响范围，确认后再写入当前商家。
   - 应用模板后，系统会记录模板 code、版本号和应用时间；后续再次应用模板前，先看预览里的版本和覆盖提示。
4. 在“店长配置”检查品牌、人设、门店、营业时间、预约规则和禁用承诺，必要时改成商家的真实信息。
5. 在“产品库”继续导入商家真实产品 CSV，并确认 FAQ 已生成；跨行业字段可写到“行业属性”，例如课时/班型、诊疗项目、装修面积、车型配置、睡感/尺寸。
   - 如导入失败，产品库工具栏会出现“错误明细”，可下载包含行号和错误原因的 CSV，按行修正后重新导入。
6. 在“活动库”继续导入商家真实活动 CSV，并确认有效期与 FAQ 已生成。
   - 如导入失败，活动库工具栏会出现“错误明细”，可下载包含行号和错误原因的 CSV，便于现场修正日期、状态或必填字段。
   - “交付初始化”会展示产品/活动 FAQ 同步覆盖率；出现未同步或索引失败时，先在产品库/活动库执行重建索引。
7. 在“知识库”补充门店 FAQ、售后政策、安装配送、价格口径；医疗、教育等敏感行业需额外补充合规禁用承诺。
   - “交付初始化”的“知识库导入助手”会按行业列出必备 FAQ，优先补齐待补充项。
   - 上线试运行后，每周查看“交付初始化”的“模板效果回收”，重点处理近 30 天高频无答案、兜底、风控和负反馈问题；可复制“模板改进包”作为版本记录，确认答案后补 FAQ 并导出行业模板 JSON，沉淀给下一家同类商家。
8. 在“交付初始化”点击“生成接待运行时”，确认数字店长 Agent、已发布流程和 Web 聊天渠道都变为完成。
9. 在“交付初始化”的“聊天入口”复制客户聊天链接；如果商家要接入官网，则复制网站嵌入代码交给网站维护人员。
10. 在“交付初始化”的“交付报告”检查上线安全自检，处理所有“阻断”项。
11. 复制 Markdown 报告，归档到本次商家交付资料。
12. 如需给商家留存 PDF，在交付报告区域点击“打印 / 保存 PDF”，浏览器打印目标选择“保存为 PDF”。
13. 验收完成后点击“保存交付记录”，将报告、验收状态和摘要留存在后台。
14. 点击“发送关键通知测试”，确认外部通知通道能收到高意向、预约、转人工、未分配和售后风险提醒。
15. 引导商家打开“运营手册”，确认老板、运营和顾问分别知道每天看哪些页面、如何跟进线索、如何处理 AI 负反馈和售后风险。

## 5. 上线验收

“交付初始化”的“交付报告”会自动生成结构化上线验收清单，并在页面预览阻断项。复制 Markdown 报告时会包含客户测试话术、期望结果、后台检查项和不通过标准；交付现场也可以单独点击“复制验收执行清单”，得到带勾选框的 Markdown 表格，贴到飞书、企微或 Notion 后逐项记录通过情况。

验收清单会按行业生成不同矩阵：家居寝具、口腔/医疗、教育培训、金融服务、家装装修和通用咨询会分别覆盖本行业的推荐、活动/权益、留资、转人工、禁用承诺和风险场景。新行业 JSON 模板导入后，先检查 `template.industry` 与 `profile.industry` 是否能命中正确行业口径。

正式验收前先查看“上线安全自检”：客户聊天密钥和首次管理员密码属于强阻断项；CORS、数据库类型、向量库、登录失败锁定、外部通知和 Webhook 签名密钥至少要形成明确交付结论。

每个商家上线前至少跑以下场景：

- 客户问品牌和门店地址。
- 客户问某个产品适合什么人。
- 客户提供预算，AI 能给出推荐。
- 客户问当前优惠，AI 能结合活动库回答。
- 交付报告里的“产品知识索引”和“活动知识索引”均为完成，且同步覆盖率为 100%。
- 客户留下手机号、微信、城市、预算或预约时间，后台能生成销售线索。
- 客户留下手机号、微信或姓名后，后台“客户管理”能看到对应客户档案；销售线索和原会话应绑定同一个客户，重复手机号/微信应复用已有客户而不是新建重复客户。
- 销售线索详情能展示已绑定客户档案摘要，并可跳转到客户管理按该客户手机号或姓名定位。
- 线索详情新增跟进记录后，销售线索列表能按“已逾期”“今日待跟进”“未来已安排”“未设置”筛选。
- 销售线索页顶部能看到逾期、今日、未分配、未设置跟进计划数量，并可点击“发送跟进提醒”生成站内通知和外部 Webhook 摘要。
- 销售线索页“预约到店看板”能看到今日预约、未来到店、逾期未到店、未定时间和未分配预约线索，并可点击“发送预约提醒”生成站内通知和外部 Webhook 摘要。
- 销售线索列表能按“逾期未到店”“今日预约”“未来到店”“未定时间”筛选预约线索，并且导出 CSV 时保留当前预约筛选条件。
- 销售线索列表能按负责人筛选顾问任务，也能筛出“未分配”线索；导出 CSV 时保留当前负责人筛选条件。
- 筛出“未分配”线索后，点击“领取未分配”能把当前筛选范围内的线索分配给当前登录顾问，并刷新列表。
- 销售线索列表能快速标记“到店”“成交”或“无效”，状态变更后列表、首页漏斗、跟进提醒和预约看板会刷新；已到店客户不应继续出现在逾期预约提醒里。
- 同一手机号或微信从新会话再次咨询时，后台应更新原有活跃线索，不应重复生成多条销售线索。
- 客户要求人工，人工客服能看到会话摘要。
- 非服务时间客户要求人工时，AI 继续接待并提示服务时间，同时后台生成未分配待跟进线索，且下次跟进时间落到下一个上午。
- 客户表达售后、投诉、退款、退货、异响或差评风险时，后台应自动生成会话来源工单；同一会话重复表达售后诉求时不应重复创建未完成工单。
- 人工处理完成后，后台可点击“恢复 AI 接待”，后续客户消息重新由 AI 数字店长承接。
- 高意向线索、预约线索、转人工、未分配线索和售后风险能触发外部通知。
- 高意向、预约、准成交、已到店和已成交销售线索能自动触发 `sales_lead_crm_sync`，低意向普通咨询不会误推 CRM；必要时可在销售线索列表手动补同步。
- “发送关键通知测试”能在商家通知群或接收系统中收到 5 类测试消息，并在页面展示逐项发送状态。
- 首页“今日经营复盘”能展示优先跟进名单，包含逾期、今日待跟进、未排计划的高意向/预约线索，复制日报时也包含这些客户。
- 首页“今日经营复盘”能突出未分配重点线索数量；无排班转人工、未分配高意向、预约、准成交、售后风险和当天应跟进线索都应进入该风险口径。
- 首页“今日经营复盘”能展示预约风险，包含逾期未到店、今日预约和未定时间预约，复制日报时也包含这些数量和处理建议。
- 首页经营概览和“今日经营复盘”能展示今日成交线索数；销售线索列表快捷标记成交后，该指标应能反映转化结果。
- 首页“今日经营复盘”能展示售后/投诉工单风险，包含未处理工单、今日新增、今日已处理和最近工单预览；复制日报时包含工单号、状态、负责人、问题摘要和最近处理进展。
- 首页“今日经营复盘”能展示 AI 质量反馈，包含反馈总数、点赞、负反馈、负反馈率和主要负反馈原因；复制日报时也包含这些质量信号。
- 不在知识库内的问题，AI 不乱承诺价格、库存、疗效、退款退货、安装时效、售后赔付或绝对结果。
- 首页经营概览和每日复盘能正常打开。

慕斯样板或同结构商家可先跑自动化冒烟：

```bash
MUSE_ACCEPTANCE_TIMEOUT_MS=70000 scripts/run-muse-chat-acceptance.mjs
```

脚本默认会把本次自动验收的场景总数、通过数、失败数和逐项结果写回后台交付记录，失败项会包含失败类型、缺失关键词、命中禁用词、回复片段、后台会话链接和处理建议；随后可在“交付初始化”的最近记录中查看失败场景定位。临时调试时可设置 `MUSE_ACCEPTANCE_RECORD_RESULT=0` 关闭回写。

## 6. 备份

推荐使用内置备份脚本：

```bash
scripts/backup-single-merchant.sh --output backups --compose docker-compose.yml
```

后台“交付初始化”的“运维与升级”卡片会显示最近一次本地备份目录、备份组成项，并提供可复制的备份命令。正式商家建议把该命令接入服务器定时任务。

正式接入定时任务前，可先 dry-run：

```bash
scripts/backup-single-merchant.sh --dry-run
```

备份内容包含：

- MySQL dump，前提是 compose 中存在并运行 `mysql` 服务。
- 本地 `data/` 目录，包含 SQLite、LanceDB、上传文件等本地数据。
- Docker 配置目录和主 compose 文件快照。
- `BACKUP-MANIFEST.txt`，记录备份时间、项目目录和 compose 文件。

如果使用云数据库或 Docker named volume 且没有本地挂载目录，需要同时配置云厂商快照或 volume 级备份。

## 7. 恢复演练

每个正式商家上线后，至少做一次 dry-run 恢复演练，确认备份目录可读、MySQL 密码可用、`data/` 快照存在。
后台“运维与升级”卡片会自动把最近一次备份目录带入 dry-run 恢复命令；如果没有检测到备份，会保留 `<备份目录>` 占位并给出提醒。

```bash
scripts/restore-single-merchant.sh \
  --backup-dir backups/20260101-120000 \
  --compose docker-compose.yml \
  --dry-run
```

正式恢复前先停止对外流量和应用服务，避免恢复 `data/` 时出现写入竞争：

```bash
docker compose --env-file .env.production stop agent-desk
```

确认要覆盖当前实例数据后执行：

```bash
export AGENT_DESK_MYSQL_PASSWORD="<商家 MySQL 应用密码>"

scripts/restore-single-merchant.sh \
  --backup-dir backups/20260101-120000 \
  --compose docker-compose.yml \
  --confirm
```

如果需要一并恢复当时的 `config/config.yaml`、`docker/` 配置和 compose 快照，增加 `--restore-config`。该选项会覆盖当前部署配置，执行前先确认域名、密钥和数据库连接仍适用于目标机器：

```bash
scripts/restore-single-merchant.sh \
  --backup-dir backups/20260101-120000 \
  --compose docker-compose.yml \
  --restore-config \
  --confirm
```

恢复后重新启动并检查：

```bash
docker compose --env-file .env.production up -d
curl -fsS http://127.0.0.1:8083/api/health
scripts/check-single-merchant-deploy.sh docker/agent-desk.production.yaml docker-compose.yml
```

再进入后台“交付初始化”，确认产品/活动知识索引、人工接待、外部通知和上线安全自检状态；必要时重新运行交付报告中的验收脚本。

## 8. 更新

更新前先在后台“交付初始化”的“运维与升级”卡片复制“升级 Runbook”。Runbook 会带出最近备份状态、恢复 dry-run 命令、升级命令、升级后模型/索引/通知复验、慕斯验收脚本和异常回滚说明。只需要快速复制命令时，也可点击“复制升级检查命令”。第一步必须先做备份：

```bash
scripts/backup-single-merchant.sh --output backups --compose docker-compose.yml
git pull
docker compose --env-file .env.production up -d --build
```

更新后检查：

```bash
curl -fsS http://127.0.0.1:8083/api/health
scripts/check-single-merchant-deploy.sh docker/agent-desk.production.yaml docker-compose.yml
```

随后回到“交付初始化”确认“模型与检索健康”无阻断项，点击“发送关键通知测试”，再运行交付报告中的自动化验收脚本。若失败，先查看最近交付记录里的失败场景定位，再决定修复或按最近备份执行恢复演练。

## 9. 交付资料

交付给商家时应包含：

- 后台地址。
- 管理员账号和首次密码。
- `.env.production` 保管位置和密钥轮换负责人，不在普通交付包中明文扩散。
- 客户聊天入口和网站嵌入代码。
- 初始化页生成的 Markdown 交付报告。
- 后台保存的最近一次交付记录。
- 上线安全自检结论及已接受的提醒项。
- 已配置模型供应商和模型名称。
- 产品 CSV、活动 CSV、FAQ 原始资料。
- 备份目录、`BACKUP-MANIFEST.txt` 和恢复演练记录。
- 人工客服通知方式。
- 不可承诺事项清单。
