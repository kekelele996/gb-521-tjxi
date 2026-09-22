package dto

type FanCurvePoint struct {
	FlowM3S    float64 `json:"flow_m3s" binding:"gte=0,lte=5000"`
	PressurePa float64 `json:"pressure_pa" binding:"gte=0,lte=200000"`
}

type CreateFanScenarioRequest struct {
	Name            string          `json:"name" binding:"required,min=2,max=120"`
	Description     string          `json:"description" binding:"required,min=4,max=600"`
	FanCurve        []FanCurvePoint `json:"fan_curve" binding:"required,min=2,max=20,dive"`
	OperatingMode   string          `json:"operating_mode" binding:"required,oneof=normal reduced emergency_test"`
	SolverTolerance float64         `json:"solver_tolerance" binding:"required,gt=0,lte=10"`
	MaxIterations   int             `json:"max_iterations" binding:"required,gte=10,lte=500"`
}

type TransitionScenarioRequest struct {
	TargetStatus string `json:"target_status" binding:"required"`
	Reason       string `json:"reason" binding:"omitempty,max=400"`
	Version      uint   `json:"version" binding:"required,gte=1"`
}

// UpdateDraftScenarioRequest 只允许修订草稿参数；待审、已批准和已归档方案禁止编辑。
type UpdateDraftScenarioRequest struct {
	Name            string          `json:"name" binding:"required,min=2,max=120"`
	Description     string          `json:"description" binding:"required,min=4,max=600"`
	FanCurve        []FanCurvePoint `json:"fan_curve" binding:"required,min=2,max=20,dive"`
	OperatingMode   string          `json:"operating_mode" binding:"required,oneof=normal reduced emergency_test"`
	SolverTolerance float64         `json:"solver_tolerance" binding:"required,gt=0,lte=10"`
	MaxIterations   int             `json:"max_iterations" binding:"required,gte=10,lte=500"`
	Version         uint            `json:"version" binding:"required,gte=1"`
}

type ScenarioVersionQuery struct {
	Page     int `form:"page" binding:"omitempty,gte=1"`
	PageSize int `form:"page_size" binding:"omitempty,gte=1,lte=100"`
}

type ScenarioVersionDiffQuery struct {
	FromVersion uint `form:"from_version" binding:"required,gte=1"`
	ToVersion   uint `form:"to_version" binding:"required,gte=1"`
}

type CurvePointDiff struct {
	Index        int     `json:"index"`
	FromFlowM3S  float64 `json:"from_flow_m3s"`
	ToFlowM3S    float64 `json:"to_flow_m3s"`
	FromPressure float64 `json:"from_pressure_pa"`
	ToPressure   float64 `json:"to_pressure_pa"`
	Changed      bool    `json:"changed"`
}

type ScenarioFieldDiff struct {
	Field   string      `json:"field"`
	Label   string      `json:"label"`
	From    interface{} `json:"from"`
	To      interface{} `json:"to"`
	Changed bool        `json:"changed"`
}

type ScenarioVersionComparison struct {
	ScenarioID uint                `json:"scenario_id"`
	From       ScenarioVersionMeta `json:"from"`
	To         ScenarioVersionMeta `json:"to"`
	Fields     []ScenarioFieldDiff `json:"fields"`
	Curve      []CurvePointDiff    `json:"curve"`
	CurveEqual bool                `json:"curve_equal"`
	HasChanges bool                `json:"has_changes"`
}

type ScenarioVersionMeta struct {
	Version         uint            `json:"version"`
	ChangeKind      string          `json:"change_kind"`
	StatusAtChange  string          `json:"status_at_change"`
	Reason          string          `json:"reason"`
	ActorEmail      string          `json:"actor_email"`
	CreatedAt       string          `json:"created_at"`
	FanCurve        []FanCurvePoint `json:"fan_curve"`
	OperatingMode   string          `json:"operating_mode"`
	SolverTolerance float64         `json:"solver_tolerance"`
	MaxIterations   int             `json:"max_iterations"`
}

type ScenarioListQuery struct {
	Page     int    `form:"page" binding:"omitempty,gte=1"`
	PageSize int    `form:"page_size" binding:"omitempty,gte=1,lte=100"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}
