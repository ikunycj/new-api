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

import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import { getGroupBenchArtifact } from '../api'

type BenchArtifactProps = {
  runId: number
  className?: string
}

// SVGs are shown through <img> so model-written markup never runs scripts on
// this origin; SMIL animations still play inside an image.
export function BenchArtifact({ runId, className }: BenchArtifactProps) {
  const { t } = useTranslation()
  const artifactQuery = useQuery({
    queryKey: ['group-bench-artifact', runId],
    queryFn: () => getGroupBenchArtifact(runId),
    staleTime: Infinity,
  })

  if (artifactQuery.isLoading) {
    return <Skeleton className={cn('aspect-square w-full', className)} />
  }
  const artifact = artifactQuery.data
  if (!artifact) {
    return (
      <div
        className={cn(
          'text-muted-foreground flex aspect-square w-full items-center justify-center text-xs',
          className
        )}
      >
        {t('Failed to load image')}
      </div>
    )
  }
  if (artifact.content_type !== 'image/svg+xml') {
    return (
      <pre
        className={cn(
          'bg-muted aspect-square w-full overflow-auto rounded-md p-2 text-[10px] whitespace-pre-wrap',
          className
        )}
      >
        {artifact.content}
      </pre>
    )
  }
  return (
    <img
      src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(artifact.content)}`}
      alt={t('Bench output')}
      loading='lazy'
      className={cn(
        'aspect-square w-full rounded-md border bg-white object-contain',
        className
      )}
    />
  )
}
