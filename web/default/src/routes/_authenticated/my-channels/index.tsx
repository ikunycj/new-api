import { createFileRoute, redirect } from '@tanstack/react-router'

import { MyChannels } from '@/features/my-channels'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/my-channels/')({
  beforeLoad: () => {
    const user = useAuthStore.getState().auth.user
    if (!user || !['vip', 'enterprise'].includes((user.group ?? '').toLowerCase())) {
      throw redirect({ to: '/403' })
    }
  },
  component: MyChannels,
})
