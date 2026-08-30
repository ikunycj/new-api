import { createFileRoute, redirect } from '@tanstack/react-router'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/group-pricing/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({ to: '/403' })
    }
    throw redirect({
      to: '/group-management',
      search: { tab: 'pricing-groups' },
    })
  },
})
