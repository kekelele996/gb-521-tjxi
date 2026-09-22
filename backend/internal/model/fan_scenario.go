package model

import (
	"time"

	"gorm.io/datatypes"
)

type FanScenario struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"size:120;not null" json:"name"`
	Description     string         `gorm:"size:600;not null" json:"description"`
	FanCurveJSON    datatypes.JSON `gorm:"type:jsonb;not null" json:"fan_curve_json"`
	OperatingMode   string         `gorm:"size:40;not null" json:"operating_mode"`
	ScenarioStatus  string         `gorm:"size:24;index;not null;check:scenario_status IN ('draft','pending_review','approved','archived')" json:"scenario_status"`
	SolverTolerance float64        `gorm:"not null;default:0.02" json:"solver_tolerance"`
	MaxIterations   int            `gorm:"not null;default:80" json:"max_iterations"`
	Version         uint           `gorm:"not null;default:1" json:"version"`
	CreatedBy       uint           `gorm:"not null;index" json:"created_by"`
	ApprovedBy      *uint          `gorm:"index" json:"approved_by"`
	RejectReason    string         `gorm:"size:400" json:"reject_reason"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (FanScenario) TableName() string { return "fan_scenarios" }

// FanScenarioVersion 是方案在每次状态变化时写下的不可变参数快照。
// 该表只追加不更新：(scenario_id, version) 唯一约束保证历史版本
// （包括已归档方案的版本）不会被后续操作或旧版本回滚覆盖。
type FanScenarioVersion struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	ScenarioID      uint           `gorm:"not null;uniqueIndex:idx_fan_scenario_version,priority:1" json:"scenario_id"`
	Version         uint           `gorm:"not null;uniqueIndex:idx_fan_scenario_version,priority:2" json:"version"`
	ScenarioStatus  string         `gorm:"size:24;not null;index" json:"scenario_status"`
	Name            string         `gorm:"size:120;not null" json:"name"`
	Description     string         `gorm:"size:600;not null" json:"description"`
	FanCurveJSON    datatypes.JSON `gorm:"type:jsonb;not null" json:"fan_curve_json"`
	OperatingMode   string         `gorm:"size:40;not null" json:"operating_mode"`
	SolverTolerance float64        `gorm:"not null" json:"solver_tolerance"`
	MaxIterations   int            `gorm:"not null" json:"max_iterations"`
	Action          string         `gorm:"size:40;not null" json:"action"`
	ActorID         uint           `gorm:"not null;index" json:"actor_id"`
	ActorEmail      string         `gorm:"size:160;not null" json:"actor_email"`
	Reason          string         `gorm:"size:400" json:"reason"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (FanScenarioVersion) TableName() string { return "fan_scenario_versions" }
