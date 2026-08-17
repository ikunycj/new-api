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
import { LoaderCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'

interface OnboardingStepPreview {
  title: string
  description: string
}

interface NewUserOnboardingDialogProps {
  open: boolean
  steps: OnboardingStepPreview[]
  isPending: boolean
  onStart: () => void
  onSkip: () => void
}

export function NewUserOnboardingDialog(props: NewUserOnboardingDialogProps) {
  const { t } = useTranslation()

  const handleOpenChange = (open: boolean) => {
    if (!open) props.onSkip()
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={handleOpenChange}
      title={t('Build on your API gateway in minutes')}
      description={t(
        'Learn what the gateway provides and complete the basic setup before making your first request.'
      )}
      contentClassName='sm:max-w-lg'
      showCloseButton={!props.isPending}
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            disabled={props.isPending}
            onClick={props.onSkip}
          >
            {t('Skip')}
          </Button>
          <Button
            type='button'
            disabled={props.isPending}
            aria-busy={props.isPending}
            onClick={props.onStart}
          >
            {props.isPending ? (
              <LoaderCircle
                data-icon='inline-start'
                className='animate-spin'
                aria-hidden='true'
              />
            ) : null}
            {t('Get Started')}
          </Button>
        </>
      }
    >
      <ol className='grid gap-2' aria-busy={props.isPending}>
        {props.steps.map((step, index) => (
          <li
            key={step.title}
            className='bg-background flex min-h-14 items-center gap-3 rounded-lg border px-3 py-2.5'
          >
            <span className='text-muted-foreground w-8 shrink-0 font-mono text-xs font-semibold tabular-nums'>
              {String(index + 1).padStart(2, '0')}
            </span>
            <span className='flex min-w-0 flex-col gap-0.5'>
              <span className='text-sm font-medium'>{step.title}</span>
              <span className='text-muted-foreground text-xs leading-relaxed'>
                {step.description}
              </span>
            </span>
          </li>
        ))}
      </ol>
    </Dialog>
  )
}
