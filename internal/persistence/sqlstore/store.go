package sqlstore

import (
	"context"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&teamRow{}, &teamControllerRow{}, &workerRow{}, &humanRow{},
		&taskRow{}, &dispatchRow{}, &workerRunRow{}, &executionPlanRow{},
		&reportRow{}, &timelineEventRow{}, &messageRow{}, &approvalRow{},
		&todoRow{}, &artifactRow{}, &knowledgeDocRow{}, &orchestrationJobRow{},
		&agentRow{}, &teamAgentRow{},
	); err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.MigrateLegacyWorkersToAgents(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) dbWithCtx(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx)
}
