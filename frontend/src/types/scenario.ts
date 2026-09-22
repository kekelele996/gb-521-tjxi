export type ScenarioStatus = 'draft' | 'pending_review' | 'approved' | 'archived';

export interface FanCurvePoint {
  flow_m3s: number;
  pressure_pa: number;
}

export interface FanScenario {
  id: number;
  name: string;
  description: string;
  fan_curve_json: FanCurvePoint[];
  operating_mode: 'normal' | 'reduced' | 'emergency_test';
  scenario_status: ScenarioStatus;
  solver_tolerance: number;
  max_iterations: number;
  version: number;
  created_by: number;
  approved_by?: number;
  reject_reason: string;
  created_at: string;
  updated_at: string;
}

export interface CreateScenarioInput {
  name: string;
  description: string;
  fan_curve: FanCurvePoint[];
  operating_mode: FanScenario['operating_mode'];
  solver_tolerance: number;
  max_iterations: number;
}

export type ScenarioVersionAction = 'created' | 'submitted' | 'approved' | 'rejected' | 'archived' | 'baseline';

export interface FanScenarioVersion {
  id: number;
  scenario_id: number;
  version: number;
  scenario_status: ScenarioStatus;
  name: string;
  description: string;
  fan_curve_json: FanCurvePoint[];
  operating_mode: FanScenario['operating_mode'];
  solver_tolerance: number;
  max_iterations: number;
  action: ScenarioVersionAction;
  actor_id: number;
  actor_email: string;
  reason: string;
  created_at: string;
}

export interface VersionFieldDiff {
  field: string;
  label: string;
  before: string;
  after: string;
  changed: boolean;
}

export interface FanCurvePointDiff {
  index: number;
  before?: FanCurvePoint;
  after?: FanCurvePoint;
  changed: boolean;
}

export interface ScenarioVersionComparison {
  scenario_id: number;
  from: FanScenarioVersion;
  to: FanScenarioVersion;
  fields: VersionFieldDiff[];
  curve: FanCurvePointDiff[];
}
