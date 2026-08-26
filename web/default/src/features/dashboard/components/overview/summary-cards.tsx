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
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import { StaggerContainer, StaggerItem } from '@/components/page-transition';
import {
  getUserQuotaDates,
  getUserQuotaSummary,
} from '@/features/dashboard/api';
import { useSummaryCardsConfig } from '@/features/dashboard/hooks/use-dashboard-config';
import type { QuotaDataItem } from '@/features/dashboard/types';
import { getApiKeys } from '@/features/keys/api';
import { API_KEY_STATUS } from '@/features/keys/constants';
import { formatNumber, formatQuota } from '@/lib/format';
import { computeTimeRange } from '@/lib/time';
import { useAuthStore } from '@/stores/auth-store';

import { StatCard } from '../ui/stat-card';

const SUMMARY_SPARKLINE_BUCKETS = 12;

type SummarySparklineKey = 'balance' | 'usage' | 'requests';

function formatTokenAmount(value: number): string {
  const safeValue = Number.isFinite(value) ? value : 0;
  return `${Number((safeValue / 1_000_000).toFixed(2))}M`;
}

function getBucketIndex(
  timestamp: number,
  start: number,
  end: number,
  bucketCount: number,
): number {
  if (end <= start) return 0;
  const ratio = (timestamp - start) / (end - start);
  return Math.min(
    bucketCount - 1,
    Math.max(0, Math.floor(ratio * bucketCount)),
  );
}

function buildSummarySparklines(
  data: QuotaDataItem[],
  currentBalance: number,
  start: number,
  end: number,
): Record<SummarySparklineKey, number[]> {
  const usage = Array.from({ length: SUMMARY_SPARKLINE_BUCKETS }, () => 0);
  const requests = Array.from({ length: SUMMARY_SPARKLINE_BUCKETS }, () => 0);

  for (const item of data) {
    const timestamp = Number(item.created_at) || start;
    const index = getBucketIndex(
      timestamp,
      start,
      end,
      SUMMARY_SPARKLINE_BUCKETS,
    );
    usage[index] += Number(item.quota) || 0;
    requests[index] += Number(item.count) || 0;
  }

  let balance = currentBalance;
  const balanceTrend = Array.from(
    { length: SUMMARY_SPARKLINE_BUCKETS },
    () => 0,
  );

  for (let index = SUMMARY_SPARKLINE_BUCKETS - 1; index >= 0; index--) {
    balanceTrend[index] = Math.max(0, balance);
    balance += usage[index];
  }

  return {
    balance: balanceTrend,
    usage,
    requests,
  };
}

function getSummarySparkline(
  key: string,
  sparklineData: Record<SummarySparklineKey, number[]>,
): number[] | undefined {
  if (key === 'usage') return sparklineData.usage;
  if (key === 'requests') return sparklineData.requests;
  return undefined;
}

export function SummaryCards() {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.auth.user);

  const summaryTimeRange = useMemo(() => computeTimeRange(1), []);
  const remainQuota = Number(user?.quota ?? 0);

  const usageTrendQuery = useQuery({
    queryKey: [
      'dashboard',
      'overview',
      'summary-sparklines',
      summaryTimeRange.start_timestamp,
      summaryTimeRange.end_timestamp,
    ],
    queryFn: async () =>
      getUserQuotaDates({
        start_timestamp: summaryTimeRange.start_timestamp,
        end_timestamp: summaryTimeRange.end_timestamp,
        default_time: 'hour',
      }),
    staleTime: 60 * 1000,
  });

  const historicalSummaryQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'summary'],
    queryFn: getUserQuotaSummary,
    enabled: Boolean(user?.id),
    staleTime: 60 * 1000,
  });

  const apiKeysQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'api-keys-summary', user?.id],
    queryFn: async () => {
      const firstPage = await getApiKeys({ p: 1, size: 100 });
      const data = firstPage.data;
      if (!data) return { total: 0, enabled: 0 };
      let items = data.items;
      const pageCount = Math.ceil(data.total / data.page_size);
      if (pageCount > 1) {
        const pages = await Promise.all(
          Array.from({ length: pageCount - 1 }, (_, index) =>
            getApiKeys({ p: index + 2, size: 100 }),
          ),
        );
        items = [...items, ...pages.flatMap((page) => page.data?.items ?? [])];
      }
      return {
        total: data.total,
        enabled: items.filter((item) => item.status === API_KEY_STATUS.ENABLED)
          .length,
      };
    },
    enabled: Boolean(user?.id),
    staleTime: 60 * 1000,
  });

  const loading =
    usageTrendQuery.isLoading ||
    historicalSummaryQuery.isLoading ||
    apiKeysQuery.isLoading;

  const sparklineData = useMemo(
    () =>
      buildSummarySparklines(
        usageTrendQuery.data?.data ?? [],
        remainQuota,
        summaryTimeRange.start_timestamp,
        summaryTimeRange.end_timestamp,
      ),
    [
      remainQuota,
      summaryTimeRange.end_timestamp,
      summaryTimeRange.start_timestamp,
      usageTrendQuery.data?.data,
    ],
  );

  const recentUsage = useMemo(
    () =>
      (usageTrendQuery.data?.data ?? []).reduce(
        (total, item) => total + (Number(item.quota) || 0),
        0,
      ),
    [usageTrendQuery.data?.data],
  );

  const recentTokens = useMemo(
    () =>
      (usageTrendQuery.data?.data ?? []).reduce(
        (total, item) => total + (Number(item.token_used) || 0),
        0,
      ),
    [usageTrendQuery.data?.data],
  );

  const recentRequests = useMemo(
    () =>
      (usageTrendQuery.data?.data ?? []).reduce(
        (total, item) => total + (Number(item.count) || 0),
        0,
      ),
    [usageTrendQuery.data?.data],
  );

  const summaryValues = useMemo(() => {
    const totalTokens = Number(
      historicalSummaryQuery.data?.data?.total_tokens ?? 0,
    );
    const totalRequests = Number(
      historicalSummaryQuery.data?.data?.total_requests ??
        user?.request_count ??
        0,
    );
    const enabledKeys = apiKeysQuery.data?.enabled ?? 0;
    const totalKeys = apiKeysQuery.data?.total ?? 0;
    return {
      usageDisplay: formatQuota(recentUsage),
      usageDescription: `${t('Current balance')}: ${formatQuota(remainQuota)} · ${t('Historical total consumed')}: ${formatQuota(Number(user?.used_quota ?? 0))}`,
      tokenDisplay: formatTokenAmount(recentTokens),
      tokenDescription: `${t('Total')}: ${formatTokenAmount(totalTokens)}`,
      requestDisplay: `${t('Today')} ${formatNumber(recentRequests)}`,
      requestDescription: `${t('Total')}: ${formatNumber(totalRequests)}`,
      apiKeysDisplay: `${enabledKeys} ${t('enabled')} / ${totalKeys} ${t('total')}`,
      apiKeysDescription: `${t('Current concurrency')}: ${enabledKeys}`,
    };
  }, [
    apiKeysQuery.data,
    historicalSummaryQuery.data,
    recentRequests,
    recentTokens,
    recentUsage,
    remainQuota,
    t,
    user?.request_count,
    user?.used_quota,
  ]);

  const items = useSummaryCardsConfig(summaryValues).map((config, index) => {
    const tones = ['accent-1', 'accent-2', 'accent-3', 'accent-1'] as const;

    return {
      key: config.key,
      title: config.title,
      value: config.value,
      desc: config.description,
      icon: config.icon,
      tone: tones[index] ?? 'accent-3',
      sparkline:
        config.key === 'todayUsage'
          ? sparklineData.usage
          : getSummarySparkline(config.key, sparklineData),
      sparklineVariant: 'line' as const,
    };
  });

  return (
    <div className="bg-card overflow-hidden rounded-2xl border shadow-xs">
      <div className="flex flex-col gap-2.5 p-3 sm:gap-3 sm:p-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex flex-col gap-1">
            <h3 className="text-sm font-semibold sm:text-base">
              {t('Usage at a glance')}
            </h3>
            <p className="text-muted-foreground text-xs sm:text-sm">
              {t('Monitor balance, usage, and request volume')}
            </p>
          </div>
        </div>
        <StaggerContainer className="grid grid-cols-2 gap-1.5 sm:grid-cols-4 sm:gap-3">
          {items.map((it) => (
            <StaggerItem
              key={it.key}
              className="bg-background/60 rounded-lg border px-2 py-1.5 sm:rounded-xl sm:p-3"
            >
              <StatCard
                title={it.title}
                value={it.value}
                description={it.desc}
                icon={it.icon}
                tone={it.tone}
                sparkline={it.sparkline}
                sparklineVariant={it.sparklineVariant}
                loading={loading}
                compactMobile
              />
            </StaggerItem>
          ))}
        </StaggerContainer>
      </div>
    </div>
  );
}
