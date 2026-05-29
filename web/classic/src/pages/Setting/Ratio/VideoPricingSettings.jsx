/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Collapse,
  Input,
  Modal,
  Popconfirm,
  Radio,
  RadioGroup,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { IconCopy, IconDelete, IconPlus } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, copy, showError, showSuccess } from '../../../helpers';

const { Text } = Typography;

const OPTION_KEY = 'video_pricing_config';

const EMPTY_MODEL_TEMPLATE = {
  billing_mode: 'per_call',
  base_price: 1.0,
  markup: 1.0,
  pricing_rules: [
    {
      conditions: {
        resolution: ['768p'],
        duration: [6],
      },
      price: 1.0,
    },
  ],
};

const DEFAULT_VIDEO_PRICING = {
  models: {
    'MiniMax-Hailuo-02': {
      billing_mode: 'per_call',
      base_price: 2.0,
      markup: 1.15,
      pricing_rules: [
        {
          conditions: {
            resolution: ['512p'],
            duration: [6],
            reference_types: ['image'],
          },
          price: 0.6,
        },
        {
          conditions: {
            resolution: ['512p'],
            duration: [10],
            reference_types: ['image'],
          },
          price: 1.0,
        },
        {
          conditions: {
            resolution: ['768p'],
            duration: [6, 10],
            reference_types: ['image', 'video'],
          },
          price: 2.0,
        },
        {
          conditions: {
            resolution: ['1080p'],
            duration: [6],
            reference_types: ['image', 'video'],
          },
          price: 3.5,
        },
      ],
    },
  },
};

function parseRawConfig(raw) {
  let config = { ...DEFAULT_VIDEO_PRICING };
  try {
    if (raw) {
      const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw;
      if (parsed && parsed.models && Object.keys(parsed.models).length > 0) {
        config = parsed;
      }
    }
  } catch {
    config = { ...DEFAULT_VIDEO_PRICING };
  }

  const models = config.models || {};
  const modelTexts = {};
  const modelErrors = {};
  for (const [name, modelConfig] of Object.entries(models)) {
    modelTexts[name] = JSON.stringify(modelConfig, null, 2);
    modelErrors[name] = '';
  }

  return { config, models, modelTexts, modelErrors };
}

function formatModelJson(obj) {
  return JSON.stringify(obj, null, 2);
}

function validateJsonText(text) {
  try {
    const parsed = JSON.parse(text);
    if (typeof parsed !== 'object' || Array.isArray(parsed) || parsed === null) {
      return { valid: false, message: 'JSON 必须是对象' };
    }
    return { valid: true, message: '' };
  } catch (e) {
    return { valid: false, message: e.message };
  }
}

export default function VideoPricingSettings({ options }) {
  const { t } = useTranslation();
  const [editMode, setEditMode] = useState('per-model');
  const [jsonText, setJsonText] = useState('');
  const [jsonError, setJsonError] = useState('');
  const [saving, setSaving] = useState(false);

  const [models, setModels] = useState({});
  const [modelTexts, setModelTexts] = useState({});
  const [modelErrors, setModelErrors] = useState({});
  const [activeKeys, setActiveKeys] = useState([]);

  const [showAddModal, setShowAddModal] = useState(false);
  const [newModelName, setNewModelName] = useState('');

  useEffect(() => {
    const raw = options?.[OPTION_KEY];
    const parsed = parseRawConfig(raw);
    setModels(parsed.models);
    setModelTexts(parsed.modelTexts);
    setModelErrors(parsed.modelErrors);
    setJsonText(formatModelJson(parsed.config));
    setJsonError('');
  }, [options]);

  const modelNames = useMemo(
    () => Object.keys(models).sort((a, b) => a.localeCompare(b)),
    [models],
  );

  const hasAnyModelError = useMemo(
    () => Object.values(modelErrors).some((e) => e),
    [modelErrors],
  );

  const handleModelTextChange = useCallback(
    (modelName, text) => {
      setModelTexts((prev) => ({ ...prev, [modelName]: text }));
      const validation = validateJsonText(text);
      setModelErrors((prev) => ({
        ...prev,
        [modelName]: validation.valid ? '' : validation.message,
      }));
    },
    [],
  );

  const assembleFullJson = useCallback(
    () => {
      const assembled = { models: {} };
      for (const name of modelNames) {
        try {
          assembled.models[name] = JSON.parse(modelTexts[name]);
        } catch {
          assembled.models[name] = {};
        }
      }
      return assembled;
    },
    [modelNames, modelTexts],
  );

  const handleAddModel = useCallback(
    () => {
      const name = newModelName.trim();
      if (!name) {
        showError(t('请输入模型名称'));
        return;
      }
      if (models[name]) {
        showError(t('模型已存在'));
        return;
      }
      const template = { ...EMPTY_MODEL_TEMPLATE };
      setModels((prev) => ({ ...prev, [name]: template }));
      setModelTexts((prev) => ({
        ...prev,
        [name]: formatModelJson(template),
      }));
      setModelErrors((prev) => ({ ...prev, [name]: '' }));
      setNewModelName('');
      setShowAddModal(false);
      setActiveKeys((prev) => [...prev, name]);
    },
    [newModelName, models, t],
  );

  const handleDeleteModel = useCallback(
    (modelName) => {
      setModels((prev) => {
        const next = { ...prev };
        delete next[modelName];
        return next;
      });
      setModelTexts((prev) => {
        const next = { ...prev };
        delete next[modelName];
        return next;
      });
      setModelErrors((prev) => {
        const next = { ...prev };
        delete next[modelName];
        return next;
      });
      setActiveKeys((prev) => prev.filter((k) => k !== modelName));
    },
    [],
  );

  const handleToggleMode = useCallback(
    (mode) => {
      if (mode === 'raw-json') {
        const assembled = assembleFullJson();
        setJsonText(formatModelJson(assembled));
        setJsonError('');
      } else {
        const validation = validateJsonText(jsonText);
        if (!validation.valid) {
          setJsonError(validation.message);
          return;
        }
        const parsed = parseRawConfig(jsonText);
        setModels(parsed.models);
        setModelTexts(parsed.modelTexts);
        setModelErrors(parsed.modelErrors);
        setJsonError('');
      }
      setEditMode(mode);
    },
    [jsonText, assembleFullJson],
  );

  const handleSave = useCallback(
    async () => {
      let finalJson;

      if (editMode === 'per-model') {
        if (hasAnyModelError) {
          showError(t('部分模型 JSON 格式错误，请修正后保存'));
          return;
        }
        const assembled = assembleFullJson();
        finalJson = formatModelJson(assembled);
      } else {
        const validation = validateJsonText(jsonText);
        if (!validation.valid) {
          setJsonError(validation.message);
          return;
        }
        finalJson = jsonText;
      }

      const saveValidation = validateJsonText(finalJson);
      if (!saveValidation.valid) {
        showError(t('JSON 格式错误'));
        return;
      }

      setSaving(true);
      try {
        const res = await API.put('/api/option/', {
          key: OPTION_KEY,
          value: finalJson,
        });
        if (res.data.success) {
          showSuccess(t('保存成功'));

          const parsed = parseRawConfig(finalJson);
          setModels(parsed.models);
          setModelTexts(parsed.modelTexts);
          setModelErrors(parsed.modelErrors);
        } else {
          showError(res.data.message || t('保存失败'));
        }
      } catch (e) {
        showError(e.message);
      } finally {
        setSaving(false);
      }
    },
    [editMode, jsonText, hasAnyModelError, assembleFullJson, t],
  );

  const handleTextChange = useCallback(
    (text) => {
      setJsonText(text);
      const validation = validateJsonText(text);
      setJsonError(validation.valid ? '' : validation.message);
    },
    [],
  );

  const handleReset = useCallback(
    () => {
      const parsed = parseRawConfig(JSON.stringify(DEFAULT_VIDEO_PRICING));
      setModels(parsed.models);
      setModelTexts(parsed.modelTexts);
      setModelErrors(parsed.modelErrors);
      setJsonText(formatModelJson(DEFAULT_VIDEO_PRICING));
      setJsonError('');
      setActiveKeys([]);
    },
    [],
  );

  return (
    <div style={{ maxWidth: 800 }}>
      <Banner
        type='info'
        description={
          <>
            <div>{t('配置视频模型的多维度定价策略，支持按分辨率、时长、参考类型等维度设置价格。')}</div>
            <div style={{ marginTop: 4 }}>
              <Text strong>{t('配置说明')}：</Text>
              <ul style={{ margin: '8px 0 0 20px', padding: 0 }}>
                <li>{t('billing_mode: per_call（按次计费）或 per_token（按token计费）')}</li>
                <li>{t('base_price: 默认基础价格')}</li>
                <li>{t('markup: 价格上浮比例（如 1.15 表示上浮15%）')}</li>
                <li>{t('pricing_rules: 定价规则列表，按条件匹配价格')}</li>
              </ul>
            </div>
          </>
        }
        style={{ marginBottom: 16 }}
      />

      <RadioGroup
        value={editMode}
        onChange={(e) => handleToggleMode(e.target.value)}
        style={{ marginBottom: 16 }}
      >
        <Radio value='per-model'>{t('按模型编辑')}</Radio>
        <Radio value='raw-json'>{t('原始JSON')}</Radio>
      </RadioGroup>

      {editMode === 'per-model' ? (
        <div>
          {modelNames.length === 0 ? (
            <Text type='tertiary'>{t('暂未配置视频定价模型')}</Text>
          ) : (
            <Collapse
              activeKey={activeKeys}
              onChange={(keys) => setActiveKeys(Array.isArray(keys) ? keys : [keys])}
              accordion
            >
              {modelNames.map((name) => (
                <Collapse.Panel
                  key={name}
                  itemKey={name}
                  header={
                    <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Text strong>{name}</Text>
                    </span>
                  }
                  extra={
                    <div onClick={(e) => e.stopPropagation()}>
                      <Popconfirm
                        title={t('确认删除该模型定价配置？')}
                        onConfirm={() => handleDeleteModel(name)}
                        okText={t('删除')}
                        cancelText={t('取消')}
                      >
                        <Button
                          icon={<IconDelete />}
                          type='danger'
                          size='small'
                          theme='borderless'
                        />
                      </Popconfirm>
                    </div>
                  }
                >
                  <TextArea
                    value={modelTexts[name]}
                    onChange={(text) => handleModelTextChange(name, text)}
                    autosize={{ minRows: 6, maxRows: 25 }}
                    style={{ fontFamily: 'monospace', fontSize: 13 }}
                  />
                  {modelErrors[name] && (
                    <Text type='danger' size='small' style={{ display: 'block', marginTop: 8 }}>
                      {modelErrors[name]}
                    </Text>
                  )}
                </Collapse.Panel>
              ))}
            </Collapse>
          )}

          <Button
            icon={<IconPlus />}
            theme='light'
            onClick={() => setShowAddModal(true)}
            style={{ marginTop: 12 }}
          >
            {t('添加模型')}
          </Button>

          <Modal
            title={t('添加模型')}
            visible={showAddModal}
            onOk={handleAddModel}
            onCancel={() => {
              setShowAddModal(false);
              setNewModelName('');
            }}
            okText={t('添加')}
            cancelText={t('取消')}
            size='small'
          >
            <div style={{ padding: '8px 0' }}>
              <Text style={{ display: 'block', marginBottom: 8 }}>{t('模型名称')}</Text>
              <Input
                placeholder={t('输入模型名称，如 kling-3.0')}
                value={newModelName}
                onChange={setNewModelName}
                onEnterPress={handleAddModel}
              />
            </div>
          </Modal>
        </div>
      ) : (
        <div>
          <TextArea
            value={jsonText}
            onChange={handleTextChange}
            autosize={{ minRows: 15, maxRows: 30 }}
            style={{ fontFamily: 'monospace', fontSize: 13 }}
            placeholder={JSON.stringify(DEFAULT_VIDEO_PRICING, null, 2)}
          />

          {jsonError && (
            <Text type='danger' size='small' style={{ display: 'block', marginTop: 8 }}>
              {jsonError}
            </Text>
          )}
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 16 }}>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button
            icon={<IconCopy />}
            size='small'
            theme='borderless'
            onClick={() => {
              const text = editMode === 'per-model'
                ? formatModelJson(assembleFullJson())
                : jsonText;
              copy(text, t('JSON'));
            }}
          >
            {t('复制')}
          </Button>
          <Button size='small' theme='borderless' onClick={handleReset}>
            {t('恢复默认')}
          </Button>
        </div>

        <Button
          theme='solid'
          type='primary'
          loading={saving}
          disabled={editMode === 'per-model' ? hasAnyModelError : !!jsonError}
          onClick={handleSave}
        >
          {t('保存')}
        </Button>
      </div>
    </div>
  );
}
