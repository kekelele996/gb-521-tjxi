package repository

import (
	"context"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"mine-ventilation-network-simulator/backend/internal/model"
)

func newScenarioTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.FanScenario{}, &model.FanScenarioVersion{}, &model.AuditEvent{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable("fan_scenario_versions", "audit_events", "fan_scenarios")
	})
	return db
}

func scenarioAudit(id uint, email string) AuditRecord {
	return AuditRecord{RequestID: "req-" + email, ActorID: id, ActorEmail: email, Action: "test", EntityType: "fan_scenario"}
}

func TestCreateAndTransitionWritesImmutableVersions(t *testing.T) {
	db := newScenarioTestDB(t)
	repo := NewFanScenarioRepository(db)
	ctx := context.Background()
	curve := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1400},{"flow_m3s":40,"pressure_pa":800}]`))
	scenario := &model.FanScenario{
		Name: "测试方案", Description: "版本留痕测试", FanCurveJSON: curve, OperatingMode: "normal",
		ScenarioStatus: "draft", SolverTolerance: 0.02, MaxIterations: 100, Version: 1, CreatedBy: 1,
	}
	if err := repo.Create(ctx, scenario, scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := repo.Transition(ctx, scenario.ID, 1, 1, "draft", "pending_review", "", scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := repo.Transition(ctx, scenario.ID, 2, 2, "pending_review", "approved", "", scenarioAudit(2, "reviewer@mine.local")); err != nil {
		t.Fatalf("approve: %v", err)
	}
	versions, total, err := repo.ListVersions(ctx, scenario.ID, 1, 100)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 version snapshots, got %d", total)
	}
	wantKinds := []string{"approved", "submitted", "created"}
	for i, want := range wantKinds {
		if versions[i].ChangeKind != want {
			t.Fatalf("version %d kind = %s, want %s", i, versions[i].ChangeKind, want)
		}
	}
	approved := versions[0]
	if approved.StatusAtChange != "approved" || approved.ActorEmail != "reviewer@mine.local" {
		t.Fatalf("approved snapshot mismatch: %+v", approved)
	}
	if string(approved.FanCurveJSON) != string(curve) {
		t.Fatalf("approved snapshot curve was not preserved")
	}
}

func TestEditDraftCreatesNewSnapshotWithoutReplacingHistory(t *testing.T) {
	db := newScenarioTestDB(t)
	repo := NewFanScenarioRepository(db)
	ctx := context.Background()
	curveV1 := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1400},{"flow_m3s":40,"pressure_pa":800}]`))
	scenario := &model.FanScenario{
		Name: "草稿方案", Description: "编辑测试", FanCurveJSON: curveV1, OperatingMode: "normal",
		ScenarioStatus: "draft", SolverTolerance: 0.05, MaxIterations: 80, Version: 1, CreatedBy: 1,
	}
	if err := repo.Create(ctx, scenario, scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("create: %v", err)
	}
	curveV2 := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1200},{"flow_m3s":45,"pressure_pa":700}]`))
	updated, err := repo.UpdateDraft(ctx, scenario.ID, 1, 1, DraftPatch{
		Name: "草稿方案", Description: "编辑测试", FanCurve: curveV2, OperatingMode: "reduced",
		SolverTolerance: 0.01, MaxIterations: 120,
	}, "", "engineer@mine.local", scenarioAudit(1, "engineer@mine.local"))
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
	v1, err := repo.FindVersion(ctx, scenario.ID, 1)
	if err != nil {
		t.Fatalf("find v1: %v", err)
	}
	if string(v1.FanCurveJSON) != string(curveV1) || v1.SolverTolerance != 0.05 || v1.MaxIterations != 80 {
		t.Fatalf("historical v1 parameters were overwritten: %+v", v1)
	}
	if v1.ChangeKind != "created" {
		t.Fatalf("v1 kind = %s, want created", v1.ChangeKind)
	}
}

func TestEditRejectedWhenScenarioIsPendingReview(t *testing.T) {
	db := newScenarioTestDB(t)
	repo := NewFanScenarioRepository(db)
	ctx := context.Background()
	curve := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1400},{"flow_m3s":40,"pressure_pa":800}]`))
	scenario := &model.FanScenario{
		Name: "待审方案", Description: "待审不可改", FanCurveJSON: curve, OperatingMode: "normal",
		ScenarioStatus: "draft", SolverTolerance: 0.02, MaxIterations: 100, Version: 1, CreatedBy: 1,
	}
	if err := repo.Create(ctx, scenario, scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := repo.Transition(ctx, scenario.ID, 1, 1, "draft", "pending_review", "", scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	_, err := repo.UpdateDraft(ctx, scenario.ID, 1, 2, DraftPatch{
		Name: "被篡改", Description: "待审不可改", FanCurve: curve, OperatingMode: "emergency_test",
		SolverTolerance: 0.9, MaxIterations: 200,
	}, "", "engineer@mine.local", scenarioAudit(1, "engineer@mine.local"))
	if err != ErrScenarioNotDraft {
		t.Fatalf("expected ErrScenarioNotDraft, got %v", err)
	}
	rows, _, err := repo.ListVersions(ctx, scenario.ID, 1, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, row := range rows {
		if row.ChangeKind == "edited" {
			t.Fatal("pending_review scenario must not gain an edited snapshot")
		}
	}
}

func TestRestoreVersionKeepsHistoryAndAdvancesVersion(t *testing.T) {
	db := newScenarioTestDB(t)
	repo := NewFanScenarioRepository(db)
	ctx := context.Background()
	curveV1 := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1400},{"flow_m3s":40,"pressure_pa":800}]`))
	scenario := &model.FanScenario{
		Name: "恢复方案", Description: "恢复测试", FanCurveJSON: curveV1, OperatingMode: "normal",
		ScenarioStatus: "draft", SolverTolerance: 0.05, MaxIterations: 80, Version: 1, CreatedBy: 1,
	}
	if err := repo.Create(ctx, scenario, scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("create: %v", err)
	}
	curveV2 := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":900},{"flow_m3s":40,"pressure_pa":500}]`))
	if _, err := repo.UpdateDraft(ctx, scenario.ID, 1, 1, DraftPatch{
		Name: "恢复方案", Description: "恢复测试", FanCurve: curveV2, OperatingMode: "reduced",
		SolverTolerance: 0.01, MaxIterations: 200,
	}, "", "engineer@mine.local", scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("edit: %v", err)
	}
	restored, err := repo.RestoreVersion(ctx, scenario.ID, 1, 1, "engineer@mine.local", scenarioAudit(1, "engineer@mine.local"))
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Version != 3 {
		t.Fatalf("expected version 3 after restore, got %d", restored.Version)
	}
	if string(restored.FanCurveJSON) != string(curveV1) || restored.MaxIterations != 80 {
		t.Fatalf("restored parameters do not match v1: %+v", restored)
	}
	v1, err := repo.FindVersion(ctx, scenario.ID, 1)
	if err != nil || string(v1.FanCurveJSON) != string(curveV1) {
		t.Fatalf("source v1 must remain intact")
	}
	v3, err := repo.FindVersion(ctx, scenario.ID, 3)
	if err != nil || v3.ChangeKind != "restored" {
		t.Fatalf("expected restored snapshot v3, got %+v err=%v", v3, err)
	}
}

func TestArchivedScenarioVersionsRemainReadable(t *testing.T) {
	db := newScenarioTestDB(t)
	repo := NewFanScenarioRepository(db)
	ctx := context.Background()
	curve := datatypes.JSON([]byte(`[{"flow_m3s":0,"pressure_pa":1400},{"flow_m3s":40,"pressure_pa":800}]`))
	scenario := &model.FanScenario{
		Name: "归档方案", Description: "归档追溯", FanCurveJSON: curve, OperatingMode: "normal",
		ScenarioStatus: "draft", SolverTolerance: 0.02, MaxIterations: 100, Version: 1, CreatedBy: 1,
	}
	if err := repo.Create(ctx, scenario, scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := repo.Transition(ctx, scenario.ID, 1, 1, "draft", "pending_review", "", scenarioAudit(1, "engineer@mine.local")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := repo.Transition(ctx, scenario.ID, 2, 2, "pending_review", "approved", "", scenarioAudit(2, "reviewer@mine.local")); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := repo.Transition(ctx, scenario.ID, 3, 3, "approved", "archived", "", scenarioAudit(3, "admin@mine.local")); err != nil {
		t.Fatalf("archive: %v", err)
	}
	approved, err := repo.FindVersion(ctx, scenario.ID, 3)
	if err != nil || approved.ChangeKind != "approved" {
		t.Fatalf("approved version must remain traceable after archive: %+v err=%v", approved, err)
	}
	versions, total, err := repo.ListVersions(ctx, scenario.ID, 1, 100)
	if err != nil || total != 4 {
		t.Fatalf("expected 4 snapshots after archive, total=%d err=%v", total, err)
	}
	if versions[0].ChangeKind != "archived" {
		t.Fatalf("latest snapshot should be archived, got %s", versions[0].ChangeKind)
	}
}
