interface ErrorResponseBody {
  error?: string
  message?: string
}

interface FetchLikeError {
  data?: ErrorResponseBody
  response?: {
    _data?: ErrorResponseBody
    status?: number
    statusText?: string
  }
  statusCode?: number
  statusMessage?: string
  message?: string
}

export const extractApiErrorMessage = (
  error: unknown,
  fallbackMessage: string,
): string => {
  const e = error as FetchLikeError

  const bodyMessage = e.data?.error ?? e.data?.message ?? e.response?._data?.error ?? e.response?._data?.message
  if (bodyMessage) {
    return bodyMessage
  }

  const code = e.statusCode ?? e.response?.status
  const statusText = e.statusMessage ?? e.response?.statusText
  if (code && statusText) {
    return `${code} ${statusText}`
  }

  if (error instanceof Error && error.message) {
    return error.message
  }

  return fallbackMessage
}

export const extractApiStatusCode = (error: unknown): number | undefined => {
  const e = error as FetchLikeError
  return e.statusCode ?? e.response?.status
}
