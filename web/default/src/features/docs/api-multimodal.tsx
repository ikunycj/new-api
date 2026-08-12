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
import { Link } from '@tanstack/react-router'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { CodeBlock } from './components/code-block'
import { DocsShell } from './components/docs-shell'
import { useDocsBaseUrl } from './hooks/use-docs-base-url'

const DOC_LINK_CLASS =
  'text-primary font-medium underline-offset-4 hover:underline'

const MULTIMODAL_TOC = [
  { id: 'images', label: '图片' },
  { id: 'audio', label: '音频' },
  { id: 'rerank', label: '重排序' },
  { id: 'moderation', label: '内容审核' },
  { id: 'realtime', label: 'Realtime' },
  { id: 'videos', label: '视频' },
]

export function DocsApiMultimodal() {
  const baseUrl = useDocsBaseUrl()
  const openAiBaseUrl = `${baseUrl}/v1`
  const imageGeneration = `curl "${openAiBaseUrl}/images/generations" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"your-image-model-id","prompt":"一座湖边的山，日出","n":1,"size":"1024x1024"}'`
  const imageEdit = `curl "${openAiBaseUrl}/images/edits" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -F "model=your-image-model-id" \\
  -F "prompt=把天空改成日落" \\
  -F "image=@input.png"`
  const transcription = `curl "${openAiBaseUrl}/audio/transcriptions" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -F "file=@audio.mp3" \\
  -F "model=your-audio-model-id"`
  const speech = `curl "${openAiBaseUrl}/audio/speech" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"your-tts-model-id","input":"欢迎使用 API","voice":"alloy","response_format":"mp3"}' \\
  --output speech.mp3`
  const rerank = `curl "${openAiBaseUrl}/rerank" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"your-rerank-model-id","query":"API 鉴权","documents":["文档 A","文档 B"],"top_n":2}'`
  const moderation = `curl "${openAiBaseUrl}/moderations" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"your-moderation-model-id","input":"需要审核的文本"}'`
  const videoCreate = `curl "${openAiBaseUrl}/videos" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -F "model=your-video-model-id" \\
  -F "prompt=A cinematic sunrise over a city" \\
  -F "seconds=8"`
  const videoFetch = `curl "${openAiBaseUrl}/videos/video_task_xxx" \\
  -H "Authorization: Bearer sk-your-api-key"

curl -L "${openAiBaseUrl}/videos/video_task_xxx/content" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  --output result.mp4`
  const websocketProtocol = baseUrl.startsWith('https://') ? 'wss' : 'ws'
  const websocketHost = baseUrl.replace(/^https?:\/\//, '')
  const websocketUrl = `${websocketProtocol}://${websocketHost}/v1/realtime?model=your-realtime-model-id`

  return (
    <DocsShell
      pageId='api-multimodal'
      title='多模态接口'
      description='图片、音频、重排序、内容审核、Realtime 和异步视频的本站路径与最小请求。'
      toc={MULTIMODAL_TOC}
    >
      <p className='text-muted-foreground leading-7'>
        开始调用前，先用 <code>GET /v1/models</code> 确认模型
        ID，再确认该模型支持目标能力。模型能完成文本对话，不代表同时支持图片、音频或视频。
      </p>
      <section id='images' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>图片</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://platform.openai.com/docs/api-reference/images'
            target='_blank'
            rel='noreferrer'
          >
            OpenAI Images API
          </a>
        </p>
        <h3 className='mt-6 text-xl font-semibold'>生成</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          使用 <code>POST /v1/images/generations</code>。
        </p>
        <div className='mt-5'>
          <CodeBlock code={imageGeneration} label='cURL' />
        </div>
        <h3 className='mt-8 text-xl font-semibold'>编辑</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          使用 <code>POST /v1/images/edits</code>，提交图片时使用{' '}
          <code>multipart/form-data</code>。<code>/v1/edits</code>{' '}
          也可作为图片编辑兼容路径。
        </p>
        <div className='mt-5'>
          <CodeBlock code={imageEdit} label='cURL' />
        </div>
        <Alert className='mt-5'>
          <AlertTitle>先确认模型请求体</AlertTitle>
          <AlertDescription>
            不同图片模型的字段可能不同。模型详情页示例优先于通用示例，不能只替换{' '}
            <code>model</code> 就假定所有图片模型都接受同一请求体。
          </AlertDescription>
        </Alert>
      </section>

      <section id='audio' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>音频</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://platform.openai.com/docs/api-reference/audio'
            target='_blank'
            rel='noreferrer'
          >
            OpenAI Audio API
          </a>
        </p>
        <h3 className='mt-6 text-xl font-semibold'>转写</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          使用 <code>POST /v1/audio/transcriptions</code>，字段通过{' '}
          <code>multipart/form-data</code> 上传。
        </p>
        <div className='mt-5'>
          <CodeBlock code={transcription} label='cURL' />
        </div>
        <h3 className='mt-8 text-xl font-semibold'>翻译</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          将转写路径改为 <code>POST /v1/audio/translations</code>
          ，其余鉴权方式保持不变。
        </p>
        <h3 className='mt-8 text-xl font-semibold'>文本转语音</h3>
        <p className='text-muted-foreground mt-3 leading-7'>
          使用 <code>POST /v1/audio/speech</code>
          ，响应是音频内容，建议直接保存到文件。
        </p>
        <div className='mt-5'>
          <CodeBlock code={speech} label='cURL' />
        </div>
      </section>

      <section id='rerank' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>重排序</h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          使用 <code>POST /v1/rerank</code>。请求体至少包含 <code>model</code>、
          <code>query</code> 和 <code>documents</code>。
        </p>
        <div className='mt-5'>
          <CodeBlock code={rerank} label='cURL' />
        </div>
      </section>

      <section id='moderation' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>内容审核</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://platform.openai.com/docs/api-reference/moderations'
            target='_blank'
            rel='noreferrer'
          >
            OpenAI Moderations API
          </a>
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          使用 <code>POST /v1/moderations</code>，请求体包含 <code>model</code>{' '}
          和 <code>input</code>。
        </p>
        <div className='mt-5'>
          <CodeBlock code={moderation} label='cURL' />
        </div>
      </section>

      <section id='realtime' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>Realtime</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://platform.openai.com/docs/api-reference/realtime'
            target='_blank'
            rel='noreferrer'
          >
            OpenAI Realtime API
          </a>
        </p>
        <CodeBlock code={websocketUrl} label='WebSocket 地址' />
        <p className='text-muted-foreground mt-5 leading-7'>
          使用 WebSocket 客户端连接。浏览器不能自定义 <code>Authorization</code>{' '}
          时，可在握手时传递以下子协议：
        </p>
        <div className='mt-5'>
          <CodeBlock
            code='Sec-WebSocket-Protocol: realtime, openai-insecure-api-key.sk-your-api-key, openai-beta.realtime-v1'
            label='WebSocket 握手头'
          />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          服务端返回的事件格式和会话流程以官方 Realtime 文档为准。
        </p>
      </section>

      <section id='videos' className='scroll-mt-28'>
        <h2 className='text-2xl font-semibold'>视频</h2>
        <p className='mt-3 leading-7'>
          官方文档：{' '}
          <a
            className={DOC_LINK_CLASS}
            href='https://platform.openai.com/docs/api-reference/videos'
            target='_blank'
            rel='noreferrer'
          >
            OpenAI Videos API
          </a>
        </p>
        <p className='text-muted-foreground mt-3 leading-7'>
          创建任务使用 <code>POST /v1/videos</code>，查询使用{' '}
          <code>GET /v1/videos/{'{video_id}'}</code>，下载使用{' '}
          <code>GET /v1/videos/{'{video_id}'}/content</code>。
        </p>
        <div className='mt-5'>
          <CodeBlock code={videoCreate} label='创建任务' />
        </div>
        <div className='mt-5'>
          <CodeBlock code={videoFetch} label='查询和下载' />
        </div>
        <p className='text-muted-foreground mt-4 leading-7'>
          只有任务状态为 <code>completed</code>{' '}
          时下载内容。Kling、Midjourney、Suno 和即梦使用各自的任务接口，路径见{' '}
          <Link className={DOC_LINK_CLASS} to='/docs/api/compatibility'>
            兼容性与限制
          </Link>
          。
        </p>
      </section>
    </DocsShell>
  )
}
