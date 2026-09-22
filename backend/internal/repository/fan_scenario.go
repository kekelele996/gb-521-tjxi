package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/model"
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

func (r *FanScenarioRepository) Create(ctx context.Context, scenario *model.FanScenario, action string, audit AuditRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(scenario).Error; err != nil {
			return fmt.Errorf("create fan scenario: %w", err)
		}
		if err := recordVersion(tx, scenario, action, "", audit); err != nil {
			return err
		}
		after, _ := json.Marshal(scenario)
		audit.EntityID = scenario.ID
		audit.AfterState = string(after)
		return writeAudit(tx, audit)
	})
}

func (r *FanScenarioRepository) Transition(ctx context.Context, id, actorID uint, expectedVersion uint, from, to, action, reason string, audit AuditRecord) (*model.FanScenario, error) {
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
		if err := recordVersion(tx, &updated, action, reason, audit); err != nil {
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

func (r *FanScenarioRepository) ListVersions(ctx context.Context, scenarioID uint) ([]model.FanScenarioVersion, error) {
	var items []model.FanScenarioVersion
	if err := r.db.WithContext(ctx).
		Where("scenario_id = ?", scenarioID).
		Order("version DESC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list fan scenario versions: %w", err)
	}
	return items, nil
}

func (r *FanScenarioRepository) FindVersion(ctx context.Context, scenarioID uint, version uint) (*model.FanScenarioVersion, error) {
	var item model.FanScenarioVersion
	if err := r.db.WithContext(ctx).
		Where("scenario_id = ? AND version = ?", scenarioID, version).
		First(&item).Error; err != nil {
		return nil, fmt.Errorf("find fan scenario version: %w", err)
	}
	return &item, nil
}

// recordVersion 在同一事务内追加当前参数的不可变快照；版本号唯一约束
// 使任何重复写入（包括用旧版本覆盖当前待审版本）直接失败并回滚事务。
func recordVersion(tx *gorm.DB, scenario *model.FanScenario, action, reason string, audit AuditRecord) error {
	snapshot := model.FanScenarioVersion{
		ScenarioID:      scenario.ID,
		Version:         scenario.Version,
		ScenarioStatus:  scenario.ScenarioStatus,
		Name:            scenario.Name,
		Description:     scenario.Description,
		FanCurveJSON:    scenario.FanCurveJSON,
		OperatingMode:   scenario.OperatingMode,
		SolverTolerance: scenario.SolverTolerance,
		MaxIterations:   scenario.MaxIterations,
		Action:          action,
		ActorID:         audit.ActorID,
		ActorEmail:      audit.ActorEmail,
		Reason:          reason,
		CreatedAt:       time.Now().UTC(),
	}
	if err := tx.Create(&snapshot).Error; err != nil {
		return fmt.Errorf("record fan scenario version: %w", err)
	}
	return nil
}
