import { request, requestPage } from './client';
import type {
  CreateScenarioInput,
  FanScenario,
  FanScenarioVersion,
  ScenarioStatus,
  ScenarioVersionComparison,
  UpdateDraftScenarioInput,
} from '../types/scenario';

export const listScenarios = () => requestPage<FanScenario>('/api/v1/scenarios?page_size=100');
export const createScenario = (input: CreateScenarioInput) => request<FanScenario>('/api/v1/scenarios', { method: 'POST', body: JSON.stringify(input) });
export const updateDraftScenario = (id: number, input: UpdateDraftScenarioInput) =>
  request<FanScenario>(`/api/v1/scenarios/${id}`, { method: 'PUT', body: JSON.stringify(input) });
export const transitionScenario = (id: number, target_status: ScenarioStatus, version: number, reason = '') =>
  request<FanScenario>(`/api/v1/scenarios/${id}/transition`, { method: 'POST', body: JSON.stringify({ target_status, version, reason }) });
export const listScenarioVersions = (id: number) =>
  requestPage<FanScenarioVersion>(`/api/v1/scenarios/${id}/versions?page_size=100`);
export const compareScenarioVersions = (id: number, fromVersion: number, toVersion: number) =>
  request<ScenarioVersionComparison>(
    `/api/v1/scenarios/${id}/versions/diff?from_version=${fromVersion}&to_version=${toVersion}`,
  );
export const restoreScenarioVersion = (id: number, version: number) =>
  request<FanScenario>(`/api/v1/scenarios/${id}/versions/${version}/restore`, { method: 'POST' });
