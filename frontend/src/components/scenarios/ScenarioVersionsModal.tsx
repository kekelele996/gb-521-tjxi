import { useEffect, useState } from 'react';
import { Modal, Table } from 'antd';
import { GitCompareArrows } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { compareScenarioVersions, listScenarioVersions } from '../../api/scenarios';
import type { FanCurvePointDiff, FanScenario, FanScenarioVersion, ScenarioVersionAction, ScenarioVersionComparison, VersionFieldDiff } from '../../types/scenario';
import { reportError } from '../../utils/errors';
import { formatDateTime, formatNumber } from '../../utils/format';
import { StatusBadge } from '../common/StatusBadge';

const actionLabel: Record<ScenarioVersionAction, string> = {
  created: '创建草稿',
  submitted: '提交复核',
  approved: '批准',
  rejected: '驳回',
  archived: '归档',
  baseline: '基线迁移',
};

const modeLabel: Record<FanScenario['operating_mode'], string> = { normal: '常规', reduced: '降载', emergency_test: '应急测试' };

const statusText: Record<string, string> = { draft: '草稿', pending_review: '待复核', approved: '已批准', archived: '已归档' };

function fieldValueText(field: string, value: string): string {
  if (field === 'scenario_status') return statusText[value] ?? value;
  if (field === 'operating_mode') return modeLabel[value as FanScenario['operating_mode']] ?? value;
  return value;
}

interface Props {
  scenario: FanScenario | null;
  onClose: () => void;
}

export function ScenarioVersionsModal({ scenario, onClose }: Props) {
  const [versions, setVersions] = useState<FanScenarioVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<number[]>([]);
  const [comparison, setComparison] = useState<ScenarioVersionComparison | null>(null);
  const [comparing, setComparing] = useState(false);

  useEffect(() => {
    if (!scenario) return;
    setLoading(true);
    setSelected([]);
    setComparison(null);
    listScenarioVersions(scenario.id)
      .then(setVersions)
      .catch((error) => reportError(error, '版本留痕加载失败'))
      .finally(() => setLoading(false));
  }, [scenario]);

  useEffect(() => {
    if (!scenario || selected.length !== 2) {
      setComparison(null);
      return;
    }
    setComparing(true);
    compareScenarioVersions(scenario.id, Math.min(...selected), Math.max(...selected))
      .then(setComparison)
      .catch((error) => reportError(error, '版本差异计算失败'))
      .finally(() => setComparing(false));
  }, [scenario, selected]);

  const versionColumns: ColumnsType<FanScenarioVersion> = [
    { title: '版本', width: 150, render: (_, row) => <div className="primary-cell"><strong>v{row.version}</strong><span>{actionLabel[row.action] ?? row.action}</span></div> },
    { title: '状态', width: 110, render: (_, row) => <StatusBadge status={row.scenario_status} /> },
    { title: '计算阈值', dataIndex: 'solver_tolerance', width: 100, render: (value) => formatNumber(value, 4) },
    { title: '迭代上限', dataIndex: 'max_iterations', width: 90 },
    { title: '运行模式', dataIndex: 'operating_mode', width: 100, render: (value) => modeLabel[value as FanScenario['operating_mode']] ?? value },
    { title: '操作者', dataIndex: 'actor_email', width: 170 },
    { title: '留痕时间', dataIndex: 'created_at', width: 170, render: formatDateTime },
    { title: '说明', dataIndex: 'reason', render: (value) => value || <span className="muted">—</span> },
  ];

  const fieldColumns: ColumnsType<VersionFieldDiff> = [
    { title: '项目', dataIndex: 'label', width: 110 },
    { title: `v${comparison?.from.version ?? ''}（前）`, render: (_, row) => fieldValueText(row.field, row.before) },
    { title: `v${comparison?.to.version ?? ''}（后）`, render: (_, row) => fieldValueText(row.field, row.after) },
  ];

  const curveColumns: ColumnsType<FanCurvePointDiff> = [
    { title: '曲线点', dataIndex: 'index', width: 80, render: (value) => `#${value}` },
    { title: `v${comparison?.from.version ?? ''} 流量 m³/s`, render: (_, row) => (row.before ? formatNumber(row.before.flow_m3s, 2) : <span className="muted">—</span>) },
    { title: `v${comparison?.from.version ?? ''} 压力 Pa`, render: (_, row) => (row.before ? formatNumber(row.before.pressure_pa, 2) : <span className="muted">—</span>) },
    { title: `v${comparison?.to.version ?? ''} 流量 m³/s`, render: (_, row) => (row.after ? formatNumber(row.after.flow_m3s, 2) : <span className="muted">—</span>) },
    { title: `v${comparison?.to.version ?? ''} 压力 Pa`, render: (_, row) => (row.after ? formatNumber(row.after.pressure_pa, 2) : <span className="muted">—</span>) },
  ];

  return (
    <Modal
      title={scenario ? `版本留痕 · ${scenario.name}` : '版本留痕'}
      open={Boolean(scenario)}
      onCancel={onClose}
      footer={null}
      width={1020}
      destroyOnClose
    >
      <p className="version-note">每次提交、驳回、批准、归档都会固化当时的曲线、运行模式、计算阈值与迭代上限。历史版本为只读快照，归档后仍可追溯，当前待审版本不会被旧版本替换。选择两个版本可查看差异。</p>
      <Table
        rowKey="version"
        size="small"
        loading={loading}
        columns={versionColumns}
        dataSource={versions}
        pagination={false}
        scroll={{ x: 1060 }}
        rowSelection={{
          selectedRowKeys: selected,
          onChange: (keys) => setSelected(keys.slice(-2) as number[]),
          getCheckboxProps: (record) => ({ disabled: selected.length >= 2 && !selected.includes(record.version) }),
        }}
        expandable={{
          expandedRowRender: (row) => (
            <div className="version-curve">
              {row.fan_curve_json.map((point, index) => (
                <span key={index}>#{index + 1} {formatNumber(point.flow_m3s, 2)} m³/s → {formatNumber(point.pressure_pa, 2)} Pa</span>
              ))}
            </div>
          ),
        }}
      />
      {comparison && (
        <section className="version-diff" aria-label="版本差异">
          <div className="section-heading">
            <div><GitCompareArrows size={18} /><h3>版本差异 v{comparison.from.version} → v{comparison.to.version}</h3></div>
            <span>高亮行为存在差异的项目</span>
          </div>
          <Table rowKey="field" size="small" loading={comparing} columns={fieldColumns} dataSource={comparison.fields} pagination={false} rowClassName={(row) => (row.changed ? 'diff-changed' : '')} />
          <Table rowKey="index" size="small" loading={comparing} columns={curveColumns} dataSource={comparison.curve} pagination={false} rowClassName={(row) => (row.changed ? 'diff-changed' : '')} />
        </section>
      )}
    </Modal>
  );
}
