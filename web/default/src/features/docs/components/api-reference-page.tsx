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
import type { DocsPageId } from '../docs-config'
import { ApiEndpointSection } from './api-endpoint-section'
import { CodeBlock } from './code-block'
import { DocsShell, type DocsTocItem } from './docs-shell'

type ApiReferencePageProps = {
  children: React.ReactNode
  description: string
  example: string
  method: 'GET' | 'POST'
  pageId: DocsPageId
  path: string
  requestDescription: string
  title: string
  toc: DocsTocItem[]
}

export function ApiReferencePage(props: ApiReferencePageProps) {
  return (
    <DocsShell
      pageId={props.pageId}
      title={props.title}
      description={props.description}
      toc={[{ id: 'request', label: '请求' }, ...props.toc]}
    >
      <ApiEndpointSection
        id='request'
        title='请求'
        description={props.requestDescription}
        method={props.method}
        path={props.path}
      >
        <CodeBlock code={props.example} label='cURL' />
      </ApiEndpointSection>
      {props.children}
    </DocsShell>
  )
}
