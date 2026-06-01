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
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { useStatus } from '@/hooks/use-status'
import { Markdown } from '@/components/ui/markdown'
import { PublicLayout } from '@/components/layout'
import { Footer } from '@/components/layout/components/footer'
import { useTheme } from '@/context/theme-provider'
import { isRegisterEnabled } from '@/features/auth'
import { CTA, Features, Hero, HowItWorks, Stats } from './components'
import { useHomePageContent } from './hooks'

export function Home() {
  const { t, i18n } = useTranslation()
  const { auth } = useAuthStore()
  const isAuthenticated = !!auth.user
  const { status } = useStatus()
  const showRegisterEntry = !!status && isRegisterEnabled(status)
  const { content, isLoaded, isUrl } = useHomePageContent()
  const { resolvedTheme } = useTheme()
  const iframeRef = useRef<HTMLIFrameElement | null>(null)

  useEffect(() => {
    if (!isUrl) return

    const postFrameState = () => {
      iframeRef.current?.contentWindow?.postMessage(
        {
          themeMode: resolvedTheme,
          lang: i18n.language,
        },
        '*'
      )
    }

    postFrameState()

    const iframe = iframeRef.current
    if (!iframe) return
    const handleMessage = (event: MessageEvent) => {
      if (event.source !== iframe.contentWindow) return
      if (event.data?.type !== 'mmodelindex-ready') return
      postFrameState()
    }

    iframe.addEventListener('load', postFrameState)
    window.addEventListener('message', handleMessage)
    return () => {
      iframe.removeEventListener('load', postFrameState)
      window.removeEventListener('message', handleMessage)
    }
  }, [content, i18n.language, isUrl, resolvedTheme])

  if (!isLoaded) {
    return (
      <PublicLayout showMainContainer={false}>
        <main className='flex min-h-screen items-center justify-center'>
          <div className='text-muted-foreground'>{t('Loading...')}</div>
        </main>
      </PublicLayout>
    )
  }

  if (content) {
    return (
      <PublicLayout
        showMainContainer={false}
        headerProps={{ alwaysElevated: true }}
      >
        <main className='overflow-x-hidden'>
          {isUrl ? (
            <iframe
              ref={iframeRef}
              src={content}
              className='h-screen w-full border-none'
              title={t('Custom Home Page')}
            />
          ) : (
            <div className='container mx-auto py-8'>
              <Markdown className='custom-home-content'>{content}</Markdown>
            </div>
          )}
        </main>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <Hero
        isAuthenticated={isAuthenticated}
        showRegisterEntry={showRegisterEntry}
      />
      <Stats />
      <Features />
      <HowItWorks />
      <CTA
        isAuthenticated={isAuthenticated}
        showRegisterEntry={showRegisterEntry}
      />
      <Footer />
    </PublicLayout>
  )
}
