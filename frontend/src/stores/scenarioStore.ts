import { create } from 'zustand';
import * as scenarioApi from '../api/scenarios';
import type {
  CreateScenarioInput,
  FanScenario,
  FanScenarioVersion,
  ScenarioStatus,
  ScenarioVersionComparison,
  UpdateDraftScenarioInput,
} from '../types/scenario';

interface ScenarioState {
  scenarios: FanScenario[];
  loading: boolean;
  load(): Promise<void>;
  create(input: CreateScenarioInput): Promise<void>;
  updateDraft(id: number, input: UpdateDraftScenarioInput): Promise<FanScenario>;
  transition(id: number, status: ScenarioStatus, version: number, reason?: string): Promise<void>;
  listVersions(id: number): Promise<FanScenarioVersion[]>;
  compareVersions(id: number, fromVersion: number, toVersion: number): Promise<ScenarioVersionComparison>;
  restoreVersion(id: number, version: number): Promise<FanScenario>;
}

export const useScenarioStore = create<ScenarioState>((set) => ({
  scenarios: [], loading: false,
  async load() {
    set({ loading: true });
    try { set({ scenarios: (await scenarioApi.listScenarios()).items }); } finally { set({ loading: false }); }
  },
  async create(input) {
    await scenarioApi.createScenario(input);
    set({ scenarios: (await scenarioApi.listScenarios()).items });
  },
  async updateDraft(id, input) {
    const updated = await scenarioApi.updateDraftScenario(id, input);
    set({ scenarios: (await scenarioApi.listScenarios()).items });
    return updated;
  },
  async transition(id, status, version, reason = '') {
    await scenarioApi.transitionScenario(id, status, version, reason);
    set({ scenarios: (await scenarioApi.listScenarios()).items });
  },
  async listVersions(id) {
    return (await scenarioApi.listScenarioVersions(id)).items;
  },
  async compareVersions(id, fromVersion, toVersion) {
    return scenarioApi.compareScenarioVersions(id, fromVersion, toVersion);
  },
  async restoreVersion(id, version) {
    const updated = await scenarioApi.restoreScenarioVersion(id, version);
    set({ scenarios: (await scenarioApi.listScenarios()).items });
    return updated;
  },
}));
