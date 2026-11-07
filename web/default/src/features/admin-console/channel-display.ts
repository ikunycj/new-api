const UNKNOWN_CHANNEL_LABEL = '未记录渠道'

export function formatChannelDisplayName(
  channelName?: string | null,
  channelId?: number | null
): string {
  const name = channelName?.trim() ?? ''
  const id = Number(channelId)

  if (!Number.isInteger(id) || id <= 0) {
    return name || UNKNOWN_CHANNEL_LABEL
  }

  const fallback = `渠道 #${id}`
  if (!name || name === fallback || name === `channel-${id}`) {
    return fallback
  }

  const suffix = ` #${id}`
  return name.endsWith(suffix) ? name : `${name}${suffix}`
}
