import { useEffect } from 'react';
import { Button, Form, Input, InputNumber, Modal, Select, message } from 'antd';
import { useScenarioStore } from '../../stores/scenarioStore';
import type { CreateScenarioInput, FanScenario } from '../../types/scenario';
import { reportError } from '../../utils/errors';

interface Props {
  scenario: FanScenario | null;
  open: boolean;
  busy: boolean;
  onBusyChange(busy: boolean): void;
  onClose(): void;
  onSaved(): void;
}

interface DraftFormValues {
  name: string;
  description: string;
  operating_mode: CreateScenarioInput['operating_mode'];
  solver_tolerance: number;
  max_iterations: number;
  pressure_zero: number;
  flow_design: number;
  pressure_design: number;
  flow_max: number;
  pressure_max: number;
}

export function EditDraftModal({ scenario, open, busy, onBusyChange, onClose, onSaved }: Props) {
  const { updateDraft } = useScenarioStore();
  const [form] = Form.useForm<DraftFormValues>();

  useEffect(() => {
    if (!open || !scenario) return;
    const points = scenario.fan_curve_json;
    form.setFieldsValue({
      name: scenario.name,
      description: scenario.description,
      operating_mode: scenario.operating_mode,
      solver_tolerance: scenario.solver_tolerance,
      max_iterations: scenario.max_iterations,
      pressure_zero: points[0]?.pressure_pa ?? 0,
      flow_design: points[1]?.flow_m3s ?? 30,
      pressure_design: points[1]?.pressure_pa ?? 1000,
      flow_max: points[2]?.flow_m3s ?? 60,
      pressure_max: points[2]?.pressure_pa ?? 700,
    });
  }, [open, scenario, form]);

  const submit = async (values: DraftFormValues) => {
    if (!scenario) return;
    onBusyChange(true);
    try {
      await updateDraft(scenario.id, {
        name: values.name,
        description: values.description,
        operating_mode: values.operating_mode,
        solver_tolerance: Number(values.solver_tolerance),
        max_iterations: Number(values.max_iterations),
        fan_curve: [
          { flow_m3s: 0, pressure_pa: Number(values.pressure_zero) },
          { flow_m3s: Number(values.flow_design), pressure_pa: Number(values.pressure_design) },
          { flow_m3s: Number(values.flow_max), pressure_pa: Number(values.pressure_max) },
        ],
        version: scenario.version,
      });
      message.success(`草稿参数已修订，v${scenario.version + 1} 已留痕`);
      onSaved();
      onClose();
    } catch (error) {
      reportError(error, '草稿修订失败');
    } finally {
      onBusyChange(false);
    }
  };

  return (
    <Modal title={scenario ? `修订草稿参数 · ${scenario.name}` : '修订草稿参数'} open={open} onCancel={onClose} footer={null} width={680} destroyOnClose>
      <p className="muted edit-note">仅草稿状态可修订；修订会生成新的版本快照，旧参数保留可查。待复核、已批准和已归档方案的参数锁定，不能被任何版本替换。</p>
      <Form form={form} layout="vertical" onFinish={submit}>
        <Form.Item name="name" label="方案名称" rules={[{ required: true, min: 2, message: '请输入至少 2 个字符的方案名称' }]}><Input /></Form.Item>
        <Form.Item name="description" label="适用边界与说明" rules={[{ required: true, min: 4, message: '请说明方案边界' }]}><Input.TextArea rows={3} /></Form.Item>
        <div className="form-grid">
          <Form.Item name="operating_mode" label="运行模式" rules={[{ required: true }]}>
            <Select options={[{ value: 'normal', label: '常规' }, { value: 'reduced', label: '降载' }, { value: 'emergency_test', label: '应急测试' }]} />
          </Form.Item>
          <Form.Item name="max_iterations" label="迭代上限" rules={[{ required: true }]}><InputNumber min={10} max={500} /></Form.Item>
        </div>
        <Form.Item name="solver_tolerance" label="残差阈值" rules={[{ required: true }]}><InputNumber min={0.0001} max={10} step={0.01} /></Form.Item>
        <fieldset>
          <legend>风机曲线三点</legend>
          <div className="curve-grid">
            <Form.Item name="pressure_zero" label="零流量压力 Pa"><InputNumber min={0} /></Form.Item>
            <Form.Item name="flow_design" label="设计流量 m³/s"><InputNumber min={1} /></Form.Item>
            <Form.Item name="pressure_design" label="设计压力 Pa"><InputNumber min={0} /></Form.Item>
            <Form.Item name="flow_max" label="最大流量 m³/s"><InputNumber min={2} /></Form.Item>
            <Form.Item name="pressure_max" label="末端压力 Pa"><InputNumber min={0} /></Form.Item>
          </div>
        </fieldset>
        <div className="modal-actions">
          <Button onClick={onClose}>放弃修订</Button>
          <Button type="primary" htmlType="submit" loading={busy}>保存为新版本（v{scenario ? scenario.version + 1 : '—'}）</Button>
        </div>
      </Form>
    </Modal>
  );
}
