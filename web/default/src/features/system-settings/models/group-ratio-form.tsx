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
import { Code2, Eye, HelpCircle } from 'lucide-react'
import { memo, useCallback, useState } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageActionsPortal } from '../components/settings-page-context'
import { GroupRatioVisualEditor } from './group-ratio-visual-editor'

type GroupFormValues = {
  GroupRatio: string
  TopupGroupRatio: string
  UserUsableGroups: string
}

type GroupRatioFormProps = {
  form: UseFormReturn<GroupFormValues>
  onSave: (values: GroupFormValues) => Promise<void>
  isSaving: boolean
}

export const GroupRatioForm = memo(function GroupRatioForm({
  form,
  onSave,
  isSaving,
}: GroupRatioFormProps) {
  const { t } = useTranslation()
  const [editMode, setEditMode] = useState<'visual' | 'json'>('visual')
  const [guideOpen, setGuideOpen] = useState(false)

  const handleFieldChange = useCallback(
    (field: keyof GroupFormValues, value: string) => {
      form.setValue(field, value, {
        shouldValidate: true,
        shouldDirty: true,
      })
    },
    [form]
  )

  const toggleEditMode = useCallback(() => {
    setEditMode((prev) => (prev === 'visual' ? 'json' : 'visual'))
  }, [])

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap justify-end gap-2'>
        <Button variant='outline' size='sm' onClick={() => setGuideOpen(true)}>
          <HelpCircle className='mr-2 h-4 w-4' />
          {t('Usage guide')}
        </Button>
        <Button variant='outline' size='sm' onClick={toggleEditMode}>
          {editMode === 'visual' ? (
            <>
              <Code2 className='mr-2 h-4 w-4' />
              {t('Switch to JSON')}
            </>
          ) : (
            <>
              <Eye className='mr-2 h-4 w-4' />
              {t('Switch to Visual')}
            </>
          )}
        </Button>
      </div>

      <GroupPricingGuide open={guideOpen} onOpenChange={setGuideOpen} />

      <Form {...form}>
        <SettingsPageActionsPortal>
          <Button
            type='button'
            size='sm'
            onClick={form.handleSubmit(onSave)}
            disabled={isSaving}
          >
            {isSaving ? t('Saving...') : t('Save group ratios')}
          </Button>
        </SettingsPageActionsPortal>
        {editMode === 'visual' ? (
          <div className='space-y-6'>
            <GroupRatioVisualEditor
              groupRatio={form.watch('GroupRatio')}
              topupGroupRatio={form.watch('TopupGroupRatio')}
              userUsableGroups={form.watch('UserUsableGroups')}
              onChange={(field, value) =>
                handleFieldChange(field as keyof GroupFormValues, value)
              }
            />
          </div>
        ) : (
          <SettingsForm onSubmit={form.handleSubmit(onSave)}>
            <FormField
              control={form.control}
              name='GroupRatio'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Group ratios')}</FormLabel>
                  <FormControl>
                    <Textarea rows={8} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'JSON map of group → ratio applied when the user selects the group explicitly.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='TopupGroupRatio'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Top-up group ratios')}</FormLabel>
                  <FormControl>
                    <Textarea rows={6} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Optional multiplier per user group used when calculating recharge pricing. Provide a JSON object such as'
                    )}
                    {` { "default": 1, "vip": 1.2 }`}.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='UserUsableGroups'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Selectable groups')}</FormLabel>
                  <FormControl>
                    <Textarea rows={6} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'JSON map of group → description exposed when users create API keys.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsForm>
        )}
      </Form>
    </div>
  )
})

type GroupPricingGuideProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function GuideCodeBlock({ children }: { children: string }) {
  return (
    <pre className='bg-muted/60 overflow-x-auto rounded-lg border px-3 py-2 text-xs leading-6 whitespace-pre-wrap'>
      {children}
    </pre>
  )
}

function GroupPricingGuide({ open, onOpenChange }: GroupPricingGuideProps) {
  const { t } = useTranslation()

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side='right'
        className={sideDrawerContentClassName('sm:max-w-2xl')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Group pricing usage guide')}</SheetTitle>
        </SheetHeader>

        <div className={sideDrawerFormClassName('gap-5')}>
          <section className='space-y-2'>
            <h3 className='text-sm font-semibold'>
              {t('The two roles of a group')}
            </h3>
            <div className='text-muted-foreground space-y-2 text-sm leading-6'>
              <p>
                {t(
                  'Pricing groups and user groups are separate concepts. Pricing groups control token routing and billing; user groups identify accounts and do not automatically become pricing groups.'
                )}
              </p>
              <p>
                <span className='text-foreground font-medium'>
                  {t('Token group')}
                </span>
                {': '}
                {t(
                  'decides which channels are used and which base ratio applies.'
                )}
              </p>
              <p>
                <span className='text-foreground font-medium'>
                  {t('User group')}
                </span>
                {': '}
                {t(
                  'identifies the account. Selectable groups decide which pricing groups the account can assign to API keys.'
                )}
              </p>
            </div>
          </section>

          <Accordion className='rounded-lg border px-3'>
            <AccordionItem value='groups'>
              <AccordionTrigger>{t('Pricing group example')}</AccordionTrigger>
              <AccordionContent className='space-y-3'>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Use the pricing group table to manage the ratio and whether the group appears in the token creation dropdown.'
                  )}
                </p>
                <GuideCodeBlock>
                  {`${t('Group name')}   ${t('Ratio')}   ${t('User selectable')}   ${t('Description')}
standard     1.0     ${t('Yes')}               ${t('Standard price')}
premium      0.5     ${t('Yes')}               ${t('Premium plan, half price')}
vip          0.5     ${t('No')}                ${t('Assigned by administrator only')}`}
                </GuideCodeBlock>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Users only see groups marked as user selectable. Non-selectable groups can still be assigned by administrators.'
                  )}
                </p>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      </SheetContent>
    </Sheet>
  )
}
