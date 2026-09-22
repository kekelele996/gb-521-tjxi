package service

import (
	"testing"

	"gorm.io/datatypes"

	"mine-ventilation-network-simulator/backend/internal/constants"
	"mine-ventilation-network-simulator/backend/internal/model"
)

func versionFixture(version uint, status constants.ScenarioStatus, curve string) model.FanScenarioVersion {
	return model.FanScenarioVersion{
		ScenarioID:      7,
		Version:         version,
		ScenarioStatus:  string(status),
		Name:            "夜班基准方案",
		Description:     "基准曲线",
		FanCurveJSON:    datatypes.JSON([]byte(curve)),
		OperatingMode:   "normal",
		SolverTolerance: 0.02,
		MaxIterations:   100,
	}
}

func TestVersionActionForStatusCoversEveryTransition(t *testing.T) {
	cases := map[constants.ScenarioStatus]constants.ScenarioVersionAction{
		constants.ScenarioStatusPendingReview: constants.ScenarioVersionSubmitted,
		constants.ScenarioStatusApproved:      constants.ScenarioVersionApproved,
		constants.ScenarioStatusDraft:         constants.ScenarioVersionRejected,
		constants.ScenarioStatusArchived:      constants.ScenarioVersionArchived,
	}
	for status, expected := range cases {
		if action := versionActionForStatus(status); action != expected {
			t.Errorf("status %s: expected action %s, got %s", status, expected, action)
		}
	}
}

func TestDiffScenarioVersionsMarksChangedFields(t *testing.T) {
	curve := `[{"flow_m3s":0,"pressure_pa":1450},{"flow_m3s":30,"pressure_pa":1180}]`
	from := versionFixture(2, constants.ScenarioStatusPendingReview, curve)
	to := versionFixture(3, constants.ScenarioStatusApproved, curve)
	to.SolverTolerance = 0.05
	to.MaxIterations = 200
	to.OperatingMode = "reduced"

	comparison := diffScenarioVersions(from, to)
	changed := map[string]bool{}
	for _, field := range comparison.Fields {
		changed[field.Field] = field.Changed
	}
	for _, field := range []string{"scenario_status", "operating_mode", "solver_tolerance", "max_iterations"} {
		if !changed[field] {
			t.Errorf("expected field %s to be marked changed", field)
		}
	}
	for _, field := range []string{"name", "description"} {
		if changed[field] {
			t.Errorf("expected field %s to be unchanged", field)
		}
	}
	for _, point := range comparison.Curve {
		if point.Changed {
			t.Errorf("curve point %d should be unchanged", point.Index)
		}
	}
}

func TestDiffFanCurvesAlignsPointsAndDetectsChanges(t *testing.T) {
	before := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1450},{"flow_m3s":30,"pressure_pa":1180}]`))
	after := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1450},{"flow_m3s":30,"pressure_pa":1100},{"flow_m3s":60,"pressure_pa":720}]`))

	points := diffFanCurves(before, after)
	if len(points) != 3 {
		t.Fatalf("expected 3 aligned points, got %d", len(points))
	}
	if points[0].Changed {
		t.Error("first point is identical and must not be marked changed")
	}
	if !points[1].Changed {
		t.Error("second point pressure changed and must be marked changed")
	}
	if !points[2].Changed || points[2].Before != nil || points[2].After == nil {
		t.Error("added point must be marked changed with only an after value")
	}
}

func TestDiffFanCurvesHandlesInvalidJSON(t *testing.T) {
	points := diffFanCurves(datatypes.JSON([]byte(`not-json`)), datatypes.JSON([]byte(`[]`)))
	if len(points) != 0 {
		t.Fatalf("invalid curves should produce no points, got %d", len(points))
	}
}
