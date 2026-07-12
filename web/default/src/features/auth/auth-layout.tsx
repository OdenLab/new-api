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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()
  const { status } = useStatus()
  const bgEnabled = status?.enable_custom_background === true
  const bgImageUrl =
    typeof status?.custom_background_url === 'string'
      ? status.custom_background_url.trim()
      : ''
  const showCustomBackground = bgEnabled && bgImageUrl.length > 0

  return (
    <div className='relative grid h-svh max-w-none overflow-hidden'>
      {showCustomBackground && (
        <>
          <img
            src={bgImageUrl}
            alt={t('Custom background image')}
            className='absolute inset-0 h-full w-full object-cover'
            loading='eager'
            decoding='async'
          />
          <div className='from-background/72 via-background/55 to-background/78 sm:from-background/70 sm:to-background/72 absolute inset-0 bg-linear-to-b' />
        </>
      )}
      <Link
        to='/'
        className='absolute top-4 left-4 z-20 flex items-center gap-2 transition-opacity hover:opacity-80 sm:top-8 sm:left-8'
      >
        <div className='relative h-8 w-8'>
          {loading ? (
            <Skeleton className='absolute inset-0 rounded-full' />
          ) : (
            <img
              src={logo}
              alt={t('Logo')}
              className='h-8 w-8 rounded-full object-cover'
            />
          )}
        </div>
        {loading ? (
          <Skeleton className='h-6 w-24' />
        ) : (
          <h1 className='text-xl font-medium'>{systemName}</h1>
        )}
      </Link>
      <div className='relative z-10 container flex items-center pt-16 sm:pt-0'>
        <div className='bg-background/78 border-border/55 shadow-primary/5 mx-auto flex w-full flex-col justify-center space-y-2 rounded-2xl border px-4 py-8 shadow-xl backdrop-blur-md sm:w-[480px] sm:p-8'>
          {children}
        </div>
      </div>
    </div>
  )
}
