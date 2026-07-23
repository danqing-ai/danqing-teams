package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"danqing-teams/core/adapter/config"
	"danqing-teams/core/adapter/weixin/ilink"
	"danqing-teams/core/domain"
	"danqing-teams/core/service"
	sqlitestore "danqing-teams/core/store/sqlite"
)

func TestWeixinBindingRepoPeerSession(t *testing.T) {
	dir := t.TempDir()
	st, err := sqlitestore.New(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	acc := domain.WeixinAccount{AccountID: "bot1", Token: "tok", BaseURL: ilink.DefaultBaseURL, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := st.WeixinAccounts().Upsert(ctx, acc); err != nil {
		t.Fatal(err)
	}
	b := domain.WeixinBinding{AccountID: "bot1", PeerUserID: "peer-a", SessionID: "session-xyz", ContextToken: "ct"}
	if err := st.WeixinBindings().Upsert(ctx, b); err != nil {
		t.Fatal(err)
	}
	got, err := st.WeixinBindings().GetByPeer(ctx, "bot1", "peer-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "session-xyz" || got.ContextToken != "ct" {
		t.Fatalf("%+v", got)
	}
	n, err := st.WeixinBindings().Count(ctx)
	if err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}
}

func TestEnsureWeixinProject(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "t.db")
	cfgPath := filepath.Join(dir, "config.yaml")
	st, err := sqlitestore.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	loader := config.NewLoader(cfgPath)
	cm := service.NewConfigManager(loader)
	pm := service.NewProjectManager(st, dir)
	bridge := service.NewWeixinBridge(st, service.NewSessionManager(st, nil, nil), pm, cm)
	id, err := bridge.EnsureWeixinProject(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty project id")
	}
	p, err := pm.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != domain.WeixinProjectName {
		t.Fatalf("name=%q", p.Name)
	}
	cfg, err := cm.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Channels.Weixin.DefaultProjectID != id {
		t.Fatalf("cfg project=%q want %q", cfg.Channels.Weixin.DefaultProjectID, id)
	}
}

func TestEnsureWeixinProjectIgnoresStaleDefaultID(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "t.db")
	cfgPath := filepath.Join(dir, "config.yaml")
	st, err := sqlitestore.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	loader := config.NewLoader(cfgPath)
	cm := service.NewConfigManager(loader)
	pm := service.NewProjectManager(st, dir)
	bridge := service.NewWeixinBridge(st, service.NewSessionManager(st, nil, nil), pm, cm)

	other, err := pm.Create(context.Background(), domain.CreateProjectRequest{Name: "DanQing-knowledge"})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := cm.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wx := cfg.Channels.Weixin
	wx.DefaultProjectID = other.ID
	sec := cfg.Channels
	sec.Weixin = wx
	if _, err := cm.Update(context.Background(), domain.UpdateConfigFileRequest{Channels: &sec}); err != nil {
		t.Fatal(err)
	}

	id, err := bridge.EnsureWeixinProject(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if id == other.ID {
		t.Fatalf("reused stale project %q", id)
	}
	p, err := pm.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != domain.WeixinProjectName {
		t.Fatalf("name=%q", p.Name)
	}
}

func TestWeixinLoginConfirmed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "get_bot_qrcode"):
			// Simulate real iLink: URL payload, not a data-image.
			_ = json.NewEncoder(w).Encode(map[string]string{
				"qrcode":             "qr1",
				"qrcode_img_content": "https://liteapp.weixin.qq.com/q/demo?qrcode=abc",
			})
		case strings.Contains(r.URL.Path, "get_qrcode_status"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":        "confirmed",
				"bot_token":     "token-1",
				"ilink_bot_id":  "bot@im.bot",
				"ilink_user_id": "user-1",
				"baseurl":       r.Host,
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	st, err := sqlitestore.New(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	loader := config.NewLoader(filepath.Join(dir, "config.yaml"))
	cm := service.NewConfigManager(loader)
	pm := service.NewProjectManager(st, dir)
	bridge := service.NewWeixinBridge(st, service.NewSessionManager(st, nil, nil), pm, cm)
	client := ilink.NewClient()
	client.HTTP = srv.Client()
	client.BaseURL = srv.URL
	bridge.SetClient(client)

	start, err := bridge.StartLogin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(start.QRCodeURL, "data:image/png;base64,") {
		t.Fatalf("expected generated png data url, got %q", start.QRCodeURL[:min(40, len(start.QRCodeURL))])
	}
	wait, err := bridge.WaitLogin(context.Background(), start.SessionKey, "", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if !wait.Connected || wait.AccountID == "" {
		t.Fatalf("%+v", wait)
	}
	acc, err := st.WeixinAccounts().Get(context.Background(), wait.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if acc.Token != "token-1" {
		t.Fatalf("token=%q", acc.Token)
	}
}
