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
import { ERROR_MESSAGES } from '../../constants'

type RequestErrorLike = {
  message?: string
  response?: {
    data?: {
      error?: {
        code?: string
        message?: string
        request_id?: string
      }
      message?: string
      request_id?: string
    }
    headers?: Record<string, string | string[] | undefined>
  }
}

export type RequestErrorDetails = {
  errorCode?: string
  errorMessage: string
  requestId?: string
}

function getRequestIdFromHeaders(
  headers?: Record<string, string | string[] | undefined>
): string | undefined {
  const entry = Object.entries(headers ?? {}).find(
    ([key]) => key.toLowerCase() === 'x-oneapi-request-id'
  )
  const value = entry?.[1]
  if (Array.isArray(value)) return value[0] || undefined
  return value || undefined
}

export function parseRequestErrorDetails(error: unknown): RequestErrorDetails {
  const requestError = error as RequestErrorLike
  const errorData = requestError?.response?.data

  return {
    errorCode: errorData?.error?.code || undefined,
    errorMessage:
      errorData?.error?.message ||
      errorData?.message ||
      requestError?.message ||
      ERROR_MESSAGES.API_REQUEST_ERROR,
    requestId:
      errorData?.error?.request_id ||
      errorData?.request_id ||
      getRequestIdFromHeaders(requestError?.response?.headers),
  }
}
