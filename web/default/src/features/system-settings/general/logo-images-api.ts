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
import { api } from '@/lib/api'

export type LogoImage = {
  name: string
  url: string
}

type LogoImagesResponse<T> = {
  success: boolean
  message?: string
  data: T
}

export async function getLogoImages(): Promise<LogoImage[]> {
  const response = await api.get<LogoImagesResponse<LogoImage[]>>(
    '/api/option/logo-images',
    { skipBusinessError: true, skipErrorHandler: true }
  )
  if (!response.data.success) {
    throw new Error(response.data.message || '加载图片失败')
  }
  return response.data.data
}

export async function uploadLogoImage(file: File): Promise<LogoImage> {
  const formData = new FormData()
  formData.append('file', file)
  const response = await api.post<LogoImagesResponse<LogoImage>>(
    '/api/option/logo-images',
    formData,
    { skipBusinessError: true }
  )
  if (!response.data.success) {
    throw new Error(response.data.message || '上传图片失败')
  }
  return response.data.data
}

export async function deleteLogoImage(name: string): Promise<void> {
  const response = await api.delete<LogoImagesResponse<null>>(
    `/api/option/logo-images/${encodeURIComponent(name)}`,
    { skipBusinessError: true }
  )
  if (!response.data.success) {
    throw new Error(response.data.message || '删除图片失败')
  }
}
