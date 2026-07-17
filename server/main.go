package main

import (
	"os"

	"danqing-teams/core/bootstrap"
	"danqing-teams/core/runtime/sandbox"
	"danqing-teams/core/service"
	apiv1 "danqing-teams/server/api/v1"
)

func main() {
	if sandbox.MaybeReexec() {
		return
	}
	core := bootstrap.New(bootstrap.Config{ConfigPath: os.Getenv("TEAMS_CONFIG")})
	h := &apiv1.Handler{
		Sessions:     core.Sessions,
		Projects:     core.Projects,
		LLMConfig:    core.LLMConfig,
		Config:       core.ConfigManager,
		SearchConfig: core.SearchConfig,
		Agents:       core.Agents,
		Skills:       core.Skills,
		SkillHandler: &apiv1.SkillHandler{
			Skills:   core.Skills,
			Importer: service.NewSkillImporter(),
		},
		TurnLogs:   core.TurnLogs,
		MCPServers: core.MCPServers,
		Sandbox:    core.Sandbox,
		Store:      core.Store,
	}
	apiv1.NewRouter(h, apiv1.RouterConfig{}).Run(core.Config.Server.ListenAddr)
}
