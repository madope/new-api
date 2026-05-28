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
import { Link, useSearch } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useStatus } from '@/hooks/use-status'
import { TermsFooter } from '../components/terms-footer'
import { UserAuthForm } from './components/user-auth-form'

export function EmbeddedSignIn() {
  const { t } = useTranslation()
  const { redirect } = useSearch({ from: '/(auth)/embedded-sign-in' })
  const { status } = useStatus()

  return (
    <main className='bg-background flex min-h-svh items-center justify-center px-4 py-6 sm:px-6 sm:py-8'>
      <section className='border-border/60 bg-background/95 w-full max-w-[30rem] rounded-[2rem] border p-6 shadow-[0_24px_80px_-32px_rgba(15,23,42,0.24)] backdrop-blur sm:p-8'>
        <div className='w-full space-y-8'>
          <div className='space-y-2'>
            <h2 className='text-center text-2xl font-semibold tracking-tight sm:text-left'>
              {t('Sign in')}
            </h2>
            {!status?.self_use_mode_enabled && (
              <p className='text-muted-foreground text-left text-sm sm:text-base'>
                {t("Don't have an account?")}{' '}
                <Link
                  to='/embedded-sign-up'
                  className='hover:text-primary font-medium underline underline-offset-4'
                >
                  {t('Sign up')}
                </Link>
                .
              </p>
            )}
          </div>

          <UserAuthForm redirectTo={redirect} />

          <TermsFooter
            variant='sign-in'
            status={status}
            className='text-center'
          />
        </div>
      </section>
    </main>
  )
}
