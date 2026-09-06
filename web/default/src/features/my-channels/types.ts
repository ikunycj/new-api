export type SelfChannel = {
  id: number
  type: number
  name: string
  base_url?: string | null
  models: string
  test_model?: string | null
  model_mapping?: string | null
  remark?: string | null
  masked_key?: string
  group: string
  status: number
}

export type SelfChannelResponse = {
  success: boolean
  message?: string
  data?: SelfChannel[]
}
