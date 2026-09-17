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
import { useEffect, useRef, useState } from 'react'

import { cn } from '@/lib/utils'

interface DeferUntilVisibleProps {
  children: React.ReactNode
  waitForScroll?: boolean
  placeholderClassName?: string
}

export function DeferUntilVisible(props: DeferUntilVisibleProps) {
  const markerRef = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    if (props.waitForScroll) {
      if (window.scrollY > 0) {
        setVisible(true)
        return
      }

      let idleCallbackId: number | undefined
      const fallbackTimerId = window.setTimeout(() => {
        if ('requestIdleCallback' in window) {
          idleCallbackId = window.requestIdleCallback(reveal, { timeout: 1000 })
          return
        }

        reveal()
      }, 2000)

      function cleanup() {
        window.removeEventListener('scroll', reveal)
        window.clearTimeout(fallbackTimerId)
        if (idleCallbackId !== undefined) {
          window.cancelIdleCallback(idleCallbackId)
        }
      }

      function reveal() {
        cleanup()
        setVisible(true)
      }

      window.addEventListener('scroll', reveal, { once: true, passive: true })
      return cleanup
    }

    const marker = markerRef.current
    if (!marker) return

    if (!('IntersectionObserver' in window)) {
      setVisible(true)
      return
    }

    const reveal = () => {
      observer.disconnect()
      window.removeEventListener('scroll', revealIfNearOrPassed)
      setVisible(true)
    }
    const revealIfNearOrPassed = () => {
      if (marker.getBoundingClientRect().top <= window.innerHeight + 200) {
        reveal()
      }
    }
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) reveal()
      },
      { rootMargin: '200px 0px' }
    )
    observer.observe(marker)
    window.addEventListener('scroll', revealIfNearOrPassed, { passive: true })
    revealIfNearOrPassed()

    return () => {
      observer.disconnect()
      window.removeEventListener('scroll', revealIfNearOrPassed)
    }
  }, [props.waitForScroll])

  if (visible) return props.children

  return (
    <div
      ref={markerRef}
      className={cn('bg-muted/30 min-h-32', props.placeholderClassName)}
      aria-hidden='true'
    />
  )
}
