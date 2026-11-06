/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export const SMTP_EMAIL_TEMPLATE_DEFAULTS = {
  SMTPVerificationSubject: '{{system_name}}邮箱验证邮件',
  SMTPVerificationContent:
    '<p>您好，你正在进行{{system_name}}邮箱验证。</p><p>您的验证码为: <strong>{{code}}</strong></p><p>验证码 {{valid_minutes}} 分钟内有效，如果不是本人操作，请忽略。</p>',
  SMTPPasswordResetSubject: '{{system_name}}密码重置',
  SMTPPasswordResetContent:
    "<p>您好，你正在进行{{system_name}}密码重置。</p><p>点击 <a href='{{reset_link}}'>此处</a> 进行密码重置。</p><p>如果链接无法点击，请尝试点击下面的链接或将其复制到浏览器中打开：<br> {{reset_link}} </p><p>重置链接 {{valid_minutes}} 分钟内有效，如果不是本人操作，请忽略。</p>",
} as const
