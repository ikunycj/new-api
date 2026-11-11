# 翻译资料

| 文档 | 用途 |
| --- | --- |
| [中英术语表](glossary.md) | 核心概念、模型、计费、用户与渠道用语 |
| [法语术语表](glossary.fr.md) | 法语翻译参考 |
| [俄语术语表](glossary.ru.md) | 俄语翻译参考 |

前端翻译文件位于 [locales](../../web/default/src/i18n/locales/)。需要翻译的用户文案使用英文源字符串作为平面 JSON key，并通过 `t()` 引用。管理员专用内部界面按项目约定使用中文；普通用户、公共页面和共享组件仍需国际化。

修改前端文案时遵守 [前端约定](../../web/default/AGENTS.md) 和 [i18n 技能](../../.agents/skills/i18n-translate/SKILL.md)。术语表用于语言一致性参考，具体界面含义和字段契约以当前源码为准。
