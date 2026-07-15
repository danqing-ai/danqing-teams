package bootstrap

import (
	"context"
	"os"
	"path/filepath"

	"danqing-teams/core/adapter/config"
	"danqing-teams/core/adapter/llm"
	"danqing-teams/core/service"
	"danqing-teams/core/domain"
	"danqing-teams/core/port"
	dqruntime "danqing-teams/core/runtime"
	"danqing-teams/core/runtime/prompt"
	"danqing-teams/core/runtime/tool/builtin"
	sqlitestore "danqing-teams/core/store/sqlite"
	"danqing-teams/core/store/turnlog"
)

type Config struct {
	ConfigPath string
	AutoApprove bool
	DataDir string
	LLM port.LLMProvider
	CompactionEnabled bool
	CompactionTurnInterval int
	CompactionSubInterval int
	CompactionMaxTokens int
	CompactionCutTokens int
}

type Core struct {
	Store         port.Repository
	Engine        port.Engine
	Config        *domain.ConfigFile
	Loader        *config.Loader
	Sessions      *service.SessionManager
	Projects      *service.ProjectManager
	LLMConfig     *service.LLMConfigManager
	ConfigManager *service.ConfigManager
	SearchConfig  *service.SearchConfigManager
	Agents        *service.AgentManager
	Skills        *service.SkillManager
	TurnLogs      *service.TurnLogManager
	MCPServers    *service.MCPManager
}

func New(cfg Config) *Core {
	loader := config.NewLoader(cfg.ConfigPath)
	appCfg, err := loader.Load(context.Background())
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	if cfg.DataDir != "" {
		appCfg.Data.Dir = cfg.DataDir
	}
	if cfg.AutoApprove {
		appCfg.Runtime.AutoApprove = true
	}
	if cfg.CompactionEnabled {
		appCfg.Runtime.Compaction.Enabled = true
	}
	if cfg.CompactionTurnInterval > 0 {
		appCfg.Runtime.Compaction.TurnInterval = cfg.CompactionTurnInterval
	}
	if cfg.CompactionSubInterval > 0 {
		appCfg.Runtime.Compaction.SubInterval = cfg.CompactionSubInterval
	}
	if cfg.CompactionMaxTokens > 0 {
		appCfg.Runtime.Compaction.MaxTokens = cfg.CompactionMaxTokens
	}
	if cfg.CompactionCutTokens > 0 {
		appCfg.Runtime.Compaction.CutTokens = cfg.CompactionCutTokens
	}

	if appCfg.Data.Dir == "" {
		appCfg.Data.Dir = "./data"
	}
	if appCfg.Data.Database == "" {
		appCfg.Data.Database = appCfg.Data.Dir + "/teams.db"
	}
	configDir := filepath.Dir(loader.Path())
	if !filepath.IsAbs(appCfg.Data.Dir) {
		appCfg.Data.Dir = filepath.Join(configDir, appCfg.Data.Dir)
	}
	if !filepath.IsAbs(appCfg.Data.Database) {
		appCfg.Data.Database = filepath.Join(configDir, appCfg.Data.Database)
	}
	if appCfg.Instance.ID == "" {
		appCfg.Instance.ID = os.Getenv("TEAMS_INSTANCE_ID")
		if appCfg.Instance.ID == "" {
			appCfg.Instance.ID, _ = os.Hostname()
		}
	}

	st, err := sqlitestore.New(appCfg.Data.Database)
	if err != nil {
		panic("failed to open database: " + err.Error())
	}

	pm := service.NewProjectManager(st, appCfg.Data.Dir)
	ensureDefaultProject(pm)

	turnLog := turnlog.NewTurnLogStore(pm.ProjectDir)
	agents := service.NewAgentManager(st.Agents())
	agents.SetTemplateLoader(prompt.LoadTemplateByID)
	skills := service.NewSkillManager(st.Skills(), st.SkillFiles())
	skills.SetTemplateLoader(prompt.LoadSkillTemplateByID)
	skills.SetFileTemplateLoader(prompt.LoadBuiltinSkillFiles)
	knowledge := buildKnowledge(st)
	turnManager := service.NewTurnManager(st.Turns())
	turnLogManager := service.NewTurnLogManager(turnLog)
	approvalManager := service.NewApprovalManager(st.Approvals())
	mcpManager := service.NewMCPManager(st.MCPServers())

	llmConfigRepo := st.LLMConfig()
	searchConfig := service.NewSearchConfigManager(loader)
	configManager := service.NewConfigManager(loader)

	// Create model config registry for generation params and context window lookups.
	modelCfg := service.NewModelConfigRegistry()
	modelCfg.LoadFromConfig(context.Background(), loader)

	llmConfig := service.NewLLMConfigManager(llmConfigRepo, modelCfg)

	client := llm.NewDefaultLLMProvider(llmConfig, modelCfg)

	provider := cfg.LLM
	if provider == nil {
		var cfgs []domain.LLMProviderConfig
		cfgs, _ = llmConfig.GetAll(context.Background())
		if len(cfgs) == 0 {
			provider = llm.NewMock()
		} else {
			provider = client
		}
	}

	ensureBuiltinAgents(agents)
	ensureBuiltinSkills(skills)

	stream := dqruntime.NewStreamEventManager(st.StreamEvents())
	checkpointStore := turnlog.NewCheckpointStore(pm.ProjectDir)

	sessions := service.NewSessionManager(st, nil, provider)
	eng := dqruntime.NewEngine(sessions, turnManager, pm, approvalManager, turnLogManager, agents, skills, knowledge, provider, stream, checkpointStore, loader, appCfg.Data.Dir)
	sessions.SetEngine(eng)

	eng.RegisterTool(&builtin.ExecShell{})
	eng.RegisterTool(&builtin.ReadFile{})
	eng.RegisterTool(&builtin.Edit{})
	eng.RegisterTool(&builtin.Write{})
	eng.RegisterTool(&builtin.ApplyPatch{})
	eng.RegisterTool(&builtin.Grep{})
	eng.RegisterTool(&builtin.Glob{})
	eng.RegisterTool(&builtin.TodoWrite{})
	eng.RegisterTool(&builtin.WebFetch{})
	eng.RegisterTool(&builtin.WebSearch{ConfigFunc: func(ctx context.Context) (domain.SearchConfig, error) {
		return searchConfig.Get(ctx)
	}})
	eng.RegisterTool(&builtin.AskUser{})
	eng.RegisterTool(&builtin.Sleep{})
	eng.RegisterTool(&builtin.ReadSkill{Skills: skills})
	eng.RecoverRunning(context.Background())

	return &Core{
		Store:         st,
		Engine:        eng,
		Config:        appCfg,
		Loader:        loader,
		Sessions:      sessions,
		Projects:      pm,
		LLMConfig:     llmConfig,
		ConfigManager: configManager,
		SearchConfig:  searchConfig,
		Agents:        agents,
		Skills:        skills,
		TurnLogs:      turnLogManager,
		MCPServers:    mcpManager,
	}
}

func ensureDefaultProject(pm *service.ProjectManager) {
	ctx := context.Background()
	projects, err := pm.List(ctx)
	if err != nil || len(projects) > 0 {
		return
	}
	pm.Create(ctx, domain.CreateProjectRequest{Name: "默认项目"})
}

func ensureBuiltinAgents(agents *service.AgentManager) {
	ctx := context.Background()
	templates, err := prompt.LoadTemplates()
	if err != nil {
		return
	}
	for _, tmpl := range templates {
		if _, err := agents.Get(ctx, tmpl.Agent.ID); err == nil {
			continue
		}
		agents.Upsert(ctx, tmpl.Agent)
	}
}

func ensureBuiltinSkills(skills *service.SkillManager) {
	ctx := context.Background()
	templates, err := prompt.LoadSkillTemplates()
	if err != nil {
		return
	}
	for _, tmpl := range templates {
		skill := tmpl.Skill
		skill.Builtin = true
		if existing, err := skills.Get(ctx, skill.ID); err == nil && existing != nil {
			// Update builtin flag on existing record if needed
			if !existing.Builtin {
				skills.Upsert(ctx, skill)
			}
		} else {
			skills.Upsert(ctx, skill)
		}
		// Import resource files (scripts, references, assets) into DB
		files, _ := prompt.LoadBuiltinSkillFiles(skill.ID)
		for _, f := range files {
			_ = skills.UpsertFile(ctx, f)
		}
	}
}
func buildKnowledge(st *sqlitestore.Store) *builtin.Knowledge {
	kb := builtin.NewKnowledge()
	for _, doc := range st.KnowledgeDocs() {
		kb.Add(builtin.Doc{KBID: doc.KBID, Title: doc.Title, Content: doc.Content})
	}
	return kb
}
