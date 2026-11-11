# 渠道设置

渠道的 `setting` 保存格式化、思考内容转换、代理等通用设置；`other_settings` 保存厂商特定选项和可选请求字段的透传策略。两者是不同的 JSON 字段，不能混用。

字段定义见 [ChannelSettings / ChannelOtherSettings](../../relaykit/dto/channel_settings.go)，持久化入口见 [渠道模型](../../model/channel.go)。

## 常用通用设置

| 字段 | 类型 | 用途 |
| --- | --- | --- |
| `force_format` | 布尔值 | 在 OpenAI 兼容响应处理路径中按网关结构重新序列化响应；不代表任意协议都能自动转换为 OpenAI |
| `thinking_to_content` | 布尔值 | 在支持该选项的 OpenAI 兼容流式处理路径中，将 `reasoning_content` 以 `<think>` 标签拼接到正文 |
| `proxy` | 字符串 | 该渠道请求使用的代理地址；支持 `http`、`https`、`socks5`、`socks5h` |

例如，下面的内容应写入渠道的 `setting`：

```json
{
  "force_format": true,
  "thinking_to_content": true,
  "proxy": "socks5://proxy.example.com:1080"
}
```

这些选项的实际适用范围由具体适配器决定。响应处理见 [OpenAI 响应处理](../../relay/channel/openai/relay-openai.go)，流式转换见 [流式格式处理](../../relay/channel/openai/helper.go)，代理支持见 [HTTP 客户端](../../service/http_client.go)。

## 其他设置

`other_settings` 中包括 `azure_responses_version`、`vertex_key_type`、`aws_key_type`、`allow_service_tier`、`disable_store` 和上游模型同步选项等。按对应渠道类型和管理界面填写，完整字段以 DTO 为准。

可选字段的过滤与透传见 [Relay 请求字段处理](../../relay/common/relay_info.go)。自动探测使用独立的路由策略设置，见 [自动探测](probes.md)。
