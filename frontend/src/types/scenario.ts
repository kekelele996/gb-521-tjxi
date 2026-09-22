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

export interface UpdateDraftScenarioInput extends CreateScenarioInput {
  version: number;
}

export type ScenarioVersionKind =
  | 'created'
  | 'submitted'
  | 'approved'
  | 'rejected'
  | 'archived'
  | 'edited'
  | 'restored'
  | 'backfill';

export interface FanScenarioVersion {
  id: number;
  scenario_id: number;
  version: number;
  change_kind: ScenarioVersionKind;
  status_at_change: ScenarioStatus;
  fan_curve_json: FanCurvePoint[];
  operating_mode: FanScenario['operating_mode'];
  solver_tolerance: number;
  max_iterations: number;
  reason: string;
  actor_id: number;
  actor_email: string;
  created_at: string;
}

export interface ScenarioVersionMeta {
  version: number;
  change_kind: ScenarioVersionKind;
  status_at_change: ScenarioStatus;
  reason: string;
  actor_email: string;
  created_at: string;
  fan_curve: FanCurvePoint[];
  operating_mode: FanScenario['operating_mode'];
  solver_tolerance: number;
  max_iterations: number;
}

export interface ScenarioFieldDiff {
  field: string;
  label: string;
  from: unknown;
  to: unknown;
  changed: boolean;
}

export interface CurvePointDiff {
  index: number;
  from_flow_m3s: number;
  to_flow_m3s: number;
  from_pressure_pa: number;
  to_pressure_pa: number;
  changed: boolean;
}

export interface ScenarioVersionComparison {
  scenario_id: number;
  from: ScenarioVersionMeta;
  to: ScenarioVersionMeta;
  fields: ScenarioFieldDiff[];
  curve: CurvePointDiff[];
  curve_equal: boolean;
  has_changes: boolean;
}
