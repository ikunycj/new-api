# Codex

可选择 CC Switch 一键导入，或手动编辑 `auth.json` 和 `config.toml` 接入 Codex。

## 1. 使用 CC Switch 一键导入

CC Switch 是最快的接入方式，无需手动编辑 TOML，即可导入 API Key、所选模型和 Codex 服务地址。

1. 安装 CC Switch，并确认系统已注册 `ccswitch://` 协议。
2. 打开 API Key 页面，找到要使用的密钥。
3. 打开该行的操作菜单，选择 CC Switch 导入。
4. 选择 Codex，再选择当前 API Key 可用的主模型，并保留自动生成的 `/v1` 地址。
5. 确认浏览器提示，然后在 CC Switch 中检查并保存导入的服务商。

- [打开 API Key](https://alltokenapi.com/keys)
- [下载 CC Switch](https://github.com/farion1231/cc-switch/releases)

> **导入的配置**
>
> Codex 导入内容包含所选 API Key、主模型和以 `/v1` 结尾的服务地址。请仅在自己的设备上确认导入。

## 2. 编辑 auth.json 和 config.toml

本流程将 Codex 的认证凭据保存到用户目录中的 `auth.json`，并从 `config.toml` 读取服务商、模型和接口设置。

| 系统 | 配置文件路径 |
| --- | --- |
| Windows | `%USERPROFILE%\.codex\auth.json`、`%USERPROFILE%\.codex\config.toml` |
| macOS / Linux | `~/.codex/auth.json`、`~/.codex/config.toml` |

### 编辑 auth.json

在用户级 `auth.json` 中写入 API Key。默认路径为 `~/.codex/auth.json`；Windows 使用 `%USERPROFILE%\.codex\auth.json`。如果文件中已有其他认证字段，只更新 `OPENAI_API_KEY`，不要覆盖整个文件。

```json
{
  "OPENAI_API_KEY": "sk-your-api-key"
}
```

保存后，关闭终端也不会清除该配置。文件模式下的 `auth.json` 包含明文密钥，请仅保存在自己的设备上，不要提交到 Git 或发送给他人。密钥只保存在 `auth.json`；`config.toml` 仅用于服务商、模型和接口设置，不要依赖当前终端的临时设置。

### 编辑 config.toml

保留现有 Codex 配置，并合并下面的服务商配置块；已有 `cli_auth_credentials_store` 时将其改为 `"file"`，不要重复添加。需要时，将示例模型替换为 `auth.json` 中 API Key 可用的模型。

```toml
cli_auth_credentials_store = "file"
model = "gpt-5.6-sol"
model_provider = "alltokenapi"

[model_providers.alltokenapi]
name = "All Token API"
base_url = "https://alltokenapi.com/v1"
requires_openai_auth = true
wire_api = "responses"
```

`cli_auth_credentials_store = "file"` 确保 Codex 使用用户目录中的 `auth.json` 保存和读取凭据。`requires_openai_auth = true` 会让自定义服务商使用该凭据；`config.toml` 只保留服务商、模型和接口设置。

> [!tip] 为什么对话历史丢失
>
> 如果当前 `config.toml` 中的 `model_provider` 与创建对话时使用的值不一致，原有对话历史可能不会显示。这通常不代表记录已被删除；将 `model_provider` 改回创建对话时的值（包括大小写），重启 Codex 后使用 `codex resume` 查找历史。

### 重启并验证

1. 保存 `auth.json` 和 `config.toml`，然后重启正在运行的 Codex 应用或 IDE 扩展。
2. 在任意终端中运行 `codex`，发起一个测试任务；认证信息由 `auth.json` 提供。
3. 如启动失败，运行 `codex doctor` 检查认证文件是否被读取，再检查模型名称和 Base URL。

```bash
codex
```

> **配置说明**
>
> 请保持 `wire_api = "responses"`，并确保本服务的 Base URL 以 `/v1` 结尾。

[Codex 认证参考](https://developers.openai.com/codex/auth) · [Codex 配置参考](https://developers.openai.com/codex/config-reference)
