package runtime

import (
	"fmt"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	svc "agent-desk/internal/services"

	"github.com/mlogclub/simple/sqls"
)

type replyCommitService struct{}

type replyCommitInput struct {
	Conversation   models.Conversation
	Message        models.Message
	AIAgent        models.AIAgent
	ReplyText      string
	ClientPrefix   string
	WorkflowRunID  int64
	IncrementRound bool
}

func newReplyCommitService() *replyCommitService {
	return &replyCommitService{}
}

func (s *replyCommitService) SendAIReply(input replyCommitInput) (*models.Message, error) {
	replyText := tailorDigitalStoreFallback(input.ReplyText, input.Message.Content)
	replyText = sanitizeCommercialReplyText(replyText)
	replyText = ensureConcretePlanWhenAsked(replyText, input.Message.Content)
	replyText = ensureLowBudgetPath(replyText, input.Message.Content)
	replyText = sanitizeReplyAgainstCustomerContext(replyText, input.Conversation.ID)
	replyText = sanitizeStoreMentionAgainstCustomerContext(replyText, input.Conversation.ID)
	replyText = sanitizeReplyAgainstKnownLeadInfo(replyText, input.Conversation.ID)
	replyText = sanitizeChitchatReply(replyText, input.Message.Content)
	if replyText == "" {
		return nil, nil
	}
	replyMessage, err := svc.MessageService.SendAIMessageWithRequestIDAndWorkflowRunID(
		input.Conversation.ID,
		input.AIAgent.ID,
		fmt.Sprintf("%s_%d", strings.TrimSpace(input.ClientPrefix), input.Message.ID),
		enums.IMMessageTypeText,
		replyText,
		"",
		s.buildAIPrincipal(input.AIAgent),
		input.Message.RequestID,
		input.WorkflowRunID,
	)
	if err != nil || !input.IncrementRound {
		return replyMessage, err
	}
	if err := s.IncrementAIReplyRounds(input.Conversation.ID, input.Conversation.AIReplyRounds+1, input.AIAgent.Name); err != nil {
		return nil, err
	}
	return replyMessage, err
}

func (s *replyCommitService) CommitAIReply(input replyCommitInput) (*models.Message, error) {
	input.IncrementRound = true
	return s.SendAIReply(input)
}

func (s *replyCommitService) IncrementAIReplyRounds(conversationID int64, nextRounds int, aiAgentName string) error {
	return repositories.ConversationRepository.Updates(sqls.DB(), conversationID, map[string]any{
		"ai_reply_rounds":  nextRounds,
		"update_user_id":   0,
		"update_user_name": strings.TrimSpace(aiAgentName),
		"updated_at":       time.Now(),
	})
}

func (s *replyCommitService) buildAIPrincipal(aiAgent models.AIAgent) *dto.AuthPrincipal {
	username := "AI"
	if strings.TrimSpace(aiAgent.Name) != "" {
		username = aiAgent.Name
	}
	return &dto.AuthPrincipal{
		UserID:   0,
		Username: username,
		Nickname: username,
	}
}

func sanitizeCommercialReplyText(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"已为您预留时段，周六见！", "我已记录这条到店意向，具体时段以门店顾问确认为准。",
		"已为您预留时段", "我已记录这条到店意向，具体时段以门店顾问确认为准",
		"已为您预留", "我已记录，具体安排以门店顾问确认为准",
		"预约成功", "预约信息已记录，待门店顾问确认",
		"绝不会白跑一趟", "建议先明确重点体验清单，减少白跑",
		"确保您到店就能好好试躺", "具体体验安排以门店顾问确认为准",
		"帮您预约周末试躺时段", "帮您记录周末试躺意向，待门店顾问确认时段",
		"帮您预约个周末试躺", "帮您记录周末试躺意向，待门店顾问确认",
		"预约周末试躺可以提前帮您留好专属时段", "周末试躺意向可以先记录，具体时段待门店顾问确认",
		"预约周末试躺", "记录周末试躺意向",
		"我帮您留个时段", "我帮您记录到店意向，具体时段待门店顾问确认",
		"周六下午两点后到店都可以", "周六下午的到店意向我先帮您记录，具体时段待门店顾问确认",
		"帮你留好周末的体验时段和礼包", "帮你记录到店意向，具体体验时段和活动权益待门店顾问确认",
		"帮您留好周末的体验时段和礼包", "帮您记录到店意向，具体体验时段和活动权益待门店顾问确认",
		"帮你留好周末的体验时段", "帮你记录到店意向，具体体验时段待门店顾问确认",
		"帮您留好周末的体验时段", "帮您记录到店意向，具体体验时段待门店顾问确认",
		"留好周末的体验时段", "记录到店意向，具体体验时段待门店顾问确认",
		"留好体验时段", "记录体验意向",
		"保留体验时段", "确认体验时段",
		"预留体验时段", "确认体验时段",
		"第一时间联系您", "结合门店安排联系您",
		"稍后安排门店顾问添加您", "会转给门店顾问确认微信联系",
		"安排专人给您做介绍", "转给门店顾问确认体验安排",
		"完全没问题", "我先帮您记录，具体安排以门店顾问确认为准",
		"3点没问题", "3点的到店意向我先帮您记录，具体时段以门店顾问确认为准",
		"下午3点没问题", "下午3点的到店意向我先帮您记录，具体时段以门店顾问确认为准",
		"周六下午3点没问题", "周六下午3点的到店意向我先帮您记录，具体时段以门店顾问确认为准",
		"我帮您安排顾问优先留好试躺位", "我会转给门店顾问确认体验安排",
		"到店前顾问会跟您确认时段", "后续由门店顾问确认具体时段",
		"也不用担心滑落", "建议到店重点体验起身稳定性",
		"老人一看就会按", "按键是否顺手建议让老人现场试一下",
		"一看就会", "建议现场试一下是否顺手",
		"安全性是有保障的", "安全细节建议到店结合实物确认",
		"减少摔倒风险", "降低起身负担",
		"缓解腰部压力", "帮助调整到更舒适的姿势",
		"减轻腰椎负担", "提升休息姿势的舒适度",
		"能很好地托住腰背", "重点看腰部承托和贴合感",
		"很适合老人家使用", "适合带老人到店试躺对比",
		"马上安排", "会转给",
		"立即安排", "会转给",
		"我会安排门店顾问在您到店前做好准备", "会转给门店顾问确认到店安排",
		"我马上安排专人联系您", "我会转给门店顾问确认",
		"我马上帮您解决", "我先帮您记录并转给售后顾问确认",
		"我这就马上帮您登记售后诉求来处理", "我先帮您记录售后诉求并转给售后顾问确认",
		"我马上帮您登记", "我先帮您记录并转给售后顾问确认",
		"我马上帮您登记转人工", "我先帮您记录并转人工确认",
		"安排售后专员直接跟您对接处理", "转给售后顾问确认处理方式",
		"直接安排售后顾问联系您确认上门检测时间", "转给售后顾问确认检测方式",
		"转给门店顾问尽快确认", "转给门店顾问确认",
		"我记录后尽快安排对接", "我记录后转给售后顾问确认",
		"会尽快给您一个反馈", "会转给售后顾问确认反馈方式",
		"如果是质量问题我们一定负责处理", "是否属于质量问题及处理方式需以售后检测和订单条款确认为准",
		"一定尽快解决问题", "继续跟进确认处理方式",
		"会立刻记录您的诉求", "会记录您的诉求",
		"彻底帮您解决", "继续帮您跟进确认",
		"很多人一开始也会这么问", "这个问题很常见",
		"很多人一躺就觉得", "试躺时可以重点感受",
		"很多怕软又怕硬的顾客试过都觉得刚好", "怕软又怕硬的话，建议重点感受支撑和表层舒适度是否平衡",
		"很多家庭选它是因为", "从产品定位看，它的特点是",
		"很多重视腰背支撑的家庭都选它", "从产品定位看，它更偏支撑承托",
		"很多朋友纠结的点", "比较常见的顾虑",
		"很多家庭到店首选", "建议到店重点试躺",
		"很多老顾客反馈", "建议到店试躺确认",
		"很多客人反馈", "从产品定位看",
		"很多客人试过都说", "试躺时可以重点感受",
		"很多对腰背有要求的人试下来都说", "从产品定位看",
		"按摩功能", "升降功能",
		"有的，慕斯智能电动床带头脚升降功能", "它主要是头脚升降功能",
		"有的，慕斯智能电动床", "慕斯智能电动床核心是头脚升降",
		"有的。慕斯智能电动床带有多模式调节功能", "慕斯智能电动床核心是头脚升降和不同休息角度调节，是否有其他功能以门店实物确认为准",
		"有的。慕斯智能电动床", "慕斯智能电动床核心是头脚升降",
		"单独换个合适的枕头效果就很明显", "单独先换合适的枕头也可能有帮助，建议试枕确认",
		"肩颈腰都能放松", "有助于提升肩颈和腰部的承托、释压体验",
		"放松颈椎", "改善颈肩支撑体验",
		"颈椎放松", "颈肩支撑",
		"减轻腰部的压力", "改善睡眠时的受力感受",
		"减少腰部的压力", "改善睡眠时的受力感受",
		"减少平躺时腰椎悬空的问题", "帮助观察平躺时腰部贴合是否更稳定",
		"改善睡姿受力", "改善睡眠时的支撑感受",
		"头、肩、腰、臀、腿五个区域做了不同硬度的弹簧排布", "不同区域做了承托差异设计",
		"头、肩、腰、臀、腿五个区域", "不同承托区域",
		"腰臀部位支撑力更强", "腰臀区域更强调承托",
		"弹簧线径更粗、支撑力更强", "腰部区域更强调承托",
		"弹簧线径更粗", "腰部区域更强调承托",
		"高密度冷泡棉", "舒适承托材料",
		"2cm厚的乳胶", "舒适释压层",
		"不会硬推高价品", "可以按预算范围推荐，并尊重您的选择",
		"李叔您好", "李先生您好",
		"李女士/先生，您好。", "您好，",
		"睡眠顾问小眠", "慕小眠",
		"老人起床时轻轻按个键，背部缓缓抬起，省力又安全", "老人可以现场体验抬背起身角度和按键手感，是否顺手以实际试用为准",
		"省力又安全", "更方便体验起身角度",
		"不容易睡塌", "支撑稳定性建议试躺确认",
		"太好了！", "好的，",
		"（您在上海的话，也欢迎到店做排骨架检测确认问题，方便时也可以留一下电话）", "也可以补充订单号、型号和异响位置，方便售后确认。",
		"到店也是一样的", "到店会以门店当天活动和顾问确认为准",
		"不会到店再变", "具体活动和最终成交价以门店确认为准",
		"门店所有产品都是明码标价", "门店产品会按标价和活动规则说明",
		"部分活动款支持体验期", "是否有体验权益需由门店顾问按活动和订单条款确认",
		"要求床垫无污损、无折痕", "具体条件以订单条款为准",
		"帮您提前留好时段", "帮您记录到店意向，具体时段待门店顾问确认",
		"护脊效果好", "更强调支撑承托",
		"脊护效果好", "更强调支撑承托",
	)
	return strings.TrimSpace(replacer.Replace(text))
}

func ensureLowBudgetPath(replyText string, customerContent string) string {
	text := strings.TrimSpace(replyText)
	content := strings.TrimSpace(customerContent)
	if text == "" || !mentionsLowBudget(content) {
		return text
	}
	if containsAny(text, "三四千", "样品", "活动款", "先确认预算是否匹配") {
		return text
	}
	return text + "\n\n如果预算最多三四千，我会先把预期说清楚：慕斯常规床垫主力款大多会高于这个范围，建议优先问门店是否有样品、阶段活动款或更基础配置；如果没有匹配款，也可以先到店只做试躺对比，暂不下单。"
}

func mentionsLowBudget(value string) bool {
	text := strings.TrimSpace(value)
	return containsAny(text, "三四千", "3-4千", "3000", "4000", "三千", "四千") ||
		(strings.Contains(text, "预算") && containsAny(text, "很低", "有限", "紧", "不高"))
}

func tailorDigitalStoreFallback(replyText string, customerContent string) string {
	text := strings.TrimSpace(replyText)
	content := strings.TrimSpace(customerContent)
	if text == "" || !isGenericDigitalStoreFallback(text) {
		return text
	}
	switch {
	case containsAny(content, "颈肩", "脖子", "枕头", "t10", "T10"):
		return "颈肩不舒服可以先从枕高和颈肩承托看起，不一定要和床垫成套买。慕斯T10释压枕偏分区承托、慢回弹释压，适合侧睡较多、颈肩紧或想调整枕头高度的人先试枕确认。预算有限的话，可以先试枕头，再看床垫是否也需要调整。"
	case containsAny(content, "老人", "起夜", "起身", "电动床"):
		return "老人起夜多、起身困难，可以重点了解慕斯智能电动床的头脚升降功能，先看起身角度、遥控按键是否顺手、床垫适配和安全细节。具体配置、库存、活动和体验时段需要门店顾问确认，建议带老人到店实际试一下。"
	case containsAny(content, "护脊", "噱头", "腰", "背", "治好", "治疗"):
		return "护脊不要只听概念，主要看支撑和贴合：仰卧时腰部是否悬空，侧卧时肩臀是否压迫，翻身是否费力。床垫不能替代医疗诊断或治疗，也不能保证治好腰背问题；如果持续疼痛建议先咨询医生，到店可重点对比脊护支撑款和云感舒睡款。"
	case containsAny(content, "预算", "贵", "1.8", "最低", "便宜", "方案"):
		return "先给您一个可对比的方向：1.8米如果预算约8000-13000元，可以看云感舒睡款，偏柔和包裹；预算约12000-18000元，可以看脊护支撑款，偏支撑承托。最终价格、库存和活动需要门店顾问确认，您可以先告诉我偏软还是偏硬。"
	case containsAny(content, "售后", "异响", "投诉", "咯吱", "退"):
		return "真的抱歉影响您休息了。异响原因需要结合订单、产品型号、床架/排骨架和现场情况确认，我先帮您记录售后诉求并转人工顾问；退换或赔付需要以订单条款和售后检测结果为准。"
	}
	return "我在的。您可以直接说预算、尺寸、使用人群或当前睡眠困扰，我会先给可执行的产品方向；涉及最终价格、库存、活动、退换或售后结论，再由门店顾问确认。"
}

func isGenericDigitalStoreFallback(text string) bool {
	return containsAny(text,
		"你的问题我已经记录",
		"你问的问题我已经记录",
		"涉及最终价格、库存、退换货、售后争议或医疗效果时",
		"会安排门店顾问进一步确认",
	)
}

func ensureConcretePlanWhenAsked(replyText string, customerContent string) string {
	text := strings.TrimSpace(replyText)
	content := strings.TrimSpace(customerContent)
	if text == "" || !containsAny(content, "什么方案", "啥方案", "给我方案", "具体方案") {
		return text
	}
	if containsAny(text, "云感", "脊护", "电动床", "方案一", "方案二", "8000", "12000") {
		return text
	}
	return "先给您一个当前可执行方案：1.8米可以先看两种方向，云感舒睡款偏柔和包裹，价格大致8000-13000元；脊护支撑款偏支撑承托，价格大致12000-18000元。库存、到店活动和最终成交价需要门店顾问确认。您只需要再确认偏软还是偏硬，我就能帮您把范围缩到一款。"
}

func sanitizeChitchatReply(replyText string, customerContent string) string {
	text := strings.TrimSpace(replyText)
	content := strings.TrimSpace(customerContent)
	if text == "" || !containsAny(content, "写诗", "闲聊", "聊天", "没反应", "在吗", "听得懂") {
		return text
	}
	replacer := strings.NewReplacer(
		"抱歉让您久等了，我一直都在。您之前提到睡眠方面有些困扰，能再具体说说是什么情况吗？比如是腰背不舒服，还是觉得现在床垫太软太硬？我帮您分析分析，看看哪款更适合您。", "抱歉让您久等了，我在的。您想继续随便看看，还是我直接帮您按预算和睡感筛两款？",
		"您之前提到睡眠方面有些困扰", "如果您想看床垫",
		"比如是腰背不舒服，还是觉得现在床垫太软太硬？", "可以从预算、偏软偏硬、给谁使用这几个点看。",
		"有没有腰酸背累或者床垫不舒服的情况？", "如果想聊睡眠或床垫，我也可以帮您看看。",
		"有没有腰酸背累", "有没有想改善的睡眠体验",
		"跟我说说，我帮您分析分析。", "您想随便看看还是了解某一类产品？",
		"写诗我可不太擅长", "可以简单来一句",
	)
	return strings.TrimSpace(replacer.Replace(text))
}

func sanitizeReplyAgainstCustomerContext(replyText string, conversationID int64) string {
	text := strings.TrimSpace(replyText)
	if text == "" || conversationID <= 0 || customerHistoryMentionsBackConcern(conversationID) {
		return text
	}
	replacer := strings.NewReplacer(
		"腰背不适的话，", "",
		"腰背不舒服，", "",
		"腰背不舒服", "支撑感",
		"有没有腰酸背疼或者喜欢侧睡？", "平时更习惯侧睡还是仰睡？",
		"有没有腰酸背疼", "有没有想改善的睡眠感受",
		"腰背很友好", "支撑承托更明确",
		"腰疼的话，", "",
		"如果腰疼，", "如果关注腰背支撑，",
		"如果您腰背不适，", "如果您关注支撑感，",
		"结合您提到腰疼的情况", "如果您关注腰背支撑",
		"您提到腰疼", "您关注腰背支撑",
		"您这样需要针对性支撑的情况", "想要更强承托的情况",
		"适合您这样需要针对性支撑的情况", "适合想要更强承托的人群",
		"对腰背压力大的人群", "对关注腰背支撑的人群",
	)
	return strings.TrimSpace(replacer.Replace(text))
}

func sanitizeReplyAgainstKnownLeadInfo(replyText string, conversationID int64) string {
	text := strings.TrimSpace(replyText)
	if text == "" || conversationID <= 0 || !customerHistoryHasPhone(conversationID) {
		return text
	}
	replacer := strings.NewReplacer(
		"麻烦您再留一下您的姓名和手机号，方便顾问提前帮您安排体验和礼包～", "我会把这些信息转给门店顾问，后续由顾问确认具体时段和体验安排。",
		"方便留一下您的姓名和手机号吗？", "我先按当前信息记录，后续由门店顾问确认。",
		"方便留个姓名和手机号吗？", "我先按当前信息记录，后续由门店顾问确认。",
		"方便留个姓名和电话吗？", "我先按当前信息记录，后续由门店顾问确认。",
		"方便留下姓名和手机号吗？", "我先按当前信息记录，后续由门店顾问确认。",
		"方便留个联系方式吗？", "我会使用您已提供的联系方式转给门店顾问确认。",
		"方便留一下联系方式吗？", "我会使用您已提供的联系方式转给门店顾问确认。",
		"您可以留下手机号", "我已看到您提供的手机号",
	)
	return strings.TrimSpace(replacer.Replace(text))
}

func sanitizeStoreMentionAgainstCustomerContext(replyText string, conversationID int64) string {
	text := strings.TrimSpace(replyText)
	if text == "" || conversationID <= 0 || customerHistoryMentionsStoreArea(conversationID) {
		return text
	}
	replacer := strings.NewReplacer(
		"徐汇门店", "门店",
		"徐汇店", "门店",
		"上海徐汇", "所在城市",
	)
	return strings.TrimSpace(replacer.Replace(text))
}

func customerHistoryMentionsBackConcern(conversationID int64) bool {
	messages := repositories.MessageRepository.Find(sqls.DB(), sqls.NewCnd().
		Where("conversation_id = ?", conversationID).
		Where("sender_type = ?", enums.IMSenderTypeCustomer).
		Desc("id").
		Limit(20))
	for _, message := range messages {
		if containsAny(message.Content, "腰", "背", "酸", "疼", "痛", "僵", "护脊", "支撑") {
			return true
		}
	}
	return false
}

func customerHistoryHasPhone(conversationID int64) bool {
	messages := repositories.MessageRepository.Find(sqls.DB(), sqls.NewCnd().
		Where("conversation_id = ?", conversationID).
		Where("sender_type = ?", enums.IMSenderTypeCustomer).
		Desc("id").
		Limit(30))
	for _, message := range messages {
		if hasMainlandPhone(message.Content) {
			return true
		}
	}
	return false
}

func customerHistoryMentionsStoreArea(conversationID int64) bool {
	messages := repositories.MessageRepository.Find(sqls.DB(), sqls.NewCnd().
		Where("conversation_id = ?", conversationID).
		Where("sender_type = ?", enums.IMSenderTypeCustomer).
		Desc("id").
		Limit(30))
	for _, message := range messages {
		if containsAny(message.Content, "徐汇", "上海", "门店", "到店", "试躺", "周六", "周日", "周末", "预约") {
			return true
		}
	}
	return false
}

func hasMainlandPhone(value string) bool {
	digits := make([]rune, 0, 16)
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	text := string(digits)
	for i := 0; i+11 <= len(text); i++ {
		if text[i] == '1' && text[i+1] >= '3' && text[i+1] <= '9' {
			return true
		}
	}
	return false
}
