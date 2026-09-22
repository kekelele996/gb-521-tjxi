package service

import (
	"testing"

	"gorm.io/datatypes"

	"mine-ventilation-network-simulator/backend/internal/dto"
	"mine-ventilation-network-simulator/backend/internal/model"
)

func versionSnap(version uint, kind, status, mode string, tolerance float64, iterations int, curve string) *model.FanScenarioVersion {
	return &model.FanScenarioVersion{
		ID: version, ScenarioID: 7, Version: version, ChangeKind: kind, StatusAtChange: status,
		FanCurveJSON: datatypes.JSON([]byte(curve)), OperatingMode: mode,
		SolverTolerance: tolerance, MaxIterations: iterations, ActorEmail: "reviewer@mine.local",
	}
}

func TestBuildVersionComparisonDetectsAllParameterChanges(t *testing.T) {
	curveV1 := `[{"flow_m3s":0,"pressure_pa":1400},{"flow_m3s":40,"pressure_pa":800}]`
	curveV2 := `[{"flow_m3s":0,"pressure_pa":1300},{"flow_m3s":45,"pressure_pa":700},{"flow_m3s":60,"pressure_pa":400}]`
	from := versionSnap(2, "submitted", "pending_review", "normal", 0.05, 80, curveV1)
	to := versionSnap(3, "approved", "approved", "reduced", 0.02, 120, curveV2)

	comparison := buildVersionComparison(from, to)
	if !comparison.HasChanges {
		t.Fatal("expected changes between revisions")
	}
	changed := map[string]bool{}
	for _, field := range comparison.Fields {
		if field.Changed {
			changed[field.Field] = true
		}
	}
	for _, field := range []string{"operating_mode", "solver_tolerance", "max_iterations", "fan_curve", "scenario_status"} {
		if !changed[field] {
			t.Fatalf("expected field %s to be marked changed", field)
		}
	}
	if comparison.CurveEqual || len(comparison.Curve) != 3 {
		t.Fatalf("expected 3 aligned curve points with difference, got %+v", comparison.Curve)
	}
	if !comparison.Curve[0].Changed || comparison.Curve[2].FromFlowM3S != 0 || comparison.Curve[2].ToFlowM3S != 60 {
		t.Fatalf("new curve point was not aligned with zero-filled from-side: %+v", comparison.Curve[2])
	}
	if comparison.From.Version != 2 || comparison.To.Version != 3 {
		t.Fatalf("comparison version metadata mismatch: from=%d to=%d", comparison.From.Version, comparison.To.Version)
	}
}

func TestBuildVersionComparisonIdenticalParametersReportsNoChange(t *testing.T) {
	curve := `[{"flow_m3s":0,"pressure_pa":1400},{"flow_m3s":40,"pressure_pa":800}]`
	from := versionSnap(3, "approved", "approved", "normal", 0.02, 100, curve)
	to := versionSnap(4, "archived", "archived", "normal", 0.02, 100, curve)
	comparison := buildVersionComparison(from, to)
	if comparison.CurveEqual != true {
		t.Fatal("curve should compare equal")
	}
	for _, field := range comparison.Fields {
		if field.Field != "scenario_status" && field.Changed {
			t.Fatalf("parameter field %s should be unchanged across approve->archive", field.Field)
		}
	}
}

func TestAlignCurvePointsFlagsOnlyChangedRows(t *testing.T) {
	from := []dto.FanCurvePoint{{FlowM3S: 0, PressurePa: 100}, {FlowM3S: 10, PressurePa: 80}}
	to := []dto.FanCurvePoint{{FlowM3S: 0, PressurePa: 100}, {FlowM3S: 10, PressurePa: 70}}
	diffs := alignCurvePoints(from, to)
	if len(diffs) != 2 || diffs[0].Changed || !diffs[1].Changed {
		t.Fatalf("unexpected point-level diff: %+v", diffs)
	}
}
