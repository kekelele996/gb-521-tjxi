import { useEffect, useState } from 'react';
import { Button, Form, Input, InputNumber, Modal, Select, Table, message } from 'antd';
import { CheckCheck, GitCompareArrows, History, PencilLine, Plus, Send, Undo2 } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import type { Key } from 'react';
import { EditDraftModal } from '../components/scenario/EditDraftModal';
import { ScenarioHistoryDrawer } from '../components/scenario/ScenarioHistoryDrawer';
import { ConfirmActionDialog } from '../components/common/ConfirmActionDialog';
import { PageHeader } from '../components/common/PageHeader';
import { StatusBadge } from '../components/common/StatusBadge';
import { useAuth } from '../hooks/useAuth';
import { useEdgeStore } from '../stores/edgeStore';
import { useScenarioStore } from '../stores/scenarioStore';
import type { CreateScenarioInput, FanScenario, ScenarioStatus } from '../types/scenario';
import { reportError } from '../utils/errors';
import { formatDateTime, formatNumber } from '../utils/format';

interface PendingTransition { scenario: FanScenario; status: ScenarioStatus }

export function ScenariosPage() {
  const { user, hasRole } = useAuth();
  const { scenarios, loading, load, create, transition } = useScenarioStore();
  const { edges, load: loadEdges } = useEdgeStore();
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<FanScenario | null>(null);
  const [historyTarget, setHistoryTarget] = useState<FanScenario | null>(null);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [pending, setPending] = useState<PendingTransition | null>(null);
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<Key[]>([]);
  const [form] = Form.useForm();
  const canCreate = hasRole('engineer', 'admin');
  const refresh = () => Promise.all([load(), loadEdges()]);
  useEffect(() => { refresh().catch((error) => reportError(error, '方案数据加载失败')); }, []);

  const submitCreate = async (values: Record<string, number | string>) => {
    const input: CreateScenarioInput = {
      name: String(values.name), description: String(values.description), operating_mode: values.operating_mode as CreateScenarioInput['operating_mode'],
      solver_tolerance: Number(values.solver_tolerance), max_iterations: Number(values.max_iterations),
      fan_curve: [
        { flow_m3s: 0, pressure_pa: Number(values.pressure_zero) },
        { flow_m3s: Number(values.flow_design), pressure_pa: Number(values.pressure_design) },
        { flow_m3s: Number(values.flow_max), pressure_pa: Number(values.pressure_max) },
      ],
    };
    setBusy(true);
    try { await create(input); message.success('方案草稿已创建，v1 参数已留痕'); setCreateOpen(false); form.resetFields(); } catch (error) { reportError(error, '方案创建失败'); } finally { setBusy(false); }
  };
  const commitTransition = async () => {
    if (!pending) return;
    setBusy(true);
    try {
      await transition(pending.scenario.id, pending.status, pending.scenario.version, note);
      message.success(pending.status === 'approved' ? '方案已批准，批准时参数已写入版本留痕，可发起推演' : pending.status === 'draft' ? '方案已驳回草稿，驳回前后参数均可追溯' : pending.status === 'archived' ? '方案已归档，全部历史版本仍可追溯' : '方案已提交复核，提交时参数已冻结留痕');
      setPending(null); setNote('');
    } catch (error) { reportError(error, '方案状态更新失败'); } finally { setBusy(false); }
  };
  const openHistory = (scenario: FanScenario) => {
    setHistoryTarget(scenario);
    setHistoryOpen(true);
  };
  const actionButtons = (scenario: FanScenario) => {
    const buttons = [
      <Button key="history" size="small" icon={<History size={15} />} onClick={() => openHistory(scenario)}>版本留痕</Button>,
    ];
    if (scenario.scenario_status === 'draft' && canCreate) {
      buttons.push(<Button key="edit" size="small" icon={<PencilLine size={15} />} onClick={() => setEditTarget(scenario)}>修订参数</Button>);
      buttons.push(<Button key="submit" size="small" type="primary" icon={<Send size={15} />} onClick={() => setPending({ scenario, status: 'pending_review' })}>提交复核</Button>);
      return <div className="table-actions">{buttons}</div>;
    }
    if (scenario.scenario_status === 'pending_review' && hasRole('reviewer', 'admin')) {
      buttons.push(<Button key="reject" size="small" icon={<Undo2 size={15} />} onClick={() => setPending({ scenario, status: 'draft' })}>驳回</Button>);
      buttons.push(<Button key="approve" size="small" type="primary" icon={<CheckCheck size={15} />} onClick={() => setPending({ scenario, status: 'approved' })}>批准</Button>);
      return <div className="table-actions">{buttons}</div>;
    }
    if (scenario.scenario_status === 'approved' && hasRole('admin')) {
      buttons.push(<Button key="archive" size="small" onClick={() => setPending({ scenario, status: 'archived' })}>归档</Button>);
      return <div className="table-actions">{buttons}</div>;
    }
    return <div className="table-actions">{buttons}<span className="muted">无待办动作</span></div>;
  };
  const columns: ColumnsType<FanScenario> = [
    { title: '方案', width: 240, render: (_, row) => <div className="primary-cell"><strong>{row.name}</strong><span>v{row.version} · {row.operating_mode}{row.reject_reason ? ` · 最近驳回：${row.reject_reason}` : ''}</span></div> },
    { title: '状态', dataIndex: 'scenario_status', width: 120, render: (value) => <StatusBadge status={value} /> },
    { title: '收敛阈值', dataIndex: 'solver_tolerance', width: 110, render: (value) => formatNumber(value, 4) },
    { title: '迭代上限', dataIndex: 'max_iterations', width: 100 },
    { title: '最近更新', dataIndex: 'updated_at', width: 170, render: formatDateTime },
    { title: '复核动作', width: 330, render: (_, row) => actionButtons(row) },
  ];
  const compared = scenarios.filter((item) => selectedKeys.includes(item.id));
  return (
    <div className="page">
      <PageHeader eyebrow="方案控制 / 人工审批" title="风机方案版本" meta={<><span>{scenarios.length} 个方案</span><span>{edges.filter((edge) => edge.enabled).length} 条启用巷道作为当前网络上下文</span><span>当前角色：{user?.role}</span></>} actions={canCreate ? <Button type="primary" icon={<Plus size={17} />} onClick={() => setCreateOpen(true)}>新建方案</Button> : undefined} />
      <section className="workspace-section">
        <div className="section-heading"><div><span className="section-index">01</span><h2>审批队列</h2></div><span>每次提交、驳回、批准、归档都保留当时曲线/模式/阈值/迭代上限；勾选两个方案可横向对比当前参数</span></div>
        <Table rowKey="id" columns={columns} dataSource={scenarios} loading={loading} size="small" scroll={{ x: 1080 }} rowSelection={{ selectedRowKeys: selectedKeys, onChange: (keys) => setSelectedKeys(keys.slice(-2)), getCheckboxProps: (record) => ({ disabled: selectedKeys.length >= 2 && !selectedKeys.includes(record.id) }) }} />
      </section>
      {compared.length === 2 && <section className="workspace-section compare-strip" aria-labelledby="compare-heading"><div className="section-heading"><div><GitCompareArrows size={19} /><h2 id="compare-heading">当前参数对比</h2></div><Button type="text" onClick={() => setSelectedKeys([])}>清除选择</Button></div><div className="compare-grid">{compared.map((item) => <article key={item.id}><StatusBadge status={item.scenario_status} /><h3>{item.name}</h3><dl><div><dt>版本</dt><dd>v{item.version}</dd></div><div><dt>阈值</dt><dd>{item.solver_tolerance}</dd></div><div><dt>迭代上限</dt><dd>{item.max_iterations}</dd></div><div><dt>模式</dt><dd>{item.operating_mode}</dd></div></dl><Button size="small" icon={<History size={14} />} onClick={() => openHistory(item)}>查看该方案历史版本</Button></article>)}</div></section>}
      <Modal title="新建风机方案草稿" open={createOpen} onCancel={() => setCreateOpen(false)} footer={null} width={680} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={submitCreate} initialValues={{ operating_mode: 'normal', solver_tolerance: 0.02, max_iterations: 100, pressure_zero: 1450, flow_design: 30, pressure_design: 1180, flow_max: 60, pressure_max: 720 }}>
          <Form.Item name="name" label="方案名称" rules={[{ required: true, min: 2, message: '请输入至少 2 个字符的方案名称' }]}><Input /></Form.Item>
          <Form.Item name="description" label="适用边界与说明" rules={[{ required: true, min: 4, message: '请说明方案边界' }]}><Input.TextArea rows={3} /></Form.Item>
          <div className="form-grid"><Form.Item name="operating_mode" label="运行模式" rules={[{ required: true }]}><Select options={[{ value: 'normal', label: '常规' }, { value: 'reduced', label: '降载' }, { value: 'emergency_test', label: '应急测试' }]} /></Form.Item><Form.Item name="max_iterations" label="迭代上限" rules={[{ required: true }]}><InputNumber min={10} max={500} /></Form.Item></div>
          <Form.Item name="solver_tolerance" label="残差阈值" rules={[{ required: true }]}><InputNumber min={0.0001} max={10} step={0.01} /></Form.Item>
          <fieldset><legend>风机曲线三点</legend><div className="curve-grid"><Form.Item name="pressure_zero" label="零流量压力 Pa"><InputNumber min={0} /></Form.Item><Form.Item name="flow_design" label="设计流量 m³/s"><InputNumber min={1} /></Form.Item><Form.Item name="pressure_design" label="设计压力 Pa"><InputNumber min={0} /></Form.Item><Form.Item name="flow_max" label="最大流量 m³/s"><InputNumber min={2} /></Form.Item><Form.Item name="pressure_max" label="末端压力 Pa"><InputNumber min={0} /></Form.Item></div></fieldset>
          <div className="modal-actions"><Button onClick={() => setCreateOpen(false)}>保留当前列表</Button><Button type="primary" htmlType="submit" loading={busy}>创建草稿</Button></div>
        </Form>
      </Modal>
      <EditDraftModal scenario={editTarget} open={Boolean(editTarget)} busy={busy} onBusyChange={setBusy} onClose={() => setEditTarget(null)} onSaved={() => { void refresh(); }} />
      <ScenarioHistoryDrawer scenario={historyTarget} open={historyOpen} canEditDraft={canCreate} onClose={() => setHistoryOpen(false)} onChanged={() => { void refresh(); }} />
      <ConfirmActionDialog open={Boolean(pending)} title={pending?.status === 'approved' ? '批准风机方案' : pending?.status === 'draft' ? '驳回方案至草稿' : pending?.status === 'archived' ? '归档已批准方案' : '提交方案复核'} consequence={pending?.status === 'approved' ? '批准后该版本曲线、运行模式、计算阈值与迭代上限写入不可变留痕，可用于离线推演。' : pending?.status === 'draft' ? '方案将回到草稿，驳回原因和驳回时参数都会保留在版本记录中。' : pending?.status === 'archived' ? '归档后不能再发起新的推演，但全部历史版本快照仍可追溯与对比。' : '提交后工程师不能直接批准；本次提交时的参数将冻结留痕，待审期间不能被旧版本替换。'} confirmLabel={pending?.status === 'approved' ? '批准方案' : pending?.status === 'draft' ? '驳回至草稿' : pending?.status === 'archived' ? '归档方案' : '提交复核'} noteLabel={pending?.status === 'draft' ? '驳回原因' : '操作说明'} note={note} requireNote={pending?.status === 'draft'} busy={busy} onNoteChange={setNote} onCancel={() => { setPending(null); setNote(''); }} onConfirm={() => void commitTransition()} />
    </div>
  );
}
