package main

import (
	"os"

	"danqing-teams/core/bootstrap"
	apiv1 "danqing-teams/server/api/v1"
)

func main() {
	core := bootstrap.New(bootstrap.Config{ConfigPath: os.Getenv("TEAMS_CONFIG")})
	h := &apiv1.Handler{
		Sessions:     core.Sessions,
		Projects:     core.Projects,
		LLMConfig:    core.LLMConfig,
		Config:       core.ConfigManager,
		SearchConfig: core.SearchConfig,
		Agents:       core.Agents,
		Skills:       core.Skills,
		TurnLogs:     core.TurnLogs,
		Store:        core.Store,
	}
	apiv1.NewRouter(h, apiv1.RouterConfig{}).Run(core.Config.Server.ListenAddr)
}
