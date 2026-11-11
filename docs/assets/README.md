# 文档源素材

本目录保留编辑和重新生成发布资源所需的源文件。它们不参与前端生产构建。

## CC Switch 原始截图

原始 PNG 集中在 [screenshots/cc-switch](screenshots/cc-switch/)。下表保留迁移稿的截图顺序，便于从源图重新生成 WebP：

| 步骤 | 原始文件 | 场景 |
| --- | --- | --- |
| 01 | [f8b8e36b0e2118aa95130e4404ddbc34.png](screenshots/cc-switch/f8b8e36b0e2118aa95130e4404ddbc34.png) | 下载安装包 |
| 02 | [180885b0db3fe0723f100047bd46f02b.png](screenshots/cc-switch/180885b0db3fe0723f100047bd46f02b.png) | 创建 API Key |
| 03 | [60dee7d0d4a8f8e7cf99e141c1b1362c.png](screenshots/cc-switch/60dee7d0d4a8f8e7cf99e141c1b1362c.png) | 填写密钥信息 |
| 04 | [c2631bec9d3d4b4ed7357054457051fb.png](screenshots/cc-switch/c2631bec9d3d4b4ed7357054457051fb.png) | 打开导入入口 |
| 05 | [ac4a9712e7a58f1ac13208986fe4bae7.png](screenshots/cc-switch/ac4a9712e7a58f1ac13208986fe4bae7.png) | 导入对话框 |
| 06 | [74e4ee8dd6f109c13a50a9c506d47ccb.png](screenshots/cc-switch/74e4ee8dd6f109c13a50a9c506d47ccb.png) | 确认导入 |
| 07 | [88840668e7792d0e7f2ab7e7344dbbc6.png](screenshots/cc-switch/88840668e7792d0e7f2ab7e7344dbbc6.png) | 启用配置 |
| 08 | [f5de7f56519a46a0f159ca9b39f9578d.png](screenshots/cc-switch/f5de7f56519a46a0f159ca9b39f9578d.png) | 添加供应商 |
| 09 | [c0d0455350ac703a01c537b4d6960468.png](screenshots/cc-switch/c0d0455350ac703a01c537b4d6960468.png) | 选择类型 |
| 10 | [7baccf19a4e37a80731edcde5bcb7368.png](screenshots/cc-switch/7baccf19a4e37a80731edcde5bcb7368.png) | 填写模型配置 |
| 11 | [3ecbd61ee8ef9c1e5aba8a1373e751c2.png](screenshots/cc-switch/3ecbd61ee8ef9c1e5aba8a1373e751c2.png) | 启用供应商 |

页面采用 760px、1520px 的带哈希 WebP，路径、尺寸及说明以 [cc-switch-screenshots.ts](../../web/default/src/features/docs/components/cc-switch-screenshots.ts) 为准，发布资源在 [public/static/image/docs/cc-switch](../../web/default/public/static/image/docs/cc-switch/)。修改源图后同步重新生成发布图和 TSX 引用，具体约定见 [前端文档维护](../development/frontend-docs.md)。
