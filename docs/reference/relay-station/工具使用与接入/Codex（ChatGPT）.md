# ChatGPT保姆级安装教程

## ChatGPT下载
### Windows版
#### 常规方法下载
1. 访问OpenAI官网下载ChatGPT安装器，前往[Codex官网：https://openai.com/zh-Hans-CN/codex/](https://openai.com/zh-Hans-CN/codex/)、[ChatGPT桌面客户端官网：https://chatgpt.com/zh-Hans-CN/download/](https://chatgpt.com/zh-Hans-CN/download/)均可下载

> [!tip] 注意
Codex和ChatGPT客户端已经合二为一，Codex注重编程能力，ChatGPT桌面客户端注重日常办公，用户可自行切换模式

Codex官网：
![](/static/image/docs/codex/codex-step-01-1520.76aec47a3542.webp)

ChatGPT桌面客户端官网
![](/static/image/docs/codex/codex-step-02-1520.448a7e2c0dae.webp)

2. 找到下载好的ChatGPT安装器，双击执行下载（必须开启全局代理）
![](/static/image/docs/codex/codex-step-03-1520.c3cc73933895.webp)

>[!tip] 注意
>目前 Windows 中的 Codex 和 ChatGPT 都只能在微软商店中下载，下载后也只能默认安装在 C 盘中
>必须开启全局代理，因为安装器是通过微软应用商城下载ChatGPT，而中国版本的微软应用商城是会根据你的使用地点检测你是否有下载权限的
>如果后续遇到“ChatGPT客户端不可用、禁用”等问题，很大概率是ChatGPT通过微软应用商城检测到了你的真实使用地址，这时候开启全局代理就能解决。
>如果实在无法解决，请使用第二个下载安装方法——直接安装Codex安装包

3. 此时已经开始下载，等待进度条结束就成功了
 ![](/static/image/docs/codex/codex-step-04-1520.bca04fa27f61.webp)


#### 微软安装包下载
Codex APP Windows版本默认只能通过微软商店进行安装，由于不同电脑的设置、网络等差异，可能会导致大家在使用微软商店的时候遇到各种各样的错误，要么报错要么一直重试然后安装失败，今天就教大家怎么直接安装Codex，不经过微软商店。

1. 打开网站：[https://store.rg-adguard.net](https://store.rg-adguard.net/) ，下拉框选择Productld，输入微软产品代码 `9PLM9XGG6VKS`，右边选择RP，点击勾
2. 下载 [OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix](http://tlu.dl.delivery.mp.microsoft.com/filestreamingservice/files/19f8e311-2a5f-4e79-9b4d-a169d4738ec1?P1=1792170754&P2=404&P3=2&P4=dq7RKlIMhJNj8CkemOXb3hET3P7vxlUbMQnRM7OjgyT%2bhY%2fEWXE50jdfVNOX3mGNkjTltHG813ZfE%2f3jx92Y3Q%3d%3d) 这个版本()，下载下来之后就是msi直接安装包，直接安装即可
![](/static/image/docs/codex/codex-step-05-1520.a00596234f9c.webp)


#### 桌面图标
>[!tip] 为什么Windows版ChatGPT没有桌面图标？
>因为图标在微软应用市场里面！平时使用需要额外搜索
>目前 Windows 中的 Codex 和 ChatGPT 都只能在微软商店中下载，默认也就只能从微软应用商城打开！

1. 同时按住Win+R，打开资源管理器
2. 输入`shell:AppsFolder`（如下图），然后按回车

   ![](/static/image/docs/codex/codex-step-06-1520.96101482ae98.webp)


3. 在打开的页面中，找到ChatGPT图标，右键单单击，点击创建快捷方式
![](/static/image/docs/codex/codex-step-07-1520.f2a43e5048db.webp)

### Mac版
Mac版本的下载就简单多了
1. 访问OpenAI官网下载ChatGPT安装器，前往[Codex官网：https://openai.com/zh-Hans-CN/codex/](https://openai.com/zh-Hans-CN/codex/)、[ChatGPT桌面客户端官网：https://chatgpt.com/zh-Hans-CN/download/](https://chatgpt.com/zh-Hans-CN/download/)均可下载

> [!tip] 注意
Codex和ChatGPT客户端已经合二为一，Codex注重编程能力，ChatGPT桌面客户端注重日常办公，用户可自行切换模式

2. 下载完成后，双击文件，将ChatGPT/Codex拖入Applications里面，即可完成安装
![](/static/image/docs/codex/codex-step-08-1520.cd4f3583f98e.webp)

## ChatGPT登录
### 1.ChatGPT官方账号登录
直接点击继续登录即可![](/static/image/docs/codex/codex-step-09-1520.ee21d5f06959.webp)
### 2.ChatGPT官方API登录
点击下方使用其他方式登录，然后输入OpenAI官方API密钥即可
![](/static/image/docs/codex/codex-step-09-1520.ee21d5f06959.webp)
### 3.第三方API、中转站登录
1. 点击使用其他方式登录，然后胡乱输入一些乱七八糟的
![](/static/image/docs/codex/codex-step-09-1520.ee21d5f06959.webp)
![](/static/image/docs/codex/codex-step-10-1520.c3af45bacd6f.webp)

2. 下载CC Switch（一款管理各种主流智能体配置的插件）或者自行修改config.toml，详情看下文《ChatGPT配置》

## ChatGPT配置第三方API

可选择使用 CC Switch 一键配置，或手动编辑 `auth.json` 和 `config.toml` 接入 Codex。

### 1. 使用 CC Switch 一键导入

CC Switch 可以导入 API Key、所选模型和 Codex 服务地址，无需手动编辑 TOML。
####  1.1 安装cc switch
从 [CC Switch 发布页](https://github.com/farion1231/cc-switch/releases) 下载并安装应用

   ![](/static/image/docs/codex/codex-step-11-1520.e2c09208f339.webp)

#### 1.2 创建密钥
   打开模型服务商/中转站的API key页面，创建密钥，并一键导入。（大部分中转站可以一键导入，少部分需要手动把密钥的base url 和 key自己放入cc switch的配置，手动导入参考[cc switch官方快速上手文档](https://ccswitch.io/zh/docs?section=getting-started&item=quickstart)）

> [!tip] 注意
> 第三方API需要支持Response协议，一般中转站/模型厂商都会支持，只有Claude官方可能不支持

![](/static/image/docs/codex/codex-step-12-1520.398671127324.webp)


#### 1.3 导入前配置
选择 Codex，再选择当前 API Key 可用的主模型，并保留自动生成的 `/v1` 地址。

名称：配置名
主模型：新开窗口默认的模型，可以后期自己调整

![](/static/image/docs/codex/codex-step-13-1520.4fdb3368c4e1.webp)

#### 1.4 CC Switch侧确认配置
确认浏览器打开链接的提示，然后在 CC Switch 中检查、保存并启用导入的服务商。
![](/static/image/docs/codex/codex-step-14-1520.55261981cb3a.webp)

#### 1.5 重启客户端
![](/static/image/docs/codex/codex-step-15-1520.e69f3ad511b4.webp)

导入内容包含所选 API Key、主模型和以 `/v1` 结尾的服务地址。请只在自己的设备上确认导入。
### 手动导入CC Switch
#### 2.1 添加供应商
点击添加供应商
![](/static/image/docs/codex/codex-step-16-1520.4fd00ac230fa.webp)

然后点击自定义配置
![](/static/image/docs/codex/codex-step-17-1520.271cb69534cd.webp)

#### 2.2 自定义配置
根据提示进行填写，重点是API Key和API请求地址不要填写错误
![](/static/image/docs/codex/codex-step-18-1520.7d1d1491d42f.webp)

### 3. 编辑 `auth.json` 和 `config.toml`

Codex 从用户目录中的 `auth.json` 读取认证凭据，从 `config.toml` 读取服务商、模型和接口设置。

| 系统 | `auth.json` | `config.toml` |
| --- | --- | --- |
| Windows | `%USERPROFILE%\.codex\auth.json` | `%USERPROFILE%\.codex\config.toml` |
| macOS / Linux | `~/.codex/auth.json` | `~/.codex/config.toml` |

#### 3.1编辑 `auth.json`
在用户级 `auth.json` 中写入 API Key。如果文件中已有其他认证字段，只更新 `OPENAI_API_KEY`，不要覆盖整个文件。
```json
{
  "OPENAI_API_KEY": "sk-your-api-key"
}
```

文件模式下的 `auth.json` 包含明文密钥，请只保存在自己的设备上，不要提交到 Git 或发送给他人。密钥保存在 `auth.json`；`config.toml` 只用于服务商、模型和接口设置，不要依赖当前终端的临时设置。

#### 3.2编辑 `config.toml`

保留现有 Codex 配置，并合并下面的服务商配置块。已有 `cli_auth_credentials_store` 时将它改为 `"file"`，不要重复添加。按 API Key 的可用模型替换示例模型。

```toml
cli_auth_credentials_store = "file"
model = "gpt-5.6-sol"
model_provider = "OpenAI"

[model_providers.OpenAI]
name = "All Token API"
base_url = "https://alltokenapi.com/v1"
requires_openai_auth = true
wire_api = "responses"
```

> [!tip] 会话历史相关问题
如果原有对话历史没有显示，检查当前 `config.toml` 中的 `model_provider` 是否与创建对话时使用的值完全一致（包括大小写）。改回原值并重启 Codex 后，可运行 `codex resume` 查找历史；这通常不代表记录已被删除。

#### 3.3重启并验证

1. 保存 `auth.json` 和 `config.toml`，重启正在运行的 Codex 应用或 IDE 扩展。
2. 在终端运行 `codex` 并发起一个测试任务；认证信息由 `auth.json` 提供。
3. 如启动失败，运行 `codex doctor` 检查认证文件是否被读取，再检查模型名称和 Base URL。

```bash
codex
```

请保持 `wire_api = "responses"`，并确保 Base URL 以 `/v1` 结尾。`cli_auth_credentials_store = "file"` 让 Codex 使用 `auth.json` 保存和读取凭据；`requires_openai_auth = true` 让自定义服务商使用该凭据。

参考：[Codex 认证说明](https://developers.openai.com/codex/auth) · [Codex 配置参考](https://developers.openai.com/codex/config-reference)
