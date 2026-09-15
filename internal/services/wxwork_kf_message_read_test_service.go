package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/utils"
)

// 企业微信客服“测试读取消息”相关常量。
// 说明：silenceper/wechat SDK 的 SyncMsg 在 errcode!=0 时只返回 errmsg 文本、丢失 errcode，
// 无法按错误码给出排查建议，因此该测试直接调用官方 API 并自行解析 errcode。
const (
	wxWorkGetTokenURLTemplate = "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s"
	wxWorkSyncMsgURLTemplate  = "https://qyapi.weixin.qq.com/cgi-bin/kf/sync_msg?access_token=%s"

	wxWorkReadTestStageGetToken = "gettoken"
	wxWorkReadTestStageSyncMsg  = "syncmsg"

	// 本地错误码（非企业微信返回），前端按这些码做本地化排查建议。
	wxWorkReadTestErrInvalidChannel = "INVALID_CHANNEL"
	wxWorkReadTestErrOpenKFMissing  = "OPENKFID_MISSING"
	wxWorkReadTestErrConfigJSON     = "INVALID_CONFIG_JSON"
	wxWorkReadTestErrDisabled       = "WXWORK_DISABLED"
	wxWorkReadTestErrNetwork        = "NETWORK_ERROR"
	wxWorkReadTestErrBadResponse    = "BAD_RESPONSE"

	wxWorkReadTestSyncLimit       uint64 = 1000 // 单次 sync_msg 拉取条数（官方最大值）
	wxWorkReadTestMaxPages             = 5    // 最多翻页次数，限制测试请求耗时
	wxWorkReadTestSampleSize           = 20   // 返回最近样例条数
	wxWorkReadTestMaxPreviewRunes      = 500  // 文本样例最大字符数
	wxWorkReadTestHTTPTimeout          = 10 * time.Second
)

var wxWorkReadTestHTTPClient = &http.Client{Timeout: wxWorkReadTestHTTPTimeout}

type wxWorkReadTestTokenResponse struct {
	ErrCode     int64  `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type wxWorkReadTestSyncRequest struct {
	Cursor   string `json:"cursor"`
	Limit    uint64 `json:"limit"`
	OpenKfID string `json:"open_kfid"`
}

type wxWorkReadTestSyncResponse struct {
	ErrCode    int64          `json:"errcode"`
	ErrMsg     string         `json:"errmsg"`
	NextCursor string         `json:"next_cursor"`
	HasMore    uint32         `json:"has_more"`
	MsgList    []map[string]any `json:"msg_list"`
}

// TestWxWorkKFReadMessages 对指定企微客服渠道执行只读消息拉取测试：
// 使用空游标从企业微信拉取该客服账号最近保留期（3 天）内的消息与事件，
// 不消费消息、不创建会话、不更新同步游标。返回结构化结果供前端展示与错误码排查。
func (s *channelService) TestWxWorkKFReadMessages(channelID int64) (*response.WxWorkKFMessageReadTestResult, error) {
	channel := s.Get(channelID)
	if channel == nil || channel.Status == enums.StatusDeleted || channel.ChannelType != enums.ChannelTypeWxWorkKF {
		return s.failReadTestResult("", wxWorkReadTestErrInvalidChannel, ""), nil
	}

	openKfID := ""
	if cfg, err := s.ParseWxWorkKFChannelConfig(channel.ConfigJSON); err != nil {
		return s.failReadTestResult("", wxWorkReadTestErrConfigJSON, err.Error()), nil
	} else if cfg != nil {
		openKfID = cfg.OpenKfID
	}
	if openKfID == "" {
		return s.failReadTestResult("", wxWorkReadTestErrOpenKFMissing, ""), nil
	}

	wxConfig := config.Current().WxWork
	corpID := strings.TrimSpace(wxConfig.CorpID)
	corpSecret := strings.TrimSpace(wxConfig.CorpSecret)
	if !wxConfig.Enabled || corpID == "" || corpSecret == "" {
		return s.failReadTestResult("", wxWorkReadTestErrDisabled, ""), nil
	}

	// 1. 获取 access_token（独立调用，不接触 SDK 令牌缓存；人工测试频率低，可接受）
	accessToken, result := s.fetchReadTestAccessToken(corpID, corpSecret)
	if result != nil {
		return result, nil
	}

	// 2. 从空游标开始分页拉取（空游标返回保留期内最早的消息，按时间升序）
	scanned := make([]map[string]any, 0)
	cursor := ""
	truncated := false
	for page := 0; page < wxWorkReadTestMaxPages; page++ {
		resp, err := s.callReadTestSyncMsg(accessToken, openKfID, cursor)
		if err != nil {
			slog.Warn("wxwork kf read test sync_msg failed",
				"channel_id", channelID,
				"open_kfid", openKfID,
				"error", err,
			)
			return s.failReadTestResult(wxWorkReadTestStageSyncMsg, wxWorkReadTestErrNetwork, err.Error()), nil
		}
		if resp.ErrCode != 0 {
			slog.Warn("wxwork kf read test sync_msg rejected",
				"channel_id", channelID,
				"open_kfid", openKfID,
				"errcode", resp.ErrCode,
				"errmsg", resp.ErrMsg,
			)
			return s.failReadTestResult(wxWorkReadTestStageSyncMsg, strconv.FormatInt(resp.ErrCode, 10), resp.ErrMsg), nil
		}

		scanned = append(scanned, resp.MsgList...)
		if resp.HasMore != 1 || strings.TrimSpace(resp.NextCursor) == "" {
			truncated = false
			break
		}
		cursor = resp.NextCursor
		if page == wxWorkReadTestMaxPages-1 {
			truncated = true
		}
	}

	result2 := s.buildReadTestResult(openKfID, scanned, truncated)
	slog.Info("wxwork kf read test succeeded",
		"channel_id", channelID,
		"open_kfid", openKfID,
		"total_scanned", result2.TotalScanned,
		"message_count", result2.MessageCount,
		"event_count", result2.EventCount,
		"truncated", result2.Truncated,
	)
	return result2, nil
}

func (s *channelService) fetchReadTestAccessToken(corpID, corpSecret string) (string, *response.WxWorkKFMessageReadTestResult) {
	requestURL := fmt.Sprintf(wxWorkGetTokenURLTemplate, url.QueryEscape(corpID), url.QueryEscape(corpSecret))
	ctx, cancel := context.WithTimeout(context.Background(), wxWorkReadTestHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", s.failReadTestResult(wxWorkReadTestStageGetToken, wxWorkReadTestErrNetwork, err.Error())
	}
	resp, err := wxWorkReadTestHTTPClient.Do(req)
	if err != nil {
		return "", s.failReadTestResult(wxWorkReadTestStageGetToken, wxWorkReadTestErrNetwork, err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", s.failReadTestResult(wxWorkReadTestStageGetToken, wxWorkReadTestErrNetwork, err.Error())
	}
	tokenResp := wxWorkReadTestTokenResponse{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", s.failReadTestResult(wxWorkReadTestStageGetToken, wxWorkReadTestErrBadResponse, truncateReadTestText(string(body), 300))
	}
	if tokenResp.ErrCode != 0 || strings.TrimSpace(tokenResp.AccessToken) == "" {
		code := strconv.FormatInt(tokenResp.ErrCode, 10)
		if tokenResp.ErrCode == 0 {
			code = wxWorkReadTestErrBadResponse
		}
		return "", s.failReadTestResult(wxWorkReadTestStageGetToken, code, tokenResp.ErrMsg)
	}
	return tokenResp.AccessToken, nil
}

func (s *channelService) callReadTestSyncMsg(accessToken, openKfID, cursor string) (*wxWorkReadTestSyncResponse, error) {
	payload, err := json.Marshal(wxWorkReadTestSyncRequest{
		Cursor:   cursor,
		Limit:    wxWorkReadTestSyncLimit,
		OpenKfID: openKfID,
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), wxWorkReadTestHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf(wxWorkSyncMsgURLTemplate, url.QueryEscape(accessToken)),
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := wxWorkReadTestHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	syncResp := wxWorkReadTestSyncResponse{}
	if err := json.Unmarshal(body, &syncResp); err != nil {
		return nil, fmt.Errorf("invalid response: %s", truncateReadTestText(string(body), 300))
	}
	return &syncResp, nil
}

// buildReadTestResult 汇总扫描结果并抽取最近若干条样例（新消息在前）。
func (s *channelService) buildReadTestResult(openKfID string, scanned []map[string]any, truncated bool) *response.WxWorkKFMessageReadTestResult {
	result := &response.WxWorkKFMessageReadTestResult{
		Success:      true,
		OpenKfID:     openKfID,
		TotalScanned: len(scanned),
		Samples:      make([]response.WxWorkKFMessageReadSample, 0),
		Truncated:    truncated,
	}

	var earliestUnix, latestUnix int64
	for _, raw := range scanned {
		msgType := readTestAsString(raw["msgtype"])
		if msgType == "event" {
			result.EventCount++
		} else {
			result.MessageCount++
		}
		sendTime := readTestAsInt64(raw["send_time"])
		if sendTime > 0 {
			if earliestUnix == 0 || sendTime < earliestUnix {
				earliestUnix = sendTime
			}
			if sendTime > latestUnix {
				latestUnix = sendTime
			}
		}
	}
	if earliestUnix > 0 {
		result.EarliestTime = utils.FormatTime(time.Unix(earliestUnix, 0))
	}
	if latestUnix > 0 {
		result.LatestTime = utils.FormatTime(time.Unix(latestUnix, 0))
	}

	// 接口按时间升序返回，取最后 N 条后反转为新消息在前
	start := len(scanned) - wxWorkReadTestSampleSize
	if start < 0 {
		start = 0
	}
	for i := len(scanned) - 1; i >= start; i-- {
		result.Samples = append(result.Samples, buildWxWorkReadSample(scanned[i]))
	}
	return result
}

func buildWxWorkReadSample(raw map[string]any) response.WxWorkKFMessageReadSample {
	sample := response.WxWorkKFMessageReadSample{
		MsgID:          readTestAsString(raw["msgid"]),
		Origin:         int(readTestAsInt64(raw["origin"])),
		MsgType:        readTestAsString(raw["msgtype"]),
		ExternalUserID: readTestAsString(raw["external_userid"]),
		ServicerUserID: readTestAsString(raw["servicer_userid"]),
	}
	if sendTime := readTestAsInt64(raw["send_time"]); sendTime > 0 {
		sample.SendTime = utils.FormatTime(time.Unix(sendTime, 0))
	}
	if sample.MsgType == "text" {
		if textMap, ok := raw["text"].(map[string]any); ok {
			sample.TextContent = truncateReadTestText(strings.TrimSpace(readTestAsString(textMap["content"])), wxWorkReadTestMaxPreviewRunes)
		}
	}
	if sample.MsgType == "event" {
		if eventMap, ok := raw["event"].(map[string]any); ok {
			sample.EventType = readTestAsString(eventMap["event_type"])
			// 事件消息的客户/客服账号在 event 对象内
			if sample.ExternalUserID == "" {
				sample.ExternalUserID = readTestAsString(eventMap["external_userid"])
			}
			if sample.ServicerUserID == "" {
				sample.ServicerUserID = readTestAsString(eventMap["new_servicer_userid"])
			}
		}
	}
	return sample
}

func (s *channelService) failReadTestResult(stage, code, message string) *response.WxWorkKFMessageReadTestResult {
	return &response.WxWorkKFMessageReadTestResult{
		Success:      false,
		Stage:        stage,
		ErrorCode:    code,
		ErrorMessage: truncateReadTestText(strings.TrimSpace(message), 1000),
		Samples:      make([]response.WxWorkKFMessageReadSample, 0),
	}
}

func readTestAsString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func readTestAsInt64(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func truncateReadTestText(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "…"
}
