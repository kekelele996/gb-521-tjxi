import { useEffect, useMemo, useState } from 'react';
import { Button, Drawer, Empty, Select, Table, Tag, message } from 'antd';
import { History, RotateCcw } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { StatusBadge } from '../common/StatusBadge';
import { useScenarioStore } from '../../stores/scenarioStore';
import type {
  FanScenario,
  FanScenarioVersion,
  ScenarioVersionComparison,
  ScenarioVersionKind,
} from '../../types/scenario';
import { reportError } from '../../utils/errors';
import { formatDateTime, formatNumber } from '../../utils/format';

const kindLabels: Record<ScenarioVersionKind, { label: string; tone: string }> = {
  created: { label: '创建草稿', tone: 'default' },
  submitted: { label: '提交复核', tone: 'processing' },
  approved: { label: '批准', tone: 'success' },
  rejected: { label: '驳回', tone: 'error' },
  archived: { label: '归档', tone: 'warning' },
  edited: { label: '草稿修订', tone: 'blue' },
  restored: { label: '恢复历史参数', tone: 'purple' },
  backfill: { label: '历史基线', tone: 'default' },
};

interface Props {
  scenario: FanScenario | null;
  open: boolean;
  canEditDraft: boolean;
  onClose(): void;
  onChanged(): void;
}

export function ScenarioHistoryDrawer({ scenario, open, canEditDraft, onClose, onChanged }: Props) {
  const { listVersions, compareVersions, restoreVersion } = useScenarioStore();
  const [versions, setVersions] = useState<FanScenarioVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [fromVersion, setFromVersion] = useState<number | undefined>();
  const [toVersion, setToVersion] = useState<number | undefined>();
  const [comparison, setComparison] = useState<ScenarioVersionComparison | null>(null);

  useEffect(() => {
    if (!open || !scenario) return;
    let active = true;
    setLoading(true);
    setComparison(null);
    setFromVersion(undefined);
    setToVersion(undefined);
    listVersions(scenario.id)
      .then((items) => {
        if (!active) return;
        setVersions(items);
        if (items.length >= 2) {
          setFromVersion(items[1].version);
          setToVersion(items[0].version);
        }
      })
      .catch((error) => reportError(error, '历史版本加载失败'))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [open, scenario]);

  useEffect(() => {
    if (!open || !scenario || !fromVersion || !toVersion || fromVersion === toVersion) {
      setComparison(null);
      return;
    }
    let active = true;
    compareVersions(scenario.id, fromVersion, toVersion)
      .then((result) => active && setComparison(result))
      .catch((error) => reportError(error, '版本差异加载失败'));
    return () => {
      active = false;
    };
  }, [open, scenario, fromVersion, toVersion]);

  const versionOptions = useMemo(
    () => versions.map((item) => ({ value: item.version, label: `v${item.version} · ${kindLabels[item.change_kind].label} · ${formatDateTime(item.created_at)}` })),
    [versions],
  );

  const restore = async (version: number) => {
    if (!scenario) return;
    setBusy(true);
    try {
      await restoreVersion(scenario.id, version);
      message.success(`已按 v${version} 参数生成新的草稿版本，历史版本保持不变`);
      onChanged();
      onClose();
    } catch (error) {
      reportError(error, '恢复历史版本失败');
    } finally {
      setBusy(false);
    }
  };

  const columns: ColumnsType<FanScenarioVersion> = [
    {
      title: '版本', width: 118,
      render: (_, row) => (
        <div className="primary-cell">
          <strong>v{row.version}</strong>
          <Tag color={kindLabels[row.change_kind].tone} style={{ width: 'fit-content', marginInlineEnd: 0 }}>{kindLabels[row.change_kind].label}</Tag>
        </div>
      ),
    },
    { title: '留痕时状态', dataIndex: 'status_at_change', width: 110, render: (value: FanScenarioVersion['status_at_change']) => <StatusBadge status={value} /> },
    { title: '运行模式', dataIndex: 'operating_mode', width: 100 },
    { title: '计算阈值', dataIndex: 'solver_tolerance', width: 100, render: (value: number) => formatNumber(value, 4) },
    { title: '迭代上限', dataIndex: 'max_iterations', width: 90 },
    { title: '操作者', dataIndex: 'actor_email', width: 180 },
    { title: '留痕时间', dataIndex: 'created_at', width: 170, render: formatDateTime },
    {
      title: '恢复', width: 92,
      render: (_, row) =>
        canEditDraft && scenario?.scenario_status === 'draft' ? (
          <Button size="small" type="text" icon={<RotateCcw size={14} />} disabled={busy || row.version === scenario.version} onClick={() => void restore(row.version)}>
            恢复
          </Button>
        ) : (
          <span className="muted">—</span>
        ),
    },
  ];

  return (
    <Drawer
      title={scenario ? <span className="drawer-title"><History size={17} />{scenario.name} · 历史版本留痕</span> : '历史版本留痕'}
      open={open}
      onClose={onClose}
      width={Math.min(1080, typeof window === 'undefined' ? 1080 : window.innerWidth - 32)}
      destroyOnClose
    >
      <p className="muted history-note">
        每条记录都是状态变化或草稿修订时刻的不可变快照，保留当时的风机曲线、运行模式、计算阈值与迭代上限；归档后仍可追溯，且任何历史版本都不会覆盖当前待审版本。
      </p>
      <Table rowKey="id" columns={columns} dataSource={versions} loading={loading} size="small" pagination={false} scroll={{ x: 960 }} locale={{ emptyText: <Empty description="暂无历史版本" /> }} />
      <section className="workspace-section version-diff" aria-label="历史版本差异">
        <div className="section-heading">
          <div><span className="section-index">DIFF</span><h2>两个历史版本差异</h2></div>
        </div>
        <div className="version-diff-pickers">
          <label>基准版本<Select value={fromVersion} onChange={setFromVersion} options={versionOptions} placeholder="选择基准版本" /></label>
          <label>对比版本<Select value={toVersion} onChange={setToVersion} options={versionOptions} placeholder="选择对比版本" /></label>
        </div>
        {comparison && <VersionDiffTable comparison={comparison} />}
      </section>
    </Drawer>
  );
}

function VersionDiffTable({ comparison }: { comparison: ScenarioVersionComparison }) {
  return (
    <div className="version-diff-body">
      <Table
        size="small"
        pagination={false}
        rowKey="field"
        dataSource={comparison.fields}
        locale={{ emptyText: <Empty description="两个版本参数完全一致" /> }}
        columns={[
          { title: '参数', dataIndex: 'label', width: 130, render: (value: string, row) => <strong className={row.changed ? 'diff-changed' : ''}>{value}</strong> },
          {
            title: `v${comparison.from.version}（${kindLabels[comparison.from.change_kind].label}）`,
            dataIndex: 'from',
            render: (value: unknown, row) => <DiffValue changed={row.changed} value={value} />,
          },
          {
            title: `v${comparison.to.version}（${kindLabels[comparison.to.change_kind].label}）`,
            dataIndex: 'to',
            render: (value: unknown, row) => <DiffValue changed={row.changed} value={value} />,
          },
        ]}
      />
      {!comparison.curve_equal && (
        <details className="curve-diff-details" open>
          <summary>风机曲线逐点差异（{comparison.curve.length} 个点）</summary>
          <Table
            size="small"
            pagination={false}
            rowKey="index"
            dataSource={comparison.curve}
            columns={[
              { title: '点位', dataIndex: 'index', width: 70, render: (value: number) => `#${value}` },
              { title: '基准流量 m³/s', dataIndex: 'from_flow_m3s', render: (value: number, row) => <DiffValue changed={row.changed} value={value} /> },
              { title: '对比流量 m³/s', dataIndex: 'to_flow_m3s', render: (value: number, row) => <DiffValue changed={row.changed} value={value} /> },
              { title: '基准压力 Pa', dataIndex: 'from_pressure_pa', render: (value: number, row) => <DiffValue changed={row.changed} value={value} /> },
              { title: '对比压力 Pa', dataIndex: 'to_pressure_pa', render: (value: number, row) => <DiffValue changed={row.changed} value={value} /> },
            ]}
          />
        </details>
      )}
    </div>
  );
}

function DiffValue({ value, changed }: { value: unknown; changed: boolean }) {
  return <span className={changed ? 'diff-changed diff-value' : 'muted diff-value'}>{value === null || value === undefined || value === '' ? '—' : String(value)}</span>;
}
