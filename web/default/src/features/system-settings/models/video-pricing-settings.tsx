/*
Copyright (C) 2023-2026 QuantumNous

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
import { memo, useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Code2, Eye, AlertCircle } from 'lucide-react'
import { Textarea } from '@/components/ui/textarea'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  formatJsonForTextarea,
  normalizeJsonString,
  validateJsonString,
} from './utils'

const OPTION_KEY = 'video_pricing_config'

const DEFAULT_VIDEO_PRICING = `{
  "models": {
    "MiniMax-Hailuo-02": {
      "billing_mode": "per_call",
      "base_price": 2.00,
      "markup": 1.15,
      "pricing_rules": [
        {
          "conditions": {
            "resolution": ["512p"],
            "duration": [6],
            "reference_types": ["image"]
          },
          "price": 0.60
        },
        {
          "conditions": {
            "resolution": ["512p"],
            "duration": [10],
            "reference_types": ["image"]
          },
          "price": 1.00
        },
        {
          "conditions": {
            "resolution": ["768p"],
            "duration": [6, 10],
            "reference_types": ["image", "video"]
          },
          "price": 2.00
        },
        {
          "conditions": {
            "resolution": ["1080p"],
            "duration": [6],
            "reference_types": ["image", "video"]
          },
          "price": 3.50
        }
      ]
    }
  }
}`

type VideoPricingSettingsProps = {
  defaultValue: string
}

export const VideoPricingSettings = memo(function VideoPricingSettings({
  defaultValue,
}: VideoPricingSettingsProps) {
  const { t } = useTranslation()
  const [editMode, setEditMode] = useState<'visual' | 'json'>('visual')
  const [jsonValue, setJsonValue] = useState(() =>
    formatJsonForTextarea(defaultValue || DEFAULT_VIDEO_PRICING)
  )
  const [jsonError, setJsonError] = useState<string | null>(null)

  const { mutate: updateOption, isPending: isUpdating } = useUpdateOption()

  useEffect(() => {
    const initValue = formatJsonForTextarea(defaultValue || DEFAULT_VIDEO_PRICING)
    setJsonValue(initValue)
  }, [defaultValue])

  const handleJsonChange = useCallback(
    (value: string) => {
      setJsonValue(value)
      const validation = validateJsonString(value)
      if (!validation.valid) {
        setJsonError(validation.message || 'Invalid JSON')
      } else {
        setJsonError(null)
      }
    },
    []
  )

  const handleSave = useCallback(() => {
    const normalized = normalizeJsonString(jsonValue)
    const validation = validateJsonString(normalized)
    if (!validation.valid) {
      toast.error(t('Invalid JSON format'))
      return
    }

    updateOption(
      { key: OPTION_KEY, value: normalized },
      {
        onSuccess: () => {
          toast.success(t('Video pricing config saved successfully'))
        },
        onError: () => {
          toast.error(t('Failed to save video pricing config'))
        },
      }
    )
  }, [jsonValue, updateOption, t])

  const handleReset = useCallback(() => {
    setJsonValue(formatJsonForTextarea(DEFAULT_VIDEO_PRICING))
    setJsonError(null)
  }, [])

  const toggleEditMode = useCallback(() => {
    setEditMode((prev) => (prev === 'visual' ? 'json' : 'visual'))
  }, [])

  return (
    <div className='space-y-6'>
      <div className='flex justify-between items-center'>
        <div className='text-sm text-muted-foreground'>
          {t('Configure video model pricing by channel, resolution, duration, and reference type')}
        </div>
        <Button variant='outline' size='sm' onClick={toggleEditMode}>
          {editMode === 'visual' ? (
            <>
              <Code2 className='mr-2 w-4 h-4' />
              {t('Switch to JSON')}
            </>
          ) : (
            <>
              <Eye className='mr-2 w-4 h-4' />
              {t('Switch to Visual')}
            </>
          )}
        </Button>
      </div>

      {editMode === 'visual' ? (
        <div className='space-y-4'>
          <Alert>
            <AlertDescription>
              {t('Video pricing configuration accepts JSON format. Click "Switch to JSON" to edit directly, or configure using the form below (coming soon).')}
            </AlertDescription>
          </Alert>

          <Textarea
            rows={20}
            value={jsonValue}
            onChange={(e) => handleJsonChange(e.target.value)}
            className='font-mono text-sm'
            placeholder={DEFAULT_VIDEO_PRICING}
          />

          {jsonError && (
            <Alert variant='destructive'>
              <AlertCircle className='w-4 h-4' />
              <AlertDescription>{jsonError}</AlertDescription>
            </Alert>
          )}

          <div className='flex flex-wrap gap-4'>
            <Button onClick={handleSave} disabled={isUpdating || !!jsonError}>
              {isUpdating ? t('Saving...') : t('Save video pricing')}
            </Button>
            <Button
              type='button'
              variant='destructive'
              onClick={handleReset}
              disabled={isUpdating}
            >
              {t('Reset to defaults')}
            </Button>
          </div>
        </div>
      ) : (
        <div className='space-y-4'>
          <Textarea
            rows={20}
            value={jsonValue}
            onChange={(e) => handleJsonChange(e.target.value)}
            className='font-mono text-sm'
            placeholder={DEFAULT_VIDEO_PRICING}
          />

          {jsonError && (
            <Alert variant='destructive'>
              <AlertCircle className='w-4 h-4' />
              <AlertDescription>{jsonError}</AlertDescription>
            </Alert>
          )}

          <div className='flex flex-wrap gap-4'>
            <Button onClick={handleSave} disabled={isUpdating || !!jsonError}>
              {isUpdating ? t('Saving...') : t('Save video pricing')}
            </Button>
            <Button
              type='button'
              variant='destructive'
              onClick={handleReset}
              disabled={isUpdating}
            >
              {t('Reset to defaults')}
            </Button>
          </div>
        </div>
      )}
    </div>
  )
})
