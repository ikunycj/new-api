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
import type { QueryClient } from '@tanstack/react-query'
import {
  createRootRouteWithContext,
  Outlet,
  redirect,
} from '@tanstack/react-router'
import { lazy, Suspense, useEffect, useState } from 'react'

import { ThemeCustomizationProvider } from '@/context/theme-customization-provider'
import { saveAffiliateCode } from '@/features/auth/lib/storage'
import { GeneralError } from '@/features/errors/general-error'
import { NotFoundError } from '@/features/errors/not-found-error'
import { getSetupStatus } from '@/features/setup/api'
import { useHydrated } from '@/hooks/use-hydrated'
import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import { getPublicBootstrap } from '@/lib/public-bootstrap'

const NavigationProgress = lazy(() =>
  import('@/components/navigation-progress').then((module) => ({
    default: module.NavigationProgress,
  }))
)
const Toaster = lazy(() =>
  import('@/components/ui/sonner').then((module) => ({
    default: module.Toaster,
  }))
)
const ReactQueryDevtools = lazy(() =>
  import('@tanstack/react-query-devtools').then((module) => ({
    default: module.ReactQueryDevtools,
  }))
)
const TanStackRouterDevtools = lazy(() =>
  import('@tanstack/react-router-devtools').then((module) => ({
    default: module.TanStackRouterDevtools,
  }))
)

function RootComponent() {
  const hydrated = useHydrated()
  const [enhancementsReady, setEnhancementsReady] = useState(false)
  // One shared React Query request owns status refresh and system config sync.
  useStatus()
  useSystemConfig()

  useEffect(() => {
    const aff = new URLSearchParams(window.location.search).get('aff')?.trim()
    if (aff) {
      saveAffiliateCode(aff)
    }
  }, [])

  useEffect(() => {
    if (!hydrated) return

    const enableEnhancements = () => setEnhancementsReady(true)
    const timeoutId = window.setTimeout(enableEnhancements, 3000)
    window.addEventListener('pointerdown', enableEnhancements, {
      once: true,
      passive: true,
    })
    window.addEventListener('keydown', enableEnhancements, { once: true })

    return () => {
      window.clearTimeout(timeoutId)
      window.removeEventListener('pointerdown', enableEnhancements)
      window.removeEventListener('keydown', enableEnhancements)
    }
  }, [hydrated])

  return (
    <ThemeCustomizationProvider>
      {enhancementsReady && (
        <Suspense fallback={null}>
          <NavigationProgress />
        </Suspense>
      )}
      {import.meta.env.SSR ? (
        <Suspense fallback={null}>
          <Outlet />
        </Suspense>
      ) : (
        <Outlet />
      )}
      {enhancementsReady && (
        <Suspense fallback={null}>
          <Toaster
            closeButton
            duration={5000}
            position='top-center'
            richColors
          />
        </Suspense>
      )}
      {enhancementsReady && import.meta.env.MODE === 'development' && (
        <Suspense fallback={null}>
          <ReactQueryDevtools buttonPosition='bottom-left' />
          <TanStackRouterDevtools position='bottom-right' />
        </Suspense>
      )}
    </ThemeCustomizationProvider>
  )
}

// 缓存 setup 状态检查结果，避免每次导航都重复调用 API
// 使用 localStorage 持久化，避免页面刷新后重复检查
const SETUP_CHECKED_KEY = 'setup_status_checked'

function getSetupStatusFromCache(): boolean {
  if (getPublicBootstrap()?.setup) return true

  try {
    if (typeof window !== 'undefined') {
      return window.localStorage.getItem(SETUP_CHECKED_KEY) === 'true'
    }
  } catch {
    /* empty */
  }
  return false
}

function setSetupStatusCache(value: boolean): void {
  try {
    if (typeof window !== 'undefined') {
      if (value) {
        window.localStorage.setItem(SETUP_CHECKED_KEY, 'true')
      } else {
        window.localStorage.removeItem(SETUP_CHECKED_KEY)
      }
    }
  } catch {
    /* empty */
  }
}

// 内存中的标记，避免同一会话中重复检查
let setupStatusChecked = getSetupStatusFromCache()

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  // 应用初始化与路由解析前统一校验会话
  beforeLoad: async ({ location }) => {
    if (typeof window === 'undefined') return

    const pathname = location?.pathname || ''
    const needsSetupCheck =
      !setupStatusChecked && !pathname.startsWith('/setup')

    // 用户信息已通过 auth-store 从 localStorage 恢复
    // 如果 auth.user 存在，说明用户已登录（有缓存的用户数据）
    // 如果 auth.user 为 null，说明用户未登录，直接让 _authenticated 路由处理重定向
    // 不再调用 getSelf() API，避免不必要的网络请求和等待

    // 只检查 setup 状态（如果需要）
    if (needsSetupCheck) {
      const status = await getSetupStatus().catch((error) => {
        if (import.meta.env.DEV) {
          // eslint-disable-next-line no-console
          console.warn('[root.beforeLoad] setup status check failed', error)
        }
        return null
      })

      if (status?.success && status.data && !status.data.status) {
        throw redirect({ to: '/setup' })
      }
      setupStatusChecked = true
      setSetupStatusCache(true)
    }
    // 用户认证状态完全依赖 localStorage 缓存
    // 如果用户有有效 session 但 localStorage 被清空，会被重定向到登录页重新登录
  },
  component: RootComponent,
  notFoundComponent: NotFoundError,
  errorComponent: GeneralError,
})
