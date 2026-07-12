package sqlite

import (
	"context"
	"os"
	"path/filepath"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var _ port.Repository = (*Store)(nil)

type Store struct {
	db *gorm.DB
}

func New(dbPath string) (*Store, error) {
	if dbPath == "" {
		dbPath = "./data/teams.db"
	}
	if abs, err := filepath.Abs(dbPath); err == nil {
		dbPath = abs
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) DB() *gorm.DB { return s.db }

func (s *Store) migrate() error {
	if err := s.db.AutoMigrate(
		&agentModel{},
		&skillModel{},
		&skillFileModel{},
		&sessionModel{},
		&projectModel{},
		&llmConfigModel{},
		&approvalModel{},
		&knowledgeDocModel{},
		&streamEventModel{},
		&turnModel{},
		&mcpServerModel{},
	); err != nil {
		return err
	}
	migrator := s.db.Migrator()
	if migrator.HasColumn(&llmConfigModel{}, "default_model") {
		if err := migrator.DropColumn(&llmConfigModel{}, "default_model"); err != nil {
			return err
		}
	}
	if migrator.HasColumn(&llmConfigModel{}, "is_active") {
		if err := migrator.DropColumn(&llmConfigModel{}, "is_active"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Agents() port.AgentRepo       { return &agentRepo{s} }
func (s *Store) Skills() port.SkillRepo       { return &skillRepo{s} }
func (s *Store) SkillFiles() port.SkillFileRepo { return &skillFileRepo{s} }
func (s *Store) Sessions() port.SessionRepo   { return &sessionRepo{s} }
func (s *Store) Projects() port.ProjectRepo   { return &projectRepo{s} }
func (s *Store) LLMConfig() port.LLMConfigRepo { return &llmConfigRepo{s} }
func (s *Store) Approvals() port.ApprovalRepo { return &approvalRepo{s} }
func (s *Store) StreamEvents() port.StreamEventRepo { return &streamEventRepo{s} }
func (s *Store) Turns() port.TurnRepo             { return &turnRepo{s} }
func (s *Store) MCPServers() port.MCPServerRepo  { return &mcpServerRepo{s} }

func (s *Store) KnowledgeDocs() []KnowledgeDoc {
	var rows []knowledgeDocModel
	if err := s.db.Find(&rows).Error; err != nil {
		return nil
	}
	out := make([]KnowledgeDoc, len(rows))
	for i, r := range rows {
		out[i] = KnowledgeDoc{KBID: r.KBID, Title: r.Title, Content: r.Content}
	}
	return out
}

type KnowledgeDoc struct {
	KBID    string
	Title   string
	Content string
}

type knowledgeDocModel struct {
	ID      uint   `gorm:"primaryKey;autoIncrement"`
	KBID    string `gorm:"column:kb_id"`
	Title   string
	Content string
}

func (knowledgeDocModel) TableName() string { return "knowledge_docs" }

// ---- AgentRepo ----

type agentRepo struct{ s *Store }

func (r *agentRepo) List(ctx context.Context) ([]domain.Agent, error) {
	var rows []agentModel
	if err := r.s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Agent, len(rows))
	for i, row := range rows {
		out[i] = agentToDomain(row)
	}
	return out, nil
}

func (r *agentRepo) Get(ctx context.Context, id string) (domain.Agent, error) {
	var row agentModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Agent{}, err
	}
	return agentToDomain(row), nil
}

func (r *agentRepo) Upsert(ctx context.Context, a domain.Agent) error {
	m := agentFromDomain(a)
	var existing agentModel
	err := r.s.db.WithContext(ctx).First(&existing, "id = ?", a.ID).Error
	if err != nil {
		return r.s.db.WithContext(ctx).Create(&m).Error
	}
	return r.s.db.WithContext(ctx).Model(&existing).Updates(&m).Error
}

func (r *agentRepo) Delete(ctx context.Context, id string) error {
	return r.s.db.WithContext(ctx).Delete(&agentModel{}, "id = ?", id).Error
}

// ---- SkillRepo ----

type skillRepo struct{ s *Store }

func (r *skillRepo) List(ctx context.Context) ([]domain.Skill, error) {
	var rows []skillModel
	if err := r.s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Skill, len(rows))
	for i, row := range rows {
		out[i] = skillToDomain(row)
	}
	return out, nil
}

func (r *skillRepo) Get(ctx context.Context, id string) (domain.Skill, error) {
	var row skillModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Skill{}, err
	}
	return skillToDomain(row), nil
}

func (r *skillRepo) Upsert(ctx context.Context, sk domain.Skill) error {
	m := skillFromDomain(sk)
	var existing skillModel
	err := r.s.db.WithContext(ctx).First(&existing, "id = ?", sk.ID).Error
	if err != nil {
		return r.s.db.WithContext(ctx).Create(&m).Error
	}
	return r.s.db.WithContext(ctx).Model(&existing).Updates(&m).Error
}

func (r *skillRepo) Delete(ctx context.Context, id string) error {
	return r.s.db.WithContext(ctx).Delete(&skillModel{}, "id = ?", id).Error
}

// ---- SkillFileRepo ----

type skillFileRepo struct{ s *Store }

func (r *skillFileRepo) ListBySkill(ctx context.Context, skillID string) ([]domain.SkillFile, error) {
	var rows []skillFileModel
	if err := r.s.db.WithContext(ctx).Where("skill_id = ?", skillID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SkillFile, len(rows))
	for i, row := range rows {
		out[i] = skillFileToDomain(row)
	}
	return out, nil
}

func (r *skillFileRepo) Get(ctx context.Context, skillID, path string) (domain.SkillFile, error) {
	id := skillID + ":" + path
	var row skillFileModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.SkillFile{}, err
	}
	return skillFileToDomain(row), nil
}

func (r *skillFileRepo) Upsert(ctx context.Context, f domain.SkillFile) error {
	m := skillFileFromDomain(f)
	var existing skillFileModel
	err := r.s.db.WithContext(ctx).First(&existing, "id = ?", m.ID).Error
	if err != nil {
		return r.s.db.WithContext(ctx).Create(&m).Error
	}
	return r.s.db.WithContext(ctx).Model(&existing).Updates(&m).Error
}

func (r *skillFileRepo) DeleteBySkill(ctx context.Context, skillID string) error {
	return r.s.db.WithContext(ctx).Delete(&skillFileModel{}, "skill_id = ?", skillID).Error
}

// ---- SessionRepo ----

type sessionRepo struct{ s *Store }

func (r *sessionRepo) Create(ctx context.Context, s domain.Session) error {
	m := sessionFromDomain(s)
	return r.s.db.WithContext(ctx).Create(&m).Error
}

func (r *sessionRepo) Update(ctx context.Context, s domain.Session) error {
	return r.s.db.WithContext(ctx).Model(&sessionModel{}).Where("id = ?", s.ID).Updates(sessionFromDomain(s)).Error
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	return r.s.db.WithContext(ctx).Delete(&sessionModel{}, "id = ?", id).Error
}

func (r *sessionRepo) Get(ctx context.Context, id string) (domain.Session, error) {
	var row sessionModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Session{}, err
	}
	return sessionToDomain(row), nil
}

func (r *sessionRepo) List(ctx context.Context) ([]domain.Session, error) {
	var rows []sessionModel
	if err := r.s.db.WithContext(ctx).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Session, len(rows))
	for i, row := range rows {
		out[i] = sessionToDomain(row)
	}
	return out, nil
}

func (r *sessionRepo) ListByProject(ctx context.Context, projectID string) ([]domain.Session, error) {
	var rows []sessionModel
	if err := r.s.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Session, len(rows))
	for i, row := range rows {
		out[i] = sessionToDomain(row)
	}
	return out, nil
}

// ---- ProjectRepo ----

type projectRepo struct{ s *Store }

func (r *projectRepo) Create(ctx context.Context, p domain.Project) error {
	m := projectFromDomain(p)
	return r.s.db.WithContext(ctx).Create(&m).Error
}

func (r *projectRepo) Update(ctx context.Context, p domain.Project) error {
	return r.s.db.WithContext(ctx).Model(&projectModel{}).Where("id = ?", p.ID).Updates(projectFromDomain(p)).Error
}

func (r *projectRepo) Get(ctx context.Context, id string) (domain.Project, error) {
	var row projectModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Project{}, err
	}
	return projectToDomain(row), nil
}

func (r *projectRepo) List(ctx context.Context) ([]domain.Project, error) {
	var rows []projectModel
	if err := r.s.db.WithContext(ctx).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Project, len(rows))
	for i, row := range rows {
		out[i] = projectToDomain(row)
	}
	return out, nil
}

func (r *projectRepo) Delete(ctx context.Context, id string) error {
	return r.s.db.WithContext(ctx).Delete(&projectModel{}, "id = ?", id).Error
}

// ---- LLMConfigRepo ----

type llmConfigRepo struct{ s *Store }

func (r *llmConfigRepo) GetAll(ctx context.Context) ([]domain.LLMProviderConfig, error) {
	var rows []llmConfigModel
	if err := r.s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.LLMProviderConfig, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *llmConfigRepo) GetByID(ctx context.Context, id string) (domain.LLMProviderConfig, error) {
	var row llmConfigModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.LLMProviderConfig{}, err
	}
	return row.toDomain(), nil
}

func (r *llmConfigRepo) Upsert(ctx context.Context, cfg domain.LLMProviderConfig) error {
	row := llmConfigModelFromDomain(cfg)
	var existing llmConfigModel
	if err := r.s.db.WithContext(ctx).First(&existing, "id = ?", cfg.ID).Error; err != nil {
		return r.s.db.WithContext(ctx).Create(&row).Error
	}
	row.CreatedAt = existing.CreatedAt
	return r.s.db.WithContext(ctx).Model(&existing).Updates(&row).Error
}

func (r *llmConfigRepo) Delete(ctx context.Context, id string) error {
	return r.s.db.WithContext(ctx).Delete(&llmConfigModel{}, "id = ?", id).Error
}

// ---- ApprovalRepo ----

type approvalRepo struct{ s *Store }

func (r *approvalRepo) Create(ctx context.Context, a domain.Approval) error {
	m := approvalFromDomain(a)
	return r.s.db.WithContext(ctx).Create(&m).Error
}

func (r *approvalRepo) Get(ctx context.Context, id string) (domain.Approval, error) {
	var row approvalModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Approval{}, err
	}
	return approvalToDomain(row), nil
}

func (r *approvalRepo) Update(ctx context.Context, a domain.Approval) error {
	return r.s.db.WithContext(ctx).Model(&approvalModel{}).Where("id = ?", a.ID).Updates(approvalFromDomain(a)).Error
}

func (r *approvalRepo) ListByStatus(ctx context.Context, status string) ([]domain.Approval, error) {
	var rows []approvalModel
	if err := r.s.db.WithContext(ctx).Where("status = ?", status).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Approval, len(rows))
	for i, row := range rows {
		out[i] = approvalToDomain(row)
	}
	return out, nil
}

// ---- StreamEventRepo ----

type streamEventRepo struct{ s *Store }

func (r *streamEventRepo) Save(ctx context.Context, ev domain.StreamEvent) error {
	m := streamEventModel{
		SessionID: ev.SessionID, TurnID: ev.TurnID, Seq: ev.Seq,
		Type: ev.Type, Payload: string(ev.Payload), CreatedAt: ev.CreatedAt,
	}
	return r.s.db.WithContext(ctx).Create(&m).Error
}

func (r *streamEventRepo) ListBySession(ctx context.Context, sessionID string, since int64) ([]domain.StreamEvent, error) {
	var rows []streamEventModel
	q := r.s.db.WithContext(ctx).Where("session_id = ?", sessionID)
	if since > 0 {
		q = q.Where("seq > ?", since)
	}
	if err := q.Order("seq asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.StreamEvent, len(rows))
	for i, row := range rows {
		out[i] = streamEventToDomain(row)
	}
	return out, nil
}

func (r *streamEventRepo) MaxSeq() int64 {
	var max int64
	r.s.db.Model(&streamEventModel{}).Select("COALESCE(MAX(seq), 0)").Scan(&max)
	return max
}

// ---- TurnRepo ----

type turnRepo struct{ s *Store }

func (r *turnRepo) Create(ctx context.Context, t domain.TurnLog) error {
	m := turnFromDomain(t)
	var existing turnModel
	if err := r.s.db.WithContext(ctx).First(&existing, "id = ?", t.ID).Error; err != nil {
		return r.s.db.WithContext(ctx).Create(&m).Error
	}
	return nil
}

func (r *turnRepo) UpdateStatus(ctx context.Context, id string, status domain.TurnStatus) error {
	return r.s.db.WithContext(ctx).Model(&turnModel{}).Where("id = ?", id).Update("status", string(status)).Error
}

func (r *turnRepo) Get(ctx context.Context, id string) (domain.TurnLog, error) {
	var row turnModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.TurnLog{}, err
	}
	return turnToDomain(row), nil
}

func (r *turnRepo) ListBySession(ctx context.Context, sessionID string) ([]domain.TurnLog, error) {
	var rows []turnModel
	if err := r.s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TurnLog, len(rows))
	for i, row := range rows {
		out[i] = turnToDomain(row)
	}
	return out, nil
}

func (r *turnRepo) ListByStatus(ctx context.Context, status domain.TurnStatus) ([]domain.TurnLog, error) {
	var rows []turnModel
	if err := r.s.db.WithContext(ctx).Where("status = ?", string(status)).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TurnLog, len(rows))
	for i, row := range rows {
		out[i] = turnToDomain(row)
	}
	return out, nil
}

// ---- MCPServerRepo ----

type mcpServerRepo struct{ s *Store }

func (r *mcpServerRepo) List(ctx context.Context) ([]domain.MCPServer, error) {
	var rows []mcpServerModel
	if err := r.s.db.WithContext(ctx).Order("rowid asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.MCPServer, len(rows))
	for i, row := range rows {
		out[i] = mcpServerToDomain(row)
	}
	return out, nil
}

func (r *mcpServerRepo) Get(ctx context.Context, id string) (domain.MCPServer, error) {
	var row mcpServerModel
	if err := r.s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.MCPServer{}, err
	}
	return mcpServerToDomain(row), nil
}

func (r *mcpServerRepo) Upsert(ctx context.Context, s domain.MCPServer) error {
	m := mcpServerFromDomain(s)
	var existing mcpServerModel
	if err := r.s.db.WithContext(ctx).First(&existing, "id = ?", s.ID).Error; err != nil {
		return r.s.db.WithContext(ctx).Create(&m).Error
	}
	return r.s.db.WithContext(ctx).Model(&existing).Updates(&m).Error
}

func (r *mcpServerRepo) Delete(ctx context.Context, id string) error {
	return r.s.db.WithContext(ctx).Delete(&mcpServerModel{}, "id = ?", id).Error
}
