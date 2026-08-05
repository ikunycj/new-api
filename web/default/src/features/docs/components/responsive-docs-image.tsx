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

type ResponsiveDocsImageProps = {
  alt: string
  caption: string
  height: number
  largeSrc: string
  smallSrc: string
  width: number
}

export function ResponsiveDocsImage(props: ResponsiveDocsImageProps) {
  return (
    <figure className='mt-5'>
      <a
        href={props.largeSrc}
        target='_blank'
        rel='noopener noreferrer'
        aria-label={`${props.alt}（打开大图）`}
        className='border-border bg-muted/20 block overflow-hidden rounded-lg border'
      >
        <img
          src={props.smallSrc}
          srcSet={`${props.smallSrc} 760w, ${props.largeSrc} 1520w`}
          sizes='(min-width: 1280px) 760px, (min-width: 768px) calc(100vw - 320px), calc(100vw - 32px)'
          width={props.width}
          height={props.height}
          loading='lazy'
          decoding='async'
          alt={props.alt}
          className='h-auto w-full'
        />
      </a>
      <figcaption className='text-muted-foreground mt-2 text-center text-xs leading-5'>
        {props.caption}
      </figcaption>
    </figure>
  )
}
