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

import React from 'react';
import { Avatar, Typography, Table, Tag } from '@douyinfe/semi-ui';
import { IconCoinMoneyStroked } from '@douyinfe/semi-icons';
import { calculateModelPrice } from '../../../../../helpers';

const { Text } = Typography;

function getBestGroup(enableGroups, groupRatio) {
  if (!enableGroups || enableGroups.length === 0) return null;
  let bestGroup = enableGroups[0];
  let bestRatio = groupRatio?.[bestGroup] ?? 1;
  for (const g of enableGroups) {
    const r = groupRatio?.[g] ?? 1;
    if (r < bestRatio) {
      bestRatio = r;
      bestGroup = g;
    }
  }
  return { group: bestGroup, ratio: bestRatio };
}

function describeResolution(resolution, defaultResolution) {
  if (!resolution || resolution.length === 0) {
    return defaultResolution || '768p';
  }
  return resolution.join('/');
}

function describeDuration(duration, defaultDuration) {
  if (!duration || duration.length === 0) {
    const d = defaultDuration || 6;
    return `${d}s`;
  }
  return duration.map((d) => `${d}s`).join('/');
}

function describeRefType(type, t) {
  switch (type) {
    case 'text': return t('文生视频');
    case 'image': return t('参考图片');
    case 'video': return t('有参考视频');
    case 'audio': return t('音频生视频');
    default: return type;
  }
}

function describeAudioCapability(audioOutput) {
  if (!audioOutput) return '';
  switch (audioOutput) {
    case 'none': return '无声';
    case 'voice': return '有声';
    case 'voice_timbre': return '有声（有音色）';
    default: return audioOutput;
  }
}

const ModelPricingRules = ({
  modelData,
  groupRatio,
  currency,
  siteDisplayType,
  tokenUnit,
  displayPrice,
  usableGroup,
  t,
}) => {
  if (!modelData) return null;

  const bestGroup = getBestGroup(
    modelData.enable_groups,
    groupRatio,
  );
  const effectiveGroup = bestGroup?.group ?? null;
  const effectiveRatio = bestGroup?.ratio ?? 1;

  const isTokenBased = modelData.quota_type === 0;
  const isDynamic = modelData.billing_mode === 'tiered_expr';
  const ratioLabel = tokenUnit === 'K' ? '1K' : '1M';

  const rawParams = [
    { label: t('计费类型'), value: isDynamic ? t('动态计费') : isTokenBased ? t('按量计费') : t('按次计费') },
    { label: 'ModelRatio', value: String(modelData.model_ratio) },
    { label: 'CompletionRatio', value: String(modelData.completion_ratio) },
    { label: 'CacheRatio', value: modelData.cache_ratio != null ? String(modelData.cache_ratio) : '-' },
    { label: t('写入缓存倍率'), value: modelData.create_cache_ratio != null ? String(modelData.create_cache_ratio) : '-' },
    { label: 'ImageRatio', value: modelData.image_ratio != null ? String(modelData.image_ratio) : '-' },
    { label: 'AudioRatio', value: modelData.audio_ratio != null ? String(modelData.audio_ratio) : '-' },
    { label: t('音频输出倍率'), value: modelData.audio_completion_ratio != null ? String(modelData.audio_completion_ratio) : '-' },
  ];

  const extraPriceTypes = [];
  if (modelData.cache_ratio != null) extraPriceTypes.push({ label: t('缓存'), type: 'cache' });
  if (modelData.create_cache_ratio != null) extraPriceTypes.push({ label: t('写入缓存'), type: 'create_cache' });
  if (modelData.image_ratio != null) extraPriceTypes.push({ label: t('图片'), type: 'image' });
  if (modelData.audio_ratio != null) extraPriceTypes.push({ label: t('音频输入'), type: 'audio_input' });
  if (modelData.audio_ratio != null && modelData.audio_completion_ratio != null) {
    extraPriceTypes.push({ label: t('音频输出'), type: 'audio_output' });
  }

  const effectivePriceTypes = [
    { label: t('输入'), type: 'input' },
    { label: t('输出'), type: 'output' },
    ...extraPriceTypes,
  ];

  const getPriceForType = (priceData, type) => {
    switch (type) {
      case 'input': return priceData.inputPrice;
      case 'output': return priceData.completionPrice;
      case 'cache': return priceData.cachePrice;
      case 'create_cache': return priceData.createCachePrice;
      case 'image': return priceData.imagePrice;
      case 'audio_input': return priceData.audioInputPrice;
      case 'audio_output': return priceData.audioOutputPrice;
      default: return '-';
    }
  };

  const videoPricing = modelData.video_pricing;
  const hasVideoPricing = videoPricing?.rules?.length > 0;
  const hasDurationConditions = videoPricing?.rules?.some(
    (r) => r.conditions?.duration?.length > 0,
  );
  const isPerSecond = videoPricing?.billing_mode === 'per_second';
  const showDurationColumn = !isPerSecond || hasDurationConditions;

  const rawColumns = [
    { title: t('参数'), dataIndex: 'label', width: 200 },
    { title: t('值'), dataIndex: 'value' },
  ];
  const rawData = rawParams.map((p) => ({ key: p.label, label: p.label, value: p.value }));

  const priceColumns = [
    { title: t('价格类型'), dataIndex: 'type', width: 200 },
    { title: `${t('价格')} / ${ratioLabel}`, dataIndex: 'price', align: 'right' },
  ];
  const priceRows = [];

  if (isTokenBased && bestGroup) {
    const priceData = calculateModelPrice({
      record: modelData,
      selectedGroup: effectiveGroup,
      groupRatio,
      tokenUnit,
      displayPrice,
      currency,
      quotaDisplayType: siteDisplayType,
    });
    effectivePriceTypes.forEach((pt) => {
      priceRows.push({
        key: pt.type,
        type: pt.label,
        price: getPriceForType(priceData, pt.type) || '-',
      });
    });
  } else if (!isTokenBased && bestGroup) {
    const priceData = calculateModelPrice({
      record: modelData,
      selectedGroup: effectiveGroup,
      groupRatio,
      tokenUnit,
      displayPrice,
      currency,
      quotaDisplayType: siteDisplayType,
    });
    priceRows.push({
      key: 'request',
      type: t('按次计费'),
      price: priceData.price || '-',
    });
  }

  const videoColumns = [
    {
      title: t('属性'),
      dataIndex: 'capability',
      width: 140,
      render: (caps) => {
        if (!caps) return '-';
        return caps;
      },
    },
    { title: t('清晰度'), dataIndex: 'resolution', width: 100 },
    ...(showDurationColumn ? [{ title: t('时长'), dataIndex: 'duration', width: 100 }] : []),
    { title: t('价格'), dataIndex: 'price', align: 'right' },
  ];
  const videoData = (videoPricing?.rules || []).map((rule, idx) => {
    const factor = effectiveGroup ? effectiveRatio : 1;
    const refs = rule.conditions?.reference_types;
    const hasRef = refs && refs.length > 0;
    const refParts = hasRef ? refs.map((r) => describeRefType(r, t)) : [];
    const audioPart = describeAudioCapability(rule.conditions?.audio_output);
    const hasAnyVideoRef = videoPricing?.rules?.some(
      (r) => r.conditions?.reference_types?.includes('video'),
    );
    let capability;
    if (refParts.length > 0 && audioPart) {
      capability = refParts.join('/') + '+' + audioPart;
    } else if (refParts.length > 0) {
      capability = refParts.join('/');
    } else if (audioPart) {
      if (hasAnyVideoRef && !refs) {
        capability = '无参考视频+' + audioPart;
      } else {
        capability = audioPart;
      }
    } else if (hasAnyVideoRef && !refs) {
      capability = '无参考视频';
    } else {
      capability = '-';
    }
    const pn = videoPricing.billing_mode;
    return {
      key: idx,
      capability,
      resolution: describeResolution(rule.conditions.resolution, videoPricing.default_resolution),
      duration: describeDuration(rule.conditions.duration, videoPricing.default_duration),
      price: pn === 'per_call'
        ? `${displayPrice(rule.price * factor)}/${t('次')}`
        : pn === 'per_second'
          ? `${displayPrice(rule.price * factor)}/${t('秒')}`
          : displayPrice(rule.price * factor),
    };
  });

  return (
    <div>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='orange' className='mr-2 shadow-md'>
          <IconCoinMoneyStroked size={16} />
        </Avatar>
        <div>
          <Text className='text-lg font-medium'>{t('价格规则')}</Text>
          <div className='text-xs text-gray-600'>
            {t('模型定价参数与倍率信息')}
          </div>
        </div>
      </div>

      <div className='space-y-6'>
        {/* 表1：定价参数（视频模型不展示 token 倍率参数） */}
        {!hasVideoPricing && <div>
          <Text className='block mb-2 text-sm font-medium'>
            {t('定价参数')}
          </Text>
          <Table
            dataSource={rawData}
            columns={rawColumns}
            pagination={false}
            size='small'
            bordered={false}
          />
        </div>}

        {/* 表2：有效价格（有视频定价时隐藏，改用视频表展示） */}
        {!hasVideoPricing && bestGroup && priceRows.length > 0 && (
          <div>
            <Text className='block mb-2 text-sm font-medium'>
              {t('有效价格')}
              <span className='ml-2 text-xs font-normal text-gray-500'>
                {t('最优分组')}: <span className='font-mono'>{effectiveGroup}</span> ({effectiveRatio}x)
              </span>
            </Text>
            <Table
              dataSource={priceRows}
              columns={priceColumns}
              pagination={false}
              size='small'
              bordered={false}
            />
          </div>
        )}

        {/* 表3：视频定价（显示为有效价格） */}
        {hasVideoPricing && (
          <div>
            <Text className='block mb-2 text-sm font-medium'>
              {t('有效价格')}
              {effectiveGroup && (
                <span className='ml-2 text-xs font-normal text-gray-500'>
                  {t('最优分组')}: <span className='font-mono'>{effectiveGroup}</span> ({effectiveRatio}x)
                </span>
              )}
            </Text>
            <Table
              dataSource={videoData}
              columns={videoColumns}
              pagination={false}
              size='small'
              bordered={false}
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default ModelPricingRules;
