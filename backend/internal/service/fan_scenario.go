package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/datatypes"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
	"mine-ventilation-network-simulator/backend/internal/repository"
	"mine-ventilation-network-simulator/backend/pkg/api"
)

type FanScenarioService struct {
	scenarios *repository.FanScenarioRepository
}

func NewFanScenarioService(scenarios *repository.FanScenarioRepository) *FanScenarioService {
	return &FanScenarioService{scenarios: scenarios}
}

func (s *FanScenarioService) List(ctx context.Context, query dto.ScenarioListQuery) ([]model.FanScenario, int64, int, int, error) {
	page, pageSize := normalizePage(query.Page, query.PageSize)
	if query.Status != "" && !constants.ValidScenarioStatus(query.Status) {
		return nil, 0, page, pageSize, api.BadRequest("INVALID_SCENARIO_STATUS", "方案状态筛选值无效", nil)
	}
	items, total, err := s.scenarios.List(ctx, page, pageSize, query.Status, strings.TrimSpace(query.Search))
	if err != nil {
		return nil, 0, page, pageSize, mapRepositoryError(err, "风机方案")
	}
	return items, total, page, pageSize, nil
}

func (s *FanScenarioService) Get(ctx context.Context, id uint) (*model.FanScenario, error) {
	item, err := s.scenarios.Find(ctx, id)
	return item, mapRepositoryError(err, "风机方案")
}

func (s *FanScenarioService) Create(ctx context.Context, input dto.CreateFanScenarioRequest, actor Actor) (*model.FanScenario, error) {
	if err := validateFanCurve(input.FanCurve); err != nil {
		return nil, err
	}
	curve, err := json.Marshal(input.FanCurve)
	if err != nil {
		return nil, api.BadRequest("INVALID_FAN_CURVE", "风机曲线无法编码", err.Error())
	}
	scenario := &model.FanScenario{
		Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description),
		FanCurveJSON: datatypes.JSON(curve), OperatingMode: input.OperatingMode,
		ScenarioStatus: string(constants.ScenarioStatusDraft), SolverTolerance: input.SolverTolerance,
		MaxIterations: input.MaxIterations, Version: 1, CreatedBy: actor.ID,
	}
	if err := s.scenarios.Create(ctx, scenario, actor.Audit("fan_scenario.created", "fan_scenario")); err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	return scenario, nil
}

func (s *FanScenarioService) UpdateDraft(ctx context.Context, id uint, input dto.UpdateDraftScenarioRequest, actor Actor) (*model.FanScenario, error) {
	if err := validateFanCurve(input.FanCurve); err != nil {
		return nil, err
	}
	curve, err := json.Marshal(input.FanCurve)
	if err != nil {
		return nil, api.BadRequest("INVALID_FAN_CURVE", "风机曲线无法编码", err.Error())
	}
	patch := repository.DraftPatch{
		Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description),
		FanCurve: datatypes.JSON(curve), OperatingMode: input.OperatingMode,
		SolverTolerance: input.SolverTolerance, MaxIterations: input.MaxIterations,
	}
	audit := actor.Audit("fan_scenario.edited", "fan_scenario")
	updated, err := s.scenarios.UpdateDraft(ctx, id, actor.ID, input.Version, patch, "", actor.Email, audit)
	if err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	return updated, nil
}

func (s *FanScenarioService) Transition(ctx context.Context, id uint, input dto.TransitionScenarioRequest, actor Actor) (*model.FanScenario, error) {
	scenario, err := s.scenarios.Find(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	from := constants.ScenarioStatus(scenario.ScenarioStatus)
	to := constants.ScenarioStatus(input.TargetStatus)
	if !constants.ValidScenarioStatus(input.TargetStatus) || !constants.CanTransitionScenario(from, to) {
		return nil, api.Conflict("ILLEGAL_SCENARIO_TRANSITION", fmt.Sprintf("方案不能从 %s 迁移到 %s", from, to))
	}
	if err := authorizeTransition(from, to, actor.Role, strings.TrimSpace(input.Reason)); err != nil {
		return nil, err
	}
	audit := actor.Audit("fan_scenario."+string(to), "fan_scenario")
	audit.Metadata = fmt.Sprintf(`{"reason":%q}`, strings.TrimSpace(input.Reason))
	updated, err := s.scenarios.Transition(ctx, id, actor.ID, input.Version, string(from), string(to), strings.TrimSpace(input.Reason), audit)
	if err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	return updated, nil
}

func (s *FanScenarioService) ListVersions(ctx context.Context, scenarioID uint, query dto.ScenarioVersionQuery) ([]model.FanScenarioVersion, int64, int, int, error) {
	page, pageSize := normalizePage(query.Page, query.PageSize)
	if _, err := s.scenarios.Find(ctx, scenarioID); err != nil {
		return nil, 0, page, pageSize, mapRepositoryError(err, "风机方案")
	}
	items, total, err := s.scenarios.ListVersions(ctx, scenarioID, page, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, mapRepositoryError(err, "方案历史版本")
	}
	return items, total, page, pageSize, nil
}

func (s *FanScenarioService) CompareVersions(ctx context.Context, scenarioID uint, query dto.ScenarioVersionDiffQuery) (*dto.ScenarioVersionComparison, error) {
	if query.FromVersion == query.ToVersion {
		return nil, api.BadRequest("IDENTICAL_VERSIONS", "请选择两个不同的历史版本进行差异对比", nil)
	}
	if _, err := s.scenarios.Find(ctx, scenarioID); err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	fromSnap, err := s.scenarios.FindVersion(ctx, scenarioID, query.FromVersion)
	if err != nil {
		return nil, mapRepositoryError(err, "方案历史版本")
	}
	toSnap, err := s.scenarios.FindVersion(ctx, scenarioID, query.ToVersion)
	if err != nil {
		return nil, mapRepositoryError(err, "方案历史版本")
	}
	return buildVersionComparison(fromSnap, toSnap), nil
}

func (s *FanScenarioService) RestoreVersion(ctx context.Context, scenarioID, sourceVersion uint, actor Actor) (*model.FanScenario, error) {
	audit := actor.Audit("fan_scenario.restored", "fan_scenario")
	audit.Metadata = fmt.Sprintf(`{"source_version":%d}`, sourceVersion)
	updated, err := s.scenarios.RestoreVersion(ctx, scenarioID, actor.ID, sourceVersion, actor.Email, audit)
	if err != nil {
		return nil, mapRepositoryError(err, "风机方案")
	}
	return updated, nil
}

func validateFanCurve(points []dto.FanCurvePoint) error {
	if len(points) < 2 {
		return api.BadRequest("FAN_CURVE_TOO_SHORT", "风机曲线至少需要两个点", nil)
	}
	for i := range points {
		if i > 0 && points[i].FlowM3S <= points[i-1].FlowM3S {
			return api.BadRequest("FAN_CURVE_FLOW_ORDER", "风机曲线流量必须严格递增", map[string]int{"point": i + 1})
		}
		if i > 0 && points[i].PressurePa > points[i-1].PressurePa {
			return api.BadRequest("FAN_CURVE_PRESSURE_ORDER", "风机曲线压力必须随流量保持不升", map[string]int{"point": i + 1})
		}
	}
	return nil
}

func authorizeTransition(from, to constants.ScenarioStatus, role, reason string) error {
	switch {
	case from == constants.ScenarioStatusDraft && to == constants.ScenarioStatusPendingReview:
		if role != string(constants.RoleEngineer) && role != string(constants.RoleAdmin) {
			return api.Forbidden("只有工程师或管理员可以提交复核")
		}
	case from == constants.ScenarioStatusPendingReview && to == constants.ScenarioStatusApproved:
		if role != string(constants.RoleReviewer) && role != string(constants.RoleAdmin) {
			return api.Forbidden("只有复核员或管理员可以批准方案")
		}
	case from == constants.ScenarioStatusPendingReview && to == constants.ScenarioStatusDraft:
		if role != string(constants.RoleReviewer) && role != string(constants.RoleAdmin) {
			return api.Forbidden("只有复核员或管理员可以驳回方案")
		}
		if len(reason) < 4 {
			return api.BadRequest("REJECT_REASON_REQUIRED", "驳回方案时必须填写至少 4 个字符的原因", nil)
		}
	case from == constants.ScenarioStatusApproved && to == constants.ScenarioStatusArchived:
		if role != string(constants.RoleAdmin) {
			return api.Forbidden("只有管理员可以归档已批准方案")
		}
	}
	return nil
}

// buildVersionComparison 计算同一方案两个历史快照的参数差异，差异仅基于留痕快照，不读取可被覆盖的当前行。
func buildVersionComparison(fromSnap, toSnap *model.FanScenarioVersion) *dto.ScenarioVersionComparison {
	fromCurve := decodeVersionCurve(fromSnap.FanCurveJSON)
	toCurve := decodeVersionCurve(toSnap.FanCurveJSON)
	modeChanged := fromSnap.OperatingMode != toSnap.OperatingMode
	toleranceChanged := !floatEqual(fromSnap.SolverTolerance, toSnap.SolverTolerance)
	iterationsChanged := fromSnap.MaxIterations != toSnap.MaxIterations
	curveEqual := curvesEqual(fromCurve, toCurve)

	fields := []dto.ScenarioFieldDiff{
		{Field: "operating_mode", Label: "运行模式", From: fromSnap.OperatingMode, To: toSnap.OperatingMode, Changed: modeChanged},
		{Field: "solver_tolerance", Label: "计算阈值", From: fromSnap.SolverTolerance, To: toSnap.SolverTolerance, Changed: toleranceChanged},
		{Field: "max_iterations", Label: "迭代上限", From: fromSnap.MaxIterations, To: toSnap.MaxIterations, Changed: iterationsChanged},
		{Field: "fan_curve", Label: "风机曲线", From: pointCount(fromCurve), To: pointCount(toCurve), Changed: !curveEqual},
	}
	curveDiffs := alignCurvePoints(fromCurve, toCurve)
	statusChanged := fromSnap.StatusAtChange != toSnap.StatusAtChange
	hasChanges := modeChanged || toleranceChanged || iterationsChanged || !curveEqual || statusChanged
	if statusChanged {
		fields = append(fields, dto.ScenarioFieldDiff{Field: "scenario_status", Label: "留痕时状态", From: fromSnap.StatusAtChange, To: toSnap.StatusAtChange, Changed: true})
	}
	return &dto.ScenarioVersionComparison{
		ScenarioID: fromSnap.ScenarioID,
		From:       versionMeta(fromSnap, fromCurve),
		To:         versionMeta(toSnap, toCurve),
		Fields:     fields, Curve: curveDiffs, CurveEqual: curveEqual, HasChanges: hasChanges,
	}
}

func versionMeta(snap *model.FanScenarioVersion, curve []dto.FanCurvePoint) dto.ScenarioVersionMeta {
	return dto.ScenarioVersionMeta{
		Version: snap.Version, ChangeKind: snap.ChangeKind, StatusAtChange: snap.StatusAtChange,
		Reason: snap.Reason, ActorEmail: snap.ActorEmail, CreatedAt: snap.CreatedAt.UTC().Format(time.RFC3339),
		FanCurve: curve, OperatingMode: snap.OperatingMode,
		SolverTolerance: snap.SolverTolerance, MaxIterations: snap.MaxIterations,
	}
}

func decodeVersionCurve(raw datatypes.JSON) []dto.FanCurvePoint {
	points := []dto.FanCurvePoint{}
	if len(raw) == 0 {
		return points
	}
	_ = json.Unmarshal(raw, &points)
	return points
}

func pointCount(points []dto.FanCurvePoint) int { return len(points) }

func floatEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func curvesEqual(a, b []dto.FanCurvePoint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !floatEqual(a[i].FlowM3S, b[i].FlowM3S) || !floatEqual(a[i].PressurePa, b[i].PressurePa) {
			return false
		}
	}
	return true
}

// alignCurvePoints 按序号对齐曲线点，长度不一致时用零值补齐并标记变化。
func alignCurvePoints(from, to []dto.FanCurvePoint) []dto.CurvePointDiff {
	length := len(from)
	if len(to) > length {
		length = len(to)
	}
	diffs := make([]dto.CurvePointDiff, 0, length)
	for i := 0; i < length; i++ {
		item := dto.CurvePointDiff{Index: i + 1}
		if i < len(from) {
			item.FromFlowM3S, item.FromPressure = from[i].FlowM3S, from[i].PressurePa
		}
		if i < len(to) {
			item.ToFlowM3S, item.ToPressure = to[i].FlowM3S, to[i].PressurePa
		}
		item.Changed = !floatEqual(item.FromFlowM3S, item.ToFlowM3S) || !floatEqual(item.FromPressure, item.ToPressure) || i >= len(from) || i >= len(to)
		diffs = append(diffs, item)
	}
	return diffs
}
