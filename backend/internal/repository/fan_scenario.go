package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/model"
)

var (
	ErrScenarioNotDraft = errors.New("scenario is not in draft status")
	ErrVersionNotFound  = errors.New("scenario version not found")
)

type FanScenarioRepository struct{ db *gorm.DB }

func NewFanScenarioRepository(db *gorm.DB) *FanScenarioRepository {
	return &FanScenarioRepository{db: db}
}

func (r *FanScenarioRepository) List(ctx context.Context, page, pageSize int, status, search string) ([]model.FanScenario, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.FanScenario{})
	if status != "" {
		query = query.Where("scenario_status = ?", status)
	}
	if search != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(search)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count fan scenarios: %w", err)
	}
	var items []model.FanScenario
	if err := query.Order("updated_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list fan scenarios: %w", err)
	}
	return items, total, nil
}

func (r *FanScenarioRepository) Find(ctx context.Context, id uint) (*model.FanScenario, error) {
	var scenario model.FanScenario
	if err := r.db.WithContext(ctx).First(&scenario, id).Error; err != nil {
		return nil, fmt.Errorf("find fan scenario: %w", err)
	}
	return &scenario, nil
}

func (r *FanScenarioRepository) FindVersion(ctx context.Context, scenarioID, version uint) (*model.FanScenarioVersion, error) {
	var snap model.FanScenarioVersion
	if err := r.db.WithContext(ctx).Where("scenario_id = ? AND version = ?", scenarioID, version).First(&snap).Error; err != nil {
		return nil, fmt.Errorf("find fan scenario version: %w", err)
	}
	return &snap, nil
}

func (r *FanScenarioRepository) ListVersions(ctx context.Context, scenarioID uint, page, pageSize int) ([]model.FanScenarioVersion, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.FanScenarioVersion{}).Where("scenario_id = ?", scenarioID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count fan scenario versions: %w", err)
	}
	var items []model.FanScenarioVersion
	if err := query.Order("version DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list fan scenario versions: %w", err)
	}
	return items, total, nil
}

func (r *FanScenarioRepository) Create(ctx context.Context, scenario *model.FanScenario, audit AuditRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(scenario).Error; err != nil {
			return fmt.Errorf("create fan scenario: %w", err)
		}
		snapshot := buildScenarioVersion(scenario, model.FanScenarioVersion{
			ScenarioID: scenario.ID, Version: scenario.Version,
			ChangeKind: "created", StatusAtChange: scenario.ScenarioStatus,
			ActorID: audit.ActorID, ActorEmail: audit.ActorEmail, CreatedAt: scenario.CreatedAt,
		})
		if err := tx.Create(&snapshot).Error; err != nil {
			return fmt.Errorf("create fan scenario version snapshot: %w", err)
		}
		after, _ := json.Marshal(scenario)
		audit.EntityID = scenario.ID
		audit.AfterState = string(after)
		return writeAudit(tx, audit)
	})
}

// DraftPatch 承载草稿修订后的参数，版本快照直接读取更新后的实体。
type DraftPatch struct {
	Name            string
	Description     string
	FanCurve        datatypes.JSON
	OperatingMode   string
	SolverTolerance float64
	MaxIterations   int
}

func (r *FanScenarioRepository) UpdateDraft(ctx context.Context, id, actorID uint, expectedVersion uint, patch DraftPatch, reason, actorEmail string, audit AuditRecord) (*model.FanScenario, error) {
	var updated model.FanScenario
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.FanScenario
		if err := tx.First(&before, id).Error; err != nil {
			return fmt.Errorf("load fan scenario before draft edit: %w", err)
		}
		if before.ScenarioStatus != "draft" {
			return ErrScenarioNotDraft
		}
		result := tx.Model(&model.FanScenario{}).
			Where("id = ? AND version = ? AND scenario_status = ?", id, expectedVersion, "draft").
			Updates(map[string]interface{}{
				"name":             patch.Name,
				"description":      patch.Description,
				"fan_curve_json":   patch.FanCurve,
				"operating_mode":   patch.OperatingMode,
				"solver_tolerance": patch.SolverTolerance,
				"max_iterations":   patch.MaxIterations,
				"reject_reason":    "",
				"version":          gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return fmt.Errorf("update draft fan scenario: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return err
		}
		if err := writeVersionSnapshot(tx, &updated, "edited", reason, actorID, actorEmail); err != nil {
			return err
		}
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(updated)
		audit.EntityID = id
		audit.BeforeState = string(beforeJSON)
		audit.AfterState = string(afterJSON)
		return writeAudit(tx, audit)
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (r *FanScenarioRepository) RestoreVersion(ctx context.Context, id, actorID, sourceVersion uint, actorEmail string, audit AuditRecord) (*model.FanScenario, error) {
	var updated model.FanScenario
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.FanScenario
		if err := tx.First(&current, id).Error; err != nil {
			return fmt.Errorf("load fan scenario before restore: %w", err)
		}
		if current.ScenarioStatus != "draft" {
			return ErrScenarioNotDraft
		}
		var source model.FanScenarioVersion
		if err := tx.Where("scenario_id = ? AND version = ?", id, sourceVersion).First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrVersionNotFound
			}
			return fmt.Errorf("load source fan scenario version: %w", err)
		}
		reason := fmt.Sprintf("从历史版本 v%d 恢复参数", sourceVersion)
		result := tx.Model(&model.FanScenario{}).
			Where("id = ? AND scenario_status = ?", id, "draft").
			Updates(map[string]interface{}{
				"fan_curve_json":   datatypes.JSON(source.FanCurveJSON),
				"operating_mode":   source.OperatingMode,
				"solver_tolerance": source.SolverTolerance,
				"max_iterations":   source.MaxIterations,
				"reject_reason":    "",
				"version":          gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return fmt.Errorf("restore fan scenario version: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return err
		}
		if err := writeVersionSnapshot(tx, &updated, "restored", reason, actorID, actorEmail); err != nil {
			return err
		}
		beforeJSON, _ := json.Marshal(current)
		afterJSON, _ := json.Marshal(updated)
		audit.EntityID = id
		audit.BeforeState = string(beforeJSON)
		audit.AfterState = string(afterJSON)
		return writeAudit(tx, audit)
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (r *FanScenarioRepository) Transition(ctx context.Context, id, actorID uint, expectedVersion uint, from, to, reason string, audit AuditRecord) (*model.FanScenario, error) {
	var updated model.FanScenario
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.FanScenario
		if err := tx.First(&before, id).Error; err != nil {
			return fmt.Errorf("load fan scenario before transition: %w", err)
		}
		values := map[string]interface{}{
			"scenario_status": to,
			"reject_reason":   "",
			"version":         gorm.Expr("version + 1"),
		}
		if to == "approved" {
			values["approved_by"] = actorID
		}
		if to == "draft" {
			values["approved_by"] = nil
			values["reject_reason"] = reason
		}
		result := tx.Model(&model.FanScenario{}).
			Where("id = ? AND version = ? AND scenario_status = ?", id, expectedVersion, from).
			Updates(values)
		if result.Error != nil {
			return fmt.Errorf("transition fan scenario: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrVersionConflict
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return err
		}
		if err := writeVersionSnapshot(tx, &updated, transitionKind(to), reason, actorID, audit.ActorEmail); err != nil {
			return err
		}
		beforeJSON, _ := json.Marshal(before)
		afterJSON, _ := json.Marshal(updated)
		audit.EntityID = id
		audit.BeforeState = string(beforeJSON)
		audit.AfterState = string(afterJSON)
		return writeAudit(tx, audit)
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func buildScenarioVersion(scenario *model.FanScenario, snap model.FanScenarioVersion) model.FanScenarioVersion {
	snap.FanCurveJSON = datatypes.JSON(scenario.FanCurveJSON)
	snap.OperatingMode = scenario.OperatingMode
	snap.SolverTolerance = scenario.SolverTolerance
	snap.MaxIterations = scenario.MaxIterations
	return snap
}

func writeVersionSnapshot(tx *gorm.DB, scenario *model.FanScenario, kind, reason string, actorID uint, actorEmail string) error {
	createdAt := scenario.UpdatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	snap := buildScenarioVersion(scenario, model.FanScenarioVersion{
		ScenarioID: scenario.ID, Version: scenario.Version,
		ChangeKind: kind, StatusAtChange: scenario.ScenarioStatus, Reason: reason,
		ActorID: actorID, ActorEmail: actorEmail, CreatedAt: createdAt,
	})
	if err := tx.Create(&snap).Error; err != nil {
		return fmt.Errorf("create fan scenario version snapshot: %w", err)
	}
	return nil
}

func transitionKind(to string) string {
	switch to {
	case "pending_review":
		return "submitted"
	case "approved":
		return "approved"
	case "archived":
		return "archived"
	default:
		return "rejected"
	}
}
