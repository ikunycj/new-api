import { DownloadIcon, ExternalLinkIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { GeneratedImage } from '../../types'

type PlaygroundGeneratedImagesProps = {
  images: GeneratedImage[]
}

export function PlaygroundGeneratedImages({
  images,
}: PlaygroundGeneratedImagesProps) {
  const { t } = useTranslation()

  return (
    <div className='grid gap-3 sm:grid-cols-2'>
      {images.map((image, index) => (
        <figure
          className='border-border/70 bg-muted/20 overflow-hidden rounded-lg border'
          key={`${image.url}-${image.revisedPrompt ?? ''}`}
        >
          <a href={image.url} rel='noreferrer' target='_blank'>
            <img
              alt={image.revisedPrompt || t('Generated image')}
              className='block aspect-square w-full object-contain'
              loading='lazy'
              src={image.url}
            />
          </a>
          <figcaption className='flex items-center justify-between gap-2 border-t px-3 py-2'>
            <span className='text-muted-foreground min-w-0 truncate text-xs'>
              {image.revisedPrompt || t('Generated image')}
            </span>
            <span className='flex shrink-0 items-center gap-1'>
              <a
                aria-label={t('Open image')}
                className='text-muted-foreground hover:text-foreground inline-flex size-7 items-center justify-center rounded-md'
                href={image.url}
                rel='noreferrer'
                target='_blank'
                title={t('Open image')}
              >
                <ExternalLinkIcon className='size-4' />
              </a>
              <a
                aria-label={t('Download image')}
                className='text-muted-foreground hover:text-foreground inline-flex size-7 items-center justify-center rounded-md'
                download={`playground-image-${index + 1}.png`}
                href={image.url}
                title={t('Download image')}
              >
                <DownloadIcon className='size-4' />
              </a>
            </span>
          </figcaption>
        </figure>
      ))}
    </div>
  )
}
