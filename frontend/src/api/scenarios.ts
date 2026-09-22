import { request, requestPage } from './client';
import type { CreateScenarioInput, FanScenario, FanScenarioVersion, ScenarioStatus, ScenarioVersionComparison } from '../types/scenario';

export const listScenarios = () => requestPage<FanScenario>('/api/v1/scenarios?page_size=100');
export const createScenario = (input: CreateScenarioInput) => request<FanScenario>('/api/v1/scenarios', { method: 'POST', body: JSON.stringify(input) });
export const transitionScenario = (id: number, target_status: ScenarioStatus, version: number, reason = '') =>
  request<FanScenario>(`/api/v1/scenarios/${id}/transition`, { method: 'POST', body: JSON.stringify({ target_status, version, reason }) });
export const listScenarioVersions = (id: number) => request<FanScenarioVersion[]>(`/api/v1/scenarios/${id}/versions`);
export const compareScenarioVersions = (id: number, from: number, to: number) =>
  request<ScenarioVersionComparison>(`/api/v1/scenarios/${id}/versions/compare?from=${from}&to=${to}`);
