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
import { useQuery, useQueryClient, useIsFetching } from '@tanstack/react-query'
import { useNavigate, getRouteApi } from '@tanstack/react-router'
import type { Table } from '@tanstack/react-table'
import { Eye, EyeOff } from 'lucide-react'
import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Combobox,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from '@/components/ui/combobox'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { searchUsers } from '@/features/users/api'
import { useDebounce } from '@/hooks'

import { LOG_TYPE_ALL_VALUE, LOG_TYPE_FILTERS } from '../constants'
import { buildSearchParams } from '../lib/filter'
import { getDefaultTimeRange } from '../lib/utils'
import type { CommonLogFilters } from '../types'
import { CommonLogsStats } from './common-logs-stats'
import { CompactDateTimeRangePicker } from './compact-date-time-range-picker'
import { LogsFilterField, LogsFilterToolbar } from './logs-filter-toolbar'
import { useLogsViewScope, useUsageLogsContext } from './usage-logs-provider'

const route = getRouteApi('/_authenticated/usage-logs/$section')

type LogTypeValue = (typeof LOG_TYPE_FILTERS)[number]['value']
const logTypeValueSet = new Set<string>(
  LOG_TYPE_FILTERS.map((type) => type.value)
)

type CommonLogDraft = {
  sourceKey: string
  filters: CommonLogFilters
  logType: LogTypeValue
}

type LogUserSuggestion = {
  id: number
  username: string
  details: string
}

function isLogTypeValue(value: string): value is LogTypeValue {
  return logTypeValueSet.has(value)
}

function getLogTypeValue(value: unknown): LogTypeValue {
  return Array.isArray(value) &&
    value.length === 1 &&
    typeof value[0] === 'string' &&
    isLogTypeValue(value[0])
    ? value[0]
    : LOG_TYPE_ALL_VALUE
}

function buildSearchSourceKey(values: {
  startTime?: unknown
  endTime?: unknown
  channel?: unknown
  keyword?: unknown
  type?: unknown
}) {
  return [
    values.startTime,
    values.endTime,
    values.channel,
    values.keyword,
    Array.isArray(values.type) ? values.type.join(',') : values.type,
  ]
    .map((value) => String(value ?? ''))
    .join('\u001f')
}

interface CommonLogsFilterBarProps<TData> {
  table: Table<TData>
}

export function CommonLogsFilterBar<TData>(
  props: CommonLogsFilterBarProps<TData>
) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const searchParams = route.useSearch()
  const { sensitiveVisible, setSensitiveVisible } = useUsageLogsContext()
  const { isAdminView } = useLogsViewScope()
  const fetchingLogs = useIsFetching({ queryKey: ['logs'] })

  const searchState = useMemo<CommonLogDraft>(() => {
    const { start, end } = getDefaultTimeRange()
    const sourceValues = {
      startTime: searchParams.startTime,
      endTime: searchParams.endTime,
      channel: searchParams.channel,
      keyword: searchParams.keyword,
      type: searchParams.type,
    }
    const filters: CommonLogFilters = {
      startTime: searchParams.startTime
        ? new Date(searchParams.startTime)
        : start,
      endTime: searchParams.endTime ? new Date(searchParams.endTime) : end,
      channel: searchParams.channel || undefined,
      keyword: searchParams.keyword || undefined,
    }
    return {
      sourceKey: buildSearchSourceKey(sourceValues),
      filters,
      logType: getLogTypeValue(searchParams.type),
    }
  }, [
    searchParams.startTime,
    searchParams.endTime,
    searchParams.channel,
    searchParams.keyword,
    searchParams.type,
  ])
  const [draft, setDraft] = useState<CommonLogDraft>(() => searchState)
  const [highlightedUser, setHighlightedUser] = useState<LogUserSuggestion>()
  const activeDraft =
    draft.sourceKey === searchState.sourceKey ? draft : searchState
  const filters = activeDraft.filters
  const logType = activeDraft.logType
  const trimmedKeyword = filters.keyword?.trim() ?? ''
  const debouncedKeyword = useDebounce(trimmedKeyword, 300)
  const userSuggestionsQuery = useQuery({
    queryKey: ['usage-log-user-suggestions', debouncedKeyword],
    queryFn: () =>
      searchUsers({ keyword: debouncedKeyword, p: 1, page_size: 10 }),
    enabled: isAdminView && debouncedKeyword.length > 0,
    staleTime: 30_000,
    select: (result) =>
      (result.data?.items ?? []).map((user) => ({
        id: user.id,
        username: user.username,
        details: [user.display_name, user.email, user.remark]
          .filter(Boolean)
          .join(' / '),
      })),
  })
  const suggestionsReady = trimmedKeyword === debouncedKeyword
  const userSuggestions = suggestionsReady
    ? (userSuggestionsQuery.data ?? [])
    : []
  const keywordValue: LogUserSuggestion | null = filters.keyword
    ? { id: 0, username: filters.keyword, details: '' }
    : null

  const handleChange = useCallback(
    (field: keyof CommonLogFilters, value: Date | string | undefined) => {
      setDraft((current) => {
        const base =
          current.sourceKey === searchState.sourceKey ? current : searchState
        return {
          sourceKey: searchState.sourceKey,
          filters: { ...base.filters, [field]: value },
          logType: base.logType,
        }
      })
    },
    [searchState]
  )

  const applyFilters = useCallback(
    (nextFilters: CommonLogFilters) => {
      const filterParams = buildSearchParams(nextFilters, 'common')
      navigate({
        to: '/usage-logs/$section',
        params: { section: 'common' },
        search: {
          ...filterParams,
          type: [logType],
          page: 1,
        },
      })
      queryClient.invalidateQueries({ queryKey: ['logs'] })
      queryClient.invalidateQueries({ queryKey: ['usage-logs-stats'] })
    },
    [logType, navigate, queryClient]
  )

  const handleApply = useCallback(() => {
    applyFilters(filters)
  }, [applyFilters, filters])

  const handleUserSelect = useCallback(
    (user: LogUserSuggestion | null) => {
      if (!user) return

      const nextFilters = { ...filters, keyword: user.username }
      handleChange('keyword', user.username)
      applyFilters(nextFilters)
    },
    [applyFilters, filters, handleChange]
  )

  const handleReset = useCallback(() => {
    const { start, end } = getDefaultTimeRange()
    const resetFilters: CommonLogFilters = { startTime: start, endTime: end }
    const resetSearch = {
      type: [LOG_TYPE_ALL_VALUE],
      startTime: start.getTime(),
      endTime: end.getTime(),
    }
    setDraft({
      sourceKey: buildSearchSourceKey(resetSearch),
      filters: resetFilters,
      logType: LOG_TYPE_ALL_VALUE,
    })

    navigate({
      to: '/usage-logs/$section',
      params: { section: 'common' },
      search: {
        page: 1,
        ...resetSearch,
      },
    })
    queryClient.invalidateQueries({ queryKey: ['logs'] })
    queryClient.invalidateQueries({ queryKey: ['usage-logs-stats'] })
  }, [navigate, queryClient])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !highlightedUser) handleApply()
    },
    [handleApply, highlightedUser]
  )

  const hasTypeFilter = logType !== LOG_TYPE_ALL_VALUE
  const hasAdditionalFilters = !!filters.keyword || hasTypeFilter
  const logTypeItems = useMemo(
    () =>
      LOG_TYPE_FILTERS.map((type) => ({
        value: type.value,
        label: t(type.label),
      })),
    [t]
  )
  const logTypeLabel =
    logTypeItems.find((type) => type.value === logType)?.label ?? t('All Types')

  const statsBar = (
    <div className='flex flex-wrap items-center gap-2'>
      <CommonLogsStats />
    </div>
  )
  const sensitiveToggle = (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant='ghost'
            size='icon'
            onClick={() => setSensitiveVisible(!sensitiveVisible)}
            aria-label={sensitiveVisible ? t('Hide') : t('Show')}
            className='text-muted-foreground hover:text-foreground size-7'
          />
        }
      >
        {sensitiveVisible ? <Eye /> : <EyeOff />}
      </TooltipTrigger>
      <TooltipContent>
        {sensitiveVisible ? t('Hide') : t('Show')}
      </TooltipContent>
    </Tooltip>
  )

  const dateRangeFilter = (
    <LogsFilterField wide>
      <CompactDateTimeRangePicker
        start={filters.startTime}
        end={filters.endTime}
        onChange={({ start, end }) => {
          handleChange('startTime', start)
          handleChange('endTime', end)
        }}
      />
    </LogsFilterField>
  )
  const keywordFilter = (
    <LogsFilterField>
      <Combobox
        items={userSuggestions}
        itemToStringLabel={(user: LogUserSuggestion) => user.username}
        itemToStringValue={(user: LogUserSuggestion) => user.username}
        isItemEqualToValue={(item, value) => item.username === value.username}
        filter={null}
        value={keywordValue}
        inputValue={filters.keyword || ''}
        onInputValueChange={(value, details) => {
          if (details.reason === 'item-press') return
          handleChange('keyword', value)
          setHighlightedUser(undefined)
        }}
        onItemHighlighted={setHighlightedUser}
        onValueChange={handleUserSelect}
      >
        <ComboboxInput
          placeholder={t(
            'Search logs by model, token, user, email, remark, or request ID...'
          )}
          showTrigger={false}
          onKeyDown={handleKeyDown}
          className='h-8 min-w-0 text-sm leading-5 sm:min-w-[18rem]'
        />
        {isAdminView && (
          <ComboboxContent>
            <ComboboxList>
              <ComboboxCollection>
                {(user: LogUserSuggestion) => (
                  <ComboboxItem key={user.id} value={user}>
                    <span className='min-w-0 flex-1'>
                      <span className='block truncate font-medium'>
                        {user.username}
                      </span>
                      {user.details && (
                        <span className='text-muted-foreground block truncate text-xs'>
                          {user.details}
                        </span>
                      )}
                    </span>
                  </ComboboxItem>
                )}
              </ComboboxCollection>
            </ComboboxList>
            <ComboboxEmpty>
              {!suggestionsReady || userSuggestionsQuery.isFetching
                ? t('Loading...')
                : t('No matching users')}
            </ComboboxEmpty>
          </ComboboxContent>
        )}
      </Combobox>
    </LogsFilterField>
  )
  const typeFilter = (
    <LogsFilterField>
      <Select
        items={logTypeItems}
        value={logType}
        onValueChange={(value) => {
          const nextLogType =
            value !== null && isLogTypeValue(value) ? value : LOG_TYPE_ALL_VALUE
          setDraft((current) => {
            const base =
              current.sourceKey === searchState.sourceKey
                ? current
                : searchState
            return {
              sourceKey: searchState.sourceKey,
              filters: base.filters,
              logType: nextLogType,
            }
          })
        }}
      >
        <SelectTrigger>
          <SelectValue>{logTypeLabel}</SelectValue>
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            {LOG_TYPE_FILTERS.map((type) => (
              <SelectItem key={type.value} value={type.value}>
                {t(type.label)}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </LogsFilterField>
  )
  return (
    <LogsFilterToolbar
      table={props.table}
      stats={statsBar}
      actionStart={sensitiveToggle}
      primaryFilters={
        <>
          {dateRangeFilter}
          {keywordFilter}
          {typeFilter}
        </>
      }
      mobilePinnedFilters={dateRangeFilter}
      mobileFilters={
        <>
          {keywordFilter}
          {typeFilter}
        </>
      }
      mobileFilterCount={
        [filters.keyword, hasTypeFilter].filter(Boolean).length
      }
      hasActiveFilters={hasAdditionalFilters}
      onSearch={handleApply}
      searchLoading={fetchingLogs > 0}
      onReset={handleReset}
    />
  )
}
