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
import React, { useEffect, useState } from 'react';
import {
  Banner,
  Button,
  Radio,
  RadioGroup,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { IconCopy } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, copy, showError, showSuccess } from '../../../helpers';

const { Text } = Typography;

const OPTION_KEY = 'video_pricing_config';

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

export default function VideoPricingSettings({ options }) {
  const { t } = useTranslation();
  const [jsonText, setJsonText] = useState('');
  const [jsonError, setJsonError] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let config = {};
    try {
      const raw = options?.[OPTION_KEY];
      if (raw) {
        config = typeof raw === 'string' ? JSON.parse(raw) : raw;
      }
    } catch {
      config = {};
    }

    if (!config || Object.keys(config).length === 0) {
      config = { ...DEFAULT_VIDEO_PRICING };
    }

    setJsonText(JSON.stringify(config, null, 2));
  }, [options]);

  const validateJson = (text) => {
    try {
      const parsed = JSON.parse(text);
      if (typeof parsed !== 'object' || Array.isArray(parsed) || parsed === null) {
        return { valid: false, message: t('JSON 必须是对象') };
      }
      return { valid: true, message: '' };
    } catch (e) {
      return { valid: false, message: e.message };
    }
  };

  const handleTextChange = (text) => {
    setJsonText(text);
    const validation = validateJson(text);
    setJsonError(validation.valid ? '' : validation.message);
  };

  const resetToDefault = () => {
    setJsonText(JSON.stringify(DEFAULT_VIDEO_PRICING, null, 2));
    setJsonError('');
  };

  const handleSave = async () => {
    const validation = validateJson(jsonText);
    if (!validation.valid) {
      setJsonError(validation.message);
      return;
    }

    setSaving(true);
    try {
      const res = await API.put('/api/option/', {
        key: OPTION_KEY,
        value: jsonText,
      });
      if (res.data.success) {
        showSuccess(t('保存成功'));
      } else {
        showError(res.data.message || t('保存失败'));
      }
    } catch (e) {
      showError(e.message);
    } finally {
      setSaving(false);
    }
  };

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

      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 16 }}>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button
            icon={<IconCopy />}
            size='small'
            theme='borderless'
            onClick={() => {
              copy(jsonText, t('JSON'));
            }}
          >
            {t('复制')}
          </Button>
          <Button size='small' theme='borderless' onClick={resetToDefault}>
            {t('恢复默认')}
          </Button>
        </div>

        <Button
          theme='solid'
          type='primary'
          loading={saving}
          disabled={!!jsonError}
          onClick={handleSave}
        >
          {t('保存')}
        </Button>
      </div>
    </div>
  );
}
