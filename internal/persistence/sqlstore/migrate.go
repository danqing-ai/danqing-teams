package sqlstore

import (
	"database/sql"
	"fmt"
	"strings"
)

const schema = `
CREATE TABLE IF NOT EXISTS teams (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS team_controllers (
	team_id TEXT PRIMARY KEY REFERENCES teams(id) ON DELETE CASCADE,
	persona TEXT NOT NULL DEFAULT '',
	system_prompt TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS workers (
	id TEXT PRIMARY KEY,
	team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	persona TEXT NOT NULL DEFAULT '',
	skills_json TEXT NOT NULL DEFAULT '[]',
	tools_json TEXT NOT NULL DEFAULT '[]',
	kb_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_workers_team ON workers(team_id);

CREATE TABLE IF NOT EXISTS humans (
	id TEXT PRIMARY KEY,
	team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	display_name TEXT NOT NULL,
	email TEXT NOT NULL DEFAULT '',
	role TEXT NOT NULL DEFAULT 'observer'
);
CREATE INDEX IF NOT EXISTS idx_humans_team ON humans(team_id);

CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY,
	team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	content TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tasks_team ON tasks(team_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

CREATE TABLE IF NOT EXISTS dispatches (
	id TEXT PRIMARY KEY,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	worker_id TEXT NOT NULL,
	worker_name TEXT NOT NULL DEFAULT '',
	intent TEXT NOT NULL,
	context_summary TEXT NOT NULL DEFAULT '',
	round_num INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_dispatches_task ON dispatches(task_id);

CREATE TABLE IF NOT EXISTS worker_runs (
	id TEXT PRIMARY KEY,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	dispatch_id TEXT NOT NULL,
	worker_id TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_runs_task ON worker_runs(task_id);

CREATE TABLE IF NOT EXISTS execution_plans (
	run_id TEXT PRIMARY KEY REFERENCES worker_runs(id) ON DELETE CASCADE,
	skill_ids_json TEXT NOT NULL DEFAULT '[]',
	tool_ids_json TEXT NOT NULL DEFAULT '[]',
	rationale TEXT NOT NULL DEFAULT '',
	evaluated_risk TEXT NOT NULL DEFAULT 'low',
	high_risk_json TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS reports (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	worker_id TEXT NOT NULL,
	worker_name TEXT NOT NULL DEFAULT '',
	content_markdown TEXT NOT NULL,
	intent TEXT NOT NULL DEFAULT 'final',
	suggested_actions_json TEXT NOT NULL DEFAULT '[]',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reports_task ON reports(task_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_run ON reports(run_id);

CREATE TABLE IF NOT EXISTS timeline_events (
	id TEXT PRIMARY KEY,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	type TEXT NOT NULL,
	payload_json TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_timeline_task ON timeline_events(task_id);

CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	team_id TEXT NOT NULL,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_task ON messages(task_id);

CREATE TABLE IF NOT EXISTS approvals (
	id TEXT PRIMARY KEY,
	team_id TEXT NOT NULL,
	task_id TEXT NOT NULL,
	run_id TEXT NOT NULL,
	summary TEXT NOT NULL,
	high_risk_json TEXT NOT NULL DEFAULT '[]',
	status TEXT NOT NULL,
	comment TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_approvals_team ON approvals(team_id);
CREATE INDEX IF NOT EXISTS idx_approvals_run ON approvals(run_id);

CREATE TABLE IF NOT EXISTS todos (
	id TEXT PRIMARY KEY,
	team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	task_id TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL,
	done INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_todos_team ON todos(team_id);

CREATE TABLE IF NOT EXISTS artifacts (
	id TEXT PRIMARY KEY,
	team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	task_id TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL,
	kind TEXT NOT NULL,
	content TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_artifacts_team ON artifacts(team_id);

CREATE TABLE IF NOT EXISTS knowledge_docs (
	team_id TEXT NOT NULL,
	worker_id TEXT NOT NULL,
	id TEXT NOT NULL,
	title TEXT NOT NULL,
	size INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (team_id, worker_id, id)
);

CREATE TABLE IF NOT EXISTS orchestration_jobs (
	id TEXT PRIMARY KEY,
	team_id TEXT NOT NULL,
	task_id TEXT NOT NULL,
	kind TEXT NOT NULL,
	payload_json TEXT NOT NULL,
	dedup_key TEXT NOT NULL,
	status TEXT NOT NULL,
	lease_owner TEXT NOT NULL DEFAULT '',
	lease_until TEXT,
	last_error TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_jobs_status_created ON orchestration_jobs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_dedup ON orchestration_jobs(dedup_key);
CREATE INDEX IF NOT EXISTS idx_jobs_task ON orchestration_jobs(task_id);

CREATE TABLE IF NOT EXISTS agents (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	role TEXT NOT NULL,
	llm_url TEXT NOT NULL DEFAULT '',
	llm_api_key TEXT NOT NULL DEFAULT '',
	default_model TEXT NOT NULL DEFAULT '',
	all_models_json TEXT NOT NULL DEFAULT '[]',
	system_prompt TEXT NOT NULL DEFAULT '',
	min_function_calling_rounds INTEGER NOT NULL DEFAULT 1,
	skills_json TEXT NOT NULL DEFAULT '[]',
	tools_json TEXT NOT NULL DEFAULT '[]',
	kb_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agents_role ON agents(role);

CREATE TABLE IF NOT EXISTS team_agents (
	team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
	PRIMARY KEY (team_id, agent_id)
);
CREATE INDEX IF NOT EXISTS idx_team_agents_agent ON team_agents(agent_id);
`

func migrate(db *sql.DB) error {
	for _, stmt := range splitStatements(schema) {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w\nstmt: %s", err, stmt)
		}
	}
	return migrateColumns(db)
}

func migrateColumns(db *sql.DB) error {
	if !columnExists(db, "tasks", "close_reason") {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN close_reason TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("migrate columns: %w", err)
		}
	}
	return nil
}

func columnExists(db *sql.DB, table, column string) bool {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

func splitStatements(schema string) []string {
	var out []string
	for _, part := range strings.Split(schema, ";") {
		s := strings.TrimSpace(part)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
