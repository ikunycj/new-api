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

import { z } from 'zod'

export const channelMonitorFormSchema = z.object({
  test_model: z
    .string()
    .trim()
    .min(1, '请输入测试模型')
    .max(200, '测试模型不能超过 200 个字符'),
  interval_seconds: z.coerce
    .number('请输入数字')
    .int('请输入整数')
    .min(1, '测试间隔不能少于 1 秒')
    .max(86400, '测试间隔不能超过 86400 秒'),
  timeout_seconds: z.coerce
    .number('请输入数字')
    .int('请输入整数')
    .min(1, '请求超时不能少于 1 秒')
    .max(120, '请求超时不能超过 120 秒'),
  retry_count: z.coerce
    .number('请输入数字')
    .int('请输入整数')
    .min(1, '重试次数不能少于 1 次')
    .max(10000, '重试次数不能超过 10000 次'),
  enabled: z.boolean(),
  visible: z.boolean(),
  availability_boost_percent: z.coerce
    .number('请输入数字')
    .min(0, '可用率加成必须在 0 到 100 之间')
    .max(100, '可用率加成必须在 0 到 100 之间'),
})

export type ChannelMonitorFormInput = z.input<typeof channelMonitorFormSchema>
export type ChannelMonitorFormValues = z.output<typeof channelMonitorFormSchema>

export const channelMonitorFormDefaults: ChannelMonitorFormInput = {
  test_model: '',
  interval_seconds: 300,
  timeout_seconds: 15,
  retry_count: 1,
  enabled: true,
  visible: true,
  availability_boost_percent: 0,
}
