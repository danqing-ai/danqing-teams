package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"danqing-teams/core/adapter/weixin/ilink"
	"danqing-teams/core/domain"
	"danqing-teams/core/port"

	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

const weixinBusyReply = "上一条消息还在处理中，请稍后再试。"

type WeixinBridge struct {
	client   *ilink.Client
	store    port.Repository
	sessions *SessionManager
	projects *ProjectManager
	config   *ConfigManager

	mu       sync.Mutex
	running  bool
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logins   map[string]*weixinLoginSession
	cfgCache domain.ConfigWeixinChannel
}

type weixinLoginSession struct {
	SessionKey string
	QRCode     string
	QRCodeURL  string
	StartedAt  time.Time
	BaseURL    string
}

func NewWeixinBridge(store port.Repository, sessions *SessionManager, projects *ProjectManager, config *ConfigManager) *WeixinBridge {
	return &WeixinBridge{
		client:   ilink.NewClient(),
		store:    store,
		sessions: sessions,
		projects: projects,
		config:   config,
		logins:   make(map[string]*weixinLoginSession),
	}
}

// SetClient replaces the iLink HTTP client (tests).
func (b *WeixinBridge) SetClient(c *ilink.Client) {
	if c != nil {
		b.client = c
	}
}

func (b *WeixinBridge) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

func (b *WeixinBridge) loadChannelCfg(ctx context.Context) (domain.ConfigWeixinChannel, error) {
	cfg, err := b.config.Get(ctx)
	if err != nil {
		return domain.ConfigWeixinChannel{}, err
	}
	wx := cfg.Channels.Weixin
	if !wx.AutoApprove && cfg.Channels.Weixin.Enabled {
		// default true when enabled unless explicitly set false in yaml;
		// zero-value AutoApprove is false — treat unset as true via Sync defaults in Ensure.
	}
	return wx, nil
}

// EnsureWeixinProject finds or creates the dedicated 「微信」 project and persists its id.
// Never reuses a default_project_id that points at a differently named project
// (e.g. a leftover business project from an earlier config).
func (b *WeixinBridge) EnsureWeixinProject(ctx context.Context) (string, error) {
	cfg, err := b.config.Get(ctx)
	if err != nil {
		return "", err
	}
	wx := cfg.Channels.Weixin
	if wx.DefaultProjectID != "" {
		if p, err := b.projects.Get(ctx, wx.DefaultProjectID); err == nil {
			if p.Name == domain.WeixinProjectName {
				return p.ID, nil
			}
			// Stale id pointing at another project — fall through and fix.
			log.Printf("[weixin] default_project_id=%s name=%q is not %q; recreating", p.ID, p.Name, domain.WeixinProjectName)
		}
	}
	projects, err := b.projects.List(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.Name == domain.WeixinProjectName {
			return b.persistWeixinProjectID(ctx, cfg, p.ID)
		}
	}
	p, err := b.projects.Create(ctx, domain.CreateProjectRequest{Name: domain.WeixinProjectName})
	if err != nil {
		return "", err
	}
	return b.persistWeixinProjectID(ctx, cfg, p.ID)
}

func (b *WeixinBridge) persistWeixinProjectID(ctx context.Context, cfg *domain.ConfigFile, projectID string) (string, error) {
	wx := cfg.Channels.Weixin
	wx.DefaultProjectID = projectID
	if !wx.AutoApprove {
		wx.AutoApprove = true
	}
	sec := cfg.Channels
	sec.Weixin = wx
	if _, err := b.config.Update(ctx, domain.UpdateConfigFileRequest{Channels: &sec}); err != nil {
		return projectID, err
	}
	b.mu.Lock()
	b.cfgCache = wx
	b.mu.Unlock()
	return projectID, nil
}

// SyncFromConfig starts or stops the bridge based on config.
func (b *WeixinBridge) SyncFromConfig(ctx context.Context) error {
	cfg, err := b.loadChannelCfg(ctx)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.cfgCache = cfg
	b.mu.Unlock()
	if cfg.Enabled {
		if cfg.DefaultAgentID == "" {
			b.Stop()
			return fmt.Errorf("channels.weixin.default_agent_id required when enabled")
		}
		if strings.TrimSpace(cfg.DefaultModelID) == "" || !strings.Contains(cfg.DefaultModelID, "/") {
			b.Stop()
			return fmt.Errorf("channels.weixin.default_model_id required when enabled (provider/model)")
		}
		if _, err := b.EnsureWeixinProject(ctx); err != nil {
			return err
		}
		// Always restart so newly logged-in accounts get a monitor goroutine.
		b.Stop()
		return b.Start(ctx)
	}
	b.Stop()
	return nil
}

func (b *WeixinBridge) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.running = true
	b.mu.Unlock()

	accounts, err := b.store.WeixinAccounts().List(ctx)
	if err != nil {
		b.Stop()
		return err
	}
	for _, acc := range accounts {
		if strings.TrimSpace(acc.Token) == "" {
			continue
		}
		acc := acc
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.monitorAccount(runCtx, acc)
		}()
	}
	log.Printf("[weixin] bridge started with %d account(s)", len(accounts))
	return nil
}

func (b *WeixinBridge) Stop() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	if b.cancel != nil {
		b.cancel()
	}
	b.running = false
	b.mu.Unlock()
	b.wg.Wait()
	log.Printf("[weixin] bridge stopped")
}

func (b *WeixinBridge) channelCfg() domain.ConfigWeixinChannel {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cfgCache
}

func (b *WeixinBridge) refreshCfg(ctx context.Context) {
	if cfg, err := b.loadChannelCfg(ctx); err == nil {
		b.mu.Lock()
		b.cfgCache = cfg
		b.mu.Unlock()
	}
}

func (b *WeixinBridge) toILinkAccount(a domain.WeixinAccount) ilink.Account {
	base := a.BaseURL
	if base == "" {
		base = ilink.DefaultBaseURL
	}
	return ilink.Account{
		AccountID: a.AccountID,
		Token:     a.Token,
		BaseURL:   base,
		UserID:    a.UserID,
	}
}

func (b *WeixinBridge) monitorAccount(ctx context.Context, acc domain.WeixinAccount) {
	ilAcc := b.toILinkAccount(acc)
	_ = b.client.NotifyStart(ctx, ilAcc)
	defer func() { _ = b.client.NotifyStop(context.Background(), ilAcc) }()

	syncBuf := acc.SyncBuf
	timeout := 35 * time.Second
	failures := 0
	for {
		if ctx.Err() != nil {
			return
		}
		b.refreshCfg(ctx)
		resp, err := b.client.GetUpdates(ctx, ilAcc, syncBuf, timeout)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			failures++
			delay := 2 * time.Second
			if failures >= 3 {
				delay = 30 * time.Second
				failures = 0
			}
			log.Printf("[weixin] getupdates account=%s err=%v", acc.AccountID, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			continue
		}
		failures = 0
		if resp.LongPollingTimeoutMs > 0 {
			timeout = time.Duration(resp.LongPollingTimeoutMs) * time.Millisecond
		}
		if resp.Ret != 0 || resp.ErrCode != 0 {
			failures++
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		if resp.GetUpdatesBuf != "" {
			syncBuf = resp.GetUpdatesBuf
			_ = b.store.WeixinAccounts().UpdateSyncBuf(ctx, acc.AccountID, syncBuf)
		}
		for _, msg := range resp.Msgs {
			if ctx.Err() != nil {
				return
			}
			if msg.MessageType == ilink.MessageTypeBot {
				continue
			}
			peer := strings.TrimSpace(msg.FromUserID)
			if peer == "" {
				continue
			}
			text := ilink.TextFromMessage(msg)
			if text == "" {
				_ = b.client.SendText(ctx, ilAcc, peer, "暂时只支持文本消息。", msg.ContextToken, "")
				continue
			}
			if msg.ContextToken != "" {
				_ = b.store.WeixinBindings().UpdateContextToken(ctx, acc.AccountID, peer, msg.ContextToken)
			}
			reply, err := b.handleInbound(ctx, acc, peer, text, msg.ContextToken)
			if err != nil {
				log.Printf("[weixin] handle inbound peer=%s: %v", peer, err)
				reply = "处理消息时出错：" + err.Error()
			}
			if strings.TrimSpace(reply) == "" {
				continue
			}
			ctxTok := msg.ContextToken
			if binding, berr := b.store.WeixinBindings().GetByPeer(ctx, acc.AccountID, peer); berr == nil && binding.ContextToken != "" {
				ctxTok = binding.ContextToken
			}
			if err := b.client.SendText(ctx, ilAcc, peer, reply, ctxTok, ""); err != nil {
				log.Printf("[weixin] send reply peer=%s: %v", peer, err)
			}
		}
	}
}

func (b *WeixinBridge) sessionHasRunningTurn(sessionID string) bool {
	for _, t := range b.sessions.ListTurns(sessionID) {
		if t.Status == domain.TurnRunning {
			return true
		}
	}
	return false
}

func (b *WeixinBridge) handleInbound(ctx context.Context, acc domain.WeixinAccount, peer, text, contextToken string) (string, error) {
	cfg := b.channelCfg()
	if cfg.DefaultAgentID == "" {
		return "", fmt.Errorf("未配置默认 Agent，请在设置中选择")
	}
	projectID, err := b.EnsureWeixinProject(ctx)
	if err != nil {
		return "", err
	}
	// Refresh cache after ensure (may have rewritten default_project_id).
	b.refreshCfg(ctx)
	cfg = b.channelCfg()

	modelID := strings.TrimSpace(cfg.DefaultModelID)
	if modelID == "" || !strings.Contains(modelID, "/") {
		return "", fmt.Errorf("未配置默认模型，请在设置 → 微信中选择模型（格式 provider/model）")
	}

	binding, err := b.store.WeixinBindings().GetByPeer(ctx, acc.AccountID, peer)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var sessionID string
	if errors.Is(err, gorm.ErrRecordNotFound) || binding.SessionID == "" {
		// Create session+first turn under subscription.
		// Title from first message — never use opaque peer openids as the label.
		s, cerr := b.sessions.Create(ctx, domain.CreateSessionRequest{
			Content:       text,
			AgentID:       cfg.DefaultAgentID,
			ProjectID:     projectID,
			ModelID:       modelID,
			Title:         weixinSessionTitle(text),
			SkipAutoTitle: true,
		})
		if cerr != nil {
			return "", cerr
		}
		sessionID = s.ID
		binding = domain.WeixinBinding{
			AccountID:    acc.AccountID,
			PeerUserID:   peer,
			SessionID:    sessionID,
			ContextToken: contextToken,
		}
		if uerr := b.store.WeixinBindings().Upsert(ctx, binding); uerr != nil {
			return "", uerr
		}
		ch := b.sessions.Subscribe(sessionID)
		defer b.sessions.Unsubscribe(sessionID, ch)
		turnID := b.waitLatestTurnID(sessionID, 2*time.Second)
		return b.collectReplyFrom(ctx, sessionID, ch, turnID, cfg.AutoApprove), nil
	}

	sessionID = binding.SessionID
	if contextToken != "" && contextToken != binding.ContextToken {
		_ = b.store.WeixinBindings().UpdateContextToken(ctx, acc.AccountID, peer, contextToken)
	}
	if b.sessionHasRunningTurn(sessionID) {
		return weixinBusyReply, nil
	}
	// Persist model onto session if earlier turns were created without one.
	if s, gerr := b.sessions.Get(ctx, sessionID); gerr == nil && strings.TrimSpace(s.ModelID) == "" {
		_, _ = b.sessions.Update(ctx, sessionID, domain.UpdateSessionRequest{ModelID: &modelID})
	}
	ch := b.sessions.Subscribe(sessionID)
	defer b.sessions.Unsubscribe(sessionID, ch)
	turnID, serr := b.sessions.StartTurn(ctx, sessionID, domain.SendMessageRequest{
		UserInput: text,
		AgentID:   cfg.DefaultAgentID,
		ModelID:   modelID,
	})
	if serr != nil {
		return "", serr
	}
	return b.collectReplyFrom(ctx, sessionID, ch, turnID, cfg.AutoApprove), nil
}

func weixinSessionTitle(text string) string {
	title := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if title == "" {
		return "微信会话"
	}
	// Rune-aware truncate for CJK titles.
	runes := []rune(title)
	if len(runes) > 24 {
		return string(runes[:24]) + "…"
	}
	return title
}

// activateAfterLogin persists login state. Enabling still requires Agent + Model
// chosen explicitly in settings (no silent defaults).
func (b *WeixinBridge) activateAfterLogin(ctx context.Context) string {
	cfg, err := b.config.Get(ctx)
	if err != nil {
		return "已连接微信。请在设置中选择 Agent 与模型并启用通道。"
	}
	wx := cfg.Channels.Weixin
	if wx.Enabled && wx.DefaultAgentID != "" && strings.Contains(wx.DefaultModelID, "/") {
		if err := b.SyncFromConfig(ctx); err != nil {
			return "已连接微信，但启动 Bridge 失败：" + err.Error()
		}
		return ""
	}
	return "已连接微信。请在设置 → 微信中选择默认 Agent 与模型并启用通道，然后给机器人发消息。"
}

func (b *WeixinBridge) waitLatestTurnID(sessionID string, wait time.Duration) string {
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		turns := b.sessions.ListTurns(sessionID)
		if len(turns) > 0 {
			return turns[len(turns)-1].ID
		}
		time.Sleep(50 * time.Millisecond)
	}
	return ""
}

func (b *WeixinBridge) applyEvent(ev domain.StreamEvent, turnID string, autoApprove bool, parts *[]string) (done bool) {
	if turnID != "" && ev.TurnID != "" && ev.TurnID != turnID {
		return false
	}
	switch ev.Type {
	case domain.EventAgentMessage:
		var p domain.AgentMessagePayload
		if json.Unmarshal(ev.Payload, &p) == nil && strings.TrimSpace(p.Text) != "" {
			*parts = append(*parts, strings.TrimSpace(p.Text))
		}
	case domain.EventPermissionAsk:
		if autoApprove {
			var p domain.PermissionAskPayload
			if json.Unmarshal(ev.Payload, &p) == nil && p.ApprovalID != "" {
				_ = b.sessions.DecideApproval(context.Background(), p.ApprovalID, true, "once")
			}
		}
	case domain.EventAskUserPending:
		var p domain.AskUserPayload
		if json.Unmarshal(ev.Payload, &p) == nil && p.AskID != "" {
			_ = b.sessions.ResolveAskUser(p.AskID, "（微信通道暂不支持交互提问，请在桌面端继续）")
		}
	case domain.EventTurnEnded, domain.EventTurnFailed, domain.EventError, domain.EventSessionCompleted:
		return true
	}
	return false
}

func (b *WeixinBridge) collectReplyFrom(ctx context.Context, sessionID string, ch <-chan domain.StreamEvent, turnID string, autoApprove bool) string {
	var parts []string
	// Backfill events published before subscribe (esp. first Create turn).
	for _, ev := range b.sessions.StreamEvents(sessionID, 0) {
		if b.applyEvent(ev, turnID, autoApprove, &parts) {
			return strings.Join(parts, "\n")
		}
	}
	deadline := time.After(10 * time.Minute)
	seen := make(map[int64]struct{})
	for _, ev := range b.sessions.StreamEvents(sessionID, 0) {
		seen[ev.Seq] = struct{}{}
	}
	for {
		select {
		case <-ctx.Done():
			return strings.Join(parts, "\n")
		case <-deadline:
			if len(parts) == 0 {
				return "处理超时，请稍后在桌面端查看。"
			}
			return strings.Join(parts, "\n")
		case ev, ok := <-ch:
			if !ok {
				return strings.Join(parts, "\n")
			}
			if _, dup := seen[ev.Seq]; dup {
				continue
			}
			seen[ev.Seq] = struct{}{}
			if b.applyEvent(ev, turnID, autoApprove, &parts) {
				return strings.Join(parts, "\n")
			}
		}
	}
}

// --- Login API helpers ---

func (b *WeixinBridge) StartLogin(ctx context.Context) (domain.WeixinLoginStartResult, error) {
	accounts, _ := b.store.WeixinAccounts().List(ctx)
	tokens := make([]string, 0, len(accounts))
	for i := len(accounts) - 1; i >= 0 && len(tokens) < 10; i-- {
		if t := strings.TrimSpace(accounts[i].Token); t != "" {
			tokens = append(tokens, t)
		}
	}
	qr, err := b.client.GetBotQRCode(ctx, ilink.DefaultBotType, tokens)
	if err != nil {
		return domain.WeixinLoginStartResult{}, err
	}
	if qr.QRCode == "" || qr.QRCodeImgContent == "" {
		return domain.WeixinLoginStartResult{}, fmt.Errorf("服务器未返回二维码")
	}
	qrImage := qr.QRCodeImgContent
	if !strings.HasPrefix(qrImage, "data:image/") {
		// iLink often returns a liteapp URL, not a PNG — render QR locally for <img>.
		payload := qr.QRCodeImgContent
		if payload == "" {
			payload = qr.QRCode
		}
		png, err := qrcode.Encode(payload, qrcode.Medium, 256)
		if err != nil {
			return domain.WeixinLoginStartResult{}, fmt.Errorf("生成二维码图片失败: %w", err)
		}
		qrImage = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	}
	sessionKey := fmt.Sprintf("wxlogin-%d", time.Now().UnixNano())
	base := ilink.DefaultBaseURL
	if b.client != nil && strings.TrimSpace(b.client.BaseURL) != "" {
		base = b.client.BaseURL
	}
	b.mu.Lock()
	b.logins[sessionKey] = &weixinLoginSession{
		SessionKey: sessionKey,
		QRCode:     qr.QRCode,
		QRCodeURL:  qr.QRCodeImgContent,
		StartedAt:  time.Now(),
		BaseURL:    base,
	}
	b.mu.Unlock()
	return domain.WeixinLoginStartResult{
		SessionKey: sessionKey,
		QRCodeURL:  qrImage,
	}, nil
}

func (b *WeixinBridge) WaitLogin(ctx context.Context, sessionKey, verifyCode string, timeoutMs int) (domain.WeixinLoginWaitResult, error) {
	b.mu.Lock()
	login := b.logins[sessionKey]
	b.mu.Unlock()
	if login == nil {
		return domain.WeixinLoginWaitResult{Connected: false, Message: "当前没有进行中的登录，请先发起登录。"}, nil
	}
	if time.Since(login.StartedAt) > 5*time.Minute {
		b.mu.Lock()
		delete(b.logins, sessionKey)
		b.mu.Unlock()
		return domain.WeixinLoginWaitResult{Connected: false, Message: "二维码已过期，请重新生成。"}, nil
	}
	if timeoutMs <= 0 {
		timeoutMs = 120_000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	baseURL := login.BaseURL
	if baseURL == "" {
		baseURL = ilink.DefaultBaseURL
	}
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return domain.WeixinLoginWaitResult{}, ctx.Err()
		}
		status, err := b.client.GetQRCodeStatus(ctx, baseURL, login.QRCode, verifyCode)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		switch status.Status {
		case "wait", "scaned":
			// keep polling
		case "need_verifycode":
			return domain.WeixinLoginWaitResult{
				Connected:       false,
				NeedsVerifyCode: true,
				Message:         "请输入手机微信显示的验证码。",
			}, nil
		case "expired":
			b.mu.Lock()
			delete(b.logins, sessionKey)
			b.mu.Unlock()
			return domain.WeixinLoginWaitResult{Connected: false, Message: "二维码已过期，请重新生成。"}, nil
		case "verify_code_blocked":
			b.mu.Lock()
			delete(b.logins, sessionKey)
			b.mu.Unlock()
			return domain.WeixinLoginWaitResult{Connected: false, Message: "多次输入错误，连接流程已停止。"}, nil
		case "binded_redirect":
			b.mu.Lock()
			delete(b.logins, sessionKey)
			b.mu.Unlock()
			return domain.WeixinLoginWaitResult{
				Connected:        true,
				AlreadyConnected: true,
				Message:          "已连接过此应用，无需重复连接。",
			}, nil
		case "scaned_but_redirect":
			if status.RedirectHost != "" {
				baseURL = "https://" + status.RedirectHost
				b.mu.Lock()
				if l := b.logins[sessionKey]; l != nil {
					l.BaseURL = baseURL
				}
				b.mu.Unlock()
			}
		case "confirmed":
			accountID := ilink.NormalizeAccountID(status.ILinkBotID)
			token := strings.TrimSpace(status.BotToken)
			if accountID == "" || token == "" {
				b.mu.Lock()
				delete(b.logins, sessionKey)
				b.mu.Unlock()
				return domain.WeixinLoginWaitResult{Connected: false, Message: "登录失败：服务器未返回完整账号信息。"}, nil
			}
			base := status.BaseURL
			if base == "" {
				base = ilink.DefaultBaseURL
			}
			now := time.Now().UTC()
			acc := domain.WeixinAccount{
				AccountID: accountID,
				Token:     token,
				BaseURL:   base,
				UserID:    status.ILinkUserID,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := b.store.WeixinAccounts().Upsert(ctx, acc); err != nil {
				return domain.WeixinLoginWaitResult{}, err
			}
			b.mu.Lock()
			delete(b.logins, sessionKey)
			b.mu.Unlock()
			// Login implies the channel should be active: ensure agent + 「微信」 project, then start.
			if msg := b.activateAfterLogin(ctx); msg != "" {
				return domain.WeixinLoginWaitResult{
					Connected: true,
					AccountID: accountID,
					UserID:    status.ILinkUserID,
					Message:   msg,
				}, nil
			}
			_ = b.SyncFromConfig(ctx)
			return domain.WeixinLoginWaitResult{
				Connected: true,
				AccountID: accountID,
				UserID:    status.ILinkUserID,
				Message:   "已连接到微信。请在微信里给机器人发一条消息，会话会出现在「微信」项目下。",
			}, nil
		}
		time.Sleep(time.Second)
	}
	return domain.WeixinLoginWaitResult{Connected: false, Message: "等待扫码超时，请重试。"}, nil
}

func (b *WeixinBridge) Logout(ctx context.Context, accountID string) error {
	if accountID == "" {
		accounts, err := b.store.WeixinAccounts().List(ctx)
		if err != nil {
			return err
		}
		for _, a := range accounts {
			_ = b.store.WeixinBindings().DeleteByAccount(ctx, a.AccountID)
			_ = b.store.WeixinAccounts().Delete(ctx, a.AccountID)
		}
	} else {
		_ = b.store.WeixinBindings().DeleteByAccount(ctx, accountID)
		if err := b.store.WeixinAccounts().Delete(ctx, accountID); err != nil {
			return err
		}
	}
	wasRunning := b.IsRunning()
	b.Stop()
	if wasRunning {
		cfg, _ := b.loadChannelCfg(ctx)
		if cfg.Enabled {
			_ = b.Start(ctx)
		}
	}
	return nil
}

func (b *WeixinBridge) Status(ctx context.Context) (domain.WeixinStatus, error) {
	cfg, err := b.loadChannelCfg(ctx)
	if err != nil {
		return domain.WeixinStatus{}, err
	}
	accounts, err := b.store.WeixinAccounts().List(ctx)
	if err != nil {
		return domain.WeixinStatus{}, err
	}
	// Strip tokens from API response
	safe := make([]domain.WeixinAccount, 0, len(accounts))
	for _, a := range accounts {
		a.Token = ""
		safe = append(safe, a)
	}
	n, _ := b.store.WeixinBindings().Count(ctx)
	return domain.WeixinStatus{
		Enabled:          cfg.Enabled,
		Running:          b.IsRunning(),
		DefaultProjectID: cfg.DefaultProjectID,
		DefaultAgentID:   cfg.DefaultAgentID,
		DefaultModelID:   cfg.DefaultModelID,
		AutoApprove:      cfg.AutoApprove,
		Accounts:         safe,
		BindingCount:     n,
	}, nil
}

func (b *WeixinBridge) ListBindings(ctx context.Context) ([]domain.WeixinBinding, error) {
	return b.store.WeixinBindings().List(ctx)
}
