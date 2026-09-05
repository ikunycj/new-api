export type SelfChannel = {
  id: number
  type: number
  name: string
  base_url?: string | null
  models: string
  model_mapping?: string | null
  openai_organization?: string | null
  remark?: string | null
  group: string
  status: number
}

export type SelfChannelResponse = {
  success: boolean
  message?: string
  data?: SelfChannel[]
}
