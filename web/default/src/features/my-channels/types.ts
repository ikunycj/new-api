export type SelfChannel = {
  id: number
  owner_user_id: number
  type: number
  name: string
  key: string
  base_url?: string | null
  models: string
  model_mapping?: string | null
  openai_organization?: string | null
  remark?: string | null
  group: string
  status: number
}

export type SelfChannelRequest = {
  type: number
  name: string
  key: string
  base_url?: string
  models: string
  model_mapping?: string
  openai_organization?: string
  remark?: string
}

export type SelfChannelResponse = {
  success: boolean
  message?: string
  data?: SelfChannel[] | SelfChannel
}
