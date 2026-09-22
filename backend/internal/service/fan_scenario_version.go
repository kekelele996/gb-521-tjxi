package service

import (
	"context"
	"encoding/json"
	"strconv"

	"gorm.io/datatypes"

	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type VersionFieldDiff struct {
	Field   string `json:"field"`
	Label   string `json:"label"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Changed bool   `json:"changed"`
}

type FanCurvePointDiff struct {
	Index   int                `json:"index"`
	Before  *dto.FanCurvePoint `json:"before"`
	After   *dto.FanCurvePoint `json:"after"`
	Changed bool               `json:"changed"`
}

type VersionComparison struct {
	ScenarioID uint                     `json:"scenario_id"`
	From       model.FanScenarioVersion `json:"from"`
	To         model.FanScenarioVersion `json:"to"`
	Fields     []VersionFieldDiff       `json:"fields"`
	Curve      []FanCurvePointDiff      `json:"curve"`
}

// ListVersions 返回同一条方案的全部历史版本（含已归档方案），按版本号倒序。
func (s *FanScenarioService) ListVersions(ctx context.Context, scenarioID uint) ([]model.FanScenarioVersion, error) {
	if _, err := s.scenarios.Find(ctx, scenarioID); err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	items, err := s.scenarios.ListVersions(ctx, scenarioID)
	if err != nil {
		return nil, mapRepositoryError(err, "方案版本")
	}
	return items, nil
}

// CompareVersions 对比同一条方案的两个历史版本，返回字段级与曲线逐点差异。
// 两个版本都必须属于该方案，避免跨方案误比。
func (s *FanScenarioService) CompareVersions(ctx context.Context, scenarioID uint, fromVersion, toVersion uint) (*VersionComparison, error) {
	if fromVersion == toVersion {
		return nil, api.BadRequest("SAME_VERSION", "请选择两个不同的版本进行对比", nil)
	}
	from, err := s.scenarios.FindVersion(ctx, scenarioID, fromVersion)
	if err != nil {
		return nil, mapRepositoryError(err, "方案版本")
	}
	to, err := s.scenarios.FindVersion(ctx, scenarioID, toVersion)
	if err != nil {
		return nil, mapRepositoryError(err, "方案版本")
	}
	comparison := diffScenarioVersions(*from, *to)
	return &comparison, nil
}

func diffScenarioVersions(from, to model.FanScenarioVersion) VersionComparison {
	return VersionComparison{
		ScenarioID: from.ScenarioID,
		From:       from,
		To:         to,
		Fields: []VersionFieldDiff{
			diffField("scenario_status", "状态", from.ScenarioStatus, to.ScenarioStatus),
			diffField("operating_mode", "运行模式", from.OperatingMode, to.OperatingMode),
			diffField("solver_tolerance", "计算阈值", formatFloat(from.SolverTolerance), formatFloat(to.SolverTolerance)),
			diffField("max_iterations", "迭代上限", strconv.Itoa(from.MaxIterations), strconv.Itoa(to.MaxIterations)),
			diffField("name", "方案名称", from.Name, to.Name),
			diffField("description", "方案说明", from.Description, to.Description),
		},
		Curve: diffFanCurves(from.FanCurveJSON, to.FanCurveJSON),
	}
}

func diffField(field, label, before, after string) VersionFieldDiff {
	return VersionFieldDiff{Field: field, Label: label, Before: before, After: after, Changed: before != after}
}

func diffFanCurves(before, after datatypes.JSON) []FanCurvePointDiff {
	beforePoints := parseFanCurve(before)
	afterPoints := parseFanCurve(after)
	length := len(beforePoints)
	if len(afterPoints) > length {
		length = len(afterPoints)
	}
	points := make([]FanCurvePointDiff, 0, length)
	for i := 0; i < length; i++ {
		entry := FanCurvePointDiff{Index: i + 1}
		if i < len(beforePoints) {
			point := beforePoints[i]
			entry.Before = &point
		}
		if i < len(afterPoints) {
			point := afterPoints[i]
			entry.After = &point
		}
		entry.Changed = !fanCurvePointsEqual(entry.Before, entry.After)
		points = append(points, entry)
	}
	return points
}

func parseFanCurve(raw datatypes.JSON) []dto.FanCurvePoint {
	points := []dto.FanCurvePoint{}
	if err := json.Unmarshal(raw, &points); err != nil {
		return []dto.FanCurvePoint{}
	}
	return points
}

func fanCurvePointsEqual(a, b *dto.FanCurvePoint) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.FlowM3S == b.FlowM3S && a.PressurePa == b.PressurePa
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
