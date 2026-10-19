import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

const newKeys = {
  en: {
    '一直重连，等待网络': 'Constantly reconnecting, waiting for network',
    用户网络故障: 'User network failure',
    检查网络是否连通: 'Check whether the network is connected',
    'Authentication email templates': 'Authentication email templates',
    'Available placeholders': 'Available placeholders',
    'Verification email subject': 'Verification email subject',
    'Verification email body': 'Verification email body',
    'Password reset email subject': 'Password reset email subject',
    'Password reset email body': 'Password reset email body',
    'Cumulative:': 'Cumulative:',
    'Today:': 'Today:',
    'If you do not receive the email, please check your spam folder':
      'If you do not receive the email, please check your spam folder',
    'Choose which fields follow global settings and which fields use this user-specific strategy.':
      'Choose which fields follow global settings and which fields use this user-specific strategy.',
    'Configure {{username}}': 'Configure {{username}}',
    'Failed to reset settings': 'Failed to reset settings',
    'Failed to save settings': 'Failed to save settings',
    'No personalized settings': 'No personalized settings',
    '{{count}} customized fields': '{{count}} customized fields',
    'The new rule applies to future reward events for this user. Existing rewards and unlock times will not be recalculated.':
      'The new rule applies to future reward events for this user. Existing rewards and unlock times will not be recalculated.',
    'Customer Service': 'Customer Service',
    'Enter customer service contact information':
      'Enter customer service contact information',
    'Information displayed to users for contacting customer service':
      'Information displayed to users for contacting customer service',
    QPS: 'QPS',
    'TPS (thousands)': 'TPS (thousands)',
    'Throughput trend': 'Throughput trend',
    Range: 'Range',
    Yesterday: 'Yesterday',
    'This Week': 'This Week',
    'This Month': 'This Month',
    Granularity: 'Granularity',
    'By Token': 'By Token',
    'By Cost': 'By Cost',
    'Select the start and end time for the dashboard.':
      'Select the start and end time for the dashboard.',
    Apply: 'Apply',
    'Current concurrency': 'Current concurrency',
    'Current balance': 'Current balance',
    "Today's balance usage": "Today's balance usage",
    "Today's usage": "Today's usage",
    'Historical total consumed': 'Historical total consumed',
    'Token Usage': 'Token Usage',
    enabled: 'enabled',
    total: 'total',
    'Error self-check guide': 'Error self-check guide',
    'Official OpenAI errors': 'Official OpenAI errors',
    'User-side errors': 'User-side errors',
    'Relay errors': 'Relay errors',
    'Other errors': 'Other errors',
    'The response stream was interrupted unexpectedly.':
      'The response stream was interrupted unexpectedly.',
    'OpenAI is temporarily unable to allocate enough compute capacity for the selected model.':
      'OpenAI is temporarily unable to allocate enough compute capacity for the selected model.',
    'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.':
      'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.',
    'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.':
      'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.',
    'Please select at least one model billing group':
      'Please select at least one model billing group',
    Order: 'Order',
    '24 hours': '24 hours',
    '7 days': '7 days',
    '30 days': '30 days',
    'Actual cost': 'Actual cost',
    'Actual cost (USD)': 'Actual cost (USD)',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.',
    'Average latency': 'Average latency',
    'Channel Reconciliation': 'Channel Reconciliation',
    'Cost entries': 'Cost entries',
    'Cost entry saved': 'Cost entry saved',
    Daily: 'Daily',
    'Daily cost trend': 'Daily cost trend',
    'Estimate variance': 'Estimate variance',
    'Estimated cost': 'Estimated cost',
    'Gross margin': 'Gross margin',
    'Group Management': 'Group Management',
    'Inbound endpoints': 'Inbound endpoints',
    Loading: 'Loading',
    Models: 'Models',
    'No cost entries': 'No cost entries',
    'Record cost': 'Record cost',
    Requests: 'Requests',
    Source: 'Source',
    Start: 'Start',
    End: 'End',
    Tokens: 'Tokens',
    'Upstream endpoints': 'Upstream endpoints',
    'Usage breakdown': 'Usage breakdown',
    'User charge': 'User charge',
    'Auto-disabled probe interval': 'Auto-disabled probe interval',
    'Probe interval': 'Probe interval',
    'Upstream max retries': 'Upstream max retries',
    'Channel price multiplier': 'Channel price multiplier',
    'Daily Cost': 'Daily Cost',
    'Monthly Cost': 'Monthly Cost',
    'Daily Cost / Monthly Cost': 'Daily Cost / Monthly Cost',
    'Daily Usage': 'Daily Usage',
    'Monthly Usage': 'Monthly Usage',
    'Daily Usage / Monthly Usage': 'Daily Usage / Monthly Usage',
    'Price multiplier mode': 'Price multiplier mode',
    'USD-equivalent': 'USD-equivalent',
    CNY: 'CNY',
    'Probe failure auto-ban': 'Probe failure auto-ban',
    'Probe success auto-enable': 'Probe success auto-enable',
    'Force priority': 'Force priority',
    'Priority: High to Low': 'Priority: High to Low',
    'Priority: Low to High': 'Priority: Low to High',
    'Force priority scope': 'Force priority scope',
    'Current group only': 'Current group only',
    'Across selected groups': 'Across selected groups',
    'Previous-day probe success rate': 'Previous-day probe success rate',
    'Previous-day average TTFT': 'Previous-day average TTFT',
    TTFT: 'TTFT',
    'Recent test': 'Recent test',
    'Yesterday average': 'Yesterday average',
    'Automatic probe': 'Automatic probe',
    'The system probe task scans the task queue every 60 seconds.':
      'The system probe task scans the task queue every 60 seconds.',
    'Automatically probe this channel in the background':
      'Automatically probe this channel in the background',
    'You can still edit non-sensitive operations fields such as models, groups, weight, and routing settings.':
      'You can still edit non-sensitive operations fields such as models, groups, weight, and routing settings.',
    'Pricing groups that can access channels with this tag':
      'Pricing groups that can access channels with this tag',
    'Randomly select a key from the configured set for each request':
      'Randomly select a key from the configured set for each request',
    'Select pricing groups that can access this channel.':
      'Select pricing groups that can access this channel.',
    'Pricing groups that can access this channel.':
      'Pricing groups that can access this channel.',
    'Interval for probing enabled channels, in seconds':
      'Interval for probing enabled channels, in seconds',
    'Interval for probing auto-disabled channels, in seconds':
      'Interval for probing auto-disabled channels, in seconds',
    'Automatically disable the channel when a probe fails':
      'Automatically disable the channel when a probe fails',
    'Automatically enable the channel after a successful probe':
      'Automatically enable the channel after a successful probe',
    'Maximum retries for this channel after the first upstream attempt':
      'Maximum retries for this channel after the first upstream attempt',
    'Relative upstream cost used for channel ranking. 1 means unchanged.':
      'Relative upstream cost used for channel ranking. 1 means unchanged.',
    'Currency used when comparing this channel price multiplier':
      'Currency used when comparing this channel price multiplier',
    'Place this channel before ordinary channels in its selected scope':
      'Place this channel before ordinary channels in its selected scope',
    'Read-only success rate from the previous natural day':
      'Read-only success rate from the previous natural day',
    'Choose whether force priority applies within one group or across groups':
      'Choose whether force priority applies within one group or across groups',
    'ID (Default)': 'ID (Default)',
    'Select a date before clearing aggregation data.':
      'Select a date before clearing aggregation data.',
    'Failed to clean aggregation data': 'Failed to clean aggregation data',
    '{{count}} aggregation rows removed.':
      '{{count}} aggregation rows removed.',
    'No aggregation rows matched the selected time.':
      'No aggregation rows matched the selected time.',
    'Clean aggregation data': 'Clean aggregation data',
    'Confirm aggregation cleanup': 'Confirm aggregation cleanup',
    'This will permanently remove aggregation buckets before {{date}}. Source logs are not affected.':
      'This will permanently remove aggregation buckets before {{date}}. Source logs are not affected.',
    'This will permanently remove aggregation buckets before the selected timestamp. Source logs are not affected.':
      'This will permanently remove aggregation buckets before the selected timestamp. Source logs are not affected.',
    'Delete aggregation data': 'Delete aggregation data',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?',
    'OpenAI Responses to Anthropic Messages':
      'OpenAI Responses to Anthropic Messages',
    'Batch IQ test': 'Batch IQ test',
    'IQ Test': 'IQ Test',
    'Run IQ test': 'Run IQ test',
    'View result': 'View result',
    'IQ test running': 'IQ test running',
    'IQ test completed: {{success}} succeeded, {{failed}} failed':
      'IQ test completed: {{success}} succeeded, {{failed}} failed',
    'IQ test completed: {{count}} succeeded':
      'IQ test completed: {{count}} succeeded',
    'IQ test stopped: {{completed}}/{{total}} completed, {{success}} succeeded, {{failed}} failed':
      'IQ test stopped: {{completed}}/{{total}} completed, {{success}} succeeded, {{failed}} failed',
    'IQ test result': 'IQ test result',
    'IQ test result view': 'IQ test result view',
    'Copy source': 'Copy source',
    'No result available': 'No result available',
    'Result not retained; run again': 'Result not retained; run again',
  },
  zh: {
    'Automatically disable channels when tests fail':
      '当对渠道的批量测试失败/普通请求失败时，自动禁用对应渠道',
    'Authentication email templates': '认证邮件模板',
    'Available placeholders': '可用占位符',
    'Verification email subject': '邮箱验证码邮件主题',
    'Verification email body': '邮箱验证码邮件正文',
    'Password reset email subject': '密码重置邮件主题',
    'Password reset email body': '密码重置邮件正文',
    'Cumulative:': '累计：',
    'Today:': '今日：',
    'If you do not receive the email, please check your spam folder':
      '若未收到邮件，请查看垃圾邮箱',
    'Choose which fields follow global settings and which fields use this user-specific strategy.':
      '选择跟随全局设置的字段，以及使用该用户独立策略的字段。',
    'Configure {{username}}': '配置 {{username}}',
    'Failed to reset settings': '重置设置失败',
    'Failed to save settings': '保存设置失败',
    'No personalized settings': '未设置个性化配置',
    '{{count}} customized fields': '已个性化 {{count}} 个字段',
    'The new rule applies to future reward events for this user. Existing rewards and unlock times will not be recalculated.':
      '新规则将应用于该用户的后续奖励事件。已生成的奖励和解冻时间不会重新计算。',
    'Customer Service': '客服信息',
    'Enter customer service contact information': '输入客服联系方式',
    'Information displayed to users for contacting customer service':
      '向用户展示的客服联系方式',
    'Copy ready-to-run curl': '复制命令到本地终端测试',
    QPS: 'QPS',
    'TPS (thousands)': 'TPS（千）',
    'Throughput trend': '吞吐趋势',
    Range: '范围',
    Yesterday: '昨天',
    'This Week': '本周',
    'This Month': '本月',
    Granularity: '粒度',
    'By Token': '按 Token',
    'By Cost': '按费用',
    'Select the start and end time for the dashboard.':
      '选择数据看板的开始和结束时间。',
    Apply: '应用',
    'Current concurrency': '当前并发',
    'Current balance': '当前余额',
    "Today's balance usage": '今日余额消耗',
    "Today's usage": '今日用量',
    'Historical total consumed': '历史总计消耗',
    'Token Usage': 'Token 使用情况',
    enabled: '启用',
    total: '总计',
    'Error self-check guide': '错误自查指南',
    'Official OpenAI errors': 'OpenAI 官方错误',
    'User-side errors': '用户自身错误',
    'Relay errors': '中转站错误',
    'Other errors': '其他错误',
    'The response stream was interrupted unexpectedly.': '响应流意外中断。',
    'OpenAI is temporarily unable to allocate enough compute capacity for the selected model.':
      'OpenAI 暂时无法为所选模型分配足够的算力。',
    'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.':
      '如果希望保留缓存命中率，请持续重试请求，等待 OpenAI 通过排队分配算力。',
    'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.':
      '切换到新对话或新分组，让后台有机会选择其他账号。不同账号可能位于不同区域，各区域的算力紧张程度可能不同，这样做可能会有所好转，也可能仍会遇到同样的算力不足。',
    'Please select at least one model billing group':
      '请选择至少一个模型计费分组',
    Order: '顺序',
    '24 hours': '24 小时',
    '7 days': '7 天',
    '30 days': '30 天',
    'Actual cost': '实际成本',
    'Actual cost (USD)': '实际成本（USD）',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      '实际成本由管理员手工录入，并按所选时间段分摊。估算成本由网关根据计费快照计算。',
    'Average latency': '平均延迟',
    'Channel Reconciliation': '渠道对账',
    'Cost entries': '成本账期',
    'Cost entry saved': '成本记录已保存',
    Daily: '每日',
    'Daily cost trend': '每日成本趋势',
    'Estimate variance': '估算差额',
    'Estimated cost': '估算成本',
    'Gross margin': '毛利差额',
    'Group Management': '分组管理',
    'Inbound endpoints': '入站端点',
    Loading: '加载中',
    Models: '模型',
    'No cost entries': '暂无成本账期',
    'Record cost': '录入成本',
    Requests: '请求数',
    Source: '来源',
    Start: '开始',
    End: '结束',
    Tokens: 'Token 数',
    'Upstream endpoints': '上游端点',
    'Usage breakdown': '使用明细',
    'User charge': '用户扣费',
    'Auto-disabled probe interval': '自动禁用渠道探测间隔',
    'Probe interval': '探测间隔',
    'Upstream max retries': '上游最大重试次数',
    'Channel price multiplier': '渠道价格倍率',
    'Daily Cost': '日成本',
    'Monthly Cost': '月成本',
    'Daily Cost / Monthly Cost': '日成本 / 月成本',
    'Daily Usage': '日用量',
    'Monthly Usage': '月用量',
    'Daily Usage / Monthly Usage': '日用量 / 月用量',
    'Price multiplier mode': '价格倍率模式',
    'USD-equivalent': '美元等值',
    CNY: '人民币',
    'Probe failure auto-ban': '探测失败自动禁用',
    'Probe success auto-enable': '探测成功自动启用',
    'Force priority': '强制优先',
    'Priority: High to Low': '优先级：从高到低',
    'Priority: Low to High': '优先级：从低到高',
    'Force priority scope': '强制优先范围',
    'Current group only': '仅当前分组',
    'Across selected groups': '跨所选分组',
    'Previous-day probe success rate': '昨日成功率',
    'Previous-day average TTFT': '昨日平均 TTFT',
    TTFT: 'TTFT',
    'Recent test': '最近测试',
    'Yesterday average': '昨日平均',
    'Automatic probe': '自动探测',
    'The system probe task scans the task queue every 60 seconds.':
      '系统探测任务每60秒扫描一次任务队列',
    'Automatically probe this channel in the background':
      '在后台自动探测此渠道',
    'You can still edit non-sensitive operations fields such as models, groups, weight, and routing settings.':
      '仍可编辑模型、分组、权重和路由设置等非敏感操作字段。',
    'Pricing groups that can access channels with this tag':
      '可访问此标签渠道的定价分组',
    'Randomly select a key from the configured set for each request':
      '每次请求从已配置的密钥集合中随机选择一个',
    'Select pricing groups that can access this channel.':
      '选择可访问此渠道的定价分组。',
    'Pricing groups that can access this channel.': '可访问此渠道的定价分组。',
    'Interval for probing enabled channels, in seconds':
      '启用渠道的探测间隔，单位为秒',
    'Interval for probing auto-disabled channels, in seconds':
      '自动禁用渠道的探测间隔，单位为秒',
    'Automatically disable the channel when a probe fails':
      '探测失败时自动禁用渠道',
    'Automatically enable the channel after a successful probe':
      '探测成功后自动启用渠道',
    'Maximum retries for this channel after the first upstream attempt':
      '首次上游请求后的最大重试次数',
    'Relative upstream cost used for channel ranking. 1 means unchanged.':
      '用于渠道排序的相对上游成本，1 表示不变。',
    'Currency used when comparing this channel price multiplier':
      '比较渠道价格倍率时使用的货币',
    'Place this channel before ordinary channels in its selected scope':
      '在所选范围内将此渠道置于普通渠道之前',
    'Read-only success rate from the previous natural day':
      '前一自然日的只读探测成功率',
    'Choose whether force priority applies within one group or across groups':
      '选择强制优先仅适用于一个分组还是跨分组',
    'ID (Default)': 'ID（默认）',
    'Select a date before clearing aggregation data.':
      '清理聚合数据前请选择日期。',
    'Failed to clean aggregation data': '清理聚合数据失败',
    '{{count}} aggregation rows removed.': '已删除 {{count}} 条聚合记录。',
    'No aggregation rows matched the selected time.':
      '没有匹配所选时间的聚合记录。',
    'Clean aggregation data': '清理聚合数据',
    'Confirm aggregation cleanup': '确认清理聚合数据',
    'This will permanently remove aggregation buckets before {{date}}. Source logs are not affected.':
      '这将永久删除 {{date}} 之前的聚合数据，不会影响源日志。',
    'This will permanently remove aggregation buckets before the selected timestamp. Source logs are not affected.':
      '这将永久删除所选时间之前的聚合数据，不会影响源日志。',
    'Delete aggregation data': '删除聚合数据',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      '这会根据所有渠道配置重建渠道路由索引，包括支持的模型、分组和权重。重建期间路由可能短暂不完整。是否继续？',
    'OpenAI Responses to Anthropic Messages':
      'OpenAI Responses 到 Anthropic Messages',
    'Batch IQ test': '批量智商测试',
    'IQ Test': '智商测试',
    'Run IQ test': '智商测试',
    'View result': '查看结果',
    'IQ test running': '智商测试进行中',
    'IQ test completed: {{success}} succeeded, {{failed}} failed':
      '智商测试完成：{{success}} 个成功，{{failed}} 个失败',
    'IQ test completed: {{count}} succeeded': '智商测试完成：{{count}} 个成功',
    'IQ test stopped: {{completed}}/{{total}} completed, {{success}} succeeded, {{failed}} failed':
      '智商测试已停止：已完成 {{completed}}/{{total}}，{{success}} 个成功，{{failed}} 个失败',
    'IQ test result': '智商测试结果',
    'IQ test result view': '智商测试结果视图',
    'Copy source': '复制源码',
    'No result available': '暂无结果',
    'Result not retained; run again': '结果未保留，请重新测试',
  },
  fr: {
    'Authentication email templates': 'Modèles d’e-mails d’authentification',
    'Available placeholders': 'Variables disponibles',
    'Verification email subject': 'Objet de l’e-mail de vérification',
    'Verification email body': 'Corps de l’e-mail de vérification',
    'Password reset email subject':
      'Objet de l’e-mail de réinitialisation du mot de passe',
    'Password reset email body':
      'Corps de l’e-mail de réinitialisation du mot de passe',
    'Cumulative:': 'Cumulé :',
    'Today:': "Aujourd'hui :",
    'If you do not receive the email, please check your spam folder':
      "Si vous ne recevez pas l'e-mail, veuillez vérifier votre dossier de courrier indésirable",
    'Choose which fields follow global settings and which fields use this user-specific strategy.':
      'Choisissez les champs qui suivent les paramètres globaux et ceux qui utilisent cette stratégie utilisateur.',
    'Configure {{username}}': 'Configurer {{username}}',
    'Failed to reset settings': 'Échec de la réinitialisation des paramètres',
    'Failed to save settings': 'Échec de l’enregistrement des paramètres',
    'No personalized settings': 'Aucun paramètre personnalisé',
    '{{count}} customized fields': '{{count}} champs personnalisés',
    'The new rule applies to future reward events for this user. Existing rewards and unlock times will not be recalculated.':
      'La nouvelle règle s’applique aux futurs événements de récompense de cet utilisateur. Les récompenses et dates de déblocage existantes ne seront pas recalculées.',
    'Customer Service': 'Service client',
    'Enter customer service contact information':
      'Saisissez les coordonnées du service client',
    'Information displayed to users for contacting customer service':
      'Informations affichées aux utilisateurs pour contacter le service client',
    QPS: 'QPS',
    'TPS (thousands)': 'TPS (milliers)',
    'Throughput trend': 'Tendance du débit',
    Range: 'Période',
    Yesterday: 'Hier',
    'This Week': 'Cette semaine',
    'This Month': 'Ce mois-ci',
    Granularity: 'Granularité',
    'By Token': 'Par token',
    'By Cost': 'Par coût',
    'Select the start and end time for the dashboard.':
      'Sélectionnez les heures de début et de fin du tableau de bord.',
    Apply: 'Appliquer',
    'Current concurrency': 'Concurrence actuelle',
    'Current balance': 'Solde actuel',
    "Today's balance usage": 'Consommation du solde du jour',
    "Today's usage": 'Utilisation du jour',
    'Historical total consumed': 'Total historique consommé',
    'Token Usage': 'Utilisation des tokens',
    enabled: 'actives',
    total: 'total',
    'Error self-check guide': 'Guide d’auto-diagnostic des erreurs',
    'Official OpenAI errors': 'Erreurs officielles d’OpenAI',
    'User-side errors': 'Erreurs côté utilisateur',
    'Relay errors': 'Erreurs du relais',
    'Other errors': 'Autres erreurs',
    'The response stream was interrupted unexpectedly.':
      'Le flux de réponse a été interrompu de manière inattendue.',
    'OpenAI is temporarily unable to allocate enough compute capacity for the selected model.':
      'OpenAI ne peut temporairement pas allouer suffisamment de capacité de calcul au modèle sélectionné.',
    'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.':
      'Pour préserver le taux de cache hit, continuez à réessayer la requête et attendez que la file d’attente d’OpenAI attribue de la capacité.',
    'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.':
      'Démarrez une nouvelle conversation ou passez à un autre groupe afin que le backend puisse sélectionner un autre compte. Les comptes peuvent être servis depuis des régions différentes, où la pression sur la capacité varie ; cela peut aider, mais la même pénurie peut aussi se reproduire.',
    'Please select at least one model billing group':
      'Veuillez sélectionner au moins un groupe de facturation de modèles',
    Order: 'Ordre',
    '24 hours': '24 heures',
    '7 days': '7 jours',
    '30 days': '30 jours',
    'Actual cost': 'Coût réel',
    'Actual cost (USD)': 'Coût réel (USD)',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      'Les coûts réels sont saisis manuellement et répartis sur la période sélectionnée. Les coûts estimés sont calculés par la passerelle à partir des instantanés de facturation.',
    'Average latency': 'Latence moyenne',
    'Channel Reconciliation': 'Rapprochement du canal',
    'Cost entries': 'Périodes de coût',
    'Cost entry saved': 'Coût enregistré',
    Daily: 'Quotidien',
    'Daily cost trend': 'Tendance quotidienne des coûts',
    'Estimate variance': 'Écart estimé',
    'Estimated cost': 'Coût estimé',
    'Gross margin': 'Marge brute',
    'Group Management': 'Gestion des groupes',
    'Inbound endpoints': 'Points d’entrée',
    Loading: 'Chargement',
    Models: 'Modèles',
    'No cost entries': 'Aucune période de coût',
    'Record cost': 'Saisir un coût',
    Requests: 'Requêtes',
    Source: 'Source',
    Start: 'Début',
    End: 'Fin',
    Tokens: 'Tokens',
    'Upstream endpoints': 'Points de sortie',
    'Usage breakdown': 'Répartition de l’utilisation',
    'User charge': 'Facturation utilisateur',
    'Auto-disabled probe interval':
      'Intervalle de sonde des canaux désactivés automatiquement',
    'Probe interval': 'Intervalle de sonde',
    'Upstream max retries': 'Nombre maximal de tentatives amont',
    'Channel price multiplier': 'Multiplicateur de prix du canal',
    'Daily Cost': 'Coût quotidien',
    'Monthly Cost': 'Coût mensuel',
    'Daily Cost / Monthly Cost': 'Coût quotidien / Coût mensuel',
    'Daily Usage': 'Utilisation quotidienne',
    'Monthly Usage': 'Utilisation mensuelle',
    'Daily Usage / Monthly Usage':
      'Utilisation quotidienne / Utilisation mensuelle',
    'Price multiplier mode': 'Mode du multiplicateur de prix',
    'USD-equivalent': 'Équivalent USD',
    CNY: 'CNY',
    'Probe failure auto-ban': 'Désactivation automatique après échec de sonde',
    'Probe success auto-enable':
      'Activation automatique après réussite de sonde',
    'Force priority': 'Priorité forcée',
    'Priority: High to Low': 'Priorité : du plus élevé au plus bas',
    'Priority: Low to High': 'Priorité : du plus bas au plus élevé',
    'Force priority scope': 'Portée de la priorité forcée',
    'Current group only': 'Groupe actuel uniquement',
    'Across selected groups': 'Tous les groupes sélectionnés',
    'Previous-day probe success rate':
      'Taux de réussite des sondes du jour précédent',
    'Previous-day average TTFT': 'TTFT moyen du jour précédent',
    TTFT: 'TTFT',
    'Recent test': 'Dernier test',
    'Yesterday average': 'Moyenne d’hier',
    'Automatic probe': 'Sonde automatique',
    'The system probe task scans the task queue every 60 seconds.':
      'La tâche de sondage système analyse la file d’attente toutes les 60 secondes.',
    'Automatically probe this channel in the background':
      'Sonder automatiquement ce canal en arrière-plan',
    'You can still edit non-sensitive operations fields such as models, groups, weight, and routing settings.':
      'Vous pouvez toujours modifier les modèles, groupes, poids et paramètres de routage.',
    'Pricing groups that can access channels with this tag':
      'Groupes tarifaires pouvant accéder aux canaux de cette étiquette',
    'Randomly select a key from the configured set for each request':
      'Sélectionner aléatoirement une clé dans l’ensemble configuré pour chaque requête',
    'Select pricing groups that can access this channel.':
      'Sélectionnez les groupes tarifaires pouvant accéder à ce canal.',
    'Pricing groups that can access this channel.':
      'Groupes tarifaires pouvant accéder à ce canal.',
    'Interval for probing enabled channels, in seconds':
      'Intervalle de sonde des canaux actifs, en secondes',
    'Interval for probing auto-disabled channels, in seconds':
      'Intervalle de sonde des canaux désactivés automatiquement, en secondes',
    'Automatically disable the channel when a probe fails':
      'Désactiver automatiquement le canal après un échec de sonde',
    'Automatically enable the channel after a successful probe':
      'Activer automatiquement le canal après une sonde réussie',
    'Maximum retries for this channel after the first upstream attempt':
      'Nombre maximal de tentatives amont après la première requête',
    'Relative upstream cost used for channel ranking. 1 means unchanged.':
      'Coût amont relatif utilisé pour classer les canaux. 1 signifie inchangé.',
    'Currency used when comparing this channel price multiplier':
      'Devise utilisée pour comparer le multiplicateur de prix du canal',
    'Place this channel before ordinary channels in its selected scope':
      'Placer ce canal avant les canaux ordinaires dans la portée choisie',
    'Read-only success rate from the previous natural day':
      'Taux de réussite en lecture seule des sondes du jour précédent',
    'Choose whether force priority applies within one group or across groups':
      'Choisissez si la priorité forcée s’applique à un groupe ou à plusieurs groupes',
    'ID (Default)': 'ID (par défaut)',
    'Select a date before clearing aggregation data.':
      'Sélectionnez une date avant de supprimer les données agrégées.',
    'Failed to clean aggregation data':
      'Échec de la suppression des données agrégées',
    '{{count}} aggregation rows removed.':
      '{{count}} lignes agrégées supprimées.',
    'No aggregation rows matched the selected time.':
      'Aucune ligne agrégée ne correspond à la période sélectionnée.',
    'Clean aggregation data': 'Supprimer les données agrégées',
    'Confirm aggregation cleanup': 'Confirmer la suppression des agrégats',
    'This will permanently remove aggregation buckets before {{date}}. Source logs are not affected.':
      'Les agrégats antérieurs au {{date}} seront supprimés définitivement. Les journaux sources ne seront pas modifiés.',
    'This will permanently remove aggregation buckets before the selected timestamp. Source logs are not affected.':
      'Les agrégats antérieurs à l’horodatage sélectionné seront supprimés définitivement. Les journaux sources ne seront pas modifiés.',
    'Delete aggregation data': 'Supprimer les données agrégées',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Cette action reconstruit l’index de routage des canaux à partir de toutes les configurations, notamment les modèles pris en charge, les groupes et les poids. Le routage peut être brièvement incomplet pendant la reconstruction. Continuer ?',
    'OpenAI Responses to Anthropic Messages':
      'OpenAI Responses vers Anthropic Messages',
    'Batch IQ test': 'Test de QI en lot',
    'IQ Test': 'Test de QI',
    'Run IQ test': 'Lancer le test de QI',
    'View result': 'Voir le résultat',
    'IQ test running': 'Test de QI en cours',
    'IQ test completed: {{success}} succeeded, {{failed}} failed':
      'Test de QI terminé : {{success}} réussi(s), {{failed}} échec(s)',
    'IQ test completed: {{count}} succeeded':
      'Test de QI terminé : {{count}} réussi(s)',
    'IQ test stopped: {{completed}}/{{total}} completed, {{success}} succeeded, {{failed}} failed':
      'Test de QI arrêté : {{completed}}/{{total}} terminés, {{success}} réussis, {{failed}} échecs',
    'IQ test result': 'Résultat du test de QI',
    'IQ test result view': 'Affichage du résultat du test de QI',
    'Copy source': 'Copier le code source',
    'No result available': 'Aucun résultat disponible',
    'Result not retained; run again':
      'Résultat non conservé ; relancez le test',
  },
  ja: {
    'Authentication email templates': '認証メールテンプレート',
    'Available placeholders': '使用できるプレースホルダー',
    'Verification email subject': '認証メールの件名',
    'Verification email body': '認証メールの本文',
    'Password reset email subject': 'パスワードリセットメールの件名',
    'Password reset email body': 'パスワードリセットメールの本文',
    'Cumulative:': '累計：',
    'Today:': '今日：',
    'If you do not receive the email, please check your spam folder':
      'メールが届かない場合は、迷惑メールフォルダーをご確認ください',
    'Choose which fields follow global settings and which fields use this user-specific strategy.':
      'グローバル設定に従うフィールドと、このユーザー固有の戦略を使うフィールドを選択します。',
    'Configure {{username}}': '{{username}} を設定',
    'Failed to reset settings': '設定のリセットに失敗しました',
    'Failed to save settings': '設定の保存に失敗しました',
    'No personalized settings': '個別設定なし',
    '{{count}} customized fields': '{{count}} 個のカスタムフィールド',
    'The new rule applies to future reward events for this user. Existing rewards and unlock times will not be recalculated.':
      '新しいルールはこのユーザーの今後の報酬イベントに適用されます。既存の報酬と解除日時は再計算されません。',
    'Customer Service': 'カスタマーサービス',
    'Enter customer service contact information':
      'カスタマーサービスの連絡先を入力',
    'Information displayed to users for contacting customer service':
      'ユーザーに表示するカスタマーサービスの連絡先情報',
    QPS: 'QPS',
    'TPS (thousands)': 'TPS（千単位）',
    'Throughput trend': 'スループット推移',
    Range: '期間',
    Yesterday: '昨日',
    'This Week': '今週',
    'This Month': '今月',
    Granularity: '粒度',
    'By Token': 'Token 別',
    'By Cost': '費用別',
    'Select the start and end time for the dashboard.':
      'ダッシュボードの開始時刻と終了時刻を選択します。',
    Apply: '適用',
    'Current concurrency': '現在の同時実行数',
    'Current balance': '現在の残高',
    "Today's balance usage": '本日の残高消費',
    "Today's usage": '本日の使用量',
    'Historical total consumed': '過去の累計消費',
    'Token Usage': 'トークン使用状況',
    enabled: '有効',
    total: '合計',
    'Error self-check guide': 'エラー自己診断ガイド',
    'Official OpenAI errors': 'OpenAI公式エラー',
    'User-side errors': 'ユーザー側のエラー',
    'Relay errors': '中継エラー',
    'Other errors': 'その他のエラー',
    'The response stream was interrupted unexpectedly.':
      'レスポンスストリームが予期せず中断されました。',
    'OpenAI is temporarily unable to allocate enough compute capacity for the selected model.':
      'OpenAI は、選択したモデルに十分な計算リソースを一時的に割り当てられません。',
    'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.':
      'キャッシュヒット率を維持したい場合は、リクエストを繰り返し再試行し、OpenAI のキューによるリソース割り当てを待ってください。',
    'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.':
      '新しい会話を開始するか別のグループに切り替えると、バックエンドが別のアカウントを選択する可能性があります。アカウントごとに利用地域が異なる場合があり、地域によって混雑状況が違うため改善することもありますが、同じ容量不足が続くこともあります。',
    'Please select at least one model billing group':
      'モデル課金グループを1つ以上選択してください',
    Order: '順序',
    '24 hours': '24時間',
    '7 days': '7日間',
    '30 days': '30日間',
    'Actual cost': '実コスト',
    'Actual cost (USD)': '実コスト（USD）',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      '実コストは手動入力され、選択期間に按分されます。推定コストはゲートウェイの課金スナップショットから計算されます。',
    'Average latency': '平均レイテンシ',
    'Channel Reconciliation': 'チャネル照合',
    'Cost entries': 'コスト期間',
    'Cost entry saved': 'コストを保存しました',
    Daily: '日別',
    'Daily cost trend': '日別コスト推移',
    'Estimate variance': '推定差額',
    'Estimated cost': '推定コスト',
    'Gross margin': '粗利益差額',
    'Group Management': 'グループ管理',
    'Inbound endpoints': '受信エンドポイント',
    Loading: '読み込み中',
    Models: 'モデル',
    'No cost entries': 'コスト期間なし',
    'Record cost': 'コストを記録',
    Requests: 'リクエスト数',
    Source: 'ソース',
    Start: '開始',
    End: '終了',
    Tokens: 'トークン',
    'Upstream endpoints': '上流エンドポイント',
    'Usage breakdown': '利用内訳',
    'User charge': 'ユーザー請求額',
    'Auto-disabled probe interval': '自動無効チャネルのプローブ間隔',
    'Probe interval': 'プローブ間隔',
    'Upstream max retries': '上流の最大再試行回数',
    'Channel price multiplier': 'チャネル価格倍率',
    'Daily Cost': '日次コスト',
    'Monthly Cost': '月次コスト',
    'Daily Cost / Monthly Cost': '日次コスト / 月次コスト',
    'Daily Usage': '日次使用量',
    'Monthly Usage': '月次使用量',
    'Daily Usage / Monthly Usage': '日次使用量 / 月次使用量',
    'Price multiplier mode': '価格倍率モード',
    'USD-equivalent': 'USD 相当',
    CNY: 'CNY',
    'Probe failure auto-ban': 'プローブ失敗時に自動無効化',
    'Probe success auto-enable': 'プローブ成功時に自動有効化',
    'Force priority': '優先を強制',
    'Priority: High to Low': '優先度: 高い順',
    'Priority: Low to High': '優先度: 低い順',
    'Force priority scope': '強制優先の範囲',
    'Current group only': '現在のグループのみ',
    'Across selected groups': '選択したグループ全体',
    'Previous-day probe success rate': '前日のプローブ成功率',
    'Previous-day average TTFT': '前日の平均 TTFT',
    TTFT: 'TTFT',
    'Recent test': '最新テスト',
    'Yesterday average': '昨日の平均',
    'Automatic probe': '自動プローブ',
    'The system probe task scans the task queue every 60 seconds.':
      'システムのプローブタスクは60秒ごとにタスクキューをスキャンします。',
    'Automatically probe this channel in the background':
      'このチャネルをバックグラウンドで自動的にプローブする',
    'You can still edit non-sensitive operations fields such as models, groups, weight, and routing settings.':
      'モデル、グループ、重み、ルーティング設定などの非機密項目は編集できます。',
    'Pricing groups that can access channels with this tag':
      'このタグのチャネルにアクセスできる料金グループ',
    'Randomly select a key from the configured set for each request':
      'リクエストごとに設定済みのキーからランダムに選択',
    'Select pricing groups that can access this channel.':
      'このチャネルにアクセスできる料金グループを選択してください。',
    'Pricing groups that can access this channel.':
      'このチャネルにアクセスできる料金グループ。',
    'Interval for probing enabled channels, in seconds':
      '有効なチャネルを検査する間隔（秒）',
    'Interval for probing auto-disabled channels, in seconds':
      '自動無効チャネルを検査する間隔（秒）',
    'Automatically disable the channel when a probe fails':
      'プローブ失敗時にチャネルを自動無効化',
    'Automatically enable the channel after a successful probe':
      'プローブ成功後にチャネルを自動有効化',
    'Maximum retries for this channel after the first upstream attempt':
      '最初の上流リクエスト後の最大再試行回数',
    'Relative upstream cost used for channel ranking. 1 means unchanged.':
      'チャネル順位付けに使う相対上流コスト。1 は変更なし。',
    'Currency used when comparing this channel price multiplier':
      'チャネル価格倍率の比較に使用する通貨',
    'Place this channel before ordinary channels in its selected scope':
      '選択した範囲でこのチャネルを通常のチャネルより前に配置',
    'Read-only success rate from the previous natural day':
      '前日のプローブ成功率（読み取り専用）',
    'Choose whether force priority applies within one group or across groups':
      '強制優先を1つのグループ内だけに適用するか、グループ間に適用するかを選択',
    'ID (Default)': 'ID（デフォルト）',
    'Select a date before clearing aggregation data.':
      '集計データを削除する前に日付を選択してください。',
    'Failed to clean aggregation data': '集計データの削除に失敗しました',
    '{{count}} aggregation rows removed.':
      '{{count}} 件の集計データを削除しました。',
    'No aggregation rows matched the selected time.':
      '選択した期間に一致する集計データはありません。',
    'Clean aggregation data': '集計データを削除',
    'Confirm aggregation cleanup': '集計データの削除を確認',
    'This will permanently remove aggregation buckets before {{date}}. Source logs are not affected.':
      '{{date}} より前の集計データを完全に削除します。元のログには影響しません。',
    'This will permanently remove aggregation buckets before the selected timestamp. Source logs are not affected.':
      '選択した時刻より前の集計データを完全に削除します。元のログには影響しません。',
    'Delete aggregation data': '集計データを削除',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'すべてのチャネル設定からルーティングインデックスを再構築します。対応モデル、グループ、重みが含まれます。再構築中はルーティングが一時的に不完全になる可能性があります。続行しますか？',
    'OpenAI Responses to Anthropic Messages':
      'OpenAI Responses から Anthropic Messages',
    'Batch IQ test': '一括IQテスト',
    'IQ Test': 'IQテスト',
    'Run IQ test': 'IQテストを実行',
    'View result': '結果を表示',
    'IQ test running': 'IQテストを実行中',
    'IQ test completed: {{success}} succeeded, {{failed}} failed':
      'IQテスト完了: {{success}}件成功、{{failed}}件失敗',
    'IQ test completed: {{count}} succeeded': 'IQテスト完了: {{count}}件成功',
    'IQ test stopped: {{completed}}/{{total}} completed, {{success}} succeeded, {{failed}} failed':
      'IQテストを停止: {{completed}}/{{total}}件完了、{{success}}件成功、{{failed}}件失敗',
    'IQ test result': 'IQテスト結果',
    'IQ test result view': 'IQテスト結果の表示',
    'Copy source': 'ソースをコピー',
    'No result available': '結果がありません',
    'Result not retained; run again':
      '結果を保持できませんでした。もう一度実行してください',
  },
  ru: {
    'Authentication email templates': 'Шаблоны писем для аутентификации',
    'Available placeholders': 'Доступные подстановки',
    'Verification email subject': 'Тема письма с кодом подтверждения',
    'Verification email body': 'Текст письма с кодом подтверждения',
    'Password reset email subject': 'Тема письма для сброса пароля',
    'Password reset email body': 'Текст письма для сброса пароля',
    'Cumulative:': 'Накоплено:',
    'Today:': 'Сегодня:',
    'If you do not receive the email, please check your spam folder':
      'Если письмо не пришло, проверьте папку «Спам»',
    'Choose which fields follow global settings and which fields use this user-specific strategy.':
      'Выберите поля, которые следуют глобальным настройкам, и поля для этой индивидуальной стратегии.',
    'Configure {{username}}': 'Настроить {{username}}',
    'Failed to reset settings': 'Не удалось сбросить настройки',
    'Failed to save settings': 'Не удалось сохранить настройки',
    'No personalized settings': 'Индивидуальные настройки отсутствуют',
    '{{count}} customized fields': '{{count}} индивидуальных полей',
    'The new rule applies to future reward events for this user. Existing rewards and unlock times will not be recalculated.':
      'Новое правило применяется к будущим событиям наград для этого пользователя. Существующие награды и сроки разблокировки не пересчитываются.',
    'Customer Service': 'Служба поддержки',
    'Enter customer service contact information':
      'Введите контактную информацию службы поддержки',
    'Information displayed to users for contacting customer service':
      'Информация для пользователей о связи со службой поддержки',
    QPS: 'QPS',
    'TPS (thousands)': 'TPS (тыс.)',
    'Throughput trend': 'Тренд пропускной способности',
    Range: 'Период',
    Yesterday: 'Вчера',
    'This Week': 'Эта неделя',
    'This Month': 'Этот месяц',
    Granularity: 'Детализация',
    'By Token': 'По токенам',
    'By Cost': 'По стоимости',
    'Select the start and end time for the dashboard.':
      'Выберите время начала и окончания для панели.',
    Apply: 'Применить',
    'Current concurrency': 'Текущая параллельность',
    'Current balance': 'Текущий баланс',
    "Today's balance usage": 'Расход баланса за сегодня',
    "Today's usage": 'Использование за сегодня',
    'Historical total consumed': 'Всего израсходовано за историю',
    'Token Usage': 'Использование токенов',
    enabled: 'активных',
    total: 'всего',
    'Error self-check guide':
      'Руководство по самостоятельной диагностике ошибок',
    'Official OpenAI errors': 'Официальные ошибки OpenAI',
    'User-side errors': 'Ошибки на стороне пользователя',
    'Relay errors': 'Ошибки ретрансляции',
    'Other errors': 'Другие ошибки',
    'The response stream was interrupted unexpectedly.':
      'Поток ответа был неожиданно прерван.',
    'OpenAI is temporarily unable to allocate enough compute capacity for the selected model.':
      'OpenAI временно не может выделить достаточно вычислительных ресурсов для выбранной модели.',
    'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.':
      'Чтобы сохранить долю попаданий в кэш, продолжайте повторять запрос и дождитесь, пока очередь OpenAI выделит ресурсы.',
    'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.':
      'Начните новый диалог или переключитесь на другую группу, чтобы backend мог выбрать другую учётную запись. Учётные записи могут обслуживаться в разных регионах, где нагрузка на вычислительные ресурсы различается; это может помочь, но та же нехватка мощности может сохраниться.',
    'Please select at least one model billing group':
      'Пожалуйста, выберите хотя бы одну группу тарификации моделей',
    Order: 'Порядок',
    '24 hours': '24 часа',
    '7 days': '7 дней',
    '30 days': '30 дней',
    'Actual cost': 'Фактическая стоимость',
    'Actual cost (USD)': 'Фактическая стоимость (USD)',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      'Фактические затраты вводятся вручную и распределяются по выбранному периоду. Расчётная стоимость вычисляется шлюзом по снимкам биллинга.',
    'Average latency': 'Средняя задержка',
    'Channel Reconciliation': 'Сверка канала',
    'Cost entries': 'Периоды затрат',
    'Cost entry saved': 'Затраты сохранены',
    Daily: 'По дням',
    'Daily cost trend': 'Динамика затрат по дням',
    'Estimate variance': 'Отклонение оценки',
    'Estimated cost': 'Расчётная стоимость',
    'Gross margin': 'Валовая маржа',
    'Group Management': 'Управление группами',
    'Inbound endpoints': 'Входные точки',
    Loading: 'Загрузка',
    Models: 'Модели',
    'No cost entries': 'Нет периодов затрат',
    'Record cost': 'Записать затраты',
    Requests: 'Запросы',
    Source: 'Источник',
    Start: 'Начало',
    End: 'Конец',
    Tokens: 'Токены',
    'Upstream endpoints': 'Внешние точки',
    'Usage breakdown': 'Разбивка использования',
    'User charge': 'Списание с пользователя',
    'Auto-disabled probe interval':
      'Интервал проверки автоматически отключённых каналов',
    'Probe interval': 'Интервал проверки',
    'Upstream max retries': 'Максимум повторных попыток upstream',
    'Channel price multiplier': 'Ценовой множитель канала',
    'Daily Cost': 'Дневная стоимость',
    'Monthly Cost': 'Месячная стоимость',
    'Daily Cost / Monthly Cost': 'Дневная / месячная стоимость',
    'Daily Usage': 'Дневное потребление',
    'Monthly Usage': 'Месячное потребление',
    'Daily Usage / Monthly Usage': 'Дневное / месячное потребление',
    'Price multiplier mode': 'Режим ценового множителя',
    'USD-equivalent': 'Эквивалент USD',
    CNY: 'CNY',
    'Probe failure auto-ban': 'Автоотключение после сбоя проверки',
    'Probe success auto-enable': 'Автовключение после успешной проверки',
    'Force priority': 'Принудительный приоритет',
    'Priority: High to Low': 'Приоритет: от высокого к низкому',
    'Priority: Low to High': 'Приоритет: от низкого к высокому',
    'Force priority scope': 'Область принудительного приоритета',
    'Current group only': 'Только текущая группа',
    'Across selected groups': 'Во всех выбранных группах',
    'Previous-day probe success rate': 'Успешность проверок за предыдущий день',
    'Previous-day average TTFT': 'Средний TTFT за предыдущий день',
    TTFT: 'TTFT',
    'Recent test': 'Последний тест',
    'Yesterday average': 'Среднее за вчера',
    'Automatic probe': 'Автоматическая проверка',
    'The system probe task scans the task queue every 60 seconds.':
      'Системная задача проверки сканирует очередь задач каждые 60 секунд.',
    'Automatically probe this channel in the background':
      'Автоматически проверять этот канал в фоновом режиме',
    'You can still edit non-sensitive operations fields such as models, groups, weight, and routing settings.':
      'Можно редактировать модели, группы, веса и параметры маршрутизации.',
    'Pricing groups that can access channels with this tag':
      'Тарифные группы, которым доступны каналы с этой меткой',
    'Randomly select a key from the configured set for each request':
      'Для каждого запроса случайно выбирать ключ из настроенного набора',
    'Select pricing groups that can access this channel.':
      'Выберите тарифные группы, которым доступен этот канал.',
    'Pricing groups that can access this channel.':
      'Тарифные группы, которым доступен этот канал.',
    'Interval for probing enabled channels, in seconds':
      'Интервал проверки включённых каналов, в секундах',
    'Interval for probing auto-disabled channels, in seconds':
      'Интервал проверки автоматически отключённых каналов, в секундах',
    'Automatically disable the channel when a probe fails':
      'Автоматически отключать канал при сбое проверки',
    'Automatically enable the channel after a successful probe':
      'Автоматически включать канал после успешной проверки',
    'Maximum retries for this channel after the first upstream attempt':
      'Максимум повторных попыток после первого upstream-запроса',
    'Relative upstream cost used for channel ranking. 1 means unchanged.':
      'Относительная стоимость upstream для ранжирования каналов. 1 — без изменений.',
    'Currency used when comparing this channel price multiplier':
      'Валюта для сравнения ценового множителя канала',
    'Place this channel before ordinary channels in its selected scope':
      'Размещать этот канал перед обычными в выбранной области',
    'Read-only success rate from the previous natural day':
      'Успешность проверок за предыдущий день (только чтение)',
    'Choose whether force priority applies within one group or across groups':
      'Выберите, действует ли принудительный приоритет внутри одной группы или между группами',
    'ID (Default)': 'ID (по умолчанию)',
    'Select a date before clearing aggregation data.':
      'Выберите дату перед удалением агрегированных данных.',
    'Failed to clean aggregation data':
      'Не удалось удалить агрегированные данные',
    '{{count}} aggregation rows removed.':
      'Удалено агрегированных строк: {{count}}.',
    'No aggregation rows matched the selected time.':
      'За выбранный период агрегированные строки не найдены.',
    'Clean aggregation data': 'Удалить агрегированные данные',
    'Confirm aggregation cleanup': 'Подтвердите удаление агрегатов',
    'This will permanently remove aggregation buckets before {{date}}. Source logs are not affected.':
      'Агрегаты до {{date}} будут удалены навсегда. Исходные логи не изменятся.',
    'This will permanently remove aggregation buckets before the selected timestamp. Source logs are not affected.':
      'Агрегаты до выбранного времени будут удалены навсегда. Исходные логи не изменятся.',
    'Delete aggregation data': 'Удалить агрегированные данные',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Индекс маршрутизации каналов будет перестроен на основе всех конфигураций каналов, включая поддерживаемые модели, группы и веса. Во время перестроения маршрутизация может быть временно неполной. Продолжить?',
    'OpenAI Responses to Anthropic Messages':
      'OpenAI Responses в Anthropic Messages',
    'Batch IQ test': 'Пакетный IQ-тест',
    'IQ Test': 'IQ-тест',
    'Run IQ test': 'Запустить IQ-тест',
    'View result': 'Посмотреть результат',
    'IQ test running': 'Выполняется IQ-тест',
    'IQ test completed: {{success}} succeeded, {{failed}} failed':
      'IQ-тест завершен: успешно {{success}}, с ошибкой {{failed}}',
    'IQ test completed: {{count}} succeeded':
      'IQ-тест завершен: успешно {{count}}',
    'IQ test stopped: {{completed}}/{{total}} completed, {{success}} succeeded, {{failed}} failed':
      'IQ-тест остановлен: завершено {{completed}}/{{total}}, успешно {{success}}, с ошибкой {{failed}}',
    'IQ test result': 'Результат IQ-теста',
    'IQ test result view': 'Просмотр результата IQ-теста',
    'Copy source': 'Копировать исходный код',
    'No result available': 'Результат недоступен',
    'Result not retained; run again':
      'Результат не сохранён; запустите тест снова',
  },
  vi: {
    'Authentication email templates': 'Mẫu email xác thực',
    'Available placeholders': 'Biến thay thế khả dụng',
    'Verification email subject': 'Tiêu đề email xác minh',
    'Verification email body': 'Nội dung email xác minh',
    'Password reset email subject': 'Tiêu đề email đặt lại mật khẩu',
    'Password reset email body': 'Nội dung email đặt lại mật khẩu',
    'Cumulative:': 'Lũy kế:',
    'Today:': 'Hôm nay:',
    'If you do not receive the email, please check your spam folder':
      'Nếu bạn chưa nhận được email, hãy kiểm tra thư mục spam',
    'Choose which fields follow global settings and which fields use this user-specific strategy.':
      'Chọn các trường theo cài đặt chung và các trường dùng chiến lược riêng cho người dùng này.',
    'Configure {{username}}': 'Cấu hình {{username}}',
    'Failed to reset settings': 'Không thể đặt lại cài đặt',
    'Failed to save settings': 'Không thể lưu cài đặt',
    'No personalized settings': 'Chưa có cài đặt riêng',
    '{{count}} customized fields': '{{count}} trường được tùy chỉnh',
    'The new rule applies to future reward events for this user. Existing rewards and unlock times will not be recalculated.':
      'Quy tắc mới áp dụng cho các sự kiện thưởng trong tương lai của người dùng này. Phần thưởng và thời gian mở khóa hiện có sẽ không được tính lại.',
    'Customer Service': 'Chăm sóc khách hàng',
    'Enter customer service contact information':
      'Nhập thông tin liên hệ bộ phận chăm sóc khách hàng',
    'Information displayed to users for contacting customer service':
      'Thông tin hiển thị cho người dùng để liên hệ bộ phận chăm sóc khách hàng',
    QPS: 'QPS',
    'TPS (thousands)': 'TPS (nghìn)',
    'Throughput trend': 'Xu hướng thông lượng',
    Range: 'Phạm vi',
    Yesterday: 'Hôm qua',
    'This Week': 'Tuần này',
    'This Month': 'Tháng này',
    Granularity: 'Độ chi tiết',
    'By Token': 'Theo token',
    'By Cost': 'Theo chi phí',
    'Select the start and end time for the dashboard.':
      'Chọn thời gian bắt đầu và kết thúc cho bảng điều khiển.',
    Apply: 'Áp dụng',
    'Current concurrency': 'Số tác vụ đồng thời hiện tại',
    'Current balance': 'Số dư hiện tại',
    "Today's balance usage": 'Mức tiêu hao số dư hôm nay',
    "Today's usage": 'Mức sử dụng hôm nay',
    'Historical total consumed': 'Tổng đã tiêu thụ trong lịch sử',
    'Token Usage': 'Mức sử dụng token',
    enabled: 'đang bật',
    total: 'tổng',
    'Error self-check guide': 'Hướng dẫn tự kiểm tra lỗi',
    'Official OpenAI errors': 'Lỗi chính thức từ OpenAI',
    'User-side errors': 'Lỗi phía người dùng',
    'Relay errors': 'Lỗi chuyển tiếp',
    'Other errors': 'Lỗi khác',
    'The response stream was interrupted unexpectedly.':
      'Luồng phản hồi bị gián đoạn đột ngột.',
    'OpenAI is temporarily unable to allocate enough compute capacity for the selected model.':
      'OpenAI tạm thời không thể phân bổ đủ năng lực tính toán cho mô hình đã chọn.',
    'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.':
      'Nếu muốn giữ tỷ lệ cache hit, hãy tiếp tục thử lại yêu cầu và chờ hàng đợi của OpenAI phân bổ năng lực.',
    'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.':
      'Hãy bắt đầu cuộc trò chuyện mới hoặc chuyển sang nhóm khác để backend có thể chọn tài khoản khác. Các tài khoản có thể được phục vụ ở những khu vực khác nhau, nơi mức độ thiếu năng lực có thể khác nhau; cách này có thể cải thiện tình hình, nhưng cũng có thể vẫn gặp cùng lỗi thiếu năng lực.',
    'Please select at least one model billing group':
      'Vui lòng chọn ít nhất một nhóm tính phí mô hình',
    Order: 'Thứ tự',
    '24 hours': '24 giờ',
    '7 days': '7 ngày',
    '30 days': '30 ngày',
    'Actual cost': 'Chi phí thực tế',
    'Actual cost (USD)': 'Chi phí thực tế (USD)',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      'Chi phí thực tế được nhập thủ công và phân bổ theo khoảng thời gian đã chọn. Chi phí ước tính được cổng tính từ ảnh chụp dữ liệu thanh toán.',
    'Average latency': 'Độ trễ trung bình',
    'Channel Reconciliation': 'Đối soát kênh',
    'Cost entries': 'Kỳ chi phí',
    'Cost entry saved': 'Đã lưu chi phí',
    Daily: 'Theo ngày',
    'Daily cost trend': 'Xu hướng chi phí hằng ngày',
    'Estimate variance': 'Chênh lệch ước tính',
    'Estimated cost': 'Chi phí ước tính',
    'Gross margin': 'Chênh lệch lợi nhuận gộp',
    'Group Management': 'Quản lý nhóm',
    'Inbound endpoints': 'Điểm cuối đầu vào',
    Loading: 'Đang tải',
    Models: 'Mô hình',
    'No cost entries': 'Chưa có kỳ chi phí',
    'Record cost': 'Ghi nhận chi phí',
    Requests: 'Yêu cầu',
    Source: 'Nguồn',
    Start: 'Bắt đầu',
    End: 'Kết thúc',
    Tokens: 'Token',
    'Upstream endpoints': 'Điểm cuối upstream',
    'Usage breakdown': 'Phân tích sử dụng',
    'User charge': 'Phí người dùng',
    'Auto-disabled probe interval':
      'Khoảng thời gian kiểm tra kênh tự động tắt',
    'Probe interval': 'Khoảng thời gian kiểm tra',
    'Upstream max retries': 'Số lần thử lại upstream tối đa',
    'Channel price multiplier': 'Hệ số giá của kênh',
    'Daily Cost': 'Chi phí ngày',
    'Monthly Cost': 'Chi phí tháng',
    'Daily Cost / Monthly Cost': 'Chi phí ngày / Chi phí tháng',
    'Daily Usage': 'Mức sử dụng ngày',
    'Monthly Usage': 'Mức sử dụng tháng',
    'Daily Usage / Monthly Usage': 'Mức sử dụng ngày / Mức sử dụng tháng',
    'Price multiplier mode': 'Chế độ hệ số giá',
    'USD-equivalent': 'Tương đương USD',
    CNY: 'CNY',
    'Probe failure auto-ban': 'Tự động tắt khi kiểm tra thất bại',
    'Probe success auto-enable': 'Tự động bật khi kiểm tra thành công',
    'Force priority': 'Ưu tiên bắt buộc',
    'Priority: High to Low': 'Ưu tiên: Cao đến thấp',
    'Priority: Low to High': 'Ưu tiên: Thấp đến cao',
    'Force priority scope': 'Phạm vi ưu tiên bắt buộc',
    'Current group only': 'Chỉ nhóm hiện tại',
    'Across selected groups': 'Trên các nhóm đã chọn',
    'Previous-day probe success rate': 'Tỷ lệ kiểm tra thành công ngày trước',
    'Previous-day average TTFT': 'TTFT trung bình ngày trước',
    TTFT: 'TTFT',
    'Recent test': 'Lần kiểm tra gần nhất',
    'Yesterday average': 'Trung bình hôm qua',
    'Automatic probe': 'Kiểm tra tự động',
    'The system probe task scans the task queue every 60 seconds.':
      'Tác vụ kiểm tra hệ thống quét hàng đợi tác vụ mỗi 60 giây.',
    'Automatically probe this channel in the background':
      'Tự động dò kênh này trong nền',
    'You can still edit non-sensitive operations fields such as models, groups, weight, and routing settings.':
      'Bạn vẫn có thể sửa mô hình, nhóm, trọng số và cài đặt định tuyến.',
    'Pricing groups that can access channels with this tag':
      'Các nhóm giá có thể truy cập kênh với thẻ này',
    'Randomly select a key from the configured set for each request':
      'Chọn ngẫu nhiên một khóa từ tập đã cấu hình cho mỗi yêu cầu',
    'Select pricing groups that can access this channel.':
      'Chọn các nhóm giá có thể truy cập kênh này.',
    'Pricing groups that can access this channel.':
      'Các nhóm giá có thể truy cập kênh này.',
    'Interval for probing enabled channels, in seconds':
      'Khoảng thời gian kiểm tra kênh đang bật, tính bằng giây',
    'Interval for probing auto-disabled channels, in seconds':
      'Khoảng thời gian kiểm tra kênh tự động tắt, tính bằng giây',
    'Automatically disable the channel when a probe fails':
      'Tự động tắt kênh khi kiểm tra thất bại',
    'Automatically enable the channel after a successful probe':
      'Tự động bật kênh sau khi kiểm tra thành công',
    'Maximum retries for this channel after the first upstream attempt':
      'Số lần thử lại tối đa của kênh sau lần gọi upstream đầu tiên',
    'Relative upstream cost used for channel ranking. 1 means unchanged.':
      'Chi phí upstream tương đối dùng để xếp hạng kênh. 1 nghĩa là không đổi.',
    'Currency used when comparing this channel price multiplier':
      'Đơn vị tiền tệ dùng khi so sánh hệ số giá của kênh',
    'Place this channel before ordinary channels in its selected scope':
      'Đặt kênh này trước các kênh thông thường trong phạm vi đã chọn',
    'Read-only success rate from the previous natural day':
      'Tỷ lệ kiểm tra thành công của ngày tự nhiên trước (chỉ đọc)',
    'Choose whether force priority applies within one group or across groups':
      'Chọn áp dụng ưu tiên bắt buộc trong một nhóm hay trên nhiều nhóm',
    'ID (Default)': 'ID (mặc định)',
    'Select a date before clearing aggregation data.':
      'Hãy chọn ngày trước khi xóa dữ liệu tổng hợp.',
    'Failed to clean aggregation data': 'Không thể xóa dữ liệu tổng hợp',
    '{{count}} aggregation rows removed.': 'Đã xóa {{count}} dòng tổng hợp.',
    'No aggregation rows matched the selected time.':
      'Không có dòng tổng hợp nào trong khoảng thời gian đã chọn.',
    'Clean aggregation data': 'Xóa dữ liệu tổng hợp',
    'Confirm aggregation cleanup': 'Xác nhận xóa dữ liệu tổng hợp',
    'This will permanently remove aggregation buckets before {{date}}. Source logs are not affected.':
      'Dữ liệu tổng hợp trước {{date}} sẽ bị xóa vĩnh viễn. Nhật ký nguồn không bị ảnh hưởng.',
    'This will permanently remove aggregation buckets before the selected timestamp. Source logs are not affected.':
      'Dữ liệu tổng hợp trước thời điểm đã chọn sẽ bị xóa vĩnh viễn. Nhật ký nguồn không bị ảnh hưởng.',
    'Delete aggregation data': 'Xóa dữ liệu tổng hợp',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Thao tác này sẽ xây dựng lại chỉ mục định tuyến kênh từ mọi cấu hình kênh, bao gồm mô hình được hỗ trợ, nhóm và trọng số. Định tuyến có thể tạm thời chưa đầy đủ trong quá trình xây dựng lại. Tiếp tục?',
    'OpenAI Responses to Anthropic Messages':
      'OpenAI Responses sang Anthropic Messages',
    'Batch IQ test': 'Kiểm tra IQ hàng loạt',
    'IQ Test': 'Kiểm tra IQ',
    'Run IQ test': 'Chạy kiểm tra IQ',
    'View result': 'Xem kết quả',
    'IQ test running': 'Đang chạy kiểm tra IQ',
    'IQ test completed: {{success}} succeeded, {{failed}} failed':
      'Đã kiểm tra IQ: {{success}} thành công, {{failed}} thất bại',
    'IQ test completed: {{count}} succeeded':
      'Đã kiểm tra IQ: {{count}} thành công',
    'IQ test stopped: {{completed}}/{{total}} completed, {{success}} succeeded, {{failed}} failed':
      'Đã dừng kiểm tra IQ: {{completed}}/{{total}} hoàn tất, {{success}} thành công, {{failed}} thất bại',
    'IQ test result': 'Kết quả kiểm tra IQ',
    'IQ test result view': 'Chế độ xem kết quả kiểm tra IQ',
    'Copy source': 'Sao chép mã nguồn',
    'No result available': 'Chưa có kết quả',
    'Result not retained; run again': 'Kết quả không được lưu; hãy chạy lại',
  },
  'zh-TW': {
    'Automatically disable channels when tests fail':
      '當對渠道的批量測試失敗/普通請求失敗時，自動停用對應渠道',
    'Cumulative:': '累計：',
    'Today:': '今日：',
    'If you do not receive the email, please check your spam folder':
      '若未收到郵件，請查看垃圾郵件匣',
    'Choose which fields follow global settings and which fields use this user-specific strategy.':
      '選擇跟隨全域設定的欄位，以及使用此使用者專屬策略的欄位。',
    'Configure {{username}}': '設定 {{username}}',
    'Failed to reset settings': '重設設定失敗',
    'Failed to save settings': '儲存設定失敗',
    'No personalized settings': '尚未設定個人化設定',
    '{{count}} customized fields': '{{count}} 個個人化欄位',
    'The new rule applies to future reward events for this user. Existing rewards and unlock times will not be recalculated.':
      '新規則將套用至此使用者未來的獎勵事件。現有獎勵與解鎖時間不會重新計算。',
    'Priority: High to Low': '優先級：從高到低',
    'Priority: Low to High': '優先級：從低到高',
    'Customer Service': '客服資訊',
    'Enter customer service contact information': '輸入客服聯絡資訊',
    'Information displayed to users for contacting customer service':
      '向使用者顯示的客服聯絡資訊',
    'Batch IQ test': '批次智商測試',
    'IQ Test': '智商測試',
    'Run IQ test': '執行智商測試',
    'View result': '查看結果',
    'IQ test running': '智商測試進行中',
    'IQ test completed: {{success}} succeeded, {{failed}} failed':
      '智商測試完成：{{success}} 個成功，{{failed}} 個失敗',
    'IQ test completed: {{count}} succeeded': '智商測試完成：{{count}} 個成功',
    'IQ test stopped: {{completed}}/{{total}} completed, {{success}} succeeded, {{failed}} failed':
      '智商測試已停止：已完成 {{completed}}/{{total}}，{{success}} 個成功，{{failed}} 個失敗',
    'IQ test result': '智商測試結果',
    'IQ test result view': '智商測試結果檢視',
    'Copy source': '複製原始碼',
    'No result available': '目前沒有結果',
    'Result not retained; run again': '結果未保留，請重新測試',
    QPS: 'QPS',
    'TPS (thousands)': 'TPS（千）',
    'Throughput trend': '吞吐趨勢',
    Range: '範圍',
    Yesterday: '昨天',
    'This Week': '本週',
    'This Month': '本月',
    Granularity: '粒度',
    'By Token': '按 Token',
    'By Cost': '按費用',
    'Select the start and end time for the dashboard.':
      '選擇資料看板的開始和結束時間。',
    Apply: '套用',
    'Current balance': '目前餘額',
    "Today's balance usage": '今日餘額消耗',
    "Today's usage": '今日用量',
    'Historical total consumed': '歷史總計消耗',
    'Error self-check guide': '錯誤自查指南',
    'Official OpenAI errors': 'OpenAI 官方錯誤',
    'User-side errors': '使用者端錯誤',
    'Relay errors': '中轉站錯誤',
    'Other errors': '其他錯誤',
    'The response stream was interrupted unexpectedly.': '回應串流意外中斷。',
    'OpenAI is temporarily unable to allocate enough compute capacity for the selected model.':
      'OpenAI 暫時無法為所選模型分配足夠的算力。',
    'To preserve cache-hit rates, keep retrying the request and wait for OpenAI to assign capacity through its queue.':
      '如果希望保留快取命中率，請持續重試請求，等待 OpenAI 透過排隊分配算力。',
    'Start a new conversation or switch to a new group so the backend may select another account. Accounts can be served from different regions, where capacity pressure may differ; this may help, but the same shortage can still occur.':
      '切換到新對話或新分組，讓後台有機會選擇其他帳號。不同帳號可能位於不同區域，各區域的算力緊張程度可能不同，這樣做可能會有所改善，也可能仍會遇到同樣的算力不足。',
    'Please select at least one model billing group':
      '請至少選擇一個模型計費分組',
    Order: '順序',
    '24 hours': '24 小時',
    '7 days': '7 天',
    '30 days': '30 天',
    'Actual cost': '實際成本',
    'Actual cost (USD)': '實際成本（USD）',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      '實際成本由管理員手動輸入，並按所選期間分攤。估算成本由閘道根據計費快照計算。',
    'Average latency': '平均延遲',
    'Channel Reconciliation': '渠道對帳',
    'Cost entries': '成本帳期',
    'Cost entry saved': '成本記錄已儲存',
    Daily: '每日',
    'Daily cost trend': '每日成本趨勢',
    'Estimate variance': '估算差額',
    'Estimated cost': '估算成本',
    'Gross margin': '毛利差額',
    'Group Management': '群組管理',
    'Inbound endpoints': '入站端點',
    Loading: '載入中',
    Models: '模型',
    'No cost entries': '暫無成本帳期',
    'Record cost': '輸入成本',
    Requests: '請求數',
    Source: '來源',
    Start: '開始',
    End: '結束',
    Tokens: 'Token 數',
    'Upstream endpoints': '上游端點',
    'Usage breakdown': '使用明細',
    'User charge': '使用者扣費',
    'Auto-disabled probe interval': '自動停用渠道探測間隔',
    'Probe interval': '探測間隔',
    'Upstream max retries': '上游最大重試次數',
    'Channel price multiplier': '渠道價格倍率',
    'Daily Cost': '日成本',
    'Monthly Cost': '月成本',
    'Daily Cost / Monthly Cost': '日成本 / 月成本',
    'Daily Usage': '日用量',
    'Monthly Usage': '月用量',
    'Daily Usage / Monthly Usage': '日用量 / 月用量',
    'Price multiplier mode': '價格倍率模式',
    'USD-equivalent': '美元等值',
    CNY: '人民幣',
    'Probe failure auto-ban': '探測失敗自動停用',
    'Probe success auto-enable': '探測成功自動啟用',
    'Force priority': '強制優先',
    'Force priority scope': '強制優先範圍',
    'Current group only': '僅目前分組',
    'Across selected groups': '跨所選分組',
    'Previous-day probe success rate': '前一自然日探測成功率',
    'Previous-day average TTFT': '前一自然日平均 TTFT',
    TTFT: 'TTFT',
    'Recent test': '最近測試',
    'Yesterday average': '昨日平均',
    'Automatic probe': '自動探測',
    'The system probe task scans the task queue every 60 seconds.':
      '系統探測任務每 60 秒掃描一次任務佇列',
    'Automatically probe this channel in the background':
      '在背景自動探測此渠道',
    'You can still edit non-sensitive operations fields such as models, groups, weight, and routing settings.':
      '仍可編輯模型、分組、權重與路由設定等非敏感操作欄位。',
    'Pricing groups that can access channels with this tag':
      '可存取此標籤渠道的計價分組',
    'Randomly select a key from the configured set for each request':
      '每次請求從已設定的金鑰集合中隨機選擇一個',
    'Select pricing groups that can access this channel.':
      '選擇可存取此渠道的計價分組。',
    'Pricing groups that can access this channel.': '可存取此渠道的計價分組。',
    'Interval for probing enabled channels, in seconds':
      '啟用渠道的探測間隔（秒）',
    'Interval for probing auto-disabled channels, in seconds':
      '自動停用渠道的探測間隔（秒）',
    'Automatically disable the channel when a probe fails':
      '探測失敗時自動停用渠道',
    'Automatically enable the channel after a successful probe':
      '探測成功後自動啟用渠道',
    'Maximum retries for this channel after the first upstream attempt':
      '首次上游請求後的最大重試次數',
    'Relative upstream cost used for channel ranking. 1 means unchanged.':
      '用於渠道排序的相對上游成本，1 代表不變。',
    'Currency used when comparing this channel price multiplier':
      '比較渠道價格倍率時使用的貨幣',
    'Place this channel before ordinary channels in its selected scope':
      '在所選範圍內將此渠道置於一般渠道之前',
    'Read-only success rate from the previous natural day':
      '前一自然日探測成功率（唯讀）',
    'Choose whether force priority applies within one group or across groups':
      '選擇強制優先僅適用於單一分組或跨分組',
    'ID (Default)': 'ID（預設）',
    'Select a date before clearing aggregation data.':
      '清理聚合資料前請選擇日期。',
    'Failed to clean aggregation data': '清理聚合資料失敗',
    '{{count}} aggregation rows removed.': '已刪除 {{count}} 筆聚合資料。',
    'No aggregation rows matched the selected time.':
      '選取的時間內沒有符合的聚合資料。',
    'Clean aggregation data': '清理聚合資料',
    'Confirm aggregation cleanup': '確認清理聚合資料',
    'This will permanently remove aggregation buckets before {{date}}. Source logs are not affected.':
      '這將永久刪除 {{date}} 之前的聚合資料，不會影響來源日誌。',
    'This will permanently remove aggregation buckets before the selected timestamp. Source logs are not affected.':
      '這將永久刪除選取時間之前的聚合資料，不會影響來源日誌。',
    'Delete aggregation data': '刪除聚合資料',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      '這會根據所有渠道設定重建渠道路由索引，包括支援模型、分組和權重。重建期間路由可能短暫不完整。是否繼續？',
    'OpenAI Responses to Anthropic Messages':
      'OpenAI Responses 到 Anthropic Messages',
  },
}

// Global model settings uses structured controls, but its persisted values
// remain JSON strings. Keep the form-specific copy in every locale in sync.
const globalModelSettingsTranslations = {
  en: {
    'Add a model name': 'Add a model name',
    'Add at least one model pattern when the policy is enabled':
      'Add at least one model pattern when the policy is enabled',
    'Add pattern': 'Add pattern',
    'Add model "{{value}}"': 'Add model "{{value}}"',
    'Add channel ID "{{value}}"': 'Add channel ID "{{value}}"',
    'All channels': 'All channels',
    'Apply to': 'Apply to',
    'Channel #{{id}}': 'Channel #{{id}}',
    'Channel type #{{id}}': 'Channel type #{{id}}',
    'Channel types': 'Channel types',
    'Convert matching Chat Completions requests to the Responses API.':
      'Convert matching Chat Completions requests to the Responses API.',
    'Convert selected Chat Completions requests to the Responses API without editing a policy object.':
      'Convert selected Chat Completions requests to the Responses API without editing a policy object.',
    'Delete {{value}}': 'Delete {{value}}',
    'Enable this policy': 'Enable this policy',
    'Enter a model name': 'Enter a model name',
    'Enter a model regular expression': 'Enter a model regular expression',
    'Enter a valid regular expression': 'Enter a valid regular expression',
    'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.':
      'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.',
    'Leave empty to match by channel type only.':
      'Leave empty to match by channel type only.',
    'Model pattern {{index}}': 'Model pattern {{index}}',
    'Model patterns': 'Model patterns',
    'Move {{value}} down': 'Move {{value}} down',
    'Move {{value}} up': 'Move {{value}} up',
    'No matching channel types': 'No matching channel types',
    'No matching channels': 'No matching channels',
    'No model patterns. Add at least one to enable matching.':
      'No model patterns. Add at least one to enable matching.',
    'No models added': 'No models added',
    'No preferred models. The default order will be used.':
      'No preferred models. The default order will be used.',
    'Preferred model {{index}}': 'Preferred model {{index}}',
    'Response conversion policy': 'Response conversion policy',
    'Search by channel name or ID': 'Search by channel name or ID',
    'Select channel types': 'Select channel types',
    'Selected channels': 'Selected channels',
    'Select a channel or channel type when using selected channels':
      'Select a channel or channel type when using selected channels',
    '{{count}} items': '{{count}} items',
    'A request matches when its channel ID or type is selected.':
      'A request matches when its channel ID or type is selected.',
    'This feature is experimental. The client request must still match the selected upstream protocol.':
      'This feature is experimental. The client request must still match the selected upstream protocol.',
    'Models are tried in this order when selecting a default model. Add one model per row.':
      'Models are tried in this order when selecting a default model. Add one model per row.',
  },
  zh: {
    'Add a model name': '添加模型名称',
    'Add at least one model pattern when the policy is enabled':
      '启用策略时至少添加一条模型匹配规则',
    'Add pattern': '添加规则',
    'Add model "{{value}}"': '添加模型“{{value}}”',
    'Add channel ID "{{value}}"': '添加渠道 ID“{{value}}”',
    'All channels': '所有渠道',
    'Apply to': '应用范围',
    'Channel #{{id}}': '渠道 #{{id}}',
    'Channel type #{{id}}': '渠道类型 #{{id}}',
    'Channel types': '渠道类型',
    'Convert matching Chat Completions requests to the Responses API.':
      '将匹配的 Chat Completions 请求转换为 Responses API。',
    'Convert selected Chat Completions requests to the Responses API without editing a policy object.':
      '将选定的 Chat Completions 请求转换为 Responses API，无需编辑策略对象。',
    'Delete {{value}}': '删除 {{value}}',
    'Enable this policy': '启用此策略',
    'Enter a model name': '输入模型名称',
    'Enter a model regular expression': '输入模型正则表达式',
    'Enter a valid regular expression': '请输入有效的正则表达式',
    'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.':
      '对支持的中转接口保留原始请求体；鉴权、路由、计费和响应处理仍会执行。',
    'Leave empty to match by channel type only.': '留空则仅按渠道类型匹配。',
    'Model pattern {{index}}': '模型规则 {{index}}',
    'Model patterns': '模型匹配规则',
    'Move {{value}} down': '下移 {{value}}',
    'Move {{value}} up': '上移 {{value}}',
    'No matching channel types': '没有匹配的渠道类型',
    'No matching channels': '没有匹配的渠道',
    'No model patterns. Add at least one to enable matching.':
      '暂无模型规则。至少添加一条规则才能匹配。',
    'No models added': '尚未添加模型',
    'No preferred models. The default order will be used.':
      '暂无偏好模型，将使用默认顺序。',
    'Preferred model {{index}}': '偏好模型 {{index}}',
    'Response conversion policy': '响应转换策略',
    'Search by channel name or ID': '按渠道名称或 ID 搜索',
    'Select channel types': '选择渠道类型',
    'Selected channels': '指定渠道',
    'Select a channel or channel type when using selected channels':
      '使用指定渠道时请选择至少一个渠道或渠道类型',
    '{{count}} items': '{{count}} 项',
    'A request matches when its channel ID or type is selected.':
      '请求的渠道 ID 或渠道类型命中任一选项时即匹配。',
    'This feature is experimental. The client request must still match the selected upstream protocol.':
      '此功能仍处于实验阶段，客户端请求仍需符合所选上游协议。',
    'Models are tried in this order when selecting a default model. Add one model per row.':
      '选择默认模型时会按此顺序尝试。每行添加一个模型。',
  },
  fr: {
    'Add a model name': 'Ajouter un nom de modèle',
    'Add at least one model pattern when the policy is enabled':
      'Ajoutez au moins un motif de modèle lorsque la stratégie est activée',
    'Add pattern': 'Ajouter un motif',
    'Add model "{{value}}"': 'Ajouter le modèle « {{value}} »',
    'Add channel ID "{{value}}"':
      'Ajouter l’identifiant de canal « {{value}} »',
    'All channels': 'Tous les canaux',
    'Apply to': 'Appliquer à',
    'Channel #{{id}}': 'Canal n° {{id}}',
    'Channel type #{{id}}': 'Type de canal n° {{id}}',
    'Channel types': 'Types de canal',
    'Convert matching Chat Completions requests to the Responses API.':
      'Convertir les requêtes Chat Completions correspondantes vers l’API Responses.',
    'Convert selected Chat Completions requests to the Responses API without editing a policy object.':
      'Convertir les requêtes Chat Completions sélectionnées vers l’API Responses sans modifier une stratégie.',
    'Delete {{value}}': 'Supprimer {{value}}',
    'Enable this policy': 'Activer cette stratégie',
    'Enter a model name': 'Saisir un nom de modèle',
    'Enter a model regular expression':
      'Saisir une expression régulière de modèle',
    'Enter a valid regular expression':
      'Saisissez une expression régulière valide',
    'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.':
      'Conserver le corps original pour les points de relais pris en charge. L’authentification, le routage, la facturation et le traitement de la réponse restent actifs.',
    'Leave empty to match by channel type only.':
      'Laisser vide pour utiliser uniquement le type de canal.',
    'Model pattern {{index}}': 'Motif de modèle {{index}}',
    'Model patterns': 'Motifs de modèle',
    'Move {{value}} down': 'Descendre {{value}}',
    'Move {{value}} up': 'Monter {{value}}',
    'No matching channel types': 'Aucun type de canal correspondant',
    'No matching channels': 'Aucun canal correspondant',
    'No model patterns. Add at least one to enable matching.':
      'Aucun motif de modèle. Ajoutez-en au moins un pour activer la correspondance.',
    'No models added': 'Aucun modèle ajouté',
    'No preferred models. The default order will be used.':
      'Aucun modèle préféré. L’ordre par défaut sera utilisé.',
    'Preferred model {{index}}': 'Modèle préféré {{index}}',
    'Response conversion policy': 'Stratégie de conversion des réponses',
    'Search by channel name or ID':
      'Rechercher par nom ou identifiant de canal',
    'Select channel types': 'Sélectionner les types de canal',
    'Selected channels': 'Canaux sélectionnés',
    'Select a channel or channel type when using selected channels':
      'Sélectionnez un canal ou un type de canal pour utiliser les canaux sélectionnés',
    '{{count}} items': '{{count}} éléments',
    'A request matches when its channel ID or type is selected.':
      'Une requête correspond lorsque son identifiant ou son type de canal est sélectionné.',
    'This feature is experimental. The client request must still match the selected upstream protocol.':
      'Cette fonctionnalité est expérimentale. La requête cliente doit respecter le protocole amont sélectionné.',
    'Models are tried in this order when selecting a default model. Add one model per row.':
      'Les modèles sont essayés dans cet ordre lors du choix d’un modèle par défaut. Ajoutez un modèle par ligne.',
  },
  ja: {
    'Add a model name': 'モデル名を追加',
    'Add at least one model pattern when the policy is enabled':
      'ポリシーを有効にするには、モデルパターンを 1 つ以上追加してください',
    'Add pattern': 'パターンを追加',
    'Add model "{{value}}"': 'モデル「{{value}}」を追加',
    'Add channel ID "{{value}}"': 'チャンネル ID「{{value}}」を追加',
    'All channels': 'すべてのチャンネル',
    'Apply to': '適用範囲',
    'Channel #{{id}}': 'チャンネル #{{id}}',
    'Channel type #{{id}}': 'チャンネル種別 #{{id}}',
    'Channel types': 'チャンネル種別',
    'Convert matching Chat Completions requests to the Responses API.':
      '一致する Chat Completions リクエストを Responses API に変換します。',
    'Convert selected Chat Completions requests to the Responses API without editing a policy object.':
      'ポリシーを編集せず、選択した Chat Completions リクエストを Responses API に変換します。',
    'Delete {{value}}': '{{value}} を削除',
    'Enable this policy': 'このポリシーを有効化',
    'Enter a model name': 'モデル名を入力',
    'Enter a model regular expression': 'モデルの正規表現を入力',
    'Enter a valid regular expression': '有効な正規表現を入力してください',
    'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.':
      '対応するリレーエンドポイントでは元のリクエスト本文を保持します。認証、ルーティング、課金、レスポンス処理は引き続き実行されます。',
    'Leave empty to match by channel type only.':
      '空欄にするとチャンネル種別だけで一致します。',
    'Model pattern {{index}}': 'モデルパターン {{index}}',
    'Model patterns': 'モデルパターン',
    'Move {{value}} down': '{{value}} を下へ移動',
    'Move {{value}} up': '{{value}} を上へ移動',
    'No matching channel types': '一致するチャンネル種別はありません',
    'No matching channels': '一致するチャンネルはありません',
    'No model patterns. Add at least one to enable matching.':
      'モデルパターンがありません。一致判定には少なくとも 1 つ追加してください。',
    'No models added': 'モデルは未追加です',
    'No preferred models. The default order will be used.':
      '優先モデルはありません。既定の順序を使用します。',
    'Preferred model {{index}}': '優先モデル {{index}}',
    'Response conversion policy': 'レスポンス変換ポリシー',
    'Search by channel name or ID': 'チャンネル名または ID で検索',
    'Select channel types': 'チャンネル種別を選択',
    'Selected channels': '指定したチャンネル',
    'Select a channel or channel type when using selected channels':
      '指定したチャンネルを使う場合は、チャンネルまたは種別を 1 つ以上選択してください',
    '{{count}} items': '{{count}} 件',
    'A request matches when its channel ID or type is selected.':
      'チャンネル ID または種別が選択項目と一致するとリクエストに適用されます。',
    'This feature is experimental. The client request must still match the selected upstream protocol.':
      'この機能は試験中です。クライアントのリクエストは選択した上流プロトコルに一致する必要があります。',
    'Models are tried in this order when selecting a default model. Add one model per row.':
      '既定モデルの選択時はこの順序で試行します。1 行に 1 モデルを追加してください。',
  },
  ru: {
    'Add a model name': 'Добавьте название модели',
    'Add at least one model pattern when the policy is enabled':
      'При включённой политике добавьте хотя бы один шаблон модели',
    'Add pattern': 'Добавить шаблон',
    'Add model "{{value}}"': 'Добавить модель «{{value}}»',
    'Add channel ID "{{value}}"': 'Добавить ID канала «{{value}}»',
    'All channels': 'Все каналы',
    'Apply to': 'Применить к',
    'Channel #{{id}}': 'Канал № {{id}}',
    'Channel type #{{id}}': 'Тип канала № {{id}}',
    'Channel types': 'Типы каналов',
    'Convert matching Chat Completions requests to the Responses API.':
      'Преобразовывать подходящие запросы Chat Completions в Responses API.',
    'Convert selected Chat Completions requests to the Responses API without editing a policy object.':
      'Преобразовывать выбранные запросы Chat Completions в Responses API без редактирования политики.',
    'Delete {{value}}': 'Удалить {{value}}',
    'Enable this policy': 'Включить эту политику',
    'Enter a model name': 'Введите название модели',
    'Enter a model regular expression': 'Введите регулярное выражение модели',
    'Enter a valid regular expression':
      'Введите корректное регулярное выражение',
    'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.':
      'Сохранять исходное тело запроса для поддерживаемых relay-endpoint. Аутентификация, маршрутизация, тарификация и обработка ответа продолжаются.',
    'Leave empty to match by channel type only.':
      'Оставьте пустым, чтобы сопоставлять только по типу канала.',
    'Model pattern {{index}}': 'Шаблон модели {{index}}',
    'Model patterns': 'Шаблоны моделей',
    'Move {{value}} down': 'Переместить {{value}} вниз',
    'Move {{value}} up': 'Переместить {{value}} вверх',
    'No matching channel types': 'Подходящие типы каналов не найдены',
    'No matching channels': 'Подходящие каналы не найдены',
    'No model patterns. Add at least one to enable matching.':
      'Шаблоны моделей отсутствуют. Добавьте хотя бы один для сопоставления.',
    'No models added': 'Модели не добавлены',
    'No preferred models. The default order will be used.':
      'Предпочтительные модели не заданы. Будет использован порядок по умолчанию.',
    'Preferred model {{index}}': 'Предпочтительная модель {{index}}',
    'Response conversion policy': 'Политика преобразования ответов',
    'Search by channel name or ID': 'Поиск по имени или ID канала',
    'Select channel types': 'Выберите типы каналов',
    'Selected channels': 'Выбранные каналы',
    'Select a channel or channel type when using selected channels':
      'При выборе конкретных каналов выберите канал или тип канала',
    '{{count}} items': '{{count}} элементов',
    'A request matches when its channel ID or type is selected.':
      'Запрос подходит, если выбран его ID или тип канала.',
    'This feature is experimental. The client request must still match the selected upstream protocol.':
      'Функция экспериментальная. Запрос клиента должен соответствовать выбранному протоколу upstream.',
    'Models are tried in this order when selecting a default model. Add one model per row.':
      'При выборе модели по умолчанию модели проверяются в этом порядке. Добавляйте по одной модели в строке.',
  },
  vi: {
    'Add a model name': 'Thêm tên mô hình',
    'Add at least one model pattern when the policy is enabled':
      'Khi bật chính sách, hãy thêm ít nhất một mẫu mô hình',
    'Add pattern': 'Thêm mẫu',
    'Add model "{{value}}"': 'Thêm mô hình “{{value}}”',
    'Add channel ID "{{value}}"': 'Thêm ID kênh “{{value}}”',
    'All channels': 'Tất cả kênh',
    'Apply to': 'Áp dụng cho',
    'Channel #{{id}}': 'Kênh #{{id}}',
    'Channel type #{{id}}': 'Loại kênh #{{id}}',
    'Channel types': 'Loại kênh',
    'Convert matching Chat Completions requests to the Responses API.':
      'Chuyển các yêu cầu Chat Completions phù hợp sang Responses API.',
    'Convert selected Chat Completions requests to the Responses API without editing a policy object.':
      'Chuyển các yêu cầu Chat Completions đã chọn sang Responses API mà không cần sửa chính sách.',
    'Delete {{value}}': 'Xóa {{value}}',
    'Enable this policy': 'Bật chính sách này',
    'Enter a model name': 'Nhập tên mô hình',
    'Enter a model regular expression': 'Nhập biểu thức chính quy của mô hình',
    'Enter a valid regular expression': 'Nhập biểu thức chính quy hợp lệ',
    'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.':
      'Giữ nguyên nội dung yêu cầu cho các endpoint chuyển tiếp được hỗ trợ. Xác thực, định tuyến, tính phí và xử lý phản hồi vẫn được thực hiện.',
    'Leave empty to match by channel type only.':
      'Để trống để chỉ khớp theo loại kênh.',
    'Model pattern {{index}}': 'Mẫu mô hình {{index}}',
    'Model patterns': 'Mẫu mô hình',
    'Move {{value}} down': 'Chuyển {{value}} xuống',
    'Move {{value}} up': 'Chuyển {{value}} lên',
    'No matching channel types': 'Không có loại kênh phù hợp',
    'No matching channels': 'Không có kênh phù hợp',
    'No model patterns. Add at least one to enable matching.':
      'Chưa có mẫu mô hình. Thêm ít nhất một mẫu để bật khớp.',
    'No models added': 'Chưa thêm mô hình',
    'No preferred models. The default order will be used.':
      'Chưa có mô hình ưu tiên. Sẽ dùng thứ tự mặc định.',
    'Preferred model {{index}}': 'Mô hình ưu tiên {{index}}',
    'Response conversion policy': 'Chính sách chuyển đổi phản hồi',
    'Search by channel name or ID': 'Tìm theo tên hoặc ID kênh',
    'Select channel types': 'Chọn loại kênh',
    'Selected channels': 'Kênh đã chọn',
    'Select a channel or channel type when using selected channels':
      'Khi chọn kênh cụ thể, hãy chọn một kênh hoặc loại kênh',
    '{{count}} items': '{{count}} mục',
    'A request matches when its channel ID or type is selected.':
      'Yêu cầu khớp khi ID hoặc loại kênh của nó được chọn.',
    'This feature is experimental. The client request must still match the selected upstream protocol.':
      'Tính năng này đang thử nghiệm. Yêu cầu từ máy khách vẫn phải phù hợp với giao thức upstream đã chọn.',
    'Models are tried in this order when selecting a default model. Add one model per row.':
      'Khi chọn mô hình mặc định, các mô hình được thử theo thứ tự này. Thêm một mô hình mỗi dòng.',
  },
  'zh-TW': {
    'Add a model name': '新增模型名稱',
    'Add at least one model pattern when the policy is enabled':
      '啟用策略時至少新增一條模型比對規則',
    'Add pattern': '新增規則',
    'Add model "{{value}}"': '新增模型「{{value}}」',
    'Add channel ID "{{value}}"': '新增頻道 ID「{{value}}」',
    'All channels': '所有頻道',
    'Apply to': '套用範圍',
    'Channel #{{id}}': '頻道 #{{id}}',
    'Channel type #{{id}}': '頻道類型 #{{id}}',
    'Channel types': '頻道類型',
    'Convert matching Chat Completions requests to the Responses API.':
      '將符合的 Chat Completions 請求轉換為 Responses API。',
    'Convert selected Chat Completions requests to the Responses API without editing a policy object.':
      '將選取的 Chat Completions 請求轉換為 Responses API，不需要編輯策略物件。',
    'Delete {{value}}': '刪除 {{value}}',
    'Enable this policy': '啟用此策略',
    'Enter a model name': '輸入模型名稱',
    'Enter a model regular expression': '輸入模型正規表示式',
    'Enter a valid regular expression': '請輸入有效的正規表示式',
    'Keep the original request body for supported relay endpoints. Authentication, routing, billing, and response handling still run.':
      '對支援的轉送端點保留原始請求內容；驗證、路由、計費和回應處理仍會執行。',
    'Leave empty to match by channel type only.': '留空則只依頻道類型比對。',
    'Model pattern {{index}}': '模型規則 {{index}}',
    'Model patterns': '模型比對規則',
    'Move {{value}} down': '下移 {{value}}',
    'Move {{value}} up': '上移 {{value}}',
    'No matching channel types': '沒有符合的頻道類型',
    'No matching channels': '沒有符合的頻道',
    'No model patterns. Add at least one to enable matching.':
      '尚無模型規則。至少新增一條規則才能進行比對。',
    'No models added': '尚未新增模型',
    'No preferred models. The default order will be used.':
      '尚無偏好模型，將使用預設順序。',
    'Preferred model {{index}}': '偏好模型 {{index}}',
    'Response conversion policy': '回應轉換策略',
    'Search by channel name or ID': '依頻道名稱或 ID 搜尋',
    'Select channel types': '選取頻道類型',
    'Selected channels': '指定頻道',
    'Select a channel or channel type when using selected channels':
      '使用指定頻道時，請至少選取一個頻道或頻道類型',
    '{{count}} items': '{{count}} 項',
    'A request matches when its channel ID or type is selected.':
      '請求的頻道 ID 或類型符合任一選項時即套用。',
    'This feature is experimental. The client request must still match the selected upstream protocol.':
      '此功能仍在實驗階段，客戶端請求仍須符合所選的上游協定。',
    'Models are tried in this order when selecting a default model. Add one model per row.':
      '選取預設模型時會依此順序嘗試，每列新增一個模型。',
  },
}

Object.assign(newKeys.en, globalModelSettingsTranslations.en)
Object.assign(newKeys.zh, globalModelSettingsTranslations.zh)
Object.assign(newKeys.fr, globalModelSettingsTranslations.fr)
Object.assign(newKeys.ja, globalModelSettingsTranslations.ja)
Object.assign(newKeys.ru, globalModelSettingsTranslations.ru)
Object.assign(newKeys.vi, globalModelSettingsTranslations.vi)
Object.assign(newKeys['zh-TW'], globalModelSettingsTranslations['zh-TW'])

const errorSelfCheckTranslations = {
  en: {
    'Official model errors': 'Official model errors',
    'User client errors': 'User client errors',
    'Streaming output interrupted due to too many pending requests':
      'Streaming output interrupted due to too many pending requests',
    'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.':
      'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.',
    'Retry the request several times or avoid peak hours.':
      'Retry the request several times or avoid peak hours.',
    'If the problem persists, click About above and contact the administrator.':
      'If the problem persists, click About above and contact the administrator.',
    'Search error keywords...': 'Search error keywords...',
    'Search error keywords': 'Search error keywords',
    'No matching errors found.': 'No matching errors found.',
    'Model capacity insufficient': 'Model capacity insufficient',
    'Authentication failed': 'Authentication failed',
    'Invalid API key': 'Invalid API key',
    'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.':
      'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.',
    'Re-import through CC Switch (simplest).':
      'Re-import through CC Switch (simplest).',
    'Confirm that the API key and Base URL match.':
      'Confirm that the API key and Base URL match.',
    'Official OpenAI error': 'Official OpenAI error',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI and other official providers temporarily lack enough compute capacity.',
    'OpenAI and other model providers are experiencing server overload.':
      'OpenAI and other model providers are experiencing server overload.',
    'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.':
      'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.',
    'The API key or authentication header is missing, expired, or invalid.':
      'The API key or authentication header is missing, expired, or invalid.',
    'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.':
      'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.',
    'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.':
      'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.',
    'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.':
      'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.',
    'Response stream interrupted': 'Response stream interrupted',
    'The upstream connection or relay path closed before the response completed.':
      'The upstream connection or relay path closed before the response completed.',
    'Retry the request. A transient upstream disconnect can succeed on a later attempt.':
      'Retry the request. A transient upstream disconnect can succeed on a later attempt.',
    'If the error persists, try another available model or group and check the channel health and usage logs.':
      'If the error persists, try another available model or group and check the channel health and usage logs.',
    'Error code': 'Error code',
    'Error description': 'Error description',
    'Error information': 'Error information',
    Cause: 'Cause',
    Solution: 'Solution',
  },
  zh: {
    'Official model errors': '模型官方错误',
    'User client errors': '用户客户端错误',
    'Streaming output interrupted due to too many pending requests':
      '流式输出中断，请求太多',
    'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.':
      '真正原因是上游渠道的排队请求数达到上限，大概号池不足，小概率OpenAI算力不足',
    'Retry the request several times or avoid peak hours.':
      '多重试，或者避开使用高峰',
    'If the problem persists, click About above and contact the administrator.':
      '如果长时间不行，点击上方“关于”，联系管理员询问',
    '一直重连，等待网络': '一直重连，等待网络',
    用户网络故障: '用户网络故障',
    检查网络是否连通: '检查网络是否连通',
    'Search error keywords...': '搜索错误关键词...',
    'Search error keywords': '搜索错误关键词',
    'No matching errors found.': '未找到匹配的错误。',
    'Model capacity insufficient': '模型容量不足',
    'Authentication failed': '认证失败',
    'Invalid API key': '提供的API密钥不正确，密钥无效',
    'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.':
      '密钥错误，或者请求的网站错误（例如直接请求 OpenAI 官网）',
    'Re-import through CC Switch (simplest).':
      '重新通过 CC Switch 导入（最简单）',
    'Confirm that the API key and Base URL match.':
      '确认密钥和网址（Base URL 是否对应）',
    'Official OpenAI error': 'OpenAI官方异常错误',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI 及其他官方厂商暂时缺乏足够的算力。',
    'OpenAI and other model providers are experiencing server overload.':
      'OpenAI 等模型厂商的服务器过载。',
    'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.':
      '可能是 OpenAI 等模型供应商服务器超载，也可能是 OpenAI 内部的临时 Bug。',
    'The API key or authentication header is missing, expired, or invalid.':
      'API Key 或鉴权请求头缺失、已过期或无效。',
    'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.':
      '确认 API Key 处于启用状态、复制时没有多余空格，并且有权使用所选模型。',
    'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.':
      '按接口要求的鉴权格式发送密钥，例如 Authorization: Bearer <API_KEY>。',
    'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.':
      '检查客户端 Base URL，确保指向本服务，并使用所选协议要求的路径。',
    'Response stream interrupted': '响应流中断',
    'The upstream connection or relay path closed before the response completed.':
      '上游连接或中转链路在响应完成前关闭。',
    'Retry the request. A transient upstream disconnect can succeed on a later attempt.':
      '重试请求；临时性的上游断开可能在稍后重试时恢复。',
    'If the error persists, try another available model or group and check the channel health and usage logs.':
      '如果持续出现，请尝试其他可用模型或分组，并检查渠道健康状态和使用日志。',
    'Error code': '错误码',
    'Error description': '错误描述',
    'Error information': '错误信息',
    Cause: '原因',
    Solution: '解决方法',
  },
  'zh-TW': {
    'Official model errors': '模型官方錯誤',
    'User client errors': '使用者端錯誤',
    'Streaming output interrupted due to too many pending requests':
      '串流輸出中斷，請求過多',
    'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.':
      '真正原因是上游渠道的排隊請求數已達上限，可能是帳號池不足，少數情況下也可能是 OpenAI 算力不足。',
    'Retry the request several times or avoid peak hours.':
      '多重試幾次，或避開使用高峰時段',
    'If the problem persists, click About above and contact the administrator.':
      '如果長時間無法恢復，請點擊上方「關於」並聯絡管理員詢問',
    '一直重连，等待网络': '一直重新連線，等待網路',
    用户网络故障: '使用者網路故障',
    检查网络是否连通: '檢查網路是否連通',
    'Search error keywords...': '搜尋錯誤關鍵字...',
    'Search error keywords': '搜尋錯誤關鍵字',
    'No matching errors found.': '找不到符合的錯誤。',
    'Model capacity insufficient': '模型容量不足',
    'Authentication failed': '驗證失敗',
    'Invalid API key': '提供的 API 金鑰不正確，金鑰無效',
    'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.':
      '金鑰錯誤，或請求的網站錯誤（例如直接請求 OpenAI 官網）',
    'Re-import through CC Switch (simplest).':
      '重新透過 CC Switch 匯入（最簡單）',
    'Confirm that the API key and Base URL match.':
      '確認金鑰與網址（Base URL）是否對應',
    'Official OpenAI error': 'OpenAI 官方異常錯誤',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI 及其他官方廠商暫時缺乏足夠的算力。',
    'OpenAI and other model providers are experiencing server overload.':
      'OpenAI 等模型供應商的伺服器過載。',
    'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.':
      '可能是 OpenAI 等模型供應商的伺服器過載，也可能是 OpenAI 內部的暫時性 Bug。',
    'The API key or authentication header is missing, expired, or invalid.':
      'API Key 或驗證標頭缺失、已過期或無效。',
    'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.':
      '確認 API Key 已啟用、複製時沒有多餘空格，且有權使用所選模型。',
    'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.':
      '依端點要求的驗證格式傳送金鑰，例如 Authorization: Bearer <API_KEY>。',
    'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.':
      '檢查用戶端 Base URL，確認其指向本服務，並使用所選協定要求的路徑。',
    'Response stream interrupted': '回應串流中斷',
    'The upstream connection or relay path closed before the response completed.':
      '上游連線或中轉鏈路在回應完成前關閉。',
    'Retry the request. A transient upstream disconnect can succeed on a later attempt.':
      '重試請求；暫時性的上游中斷可能在稍後重試時恢復。',
    'If the error persists, try another available model or group and check the channel health and usage logs.':
      '如果持續發生，請嘗試其他可用模型或分組，並檢查渠道健康狀態與使用記錄。',
    'Error code': '錯誤碼',
    'Error description': '錯誤描述',
    'Error information': '錯誤資訊',
    Cause: '原因',
    Solution: '解決方法',
  },
  fr: {
    'Official model errors': 'Erreurs officielles des modèles',
    'User client errors': 'Erreurs côté client',
    'Streaming output interrupted due to too many pending requests':
      'Flux interrompu en raison d’un trop grand nombre de requêtes en attente',
    'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.':
      'La file du canal amont a atteint sa limite de requêtes en attente, généralement parce que le pool de comptes est insuffisant ; dans de rares cas, la capacité de calcul d’OpenAI peut être insuffisante.',
    'Retry the request several times or avoid peak hours.':
      'Réessayez plusieurs fois ou évitez les heures de pointe.',
    'If the problem persists, click About above and contact the administrator.':
      'Si le problème persiste, cliquez sur « À propos » ci-dessus et contactez l’administrateur.',
    '一直重连，等待网络': 'Reconnexion continue, en attente du réseau',
    用户网络故障: 'Défaillance du réseau de l’utilisateur',
    检查网络是否连通: 'Vérifiez que le réseau est accessible',
    'Search error keywords...': 'Rechercher des mots-clés d’erreur...',
    'Search error keywords': 'Rechercher des mots-clés d’erreur',
    'No matching errors found.': 'Aucune erreur correspondante trouvée.',
    'Model capacity insufficient': 'Capacité du modèle insuffisante',
    'Authentication failed': 'Échec de l’authentification',
    'Invalid API key': 'Clé API invalide',
    'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.':
      'La clé API est incorrecte ou la requête est envoyée vers le mauvais site, par exemple directement vers le site d’OpenAI.',
    'Re-import through CC Switch (simplest).':
      'Réimportez via CC Switch (le plus simple).',
    'Confirm that the API key and Base URL match.':
      'Vérifiez que la clé API et la Base URL correspondent.',
    'Official OpenAI error': 'Erreur officielle d’OpenAI',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI et les autres fournisseurs officiels ne disposent temporairement pas d’une capacité de calcul suffisante.',
    'OpenAI and other model providers are experiencing server overload.':
      'Les serveurs d’OpenAI et d’autres fournisseurs de modèles sont surchargés.',
    'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.':
      'Les serveurs d’OpenAI ou d’autres fournisseurs de modèles peuvent être surchargés, ou OpenAI peut rencontrer un bug interne temporaire.',
    'The API key or authentication header is missing, expired, or invalid.':
      'La clé API ou l’en-tête d’authentification est manquant, expiré ou invalide.',
    'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.':
      'Vérifiez que la clé API est active, copiée sans espaces superflus et autorisée à utiliser le modèle sélectionné.',
    'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.':
      'Envoyez la clé au format d’authentification requis par le point de terminaison, par exemple Authorization: Bearer <API_KEY>.',
    'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.':
      'Vérifiez la Base URL du client : elle doit pointer vers ce service et utiliser le chemin attendu par le protocole sélectionné.',
    'Response stream interrupted': 'Flux de réponse interrompu',
    'The upstream connection or relay path closed before the response completed.':
      'La connexion amont ou le relais s’est fermé avant la fin de la réponse.',
    'Retry the request. A transient upstream disconnect can succeed on a later attempt.':
      'Réessayez la requête : une coupure amont temporaire peut disparaître lors d’une nouvelle tentative.',
    'If the error persists, try another available model or group and check the channel health and usage logs.':
      'Si l’erreur persiste, essayez un autre modèle ou groupe disponible et consultez l’état du canal et les journaux d’utilisation.',
    'Error code': 'Code d’erreur',
    'Error description': 'Description de l’erreur',
    'Error information': 'Informations sur l’erreur',
    Cause: 'Cause',
    Solution: 'Solution',
  },
  ja: {
    'Official model errors': 'モデル公式エラー',
    'User client errors': 'ユーザー側クライアントエラー',
    'Streaming output interrupted due to too many pending requests':
      '保留中のリクエストが多すぎるためストリーミング出力が中断されました',
    'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.':
      '主な原因は上流チャンネルの待機リクエスト数が上限に達したことです。通常はアカウントプールの不足で、まれに OpenAI の計算リソース不足の場合もあります。',
    'Retry the request several times or avoid peak hours.':
      '何度か再試行するか、混雑する時間帯を避けてください。',
    'If the problem persists, click About above and contact the administrator.':
      '長時間解消しない場合は、上部の「概要」をクリックして管理者にお問い合わせください。',
    '一直重连，等待网络': '再接続を繰り返しています。ネットワークを待機中',
    用户网络故障: 'ユーザーのネットワーク障害',
    检查网络是否连通: 'ネットワーク接続を確認してください',
    'Search error keywords...': 'エラーキーワードを検索...',
    'Search error keywords': 'エラーキーワードを検索',
    'No matching errors found.': '一致するエラーが見つかりません。',
    'Model capacity insufficient': 'モデルの容量不足',
    'Authentication failed': '認証に失敗しました',
    'Invalid API key': '無効な API キー',
    'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.':
      'API キーが正しくないか、誤ったウェブサイトにリクエストを送信しています（OpenAI の公式サイトへ直接リクエストする場合など）。',
    'Re-import through CC Switch (simplest).':
      'CC Switch から再インポートしてください（最も簡単）。',
    'Confirm that the API key and Base URL match.':
      'API キーと Base URL が対応していることを確認してください。',
    'Official OpenAI error': 'OpenAI 公式の異常エラー',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI などの公式プロバイダーで、選択したモデルに割り当てる計算リソースが一時的に不足しています。',
    'OpenAI and other model providers are experiencing server overload.':
      'OpenAI などのモデルプロバイダーのサーバーが過負荷になっています。',
    'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.':
      'OpenAI などのモデルプロバイダーのサーバーが過負荷になっているか、OpenAI 内部で一時的な不具合が発生している可能性があります。',
    'The API key or authentication header is missing, expired, or invalid.':
      'API キーまたは認証ヘッダーがないか、期限切れ、または無効です。',
    'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.':
      'API キーが有効で余分な空白がなく、選択したモデルの利用を許可されていることを確認してください。',
    'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.':
      'エンドポイントが要求する認証形式でキーを送信してください（例: Authorization: Bearer <API_KEY>）。',
    'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.':
      'クライアントの Base URL がこのサービスを指し、選択したプロトコルに合ったパスを使用していることを確認してください。',
    'Response stream interrupted': 'レスポンスストリームの中断',
    'The upstream connection or relay path closed before the response completed.':
      'レスポンスが完了する前に、上流接続またはリレー経路が閉じられました。',
    'Retry the request. A transient upstream disconnect can succeed on a later attempt.':
      'リクエストを再試行してください。一時的な上流切断は、後の試行で解消する場合があります。',
    'If the error persists, try another available model or group and check the channel health and usage logs.':
      '解消しない場合は、別の利用可能なモデルまたはグループを試し、チャンネルの状態と利用ログを確認してください。',
    'Error code': 'エラーコード',
    'Error description': 'エラーの説明',
    'Error information': 'エラー情報',
    Cause: '原因',
    Solution: '解決方法',
  },
  ru: {
    'Official model errors': 'Официальные ошибки моделей',
    'User client errors': 'Ошибки клиентской части пользователя',
    'Streaming output interrupted due to too many pending requests':
      'Потоковый вывод прерван из-за слишком большого числа ожидающих запросов',
    'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.':
      'Очередь запросов upstream-канала достигла лимита; обычно это означает нехватку пула аккаунтов, а в редких случаях — недостаток вычислительных ресурсов OpenAI.',
    'Retry the request several times or avoid peak hours.':
      'Повторите запрос несколько раз или избегайте часов пик.',
    'If the problem persists, click About above and contact the administrator.':
      'Если проблема долго не устраняется, нажмите «О программе» выше и обратитесь к администратору.',
    '一直重连，等待网络': 'Постоянное переподключение, ожидание сети',
    用户网络故障: 'Сбой сети пользователя',
    检查网络是否连通: 'Проверьте подключение к сети',
    'Search error keywords...': 'Поиск по ключевым словам ошибок...',
    'Search error keywords': 'Поиск по ключевым словам ошибок',
    'No matching errors found.': 'Подходящие ошибки не найдены.',
    'Model capacity insufficient': 'Недостаточная ёмкость модели',
    'Authentication failed': 'Ошибка аутентификации',
    'Invalid API key': 'Недействительный ключ API',
    'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.':
      'Ключ API указан неверно либо запрос отправляется не на тот сайт, например напрямую на сайт OpenAI.',
    'Re-import through CC Switch (simplest).':
      'Повторно импортируйте через CC Switch (это проще всего).',
    'Confirm that the API key and Base URL match.':
      'Убедитесь, что ключ API и Base URL соответствуют друг другу.',
    'Official OpenAI error': 'Официальная ошибка OpenAI',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'У OpenAI и других официальных поставщиков временно недостаточно вычислительных ресурсов.',
    'OpenAI and other model providers are experiencing server overload.':
      'Серверы OpenAI и других поставщиков моделей перегружены.',
    'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.':
      'Возможно, серверы OpenAI или других поставщиков моделей перегружены либо в OpenAI возникла временная внутренняя ошибка.',
    'The API key or authentication header is missing, expired, or invalid.':
      'Ключ API или заголовок аутентификации отсутствует, просрочен или недействителен.',
    'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.':
      'Убедитесь, что ключ API активен, скопирован без лишних пробелов и имеет доступ к выбранной модели.',
    'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.':
      'Передавайте ключ в формате аутентификации, который требует конечная точка, например Authorization: Bearer <API_KEY>.',
    'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.':
      'Проверьте Base URL клиента: он должен указывать на этот сервис и использовать путь, предусмотренный выбранным протоколом.',
    'Response stream interrupted': 'Поток ответа прерван',
    'The upstream connection or relay path closed before the response completed.':
      'Верхнеуровневое соединение или путь ретрансляции закрылся до завершения ответа.',
    'Retry the request. A transient upstream disconnect can succeed on a later attempt.':
      'Повторите запрос. Временный разрыв соединения с upstream может исчезнуть при следующей попытке.',
    'If the error persists, try another available model or group and check the channel health and usage logs.':
      'Если ошибка сохраняется, попробуйте другую доступную модель или группу и проверьте состояние канала и журналы использования.',
    'Error code': 'Код ошибки',
    'Error description': 'Описание ошибки',
    'Error information': 'Информация об ошибке',
    Cause: 'Причина',
    Solution: 'Решение',
  },
  vi: {
    'Official model errors': 'Lỗi chính thức của mô hình',
    'User client errors': 'Lỗi phía máy khách người dùng',
    'Streaming output interrupted due to too many pending requests':
      'Luồng đầu ra bị gián đoạn do có quá nhiều yêu cầu đang chờ',
    'The upstream channel queue has reached its pending-request limit, usually because the account pool is insufficient; in rare cases, OpenAI compute capacity may be insufficient.':
      'Hàng đợi của kênh upstream đã đạt giới hạn yêu cầu đang chờ, thường do pool tài khoản không đủ; trong một số ít trường hợp, năng lực tính toán của OpenAI có thể không đủ.',
    'Retry the request several times or avoid peak hours.':
      'Thử lại yêu cầu vài lần hoặc tránh giờ cao điểm.',
    'If the problem persists, click About above and contact the administrator.':
      'Nếu vấn đề kéo dài, hãy nhấp vào “Giới thiệu” ở phía trên và liên hệ quản trị viên để hỏi thêm.',
    '一直重连，等待网络': 'Liên tục kết nối lại, đang chờ mạng',
    用户网络故障: 'Sự cố mạng của người dùng',
    检查网络是否连通: 'Kiểm tra kết nối mạng',
    'Search error keywords...': 'Tìm kiếm từ khóa lỗi...',
    'Search error keywords': 'Tìm kiếm từ khóa lỗi',
    'No matching errors found.': 'Không tìm thấy lỗi phù hợp.',
    'Model capacity insufficient': 'Mô hình không đủ năng lực',
    'Authentication failed': 'Xác thực không thành công',
    'Invalid API key': 'Khóa API không hợp lệ',
    'The API key is incorrect, or the request is being sent to the wrong website, such as directly calling the OpenAI website.':
      'Khóa API không đúng hoặc yêu cầu đang được gửi đến sai trang web, chẳng hạn gọi trực tiếp trang web OpenAI.',
    'Re-import through CC Switch (simplest).':
      'Nhập lại qua CC Switch (đơn giản nhất).',
    'Confirm that the API key and Base URL match.':
      'Xác nhận khóa API và Base URL tương ứng với nhau.',
    'Official OpenAI error': 'Lỗi chính thức từ OpenAI',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI và các nhà cung cấp chính thức khác tạm thời không có đủ năng lực tính toán.',
    'OpenAI and other model providers are experiencing server overload.':
      'Máy chủ của OpenAI và các nhà cung cấp mô hình khác đang bị quá tải.',
    'OpenAI and other model providers may be experiencing server overload, or OpenAI may have encountered a temporary internal bug.':
      'Có thể máy chủ của OpenAI hoặc các nhà cung cấp mô hình khác đang quá tải, hoặc OpenAI gặp lỗi nội bộ tạm thời.',
    'The API key or authentication header is missing, expired, or invalid.':
      'API Key hoặc tiêu đề xác thực bị thiếu, hết hạn hoặc không hợp lệ.',
    'Confirm that the API key is active, copied without extra spaces, and allowed to use the selected model.':
      'Xác nhận API Key đang hoạt động, được sao chép không có khoảng trắng thừa và được phép dùng mô hình đã chọn.',
    'Send the key in the authentication format required by the endpoint, such as Authorization: Bearer <API_KEY>.':
      'Gửi khóa theo định dạng xác thực mà endpoint yêu cầu, chẳng hạn Authorization: Bearer <API_KEY>.',
    'Check the client Base URL and make sure it targets this service and uses the path expected by the selected protocol.':
      'Kiểm tra Base URL của máy khách để bảo đảm URL trỏ đến dịch vụ này và dùng đường dẫn phù hợp với giao thức đã chọn.',
    'Response stream interrupted': 'Luồng phản hồi bị gián đoạn',
    'The upstream connection or relay path closed before the response completed.':
      'Kết nối upstream hoặc đường chuyển tiếp đã đóng trước khi phản hồi hoàn tất.',
    'Retry the request. A transient upstream disconnect can succeed on a later attempt.':
      'Thử lại yêu cầu. Sự cố ngắt kết nối upstream tạm thời có thể thành công ở lần thử sau.',
    'If the error persists, try another available model or group and check the channel health and usage logs.':
      'Nếu lỗi vẫn tiếp diễn, hãy thử mô hình hoặc nhóm khác đang khả dụng và kiểm tra trạng thái kênh cùng nhật ký sử dụng.',
    'Error code': 'Mã lỗi',
    'Error description': 'Mô tả lỗi',
    'Error information': 'Thông tin lỗi',
    Cause: 'Nguyên nhân',
    Solution: 'Cách giải quyết',
  },
}

Object.assign(newKeys.en, errorSelfCheckTranslations.en)
Object.assign(newKeys.zh, errorSelfCheckTranslations.zh)
Object.assign(newKeys['zh-TW'], errorSelfCheckTranslations['zh-TW'])
Object.assign(newKeys.fr, errorSelfCheckTranslations.fr)
Object.assign(newKeys.ja, errorSelfCheckTranslations.ja)
Object.assign(newKeys.ru, errorSelfCheckTranslations.ru)
Object.assign(newKeys.vi, errorSelfCheckTranslations.vi)

const homeLandingTranslations = {
  en: {
    'Model gateway address': 'Model gateway address',
    'Import popular clients in one click with CC Switch':
      'Import popular clients in one click with CC Switch',
    'Learn how to configure CC Switch and import in one click':
      'Learn how to configure CC Switch and import in one click',
    'Support multiple clients': 'Support multiple clients',
    'Usage documentation': 'Usage documentation',
    'Various prices at a glance': 'Various prices at a glance',
    'View more in Model Square': 'View more in Model Square',
    'Configure multi-provider models in your client':
      'Configure multi-provider models in your client',
    'Codex, Claude Code, and any other client can connect to any model without local routing conversion':
      'Codex, Claude Code, and any other client can connect to any model without local routing conversion',
    'All models on this site can be used in clients using any API protocol, without local routing or forwarding.':
      'All models on this site can be used in clients using any API protocol, without local routing or forwarding.',
    'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.':
      'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.',
  },
  zh: {
    'Model gateway address': '模型网关地址',
    'Import popular clients in one click with CC Switch':
      '配合 CC Switch 一键导入各种主流客户端',
    'Learn how to configure CC Switch and import in one click':
      '查看如何配置 CC Switch 并一键导入',
    'Support multiple clients': '支持多种客户端',
    'Usage documentation': '使用文档',
    'Various prices at a glance': '各种价格一目了然',
    'View more in Model Square': '去模型广场查看更多',
    'Configure multi-provider models in your client': '客户端配置多厂商模型',
    'Codex, Claude Code, and any other client can connect to any model without local routing conversion':
      'Codex、Claude Code等任意客户端可以接入任意模型，无需本地路由转换',
    'All models on this site can be used in clients using any API protocol, without local routing or forwarding.':
      '本站的所有模型，均可接入任意接口协议的客户端，无需本地路由转发',
    'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.':
      '如右图，在codex客户端里面同时使用deepseek、claude、gpt三家厂商的模型',
  },
  'zh-TW': {
    'Model gateway address': '模型閘道位址',
    'Import popular clients in one click with CC Switch':
      '搭配 CC Switch 一鍵匯入各種主流用戶端',
    'Learn how to configure CC Switch and import in one click':
      '查看如何設定 CC Switch 並一鍵匯入',
    'Support multiple clients': '支援多種用戶端',
    'Usage documentation': '使用文件',
    'Various prices at a glance': '各種價格一目了然',
    'View more in Model Square': '前往模型廣場查看更多',
    'Configure multi-provider models in your client': '用戶端配置多廠商模型',
    'Codex, Claude Code, and any other client can connect to any model without local routing conversion':
      'Codex、Claude Code 等任意用戶端皆可接入任意模型，無需本地路由轉換',
    'All models on this site can be used in clients using any API protocol, without local routing or forwarding.':
      '本站的所有模型，均可接入使用任意介面協定的用戶端，無需本地路由轉發',
    'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.':
      '如右圖，在 Codex 用戶端中同時使用 DeepSeek、Claude、GPT 三家廠商的模型',
  },
  fr: {
    'Model gateway address': 'Adresse de la passerelle de modèles',
    'Import popular clients in one click with CC Switch':
      'Importez vos clients favoris en un clic avec CC Switch',
    'Learn how to configure CC Switch and import in one click':
      'Configurer CC Switch et importer en un clic',
    'Support multiple clients': 'Plusieurs clients compatibles',
    'Usage documentation': 'Guide d’utilisation',
    'Various prices at a glance': 'Aperçu des différents prix',
    'View more in Model Square': 'Voir plus dans la place des modèles',
    'Configure multi-provider models in your client':
      'Configurer plusieurs fournisseurs dans votre client',
    'Codex, Claude Code, and any other client can connect to any model without local routing conversion':
      'Codex, Claude Code et tout client peuvent se connecter à n’importe quel modèle sans routage local',
    'All models on this site can be used in clients using any API protocol, without local routing or forwarding.':
      'Tous les modèles de ce site sont utilisables dans des clients avec n’importe quel protocole API, sans routage ni transfert local.',
    'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.':
      'Comme illustré à droite, utilisez simultanément DeepSeek, Claude et GPT de trois fournisseurs dans Codex.',
  },
  ja: {
    'Model gateway address': 'モデルゲートウェイのアドレス',
    'Import popular clients in one click with CC Switch':
      'CC Switch で主要クライアントにワンクリックでインポート',
    'Learn how to configure CC Switch and import in one click':
      'CC Switch の設定とワンクリックインポートの方法を見る',
    'Support multiple clients': 'さまざまなクライアントに対応',
    'Usage documentation': '利用ガイド',
    'Various prices at a glance': 'さまざまな価格をひと目で確認',
    'View more in Model Square': 'モデル広場でもっと見る',
    'Configure multi-provider models in your client':
      'クライアントで複数プロバイダーのモデルを設定',
    'Codex, Claude Code, and any other client can connect to any model without local routing conversion':
      'Codex、Claude Code など任意のクライアントで、ローカルルーティングなしに任意のモデルへ接続',
    'All models on this site can be used in clients using any API protocol, without local routing or forwarding.':
      'このサイトのすべてのモデルは、任意の API プロトコルを使用するクライアントから、ローカルルーティングや転送なしで利用できます。',
    'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.':
      '右図のように、Codex クライアントで 3 社の DeepSeek、Claude、GPT を同時に利用できます。',
  },
  ru: {
    'Model gateway address': 'Адрес шлюза моделей',
    'Import popular clients in one click with CC Switch':
      'Импорт в популярные клиенты одним нажатием с CC Switch',
    'Learn how to configure CC Switch and import in one click':
      'Как настроить CC Switch и выполнить импорт одним нажатием',
    'Support multiple clients': 'Поддержка разных клиентов',
    'Usage documentation': 'Руководство',
    'Various prices at a glance': 'Разные цены с первого взгляда',
    'View more in Model Square': 'Подробнее на витрине моделей',
    'Configure multi-provider models in your client':
      'Настройка моделей разных поставщиков в клиенте',
    'Codex, Claude Code, and any other client can connect to any model without local routing conversion':
      'Codex, Claude Code и любой клиент могут подключаться к любой модели без локальной маршрутизации',
    'All models on this site can be used in clients using any API protocol, without local routing or forwarding.':
      'Все модели на этом сайте доступны в клиентах с любым протоколом API без локальной маршрутизации или перенаправления.',
    'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.':
      'Как показано справа, в клиенте Codex можно одновременно использовать DeepSeek, Claude и GPT от трех поставщиков.',
  },
  vi: {
    'Model gateway address': 'Địa chỉ cổng mô hình',
    'Import popular clients in one click with CC Switch':
      'Nhập vào các ứng dụng phổ biến bằng một cú nhấp với CC Switch',
    'Learn how to configure CC Switch and import in one click':
      'Xem cách cấu hình CC Switch và nhập bằng một cú nhấp',
    'Support multiple clients': 'Hỗ trợ nhiều ứng dụng',
    'Usage documentation': 'Hướng dẫn sử dụng',
    'Various prices at a glance': 'Các mức giá khác nhau trong nháy mắt',
    'View more in Model Square': 'Xem thêm trong Model Square',
    'Configure multi-provider models in your client':
      'Cấu hình mô hình nhiều nhà cung cấp trong ứng dụng',
    'Codex, Claude Code, and any other client can connect to any model without local routing conversion':
      'Codex, Claude Code và mọi ứng dụng đều có thể kết nối với bất kỳ mô hình nào mà không cần định tuyến cục bộ',
    'All models on this site can be used in clients using any API protocol, without local routing or forwarding.':
      'Tất cả mô hình trên trang này đều có thể dùng trong ứng dụng sử dụng bất kỳ giao thức API nào mà không cần định tuyến hoặc chuyển tiếp cục bộ.',
    'As shown on the right, use DeepSeek, Claude, and GPT from three providers in Codex at the same time.':
      'Như hình bên phải, có thể dùng đồng thời DeepSeek, Claude và GPT từ ba nhà cung cấp trong ứng dụng Codex.',
  },
}

for (const [locale, translations] of Object.entries(homeLandingTranslations)) {
  Object.assign(newKeys[locale], translations)
}
const consoleMonitoringTranslations = {
  en: {
    'Console and monitoring': 'Console and monitoring',
    'Clear call records, visible group status':
      'Clear call records, visible group status',
    'Review model, group, token, cost, and latency for every call, then check group availability from the same console.':
      'Review model, group, token, cost, and latency for every call, then check group availability from the same console.',
    'Model, group, tokens, cost and latency':
      'Model, group, tokens, cost and latency',
    'Tokens / cost': 'Tokens / cost',
    'Group status query': 'Group status query',
    'Check available groups, latency, and recent test results for an API key.':
      'Check available groups, latency, and recent test results for an API key.',
    'Check status': 'Check status',
  },
  zh: {
    'Console and monitoring': '控制台与监控',
    'Clear call records, visible group status': '调用记录清晰，分组状态可查',
    'Review model, group, token, cost, and latency for every call, then check group availability from the same console.':
      '在控制台查看每次调用的模型、分组、Token、费用和耗时，并查询分组可用状态。',
    'Model, group, tokens, cost and latency': '模型、分组、Token、费用与耗时',
    'Tokens / cost': 'Token / 费用',
    'Group status query': '分组状态查询',
    'Check available groups, latency, and recent test results for an API key.':
      '按 API 密钥查询可用分组、延迟和最近检测结果。',
    'Check status': '查询状态',
  },
  'zh-TW': {
    'Console and monitoring': '控制台與監控',
    'Clear call records, visible group status': '呼叫記錄清楚，分組狀態可查',
    'Review model, group, token, cost, and latency for every call, then check group availability from the same console.':
      '在控制台查看每次呼叫的模型、分組、Token、費用和耗時，並查詢分組可用狀態。',
    'Model, group, tokens, cost and latency': '模型、分組、Token、費用與耗時',
    'Tokens / cost': 'Token / 費用',
    'Group status query': '分組狀態查詢',
    'Check available groups, latency, and recent test results for an API key.':
      '依 API 金鑰查詢可用分組、延遲和最近檢測結果。',
    'Check status': '查詢狀態',
  },
  fr: {
    'Console and monitoring': 'Console et surveillance',
    'Clear call records, visible group status':
      'Appels clairement suivis, état des groupes visible',
    'Review model, group, token, cost, and latency for every call, then check group availability from the same console.':
      'Consultez le modèle, le groupe, les tokens, le coût et la latence de chaque appel, puis vérifiez la disponibilité des groupes depuis la même console.',
    'Model, group, tokens, cost and latency':
      'Modèle, groupe, tokens, coût et latence',
    'Tokens / cost': 'Tokens / coût',
    'Group status query': 'État des groupes',
    'Check available groups, latency, and recent test results for an API key.':
      'Vérifiez les groupes disponibles, la latence et les derniers résultats de test pour une clé API.',
    'Check status': 'Vérifier l’état',
  },
  ja: {
    'Console and monitoring': 'コンソールと監視',
    'Clear call records, visible group status':
      '呼び出し履歴を明確に、グループ状態を確認可能に',
    'Review model, group, token, cost, and latency for every call, then check group availability from the same console.':
      '各呼び出しのモデル、グループ、トークン、費用、レイテンシーを確認し、同じコンソールからグループの可用性を照会できます。',
    'Model, group, tokens, cost and latency':
      'モデル、グループ、トークン、費用、レイテンシー',
    'Tokens / cost': 'トークン / 費用',
    'Group status query': 'グループ状態の照会',
    'Check available groups, latency, and recent test results for an API key.':
      'API キーごとに利用可能なグループ、レイテンシー、最新のテスト結果を確認します。',
    'Check status': '状態を確認',
  },
  ru: {
    'Console and monitoring': 'Консоль и мониторинг',
    'Clear call records, visible group status':
      'Понятный журнал вызовов и доступный статус групп',
    'Review model, group, token, cost, and latency for every call, then check group availability from the same console.':
      'Просматривайте модель, группу, токены, стоимость и задержку каждого вызова, а также доступность групп в одной консоли.',
    'Model, group, tokens, cost and latency':
      'Модель, группа, токены, стоимость и задержка',
    'Tokens / cost': 'Токены / стоимость',
    'Group status query': 'Проверка статуса групп',
    'Check available groups, latency, and recent test results for an API key.':
      'Проверяйте доступные группы, задержку и последние результаты тестов для API-ключа.',
    'Check status': 'Проверить статус',
  },
  vi: {
    'Console and monitoring': 'Bảng điều khiển và giám sát',
    'Clear call records, visible group status':
      'Lịch sử gọi rõ ràng, trạng thái nhóm dễ kiểm tra',
    'Review model, group, token, cost, and latency for every call, then check group availability from the same console.':
      'Xem mô hình, nhóm, token, chi phí và độ trễ của từng lần gọi, đồng thời kiểm tra tính khả dụng của nhóm ngay trong cùng bảng điều khiển.',
    'Model, group, tokens, cost and latency':
      'Mô hình, nhóm, token, chi phí và độ trễ',
    'Tokens / cost': 'Token / chi phí',
    'Group status query': 'Tra cứu trạng thái nhóm',
    'Check available groups, latency, and recent test results for an API key.':
      'Kiểm tra các nhóm khả dụng, độ trễ và kết quả kiểm tra gần nhất cho một API key.',
    'Check status': 'Kiểm tra trạng thái',
  },
}

for (const [locale, translations] of Object.entries(
  consoleMonitoringTranslations
)) {
  Object.assign(newKeys[locale], translations)
}

const multiGroupRoutingTranslations = {
  en: {
    'Model group routing': 'Model group routing',
    'One API key, multiple model groups': 'One API key, multiple model groups',
    'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.':
      'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.',
    'Example configuration': 'Example configuration',
    'Selected group order': 'Selected group order',
    'Drag to change priority': 'Drag to change priority',
    'Priority {{priority}}': 'Priority {{priority}}',
    Selected: 'Selected',
    Standby: 'Standby',
    'Model-aware routing': 'Model-aware routing',
    'The request only tries groups that support its model.':
      'The request only tries groups that support its model.',
    'Automatic fallback': 'Automatic fallback',
    'Next in order': 'Next in order',
    'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.':
      'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.',
  },
  zh: {
    'Model group routing': '模型分组路由',
    'One API key, multiple model groups': '一个API密钥，多种模型分组选择',
    'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.':
      '创建密钥时可选择多个分组；请求会按顺序匹配支持该模型的分组，并在需要时自动兜底。',
    'Example configuration': '示例配置',
    'Selected group order': '已选分组顺序',
    'Drag to change priority': '拖动调整优先级',
    'Priority {{priority}}': '优先级 {{priority}}',
    Selected: '已选中',
    Standby: '备用',
    'Model-aware routing': '按模型自动路由',
    'The request only tries groups that support its model.':
      '请求只会尝试支持该模型的分组。',
    'Automatic fallback': '自动兜底',
    'Next in order': '顺序中的下一组',
    'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.':
      '优先尝试第一个兼容分组；不可用时由下一个兼容分组接管，并按最终成功的分组计费。',
  },
  'zh-TW': {
    'Model group routing': '模型分組路由',
    'One API key, multiple model groups': '一個 API 金鑰，多種模型分組選擇',
    'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.':
      '建立金鑰時可選擇多個分組；請求會依序比對支援該模型的分組，並在需要時自動備援。',
    'Example configuration': '範例設定',
    'Selected group order': '已選分組順序',
    'Drag to change priority': '拖曳調整優先順序',
    'Priority {{priority}}': '優先順序 {{priority}}',
    Selected: '已選取',
    Standby: '備用',
    'Model-aware routing': '依模型自動路由',
    'The request only tries groups that support its model.':
      '請求只會嘗試支援該模型的分組。',
    'Automatic fallback': '自動備援',
    'Next in order': '順序中的下一組',
    'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.':
      '優先嘗試第一個相容分組；不可用時由下一個相容分組接手，並依最終成功的分組計費。',
  },
  fr: {
    'Model group routing': 'Routage par groupe de modèles',
    'One API key, multiple model groups':
      'Une clé API, plusieurs groupes de modèles',
    'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.':
      'Sélectionnez plusieurs groupes lors de la création d’une clé. Les requêtes utilisent les groupes compatibles dans l’ordre et basculent automatiquement si nécessaire.',
    'Example configuration': 'Exemple de configuration',
    'Selected group order': 'Ordre des groupes sélectionnés',
    'Drag to change priority': 'Faites glisser pour modifier la priorité',
    'Priority {{priority}}': 'Priorité {{priority}}',
    Selected: 'Sélectionné',
    Standby: 'En attente',
    'Model-aware routing': 'Routage selon le modèle',
    'The request only tries groups that support its model.':
      'La requête essaie uniquement les groupes compatibles avec son modèle.',
    'Automatic fallback': 'Basculement automatique',
    'Next in order': 'Suivant dans l’ordre',
    'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.':
      'Le premier groupe compatible est essayé en priorité. S’il est indisponible, le groupe compatible suivant prend le relais et la facturation utilise le groupe ayant réussi.',
  },
  ja: {
    'Model group routing': 'モデルグループルーティング',
    'One API key, multiple model groups':
      '1 つの API キーで複数のモデルグループを選択',
    'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.':
      'キー作成時に複数のグループを選択できます。リクエストは対応グループを順番に照合し、必要に応じて自動的にフォールバックします。',
    'Example configuration': '設定例',
    'Selected group order': '選択したグループの順序',
    'Drag to change priority': 'ドラッグして優先順位を変更',
    'Priority {{priority}}': '優先順位 {{priority}}',
    Selected: '選択済み',
    Standby: '待機',
    'Model-aware routing': 'モデル対応ルーティング',
    'The request only tries groups that support its model.':
      'リクエストは対象モデルに対応するグループのみを試します。',
    'Automatic fallback': '自動フォールバック',
    'Next in order': '次のグループ',
    'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.':
      '最初の対応グループを優先し、利用できない場合は次の対応グループへ切り替え、成功したグループの倍率で課金します。',
  },
  ru: {
    'Model group routing': 'Маршрутизация по группам моделей',
    'One API key, multiple model groups':
      'Один API-ключ, несколько групп моделей',
    'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.':
      'При создании ключа выберите несколько групп. Запросы последовательно используют совместимые группы и автоматически переключаются при необходимости.',
    'Example configuration': 'Пример конфигурации',
    'Selected group order': 'Порядок выбранных групп',
    'Drag to change priority': 'Перетащите, чтобы изменить приоритет',
    'Priority {{priority}}': 'Приоритет {{priority}}',
    Selected: 'Выбрано',
    Standby: 'Резерв',
    'Model-aware routing': 'Маршрутизация по модели',
    'The request only tries groups that support its model.':
      'Запрос использует только группы, поддерживающие его модель.',
    'Automatic fallback': 'Автоматическое переключение',
    'Next in order': 'Следующая по порядку',
    'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.':
      'Сначала используется первая совместимая группа. Если она недоступна, запрос переходит к следующей, а списание выполняется по группе, обработавшей запрос.',
  },
  vi: {
    'Model group routing': 'Định tuyến theo nhóm mô hình',
    'One API key, multiple model groups': 'Một API key, nhiều nhóm mô hình',
    'Choose multiple groups when creating a key. Requests match compatible groups in order and fall back automatically when needed.':
      'Chọn nhiều nhóm khi tạo khóa. Yêu cầu sẽ lần lượt khớp với các nhóm hỗ trợ mô hình và tự động chuyển dự phòng khi cần.',
    'Example configuration': 'Cấu hình minh họa',
    'Selected group order': 'Thứ tự nhóm đã chọn',
    'Drag to change priority': 'Kéo để thay đổi độ ưu tiên',
    'Priority {{priority}}': 'Ưu tiên {{priority}}',
    Selected: 'Đã chọn',
    Standby: 'Dự phòng',
    'Model-aware routing': 'Định tuyến theo mô hình',
    'The request only tries groups that support its model.':
      'Yêu cầu chỉ thử các nhóm hỗ trợ mô hình tương ứng.',
    'Automatic fallback': 'Tự động chuyển dự phòng',
    'Next in order': 'Nhóm tiếp theo',
    'The first compatible group is tried first. If it is unavailable, the next compatible group takes over and billing uses the group that succeeds.':
      'Nhóm tương thích đầu tiên được ưu tiên. Nếu không khả dụng, nhóm tương thích tiếp theo sẽ tiếp quản và chi phí được tính theo nhóm xử lý thành công.',
  },
}

for (const [locale, translations] of Object.entries(
  multiGroupRoutingTranslations
)) {
  Object.assign(newKeys[locale], translations)
}

const gatewaySectionTranslations = {
  en: {
    'One gateway interface connects every model':
      'One gateway interface connects every model',
  },
  zh: {
    'One gateway interface connects every model': '一个网关接口，贯通所有模型',
  },
  'zh-TW': {
    'One gateway interface connects every model': '一個閘道介面，串聯所有模型',
  },
  fr: {
    'One gateway interface connects every model':
      'Une seule interface de passerelle pour tous les modèles',
  },
  ja: {
    'One gateway interface connects every model':
      '1 つのゲートウェイで、あらゆるモデルへ',
  },
  ru: {
    'One gateway interface connects every model':
      'Один интерфейс шлюза для всех моделей',
  },
  vi: {
    'One gateway interface connects every model':
      'Một giao diện cổng kết nối mọi mô hình',
  },
}

for (const [locale, translations] of Object.entries(
  gatewaySectionTranslations
)) {
  Object.assign(newKeys[locale], translations)
}

const featuredPricingTranslations = {
  en: {
    'Text input': 'Text input',
    'Image output': 'Image output',
    'Standard pricing': 'Standard pricing',
    'Official price': 'Official price',
    'Official price: {{price}}': 'Official price: {{price}}',
    '×{{ratio}} of official price': '×{{ratio}} of official price',
    'Input tokens: ≤272K / >272K': 'Input tokens: ≤272K / >272K',
    'Off-peak / peak pricing (UTC)': 'Off-peak / peak pricing (UTC)',
    'Text and image tokens are priced separately':
      'Text and image tokens are priced separately',
    'Input tokens: <200K / ≥200K': 'Input tokens: <200K / ≥200K',
  },
  zh: {
    'Text input': '文本输入',
    'Image output': '图片输出',
    'Standard pricing': '标准定价',
    'Official price': '官方原价',
    'Official price: {{price}}': '官方原价：{{price}}',
    '×{{ratio}} of official price': 'x{{folds}}折',
    'Input tokens: ≤272K / >272K': '输入 Token：≤272K / >272K',
    'Off-peak / peak pricing (UTC)': '低峰 / 高峰价格（UTC）',
    'Text and image tokens are priced separately': '文本和图片 Token 分别计价',
    'Input tokens: <200K / ≥200K': '输入 Token：<200K / ≥200K',
  },
  'zh-TW': {
    'Text input': '文字輸入',
    'Image output': '圖片輸出',
    'Standard pricing': '標準定價',
    'Official price': '官方原價',
    'Official price: {{price}}': '官方原價：{{price}}',
    '×{{ratio}} of official price': 'x{{folds}}折',
    'Input tokens: ≤272K / >272K': '輸入 Token：≤272K / >272K',
    'Off-peak / peak pricing (UTC)': '離峰 / 尖峰價格（UTC）',
    'Text and image tokens are priced separately': '文字和圖片 Token 分別計價',
    'Input tokens: <200K / ≥200K': '輸入 Token：<200K / ≥200K',
  },
  fr: {
    'Text input': 'Entrée texte',
    'Image output': 'Sortie image',
    'Standard pricing': 'Tarif standard',
    'Official price': 'Prix officiel',
    'Official price: {{price}}': 'Prix officiel : {{price}}',
    '×{{ratio}} of official price': '×{{ratio}} du prix officiel',
    'Input tokens: ≤272K / >272K': 'Tokens en entrée : ≤272K / >272K',
    'Off-peak / peak pricing (UTC)': 'Tarifs heures creuses / pleines (UTC)',
    'Text and image tokens are priced separately':
      'Tokens de texte et d’image facturés séparément',
    'Input tokens: <200K / ≥200K': 'Tokens en entrée : <200K / ≥200K',
  },
  ja: {
    'Text input': 'テキスト入力',
    'Image output': '画像出力',
    'Standard pricing': '標準料金',
    'Official price': '公式価格',
    'Official price: {{price}}': '公式価格：{{price}}',
    '×{{ratio}} of official price': '公式価格の{{ratio}}倍',
    'Input tokens: ≤272K / >272K': '入力トークン：≤272K / >272K',
    'Off-peak / peak pricing (UTC)': 'オフピーク / ピーク料金（UTC）',
    'Text and image tokens are priced separately':
      'テキストと画像のトークンは別料金',
    'Input tokens: <200K / ≥200K': '入力トークン：<200K / ≥200K',
  },
  ru: {
    'Text input': 'Ввод текста',
    'Image output': 'Вывод изображения',
    'Standard pricing': 'Стандартный тариф',
    'Official price': 'Официальная цена',
    'Official price: {{price}}': 'Официальная цена: {{price}}',
    '×{{ratio}} of official price': '×{{ratio}} от официальной цены',
    'Input tokens: ≤272K / >272K': 'Входные токены: ≤272K / >272K',
    'Off-peak / peak pricing (UTC)': 'Вне пика / в пиковые часы (UTC)',
    'Text and image tokens are priced separately':
      'Текстовые и графические токены тарифицируются отдельно',
    'Input tokens: <200K / ≥200K': 'Входные токены: <200K / ≥200K',
  },
  vi: {
    'Text input': 'Văn bản đầu vào',
    'Image output': 'Hình ảnh đầu ra',
    'Standard pricing': 'Giá tiêu chuẩn',
    'Official price': 'Giá gốc chính thức',
    'Official price: {{price}}': 'Giá chính thức: {{price}}',
    '×{{ratio}} of official price': '×{{ratio}} giá chính thức',
    'Input tokens: ≤272K / >272K': 'Token đầu vào: ≤272K / >272K',
    'Off-peak / peak pricing (UTC)': 'Giá thấp điểm / cao điểm (UTC)',
    'Text and image tokens are priced separately':
      'Token văn bản và hình ảnh được tính giá riêng',
    'Input tokens: <200K / ≥200K': 'Token đầu vào: <200K / ≥200K',
  },
}

for (const [locale, translations] of Object.entries(
  featuredPricingTranslations
)) {
  Object.assign(newKeys[locale], translations)
}

// Illustrated Codex tutorial, imported from the archived article.
const codexTutorialTranslations = {
  en: {
    'ChatGPT download': 'ChatGPT download',
    'Windows version': 'Windows version',
    'Standard download': 'Standard download',
    'Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.':
      'Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.',
    'Tutorial note': 'Tutorial note',
    'Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.':
      'Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.',
    'Codex website:': 'Codex website:',
    'ChatGPT desktop website': 'ChatGPT desktop website',
    'Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.':
      'Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.',
    'On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.':
      'On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.',
    'Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.':
      'Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.',
    'If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.':
      'If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.',
    'If the problem persists, use the second method below to install the Codex package directly.':
      'If the problem persists, use the second method below to install the Codex package directly.',
    'The download has started. Wait for the progress bar to finish.':
      'The download has started. Wait for the progress bar to finish.',
    'Download a Microsoft installation package':
      'Download a Microsoft installation package',
    'The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.':
      'The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.',
    'Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.':
      'Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.',
    'Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.':
      'Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.',
    'Desktop shortcut': 'Desktop shortcut',
    'Why is there no ChatGPT desktop shortcut on Windows?':
      'Why is there no ChatGPT desktop shortcut on Windows?',
    'The icon is in Microsoft Store, so you need to search for it when opening the app.':
      'The icon is in Microsoft Store, so you need to search for it when opening the app.',
    'Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.':
      'Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.',
    'Press Win+R to open the Run dialog.':
      'Press Win+R to open the Run dialog.',
    'Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.':
      'Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.',
    'Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.':
      'Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.',
    'macOS version': 'macOS version',
    'Downloading the macOS version is simpler.':
      'Downloading the macOS version is simpler.',
    'After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.':
      'After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.',
    'ChatGPT sign-in': 'ChatGPT sign-in',
    '1. Sign in with an official ChatGPT account':
      '1. Sign in with an official ChatGPT account',
    'Click Continue to sign in.': 'Click Continue to sign in.',
    '2. Sign in with the official ChatGPT API':
      '2. Sign in with the official ChatGPT API',
    'Click Use another sign-in method below, then enter your official OpenAI API key.':
      'Click Use another sign-in method below, then enter your official OpenAI API key.',
    '3. Sign in with a third-party API or gateway':
      '3. Sign in with a third-party API or gateway',
    'Click Use another sign-in method and enter an arbitrary value.':
      'Click Use another sign-in method and enter an arbitrary value.',
    'Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.':
      'Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.',
    'Configure a third-party API in ChatGPT':
      'Configure a third-party API in ChatGPT',
    'Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.':
      'Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.',
    '1. One-click import with CC Switch': '1. One-click import with CC Switch',
    'CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.':
      'CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.',
    '1.1 Install CC Switch': '1.1 Install CC Switch',
    'Download and install the app from the <a0>CC Switch releases page</a0>.':
      'Download and install the app from the <a0>CC Switch releases page</a0>.',
    '1.2 Create an API key': '1.2 Create an API key',
    'Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.':
      'Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.',
    'The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.':
      'The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.',
    '1.3 Configure before importing': '1.3 Configure before importing',
    'Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.':
      'Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.',
    'Name: the configuration name.': 'Name: the configuration name.',
    'Primary model: the default model for new windows. You can change it later.':
      'Primary model: the default model for new windows. You can change it later.',
    '1.4 Confirm the configuration in CC Switch':
      '1.4 Confirm the configuration in CC Switch',
    'Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.':
      'Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.',
    '1.5 Restart the client': '1.5 Restart the client',
    'The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.':
      'The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.',
    'Manual import in CC Switch': 'Manual import in CC Switch',
    '2.1 Add a provider': '2.1 Add a provider',
    'Click Add provider.': 'Click Add provider.',
    'Then choose Custom configuration.': 'Then choose Custom configuration.',
    '2.2 Custom configuration': '2.2 Custom configuration',
    'Fill in the fields as prompted. Make sure the API key and API request URL are correct.':
      'Fill in the fields as prompted. Make sure the API key and API request URL are correct.',
    '3. Edit auth.json and config.toml': '3. Edit auth.json and config.toml',
    'Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.':
      'Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.',
    '3.1 Edit auth.json': '3.1 Edit auth.json',
    'Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.':
      'Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.',
    'In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.':
      'In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.',
    '3.2 Edit config.toml': '3.2 Edit config.toml',
    'Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.':
      'Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.',
    'Conversation history issues': 'Conversation history issues',
    'If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.':
      'If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.',
    '3.3 Restart and verify': '3.3 Restart and verify',
    'Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.':
      'Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.',
    'Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.':
      'Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.',
    'If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.':
      'If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.',
    'Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.':
      'Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.',
    'References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>':
      'References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>',
    'ChatGPT beginner installation guide':
      'ChatGPT beginner installation guide',
    'Download, install, sign in, and configure third-party APIs with this illustrated guide.':
      'Download, install, sign in, and configure third-party APIs with this illustrated guide.',
    'The Codex website': 'The Codex website',
    'The ChatGPT desktop website': 'The ChatGPT desktop website',
    'Run the Windows installer': 'Run the Windows installer',
    'Wait for the installation to finish':
      'Wait for the installation to finish',
    'Microsoft Store package download': 'Microsoft Store package download',
    'Open shell:AppsFolder': 'Open shell:AppsFolder',
    'Create a desktop shortcut': 'Create a desktop shortcut',
    'Install ChatGPT on macOS': 'Install ChatGPT on macOS',
    'ChatGPT sign-in screen': 'ChatGPT sign-in screen',
    'API key sign-in screen': 'API key sign-in screen',
    'CC Switch release downloads': 'CC Switch release downloads',
    'Create and import an API key': 'Create and import an API key',
    'Configure the Codex import': 'Configure the Codex import',
    'Confirm the provider in CC Switch': 'Confirm the provider in CC Switch',
    'Enable the provider and restart the client':
      'Enable the provider and restart the client',
    'Add a provider in CC Switch': 'Add a provider in CC Switch',
    'Choose a custom provider': 'Choose a custom provider',
    'Enter the API key and request URL': 'Enter the API key and request URL',
  },
  zh: {
    'ChatGPT download': 'ChatGPT下载',
    'Windows version': 'Windows版',
    'Standard download': '常规方法下载',
    'Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.':
      '访问OpenAI官网下载ChatGPT安装器，前往<a0>Codex官网：https://openai.com/zh-Hans-CN/codex/</a0>、<a1>ChatGPT桌面客户端官网：https://chatgpt.com/zh-Hans-CN/download/</a1>均可下载',
    'Tutorial note': '注意',
    'Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.':
      'Codex和ChatGPT客户端已经合二为一，Codex注重编程能力，ChatGPT桌面客户端注重日常办公，用户可自行切换模式',
    'Codex website:': 'Codex官网：',
    'ChatGPT desktop website': 'ChatGPT桌面客户端官网',
    'Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.':
      '找到下载好的ChatGPT安装器，双击执行下载（必须开启全局代理）',
    'On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.':
      '目前 Windows 中的 Codex 和 ChatGPT 都只能在微软商店中下载，下载后也只能默认安装在 C 盘中',
    'Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.':
      '必须开启全局代理，因为安装器是通过微软应用商城下载ChatGPT，而中国版本的微软应用商城是会根据你的使用地点检测你是否有下载权限的',
    'If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.':
      '如果后续遇到“ChatGPT客户端不可用、禁用”等问题，很大概率是ChatGPT通过微软应用商城检测到了你的真实使用地址，这时候开启全局代理就能解决。',
    'If the problem persists, use the second method below to install the Codex package directly.':
      '如果实在无法解决，请使用第二个下载安装方法——直接安装Codex安装包',
    'The download has started. Wait for the progress bar to finish.':
      '此时已经开始下载，等待进度条结束就成功了',
    'Download a Microsoft installation package': '微软安装包下载',
    'The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.':
      'Codex APP Windows版本默认只能通过微软商店进行安装，由于不同电脑的设置、网络等差异，可能会导致大家在使用微软商店的时候遇到各种各样的错误，要么报错要么一直重试然后安装失败，今天就教大家怎么直接安装Codex，不经过微软商店。',
    'Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.':
      '打开网站：<a0>https://store.rg-adguard.net</a0> ，下拉框选择Productld，输入微软产品代码 <code>9PLM9XGG6VKS</code>，右边选择RP，点击勾',
    'Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.':
      '下载 <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0> 这个版本()，下载下来之后就是msi直接安装包，直接安装即可',
    'Desktop shortcut': '桌面图标',
    'Why is there no ChatGPT desktop shortcut on Windows?':
      '为什么Windows版ChatGPT没有桌面图标？',
    'The icon is in Microsoft Store, so you need to search for it when opening the app.':
      '因为图标在微软应用市场里面！平时使用需要额外搜索',
    'Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.':
      '目前 Windows 中的 Codex 和 ChatGPT 都只能在微软商店中下载，默认也就只能从微软应用商城打开！',
    'Press Win+R to open the Run dialog.': '同时按住Win+R，打开资源管理器',
    'Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.':
      '输入<code>shell:AppsFolder</code>（如下图），然后按回车',
    'Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.':
      '在打开的页面中，找到ChatGPT图标，右键单单击，点击创建快捷方式',
    'macOS version': 'Mac版',
    'Downloading the macOS version is simpler.': 'Mac版本的下载就简单多了',
    'After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.':
      '下载完成后，双击文件，将ChatGPT/Codex拖入Applications里面，即可完成安装',
    'ChatGPT sign-in': 'ChatGPT登录',
    '1. Sign in with an official ChatGPT account': '1.ChatGPT官方账号登录',
    'Click Continue to sign in.': '直接点击继续登录即可',
    '2. Sign in with the official ChatGPT API': '2.ChatGPT官方API登录',
    'Click Use another sign-in method below, then enter your official OpenAI API key.':
      '点击下方使用其他方式登录，然后输入OpenAI官方API密钥即可',
    '3. Sign in with a third-party API or gateway': '3.第三方API、中转站登录',
    'Click Use another sign-in method and enter an arbitrary value.':
      '点击使用其他方式登录，然后胡乱输入一些乱七八糟的',
    'Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.':
      '下载CC Switch（一款管理各种主流智能体配置的插件）或者自行修改config.toml，详情看下文《ChatGPT配置》',
    'Configure a third-party API in ChatGPT': 'ChatGPT配置第三方API',
    'Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.':
      '可选择使用 CC Switch 一键配置，或手动编辑 <code>auth.json</code> 和 <code>config.toml</code> 接入 Codex。',
    '1. One-click import with CC Switch': '1. 使用 CC Switch 一键导入',
    'CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.':
      'CC Switch 可以导入 API Key、所选模型和 Codex 服务地址，无需手动编辑 TOML。',
    '1.1 Install CC Switch': '1.1 安装cc switch',
    'Download and install the app from the <a0>CC Switch releases page</a0>.':
      '从 <a0>CC Switch 发布页</a0> 下载并安装应用',
    '1.2 Create an API key': '1.2 创建密钥',
    'Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.':
      '打开模型服务商/中转站的API key页面，创建密钥，并一键导入。（大部分中转站可以一键导入，少部分需要手动把密钥的base url 和 key自己放入cc switch的配置，手动导入参考<a0>cc switch官方快速上手文档</a0>）',
    'The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.':
      '第三方API需要支持Response协议，一般中转站/模型厂商都会支持，只有Claude官方可能不支持',
    '1.3 Configure before importing': '1.3 导入前配置',
    'Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.':
      '选择 Codex，再选择当前 API Key 可用的主模型，并保留自动生成的 <code>/v1</code> 地址。',
    'Name: the configuration name.': '名称：配置名',
    'Primary model: the default model for new windows. You can change it later.':
      '主模型：新开窗口默认的模型，可以后期自己调整',
    '1.4 Confirm the configuration in CC Switch': '1.4 CC Switch侧确认配置',
    'Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.':
      '确认浏览器打开链接的提示，然后在 CC Switch 中检查、保存并启用导入的服务商。',
    '1.5 Restart the client': '1.5 重启客户端',
    'The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.':
      '导入内容包含所选 API Key、主模型和以 <code>/v1</code> 结尾的服务地址。请只在自己的设备上确认导入。',
    'Manual import in CC Switch': '手动导入CC Switch',
    '2.1 Add a provider': '2.1 添加供应商',
    'Click Add provider.': '点击添加供应商',
    'Then choose Custom configuration.': '然后点击自定义配置',
    '2.2 Custom configuration': '2.2 自定义配置',
    'Fill in the fields as prompted. Make sure the API key and API request URL are correct.':
      '根据提示进行填写，重点是API Key和API请求地址不要填写错误',
    '3. Edit auth.json and config.toml':
      '3. 编辑 <code>auth.json</code> 和 <code>config.toml</code>',
    'Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.':
      'Codex 从用户目录中的 <code>auth.json</code> 读取认证凭据，从 <code>config.toml</code> 读取服务商、模型和接口设置。',
    '3.1 Edit auth.json': '3.1编辑 <code>auth.json</code>',
    'Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.':
      '在用户级 <code>auth.json</code> 中写入 API Key。如果文件中已有其他认证字段，只更新 <code>OPENAI_API_KEY</code>，不要覆盖整个文件。',
    'In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.':
      '文件模式下的 <code>auth.json</code> 包含明文密钥，请只保存在自己的设备上，不要提交到 Git 或发送给他人。密钥保存在 <code>auth.json</code>；<code>config.toml</code> 只用于服务商、模型和接口设置，不要依赖当前终端的临时设置。',
    '3.2 Edit config.toml': '3.2编辑 <code>config.toml</code>',
    'Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.':
      '保留现有 Codex 配置，并合并下面的服务商配置块。已有 <code>cli_auth_credentials_store</code> 时将它改为 <code>"file"</code>，不要重复添加。按 API Key 的可用模型替换示例模型。',
    'Conversation history issues': '会话历史相关问题',
    'If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.':
      '如果原有对话历史没有显示，检查当前 <code>config.toml</code> 中的 <code>model_provider</code> 是否与创建对话时使用的值完全一致（包括大小写）。改回原值并重启 Codex 后，可运行 <code>codex resume</code> 查找历史；这通常不代表记录已被删除。',
    '3.3 Restart and verify': '3.3重启并验证',
    'Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.':
      '保存 <code>auth.json</code> 和 <code>config.toml</code>，重启正在运行的 Codex 应用或 IDE 扩展。',
    'Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.':
      '在终端运行 <code>codex</code> 并发起一个测试任务；认证信息由 <code>auth.json</code> 提供。',
    'If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.':
      '如启动失败，运行 <code>codex doctor</code> 检查认证文件是否被读取，再检查模型名称和 Base URL。',
    'Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.':
      '请保持 <code>wire_api = "responses"</code>，并确保 Base URL 以 <code>/v1</code> 结尾。<code>cli_auth_credentials_store = "file"</code> 让 Codex 使用 <code>auth.json</code> 保存和读取凭据；<code>requires_openai_auth = true</code> 让自定义服务商使用该凭据。',
    'References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>':
      '参考：<a0>Codex 认证说明</a0> · <a1>Codex 配置参考</a1>',
    'ChatGPT beginner installation guide': 'ChatGPT保姆级安装教程',
    'Download, install, sign in, and configure third-party APIs with this illustrated guide.':
      '下载、安装、登录和第三方 API 配置完整图文教程。',
    'The Codex website': 'Codex 官网',
    'The ChatGPT desktop website': 'ChatGPT 桌面客户端官网',
    'Run the Windows installer': '运行 Windows 安装器',
    'Wait for the installation to finish': '等待安装完成',
    'Microsoft Store package download': '微软商店安装包下载',
    'Open shell:AppsFolder': '打开 shell:AppsFolder',
    'Create a desktop shortcut': '创建桌面快捷方式',
    'Install ChatGPT on macOS': '在 macOS 中安装 ChatGPT',
    'ChatGPT sign-in screen': 'ChatGPT 登录界面',
    'API key sign-in screen': 'API Key 登录界面',
    'CC Switch release downloads': 'CC Switch 发布页安装包列表',
    'Create and import an API key': '创建并导入 API Key',
    'Configure the Codex import': '配置 Codex 导入信息',
    'Confirm the provider in CC Switch': '在 CC Switch 中确认服务商',
    'Enable the provider and restart the client': '启用服务商并重启客户端',
    'Add a provider in CC Switch': '在 CC Switch 中添加供应商',
    'Choose a custom provider': '选择自定义供应商',
    'Enter the API key and request URL': '填写 API Key 和 API 请求地址',
  },
  fr: {
    'ChatGPT download': 'Télécharger ChatGPT',
    'Windows version': 'Version Windows',
    'Standard download': 'Téléchargement standard',
    'Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.':
      'Téléchargez le programme d’installation de ChatGPT depuis le <a0>site officiel de Codex : https://openai.com/zh-Hans-CN/codex/</a0> ou le <a1>site de ChatGPT pour ordinateur : https://chatgpt.com/zh-Hans-CN/download/</a1>.',
    'Tutorial note': 'Remarque',
    'Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.':
      'Codex et ChatGPT sont réunis dans un même client. Codex privilégie la programmation, tandis que ChatGPT se concentre sur le travail quotidien. Vous pouvez changer de mode.',
    'Codex website:': 'Site officiel de Codex :',
    'ChatGPT desktop website': 'Site de ChatGPT pour ordinateur',
    'Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.':
      'Retrouvez le programme d’installation téléchargé et double-cliquez dessus pour lancer le téléchargement. Le mode proxy global doit être activé.',
    'On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.':
      'Sous Windows, Codex et ChatGPT ne se téléchargent actuellement que dans le Microsoft Store et s’installent par défaut sur le disque C:.',
    'Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.':
      'Activez le mode proxy global : le programme d’installation télécharge ChatGPT via le Microsoft Store, dont la version chinoise vérifie votre droit au téléchargement selon votre localisation.',
    'If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.':
      'Si ChatGPT indique ensuite que le client est indisponible ou désactivé, le Microsoft Store a peut-être détecté votre position réelle. Le mode proxy global peut résoudre ce problème.',
    'If the problem persists, use the second method below to install the Codex package directly.':
      'Si le problème persiste, utilisez la deuxième méthode ci-dessous pour installer directement le paquet Codex.',
    'The download has started. Wait for the progress bar to finish.':
      'Le téléchargement a commencé. Attendez que la barre de progression soit terminée.',
    'Download a Microsoft installation package':
      'Télécharger un paquet d’installation Microsoft',
    'The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.':
      'L’application Codex pour Windows s’installe normalement via le Microsoft Store. Les paramètres des ordinateurs et les réseaux peuvent provoquer des erreurs, des tentatives répétées ou des échecs d’installation. Cette méthode installe Codex directement, sans passer par le Store.',
    'Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.':
      'Ouvrez <a0>https://store.rg-adguard.net</a0>, choisissez ProductId, saisissez le code produit Microsoft <code>9PLM9XGG6VKS</code>, choisissez RP à droite, puis cliquez sur la coche.',
    'Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.':
      'Téléchargez <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. Le fichier téléchargé est un programme d’installation autonome ; ouvrez-le pour installer l’application.',
    'Desktop shortcut': 'Raccourci sur le bureau',
    'Why is there no ChatGPT desktop shortcut on Windows?':
      'Pourquoi ChatGPT n’a-t-il pas de raccourci sur le bureau sous Windows ?',
    'The icon is in Microsoft Store, so you need to search for it when opening the app.':
      'L’icône se trouve dans le Microsoft Store ; il faut donc la rechercher pour ouvrir l’application.',
    'Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.':
      'Sous Windows, Codex et ChatGPT se téléchargent actuellement via le Microsoft Store et s’ouvrent normalement depuis celui-ci.',
    'Press Win+R to open the Run dialog.':
      'Appuyez sur Win+R pour ouvrir la boîte de dialogue Exécuter.',
    'Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.':
      'Saisissez <code>shell:AppsFolder</code>, comme ci-dessous, puis appuyez sur Entrée.',
    'Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.':
      'Dans la fenêtre qui s’ouvre, trouvez ChatGPT, faites un clic droit, puis choisissez Créer un raccourci.',
    'macOS version': 'Version macOS',
    'Downloading the macOS version is simpler.':
      'Le téléchargement de la version macOS est plus simple.',
    'After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.':
      'Après le téléchargement, double-cliquez sur le fichier et faites glisser ChatGPT/Codex dans Applications pour terminer l’installation.',
    'ChatGPT sign-in': 'Connexion à ChatGPT',
    '1. Sign in with an official ChatGPT account':
      '1. Se connecter avec un compte ChatGPT officiel',
    'Click Continue to sign in.': 'Cliquez sur Continuer pour vous connecter.',
    '2. Sign in with the official ChatGPT API':
      '2. Se connecter avec l’API officielle de ChatGPT',
    'Click Use another sign-in method below, then enter your official OpenAI API key.':
      'Cliquez sur Utiliser une autre méthode de connexion, puis saisissez votre clé API OpenAI officielle.',
    '3. Sign in with a third-party API or gateway':
      '3. Se connecter avec une API tierce ou une passerelle',
    'Click Use another sign-in method and enter an arbitrary value.':
      'Cliquez sur Utiliser une autre méthode de connexion et saisissez une valeur quelconque.',
    'Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.':
      'Téléchargez CC Switch, un gestionnaire de configuration pour les agents courants, ou modifiez vous-même config.toml. Consultez la section de configuration ci-dessous.',
    'Configure a third-party API in ChatGPT':
      'Configurer une API tierce dans ChatGPT',
    'Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.':
      'Utilisez CC Switch pour une configuration en un clic ou modifiez manuellement <code>auth.json</code> et <code>config.toml</code> pour connecter Codex.',
    '1. One-click import with CC Switch':
      '1. Importer en un clic avec CC Switch',
    'CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.':
      'CC Switch importe votre clé API, le modèle choisi et l’URL du service Codex sans modification manuelle du TOML.',
    '1.1 Install CC Switch': '1.1 Installer CC Switch',
    'Download and install the app from the <a0>CC Switch releases page</a0>.':
      'Téléchargez et installez l’application depuis la <a0>page des versions de CC Switch</a0>.',
    '1.2 Create an API key': '1.2 Créer une clé API',
    'Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.':
      'Ouvrez la page des clés API de votre fournisseur ou passerelle, créez une clé et importez-la en un clic. La plupart des passerelles le permettent ; pour les autres, saisissez manuellement l’URL de base et la clé dans CC Switch. Consultez le <a0>guide de démarrage officiel de CC Switch</a0> pour l’importation manuelle.',
    'The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.':
      'L’API tierce doit prendre en charge le protocole Responses. La plupart des passerelles et fournisseurs le font ; l’API officielle de Claude peut ne pas le prendre en charge.',
    '1.3 Configure before importing': '1.3 Configurer avant l’importation',
    'Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.':
      'Sélectionnez Codex, choisissez un modèle principal accessible avec cette clé API et conservez l’URL <code>/v1</code> générée automatiquement.',
    'Name: the configuration name.': 'Nom : le nom de la configuration.',
    'Primary model: the default model for new windows. You can change it later.':
      'Modèle principal : le modèle par défaut des nouvelles fenêtres. Vous pourrez le modifier ultérieurement.',
    '1.4 Confirm the configuration in CC Switch':
      '1.4 Confirmer la configuration dans CC Switch',
    'Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.':
      'Acceptez la demande du navigateur pour ouvrir le lien, puis vérifiez, enregistrez et activez le fournisseur importé dans CC Switch.',
    '1.5 Restart the client': '1.5 Redémarrer le client',
    'The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.':
      'L’importation comprend la clé API choisie, le modèle principal et une URL de service se terminant par <code>/v1</code>. Confirmez l’importation uniquement sur votre propre appareil.',
    'Manual import in CC Switch': 'Importation manuelle dans CC Switch',
    '2.1 Add a provider': '2.1 Ajouter un fournisseur',
    'Click Add provider.': 'Cliquez sur Ajouter un fournisseur.',
    'Then choose Custom configuration.':
      'Choisissez ensuite Configuration personnalisée.',
    '2.2 Custom configuration': '2.2 Configuration personnalisée',
    'Fill in the fields as prompted. Make sure the API key and API request URL are correct.':
      'Remplissez les champs indiqués. Vérifiez surtout la clé API et l’URL de requête de l’API.',
    '3. Edit auth.json and config.toml': '3. Modifier auth.json et config.toml',
    'Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.':
      'Codex lit les identifiants dans <code>auth.json</code>, dans votre dossier utilisateur, et les paramètres de fournisseur, de modèle et d’API dans <code>config.toml</code>.',
    '3.1 Edit auth.json': '3.1 Modifier auth.json',
    'Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.':
      'Saisissez votre clé API dans le fichier utilisateur <code>auth.json</code>. S’il contient déjà d’autres champs d’authentification, modifiez uniquement <code>OPENAI_API_KEY</code> sans écraser tout le fichier.',
    'In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.':
      'En mode fichier, <code>auth.json</code> contient une clé API en clair. Conservez-le uniquement sur votre appareil, ne le partagez pas et ne le publiez jamais dans Git. Stockez la clé dans <code>auth.json</code> et réservez <code>config.toml</code> aux paramètres de fournisseur, de modèle et d’API, sans dépendre des réglages temporaires du terminal.',
    '3.2 Edit config.toml': '3.2 Modifier config.toml',
    'Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.':
      'Conservez vos paramètres Codex existants et fusionnez le bloc fournisseur ci-dessous. Si <code>cli_auth_credentials_store</code> existe déjà, remplacez sa valeur par <code>"file"</code> sans ajouter de doublon. Remplacez le modèle d’exemple par un modèle accessible avec votre clé API.',
    'Conversation history issues': 'Problèmes d’historique des conversations',
    'If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.':
      'Si vos anciennes conversations n’apparaissent pas, vérifiez que <code>model_provider</code> dans <code>config.toml</code> correspond exactement à la valeur utilisée lors de leur création, majuscules comprises. Rétablissez cette valeur, redémarrez Codex et utilisez <code>codex resume</code> pour retrouver l’historique. Les données ne sont généralement pas supprimées.',
    '3.3 Restart and verify': '3.3 Redémarrer et vérifier',
    'Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.':
      'Enregistrez <code>auth.json</code> et <code>config.toml</code>, puis redémarrez l’application Codex ou l’extension IDE en cours d’exécution.',
    'Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.':
      'Exécutez <code>codex</code> dans un terminal et lancez une tâche de test. Les identifiants proviennent de <code>auth.json</code>.',
    'If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.':
      'Si le démarrage échoue, exécutez <code>codex doctor</code> pour vérifier la lecture du fichier d’identifiants, puis contrôlez le nom du modèle et l’URL de base.',
    'Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.':
      'Conservez <code>wire_api = "responses"</code> et une URL de base se terminant par <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> permet à Codex de lire et d’enregistrer les identifiants dans <code>auth.json</code> ; <code>requires_openai_auth = true</code> indique au fournisseur personnalisé de les utiliser.',
    'References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>':
      'Références : <a0>Authentification Codex</a0> · <a1>Référence de configuration Codex</a1>',
    'ChatGPT beginner installation guide':
      'Guide d’installation de ChatGPT pour débutants',
    'Download, install, sign in, and configure third-party APIs with this illustrated guide.':
      'Guide illustré pour télécharger, installer et configurer les API tierces, et se connecter.',
    'The Codex website': 'Le site officiel de Codex',
    'The ChatGPT desktop website': 'Le site de ChatGPT pour ordinateur',
    'Run the Windows installer': 'Exécuter le programme d’installation Windows',
    'Wait for the installation to finish': 'Attendre la fin de l’installation',
    'Microsoft Store package download':
      'Téléchargement du paquet Microsoft Store',
    'Open shell:AppsFolder': 'Ouvrir shell:AppsFolder',
    'Create a desktop shortcut': 'Créer un raccourci sur le bureau',
    'Install ChatGPT on macOS': 'Installer ChatGPT sur macOS',
    'ChatGPT sign-in screen': 'Écran de connexion à ChatGPT',
    'API key sign-in screen': 'Écran de connexion par clé API',
    'CC Switch release downloads': 'Paquets d’installation de CC Switch',
    'Create and import an API key': 'Créer et importer une clé API',
    'Configure the Codex import': 'Configurer l’importation Codex',
    'Confirm the provider in CC Switch':
      'Confirmer le fournisseur dans CC Switch',
    'Enable the provider and restart the client':
      'Activer le fournisseur et redémarrer le client',
    'Add a provider in CC Switch': 'Ajouter un fournisseur dans CC Switch',
    'Choose a custom provider': 'Choisir un fournisseur personnalisé',
    'Enter the API key and request URL':
      'Saisir la clé API et l’URL de requête',
  },
  ja: {
    'ChatGPT download': 'ChatGPT のダウンロード',
    'Windows version': 'Windows 版',
    'Standard download': '通常のダウンロード方法',
    'Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.':
      '公式の <a0>Codex サイト：https://openai.com/zh-Hans-CN/codex/</a0> または <a1>ChatGPT デスクトップ版サイト：https://chatgpt.com/zh-Hans-CN/download/</a1> から ChatGPT インストーラーをダウンロードします。',
    'Tutorial note': '注意',
    'Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.':
      'Codex と ChatGPT は一つのクライアントに統合されています。Codex はコーディング、ChatGPT は日常業務を重視しており、モードを切り替えて利用できます。',
    'Codex website:': 'Codex 公式サイト：',
    'ChatGPT desktop website': 'ChatGPT デスクトップ版公式サイト',
    'Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.':
      'ダウンロードした ChatGPT インストーラーをダブルクリックして、ダウンロードを開始します。プロキシのグローバルモードを有効にしてください。',
    'On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.':
      'Windows の Codex と ChatGPT は、現在 Microsoft Store からのみダウンロードでき、標準では C ドライブにインストールされます。',
    'Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.':
      'グローバルプロキシを有効にしてください。インストーラーは Microsoft Store 経由で ChatGPT をダウンロードし、中国版のストアは利用場所に基づいてダウンロード資格を確認します。',
    'If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.':
      'その後 ChatGPT が利用不可または無効と表示された場合、Microsoft Store が実際の利用場所を検出した可能性があります。グローバルプロキシを有効にすると解決する場合があります。',
    'If the problem persists, use the second method below to install the Codex package directly.':
      '解決しない場合は、以下の二つ目の方法で Codex のパッケージを直接インストールしてください。',
    'The download has started. Wait for the progress bar to finish.':
      'ダウンロードが始まりました。進行状況バーが完了するまで待ちます。',
    'Download a Microsoft installation package':
      'Microsoft インストールパッケージのダウンロード',
    'The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.':
      'Windows 版 Codex アプリは通常 Microsoft Store からインストールします。PC の設定やネットワークの違いによって、エラー、繰り返しの再試行、インストール失敗が発生することがあります。ここではストアを使わずに直接インストールします。',
    'Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.':
      '<a0>https://store.rg-adguard.net</a0> を開き、ProductId を選択して製品コード <code>9PLM9XGG6VKS</code> を入力します。右側で RP を選び、チェックマークをクリックします。',
    'Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.':
      '<a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0> をダウンロードします。単体のインストールファイルなので、開いてインストールできます。',
    'Desktop shortcut': 'デスクトップショートカット',
    'Why is there no ChatGPT desktop shortcut on Windows?':
      'Windows 版 ChatGPT にデスクトップショートカットがないのはなぜですか？',
    'The icon is in Microsoft Store, so you need to search for it when opening the app.':
      'アイコンは Microsoft Store にあるため、普段は検索してアプリを開く必要があります。',
    'Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.':
      'Windows の Codex と ChatGPT は現在 Microsoft Store からダウンロードし、通常はストアから起動します。',
    'Press Win+R to open the Run dialog.':
      'Win+R を押して「ファイル名を指定して実行」を開きます。',
    'Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.':
      '下図のように <code>shell:AppsFolder</code> を入力し、Enter を押します。',
    'Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.':
      '開いたウィンドウで ChatGPT を探し、右クリックして「ショートカットの作成」を選びます。',
    'macOS version': 'macOS 版',
    'Downloading the macOS version is simpler.':
      'macOS 版のダウンロードは簡単です。',
    'After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.':
      'ダウンロードしたファイルをダブルクリックし、ChatGPT/Codex を Applications にドラッグするとインストール完了です。',
    'ChatGPT sign-in': 'ChatGPT へのログイン',
    '1. Sign in with an official ChatGPT account':
      '1. 公式 ChatGPT アカウントでログイン',
    'Click Continue to sign in.': '「続ける」をクリックしてログインします。',
    '2. Sign in with the official ChatGPT API':
      '2. 公式 ChatGPT API でログイン',
    'Click Use another sign-in method below, then enter your official OpenAI API key.':
      '下部の「別の方法でログイン」をクリックし、公式 OpenAI API キーを入力します。',
    '3. Sign in with a third-party API or gateway':
      '3. サードパーティ API・ゲートウェイでログイン',
    'Click Use another sign-in method and enter an arbitrary value.':
      '「別の方法でログイン」をクリックし、任意の値を入力します。',
    'Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.':
      '主要なエージェントの設定を管理する CC Switch をダウンロードするか、config.toml を直接編集します。以下の設定セクションを参照してください。',
    'Configure a third-party API in ChatGPT':
      'ChatGPT にサードパーティ API を設定',
    'Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.':
      'CC Switch でワンクリック設定するか、<code>auth.json</code> と <code>config.toml</code> を手動編集して Codex に接続します。',
    '1. One-click import with CC Switch':
      '1. CC Switch でワンクリックインポート',
    'CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.':
      'CC Switch は API キー、選択したモデル、Codex のサービス URL をインポートでき、TOML の手動編集は不要です。',
    '1.1 Install CC Switch': '1.1 CC Switch のインストール',
    'Download and install the app from the <a0>CC Switch releases page</a0>.':
      '<a0>CC Switch のリリースページ</a0>からアプリをダウンロードしてインストールします。',
    '1.2 Create an API key': '1.2 API キーの作成',
    'Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.':
      'モデルプロバイダーまたはゲートウェイの API キーページを開き、キーを作成してワンクリックでインポートします。多くのゲートウェイが対応していますが、一部ではベース URL とキーを CC Switch に手動入力します。手動の場合は <a0>CC Switch 公式クイックスタート</a0>を参照してください。',
    'The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.':
      'サードパーティ API は Responses プロトコルに対応している必要があります。通常はゲートウェイやモデルプロバイダーが対応していますが、公式 Claude API は未対応の場合があります。',
    '1.3 Configure before importing': '1.3 インポート前の設定',
    'Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.':
      'Codex を選択し、API キーで利用できるメインモデルを選びます。自動生成された <code>/v1</code> URL はそのままにします。',
    'Name: the configuration name.': '名前：設定の名前です。',
    'Primary model: the default model for new windows. You can change it later.':
      'メインモデル：新しいウィンドウで使う標準モデルです。後から変更できます。',
    '1.4 Confirm the configuration in CC Switch':
      '1.4 CC Switch 側で設定を確認',
    'Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.':
      'リンクを開くブラウザーの確認に同意し、CC Switch でインポートされたプロバイダーを確認・保存して有効にします。',
    '1.5 Restart the client': '1.5 クライアントを再起動',
    'The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.':
      'インポートには、選択した API キー、メインモデル、<code>/v1</code> で終わるサービス URL が含まれます。必ず自分の端末でのみ承認してください。',
    'Manual import in CC Switch': 'CC Switch への手動インポート',
    '2.1 Add a provider': '2.1 プロバイダーの追加',
    'Click Add provider.': '「プロバイダーを追加」をクリックします。',
    'Then choose Custom configuration.': '続いて「カスタム設定」を選びます。',
    '2.2 Custom configuration': '2.2 カスタム設定',
    'Fill in the fields as prompted. Make sure the API key and API request URL are correct.':
      '案内に従って入力します。特に API キーと API リクエスト URL に間違いがないことを確認してください。',
    '3. Edit auth.json and config.toml': '3. auth.json と config.toml の編集',
    'Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.':
      'Codex はユーザーディレクトリ内の <code>auth.json</code> から認証情報を、<code>config.toml</code> からプロバイダー・モデル・API の設定を読み込みます。',
    '3.1 Edit auth.json': '3.1 auth.json の編集',
    'Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.':
      'ユーザー単位の <code>auth.json</code> に API キーを入力します。他の認証項目が既にある場合は、<code>OPENAI_API_KEY</code> だけを更新し、ファイル全体を上書きしないでください。',
    'In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.':
      'ファイルモードの <code>auth.json</code> には平文の API キーが含まれます。自分の端末だけに保存し、Git にコミットしたり他人に送ったりしないでください。キーは <code>auth.json</code> に保存し、<code>config.toml</code> はプロバイダー・モデル・API の設定だけに使用します。一時的なターミナル設定に依存しないでください。',
    '3.2 Edit config.toml': '3.2 config.toml の編集',
    'Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.':
      '既存の Codex 設定を残し、以下のプロバイダーブロックを統合します。<code>cli_auth_credentials_store</code> が既にあれば、重複追加せず <code>"file"</code> に変更します。サンプルのモデル名は API キーで利用可能なものに置き換えてください。',
    'Conversation history issues': '会話履歴に関する問題',
    'If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.':
      '以前の会話が表示されない場合、<code>config.toml</code> 内の <code>model_provider</code> が会話作成時の値と大文字・小文字も含めて一致しているか確認します。元の値に戻して Codex を再起動し、<code>codex resume</code> で履歴を探してください。通常、記録自体が削除されたわけではありません。',
    '3.3 Restart and verify': '3.3 再起動して確認',
    'Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.':
      '<code>auth.json</code> と <code>config.toml</code> を保存し、実行中の Codex アプリまたは IDE 拡張を再起動します。',
    'Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.':
      'ターミナルで <code>codex</code> を実行し、テストタスクを開始します。認証情報は <code>auth.json</code> から提供されます。',
    'If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.':
      '起動できない場合は <code>codex doctor</code> を実行し、認証ファイルが読み込まれているか確認してから、モデル名と Base URL を確認します。',
    'Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.':
      '<code>wire_api = "responses"</code> を維持し、Base URL を <code>/v1</code> で終わらせてください。<code>cli_auth_credentials_store = "file"</code> は <code>auth.json</code> での認証情報の保存・読み込みを指定し、<code>requires_openai_auth = true</code> はカスタムプロバイダーでその情報を使用するよう指定します。',
    'References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>':
      '参考：<a0>Codex 認証ガイド</a0> · <a1>Codex 設定リファレンス</a1>',
    'ChatGPT beginner installation guide':
      'ChatGPT 初心者向けインストールガイド',
    'Download, install, sign in, and configure third-party APIs with this illustrated guide.':
      'ダウンロード、インストール、ログイン、サードパーティ API 設定を画像付きで解説します。',
    'The Codex website': 'Codex 公式サイト',
    'The ChatGPT desktop website': 'ChatGPT デスクトップ版公式サイト',
    'Run the Windows installer': 'Windows インストーラーを実行',
    'Wait for the installation to finish': 'インストールの完了を待つ',
    'Microsoft Store package download':
      'Microsoft Store パッケージのダウンロード',
    'Open shell:AppsFolder': 'shell:AppsFolder を開く',
    'Create a desktop shortcut': 'デスクトップショートカットを作成',
    'Install ChatGPT on macOS': 'macOS に ChatGPT をインストール',
    'ChatGPT sign-in screen': 'ChatGPT のログイン画面',
    'API key sign-in screen': 'API キーのログイン画面',
    'CC Switch release downloads': 'CC Switch のリリースファイル一覧',
    'Create and import an API key': 'API キーを作成してインポート',
    'Configure the Codex import': 'Codex のインポート設定',
    'Confirm the provider in CC Switch': 'CC Switch でプロバイダーを確認',
    'Enable the provider and restart the client':
      'プロバイダーを有効にしてクライアントを再起動',
    'Add a provider in CC Switch': 'CC Switch にプロバイダーを追加',
    'Choose a custom provider': 'カスタムプロバイダーを選択',
    'Enter the API key and request URL': 'API キーとリクエスト URL を入力',
  },
  ru: {
    'ChatGPT download': 'Скачивание ChatGPT',
    'Windows version': 'Версия для Windows',
    'Standard download': 'Обычное скачивание',
    'Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.':
      'Скачайте установщик ChatGPT с <a0>официального сайта Codex: https://openai.com/zh-Hans-CN/codex/</a0> или с <a1>сайта настольного ChatGPT: https://chatgpt.com/zh-Hans-CN/download/</a1>.',
    'Tutorial note': 'Примечание',
    'Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.':
      'Codex и ChatGPT объединены в одном клиенте. Codex предназначен прежде всего для программирования, а ChatGPT — для повседневной работы. Режимы можно переключать.',
    'Codex website:': 'Официальный сайт Codex:',
    'ChatGPT desktop website': 'Официальный сайт настольного ChatGPT',
    'Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.':
      'Найдите скачанный установщик ChatGPT и запустите его двойным щелчком. Должен быть включён глобальный режим прокси.',
    'On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.':
      'В Windows Codex и ChatGPT сейчас доступны для скачивания только через Microsoft Store и по умолчанию устанавливаются на диск C:.',
    'Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.':
      'Включите глобальный режим прокси: установщик скачивает ChatGPT через Microsoft Store, а китайская версия магазина проверяет доступность скачивания по вашему местоположению.',
    'If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.':
      'Если позже ChatGPT сообщает, что клиент недоступен или отключён, Microsoft Store мог определить ваше реальное местоположение. Глобальный режим прокси может помочь.',
    'If the problem persists, use the second method below to install the Codex package directly.':
      'Если проблему не удалось решить, воспользуйтесь вторым способом ниже и установите пакет Codex напрямую.',
    'The download has started. Wait for the progress bar to finish.':
      'Скачивание началось. Дождитесь заполнения индикатора выполнения.',
    'Download a Microsoft installation package':
      'Скачивание установочного пакета Microsoft',
    'The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.':
      'Обычно Codex для Windows устанавливается через Microsoft Store. Различия в настройках компьютера и сети могут вызывать ошибки, повторные попытки и сбои установки. Этот способ позволяет установить Codex напрямую, без магазина.',
    'Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.':
      'Откройте <a0>https://store.rg-adguard.net</a0>, выберите ProductId, введите код продукта Microsoft <code>9PLM9XGG6VKS</code>, справа выберите RP и нажмите галочку.',
    'Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.':
      'Скачайте <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. Это самостоятельный установочный файл; откройте его для установки.',
    'Desktop shortcut': 'Ярлык на рабочем столе',
    'Why is there no ChatGPT desktop shortcut on Windows?':
      'Почему в Windows нет ярлыка ChatGPT на рабочем столе?',
    'The icon is in Microsoft Store, so you need to search for it when opening the app.':
      'Значок находится в Microsoft Store, поэтому для запуска приложения его обычно приходится искать.',
    'Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.':
      'В Windows Codex и ChatGPT сейчас скачиваются через Microsoft Store и обычно запускаются оттуда.',
    'Press Win+R to open the Run dialog.':
      'Нажмите Win+R, чтобы открыть окно «Выполнить».',
    'Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.':
      'Введите <code>shell:AppsFolder</code>, как показано ниже, и нажмите Enter.',
    'Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.':
      'В открывшемся окне найдите ChatGPT, нажмите правой кнопкой и выберите «Создать ярлык».',
    'macOS version': 'Версия для macOS',
    'Downloading the macOS version is simpler.':
      'Скачать версию для macOS проще.',
    'After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.':
      'После скачивания откройте файл двойным щелчком и перетащите ChatGPT/Codex в Applications, чтобы завершить установку.',
    'ChatGPT sign-in': 'Вход в ChatGPT',
    '1. Sign in with an official ChatGPT account':
      '1. Вход с официальным аккаунтом ChatGPT',
    'Click Continue to sign in.': 'Нажмите «Продолжить», чтобы войти.',
    '2. Sign in with the official ChatGPT API':
      '2. Вход через официальный API ChatGPT',
    'Click Use another sign-in method below, then enter your official OpenAI API key.':
      'Нажмите внизу «Другой способ входа», затем введите официальный API-ключ OpenAI.',
    '3. Sign in with a third-party API or gateway':
      '3. Вход через сторонний API или шлюз',
    'Click Use another sign-in method and enter an arbitrary value.':
      'Нажмите «Другой способ входа» и введите произвольное значение.',
    'Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.':
      'Скачайте CC Switch — менеджер конфигурации популярных агентов — или отредактируйте config.toml вручную. См. раздел настройки ниже.',
    'Configure a third-party API in ChatGPT':
      'Настройка стороннего API в ChatGPT',
    'Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.':
      'Используйте CC Switch для настройки в один клик или вручную измените <code>auth.json</code> и <code>config.toml</code> для подключения Codex.',
    '1. One-click import with CC Switch':
      '1. Импорт в один клик через CC Switch',
    'CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.':
      'CC Switch импортирует API-ключ, выбранную модель и адрес сервиса Codex без ручного редактирования TOML.',
    '1.1 Install CC Switch': '1.1 Установка CC Switch',
    'Download and install the app from the <a0>CC Switch releases page</a0>.':
      'Скачайте и установите приложение со <a0>страницы релизов CC Switch</a0>.',
    '1.2 Create an API key': '1.2 Создание API-ключа',
    'Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.':
      'Откройте страницу API-ключей провайдера или шлюза, создайте ключ и импортируйте его одним нажатием. Большинство шлюзов поддерживает такой импорт; в остальных случаях базовый URL и ключ вводятся в CC Switch вручную. См. <a0>официальное краткое руководство CC Switch</a0>.',
    'The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.':
      'Сторонний API должен поддерживать протокол Responses. Большинство шлюзов и провайдеров его поддерживает; официальный API Claude может не поддерживать.',
    '1.3 Configure before importing': '1.3 Настройка перед импортом',
    'Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.':
      'Выберите Codex и основную модель, доступную вашему API-ключу. Сохраните автоматически сформированный адрес <code>/v1</code>.',
    'Name: the configuration name.': 'Имя: название конфигурации.',
    'Primary model: the default model for new windows. You can change it later.':
      'Основная модель: модель по умолчанию для новых окон. Её можно изменить позже.',
    '1.4 Confirm the configuration in CC Switch':
      '1.4 Подтверждение конфигурации в CC Switch',
    'Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.':
      'Подтвердите запрос браузера на открытие ссылки, затем проверьте, сохраните и включите импортированного провайдера в CC Switch.',
    '1.5 Restart the client': '1.5 Перезапуск клиента',
    'The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.':
      'Импорт включает выбранный API-ключ, основную модель и адрес сервиса с окончанием <code>/v1</code>. Подтверждайте импорт только на своём устройстве.',
    'Manual import in CC Switch': 'Ручной импорт в CC Switch',
    '2.1 Add a provider': '2.1 Добавление провайдера',
    'Click Add provider.': 'Нажмите «Добавить провайдера».',
    'Then choose Custom configuration.':
      'Затем выберите «Пользовательская конфигурация».',
    '2.2 Custom configuration': '2.2 Пользовательская конфигурация',
    'Fill in the fields as prompted. Make sure the API key and API request URL are correct.':
      'Заполните поля по подсказкам. Особенно внимательно проверьте API-ключ и URL запросов API.',
    '3. Edit auth.json and config.toml':
      '3. Редактирование auth.json и config.toml',
    'Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.':
      'Codex читает учётные данные из <code>auth.json</code> в каталоге пользователя, а настройки провайдера, модели и API — из <code>config.toml</code>.',
    '3.1 Edit auth.json': '3.1 Редактирование auth.json',
    'Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.':
      'Укажите API-ключ в пользовательском файле <code>auth.json</code>. Если в нём уже есть другие поля аутентификации, измените только <code>OPENAI_API_KEY</code>, не перезаписывая весь файл.',
    'In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.':
      'В файловом режиме <code>auth.json</code> содержит API-ключ открытым текстом. Храните его только на своём устройстве, не публикуйте в Git и не передавайте другим. Ключ хранится в <code>auth.json</code>; <code>config.toml</code> нужен только для настроек провайдера, модели и API. Не полагайтесь на временные настройки терминала.',
    '3.2 Edit config.toml': '3.2 Редактирование config.toml',
    'Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.':
      'Сохраните имеющиеся настройки Codex и добавьте блок провайдера ниже. Если <code>cli_auth_credentials_store</code> уже задан, измените его на <code>"file"</code>, не создавая дубликат. Замените пример модели на доступную вашему API-ключу.',
    'Conversation history issues': 'Проблемы с историей диалогов',
    'If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.':
      'Если старые диалоги не отображаются, проверьте, что <code>model_provider</code> в <code>config.toml</code> точно совпадает со значением при их создании, включая регистр. Верните прежнее значение, перезапустите Codex и найдите историю через <code>codex resume</code>. Обычно записи при этом не удалены.',
    '3.3 Restart and verify': '3.3 Перезапуск и проверка',
    'Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.':
      'Сохраните <code>auth.json</code> и <code>config.toml</code>, затем перезапустите работающий клиент Codex или расширение IDE.',
    'Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.':
      'Запустите <code>codex</code> в терминале и начните тестовую задачу. Учётные данные берутся из <code>auth.json</code>.',
    'If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.':
      'При ошибке запуска выполните <code>codex doctor</code>, чтобы проверить чтение файла учётных данных, а затем проверьте имя модели и Base URL.',
    'Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.':
      'Сохраните <code>wire_api = "responses"</code> и Base URL с окончанием <code>/v1</code>. Параметр <code>cli_auth_credentials_store = "file"</code> задаёт хранение и чтение учётных данных в <code>auth.json</code>; <code>requires_openai_auth = true</code> включает их использование пользовательским провайдером.',
    'References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>':
      'Ссылки: <a0>Аутентификация Codex</a0> · <a1>Справочник конфигурации Codex</a1>',
    'ChatGPT beginner installation guide':
      'Руководство по установке ChatGPT для начинающих',
    'Download, install, sign in, and configure third-party APIs with this illustrated guide.':
      'Иллюстрированное руководство: скачивание, установка, вход и настройка сторонних API.',
    'The Codex website': 'Официальный сайт Codex',
    'The ChatGPT desktop website': 'Сайт настольного ChatGPT',
    'Run the Windows installer': 'Запуск установщика Windows',
    'Wait for the installation to finish': 'Ожидание завершения установки',
    'Microsoft Store package download': 'Скачивание пакета Microsoft Store',
    'Open shell:AppsFolder': 'Открытие shell:AppsFolder',
    'Create a desktop shortcut': 'Создание ярлыка на рабочем столе',
    'Install ChatGPT on macOS': 'Установка ChatGPT на macOS',
    'ChatGPT sign-in screen': 'Экран входа в ChatGPT',
    'API key sign-in screen': 'Экран входа с API-ключом',
    'CC Switch release downloads': 'Установочные пакеты CC Switch',
    'Create and import an API key': 'Создание и импорт API-ключа',
    'Configure the Codex import': 'Настройка импорта Codex',
    'Confirm the provider in CC Switch': 'Подтверждение провайдера в CC Switch',
    'Enable the provider and restart the client':
      'Включение провайдера и перезапуск клиента',
    'Add a provider in CC Switch': 'Добавление провайдера в CC Switch',
    'Choose a custom provider': 'Выбор пользовательского провайдера',
    'Enter the API key and request URL': 'Ввод API-ключа и URL запроса',
  },
  vi: {
    'ChatGPT download': 'Tải ChatGPT',
    'Windows version': 'Phiên bản Windows',
    'Standard download': 'Cách tải thông thường',
    'Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.':
      'Tải trình cài đặt ChatGPT từ <a0>trang Codex chính thức: https://openai.com/zh-Hans-CN/codex/</a0> hoặc <a1>trang ChatGPT cho máy tính: https://chatgpt.com/zh-Hans-CN/download/</a1>.',
    'Tutorial note': 'Lưu ý',
    'Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.':
      'Codex và ChatGPT đã được hợp nhất trong một ứng dụng. Codex tập trung vào lập trình, còn ChatGPT tập trung vào công việc hằng ngày. Bạn có thể chuyển đổi chế độ.',
    'Codex website:': 'Trang Codex chính thức:',
    'ChatGPT desktop website': 'Trang chính thức của ChatGPT cho máy tính',
    'Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.':
      'Tìm trình cài đặt ChatGPT đã tải và nhấp đúp để bắt đầu tải. Phải bật chế độ proxy toàn cục.',
    'On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.':
      'Trên Windows, hiện chỉ có thể tải Codex và ChatGPT qua Microsoft Store và mặc định cài đặt vào ổ C:.',
    'Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.':
      'Bật proxy toàn cục: trình cài đặt tải ChatGPT qua Microsoft Store, còn phiên bản cửa hàng Trung Quốc kiểm tra quyền tải theo vị trí của bạn.',
    'If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.':
      'Nếu ChatGPT báo ứng dụng không khả dụng hoặc bị vô hiệu hóa, Microsoft Store có thể đã phát hiện vị trí thực của bạn. Bật proxy toàn cục có thể khắc phục vấn đề này.',
    'If the problem persists, use the second method below to install the Codex package directly.':
      'Nếu vẫn không khắc phục được, hãy dùng cách thứ hai bên dưới để cài trực tiếp gói Codex.',
    'The download has started. Wait for the progress bar to finish.':
      'Quá trình tải đã bắt đầu. Hãy đợi thanh tiến trình hoàn tất.',
    'Download a Microsoft installation package': 'Tải gói cài đặt Microsoft',
    'The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.':
      'Ứng dụng Codex cho Windows thường được cài qua Microsoft Store. Khác biệt về cấu hình máy và mạng có thể gây lỗi, thử lại liên tục hoặc cài đặt thất bại. Cách này cài Codex trực tiếp mà không qua cửa hàng.',
    'Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.':
      'Mở <a0>https://store.rg-adguard.net</a0>, chọn ProductId, nhập mã sản phẩm Microsoft <code>9PLM9XGG6VKS</code>, chọn RP ở bên phải rồi nhấp dấu tích.',
    'Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.':
      'Tải <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. Đây là tệp cài đặt độc lập; hãy mở tệp để cài ứng dụng.',
    'Desktop shortcut': 'Lối tắt trên màn hình',
    'Why is there no ChatGPT desktop shortcut on Windows?':
      'Tại sao ChatGPT trên Windows không có lối tắt trên màn hình?',
    'The icon is in Microsoft Store, so you need to search for it when opening the app.':
      'Biểu tượng nằm trong Microsoft Store nên bạn thường phải tìm kiếm để mở ứng dụng.',
    'Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.':
      'Codex và ChatGPT trên Windows hiện được tải qua Microsoft Store và thường được mở từ cửa hàng.',
    'Press Win+R to open the Run dialog.': 'Nhấn Win+R để mở hộp thoại Run.',
    'Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.':
      'Nhập <code>shell:AppsFolder</code> như hình bên dưới rồi nhấn Enter.',
    'Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.':
      'Trong cửa sổ vừa mở, tìm ChatGPT, nhấp chuột phải rồi chọn Create shortcut.',
    'macOS version': 'Phiên bản macOS',
    'Downloading the macOS version is simpler.':
      'Tải phiên bản macOS đơn giản hơn.',
    'After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.':
      'Sau khi tải xong, nhấp đúp vào tệp và kéo ChatGPT/Codex vào Applications để hoàn tất cài đặt.',
    'ChatGPT sign-in': 'Đăng nhập ChatGPT',
    '1. Sign in with an official ChatGPT account':
      '1. Đăng nhập bằng tài khoản ChatGPT chính thức',
    'Click Continue to sign in.': 'Nhấp Continue để đăng nhập.',
    '2. Sign in with the official ChatGPT API':
      '2. Đăng nhập bằng API ChatGPT chính thức',
    'Click Use another sign-in method below, then enter your official OpenAI API key.':
      'Nhấp Use another sign-in method ở bên dưới rồi nhập khóa API OpenAI chính thức.',
    '3. Sign in with a third-party API or gateway':
      '3. Đăng nhập bằng API bên thứ ba hoặc cổng chuyển tiếp',
    'Click Use another sign-in method and enter an arbitrary value.':
      'Nhấp Use another sign-in method rồi nhập một giá trị bất kỳ.',
    'Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.':
      'Tải CC Switch, công cụ quản lý cấu hình cho các agent phổ biến, hoặc tự chỉnh sửa config.toml. Xem phần cấu hình bên dưới.',
    'Configure a third-party API in ChatGPT':
      'Cấu hình API bên thứ ba trong ChatGPT',
    'Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.':
      'Dùng CC Switch để cấu hình bằng một lần nhấp hoặc chỉnh sửa thủ công <code>auth.json</code> và <code>config.toml</code> để kết nối Codex.',
    '1. One-click import with CC Switch':
      '1. Nhập bằng một lần nhấp với CC Switch',
    'CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.':
      'CC Switch nhập khóa API, mô hình đã chọn và URL dịch vụ Codex mà không cần sửa TOML thủ công.',
    '1.1 Install CC Switch': '1.1 Cài đặt CC Switch',
    'Download and install the app from the <a0>CC Switch releases page</a0>.':
      'Tải và cài ứng dụng từ <a0>trang phát hành CC Switch</a0>.',
    '1.2 Create an API key': '1.2 Tạo khóa API',
    'Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.':
      'Mở trang khóa API của nhà cung cấp mô hình hoặc cổng chuyển tiếp, tạo khóa rồi nhập bằng một lần nhấp. Đa số cổng hỗ trợ cách này; một số cần nhập thủ công Base URL và khóa vào CC Switch. Với cách nhập thủ công, xem <a0>hướng dẫn bắt đầu nhanh chính thức của CC Switch</a0>.',
    'The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.':
      'API bên thứ ba phải hỗ trợ giao thức Responses. Hầu hết cổng chuyển tiếp và nhà cung cấp mô hình đều hỗ trợ; API Claude chính thức có thể không hỗ trợ.',
    '1.3 Configure before importing': '1.3 Cấu hình trước khi nhập',
    'Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.':
      'Chọn Codex, chọn mô hình chính mà khóa API được phép dùng và giữ URL <code>/v1</code> được tạo tự động.',
    'Name: the configuration name.': 'Tên: tên cấu hình.',
    'Primary model: the default model for new windows. You can change it later.':
      'Mô hình chính: mô hình mặc định cho cửa sổ mới. Bạn có thể thay đổi sau.',
    '1.4 Confirm the configuration in CC Switch':
      '1.4 Xác nhận cấu hình trong CC Switch',
    'Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.':
      'Xác nhận yêu cầu mở liên kết của trình duyệt, sau đó kiểm tra, lưu và bật nhà cung cấp đã nhập trong CC Switch.',
    '1.5 Restart the client': '1.5 Khởi động lại ứng dụng',
    'The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.':
      'Dữ liệu nhập gồm khóa API đã chọn, mô hình chính và URL dịch vụ kết thúc bằng <code>/v1</code>. Chỉ xác nhận nhập trên thiết bị của bạn.',
    'Manual import in CC Switch': 'Nhập thủ công trong CC Switch',
    '2.1 Add a provider': '2.1 Thêm nhà cung cấp',
    'Click Add provider.': 'Nhấp Add provider.',
    'Then choose Custom configuration.': 'Sau đó chọn Custom configuration.',
    '2.2 Custom configuration': '2.2 Cấu hình tùy chỉnh',
    'Fill in the fields as prompted. Make sure the API key and API request URL are correct.':
      'Điền các trường theo hướng dẫn. Đặc biệt kiểm tra khóa API và URL yêu cầu API cho chính xác.',
    '3. Edit auth.json and config.toml':
      '3. Chỉnh sửa auth.json và config.toml',
    'Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.':
      'Codex đọc thông tin xác thực từ <code>auth.json</code> trong thư mục người dùng, còn cấu hình nhà cung cấp, mô hình và API được đọc từ <code>config.toml</code>.',
    '3.1 Edit auth.json': '3.1 Chỉnh sửa auth.json',
    'Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.':
      'Nhập khóa API vào tệp <code>auth.json</code> cấp người dùng. Nếu tệp đã có các trường xác thực khác, chỉ cập nhật <code>OPENAI_API_KEY</code>, không ghi đè toàn bộ tệp.',
    'In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.':
      'Ở chế độ tệp, <code>auth.json</code> chứa khóa API dạng văn bản thuần. Chỉ lưu trên thiết bị của bạn, không đưa vào Git hoặc chia sẻ. Lưu khóa trong <code>auth.json</code>; <code>config.toml</code> chỉ dành cho cấu hình nhà cung cấp, mô hình và API, không dựa vào thiết lập tạm thời của terminal.',
    '3.2 Edit config.toml': '3.2 Chỉnh sửa config.toml',
    'Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.':
      'Giữ cấu hình Codex hiện có và hợp nhất khối nhà cung cấp bên dưới. Nếu đã có <code>cli_auth_credentials_store</code>, đổi thành <code>"file"</code> thay vì thêm trùng. Thay mô hình mẫu bằng mô hình mà khóa API có thể sử dụng.',
    'Conversation history issues': 'Vấn đề về lịch sử hội thoại',
    'If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.':
      'Nếu hội thoại cũ không hiển thị, hãy kiểm tra <code>model_provider</code> trong <code>config.toml</code> có khớp chính xác giá trị khi tạo hội thoại, kể cả chữ hoa và chữ thường. Khôi phục giá trị đó, khởi động lại Codex và dùng <code>codex resume</code> để tìm lịch sử. Thông thường dữ liệu chưa bị xóa.',
    '3.3 Restart and verify': '3.3 Khởi động lại và kiểm tra',
    'Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.':
      'Lưu <code>auth.json</code> và <code>config.toml</code>, rồi khởi động lại ứng dụng Codex hoặc tiện ích IDE đang chạy.',
    'Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.':
      'Chạy <code>codex</code> trong terminal và bắt đầu một tác vụ thử nghiệm. Thông tin xác thực được lấy từ <code>auth.json</code>.',
    'If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.':
      'Nếu không khởi động được, chạy <code>codex doctor</code> để kiểm tra việc đọc tệp xác thực, rồi kiểm tra tên mô hình và Base URL.',
    'Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.':
      'Giữ <code>wire_api = "responses"</code> và Base URL kết thúc bằng <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> cho phép Codex lưu và đọc thông tin xác thực trong <code>auth.json</code>; <code>requires_openai_auth = true</code> cho phép nhà cung cấp tùy chỉnh sử dụng thông tin đó.',
    'References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>':
      'Tham khảo: <a0>Xác thực Codex</a0> · <a1>Tài liệu cấu hình Codex</a1>',
    'ChatGPT beginner installation guide':
      'Hướng dẫn cài đặt ChatGPT cho người mới',
    'Download, install, sign in, and configure third-party APIs with this illustrated guide.':
      'Hướng dẫn bằng hình ảnh về tải xuống, cài đặt, đăng nhập và cấu hình API bên thứ ba.',
    'The Codex website': 'Trang Codex chính thức',
    'The ChatGPT desktop website': 'Trang ChatGPT cho máy tính',
    'Run the Windows installer': 'Chạy trình cài đặt Windows',
    'Wait for the installation to finish': 'Đợi cài đặt hoàn tất',
    'Microsoft Store package download': 'Tải gói cài đặt Microsoft Store',
    'Open shell:AppsFolder': 'Mở shell:AppsFolder',
    'Create a desktop shortcut': 'Tạo lối tắt trên màn hình',
    'Install ChatGPT on macOS': 'Cài ChatGPT trên macOS',
    'ChatGPT sign-in screen': 'Màn hình đăng nhập ChatGPT',
    'API key sign-in screen': 'Màn hình đăng nhập bằng khóa API',
    'CC Switch release downloads': 'Danh sách bản cài đặt CC Switch',
    'Create and import an API key': 'Tạo và nhập khóa API',
    'Configure the Codex import': 'Cấu hình nhập Codex',
    'Confirm the provider in CC Switch':
      'Xác nhận nhà cung cấp trong CC Switch',
    'Enable the provider and restart the client':
      'Bật nhà cung cấp và khởi động lại ứng dụng',
    'Add a provider in CC Switch': 'Thêm nhà cung cấp trong CC Switch',
    'Choose a custom provider': 'Chọn nhà cung cấp tùy chỉnh',
    'Enter the API key and request URL': 'Nhập khóa API và URL yêu cầu',
  },
  'zh-TW': {
    'ChatGPT download': 'ChatGPT下載',
    'Windows version': 'Windows版',
    'Standard download': '常規方法下載',
    'Download the ChatGPT installer from the official <a0>Codex website: https://openai.com/zh-Hans-CN/codex/</a0> or the <a1>ChatGPT desktop website: https://chatgpt.com/zh-Hans-CN/download/</a1>.':
      '訪問OpenAI官網下載ChatGPT安裝器，前往<a0>Codex官網：https://openai.com/zh-Hans-CN/codex/</a0>、<a1>ChatGPT桌面客戶端官網：https://chatgpt.com/zh-Hans-CN/download/</a1>均可下載',
    'Tutorial note': '注意',
    'Codex and ChatGPT have been combined into one client. Codex focuses on coding, while ChatGPT focuses on everyday work. You can switch between modes.':
      'Codex和ChatGPT客戶端已經合二為一，Codex注重編程能力，ChatGPT桌面客戶端注重日常辦公，用戶可自行切換模式',
    'Codex website:': 'Codex官網：',
    'ChatGPT desktop website': 'ChatGPT桌面客戶端官網',
    'Find the downloaded ChatGPT installer and double-click it to start the download. Global proxy mode must be enabled.':
      '找到下載好的ChatGPT安裝器，雙擊執行下載（必須開啓全局代理）',
    'On Windows, Codex and ChatGPT can currently only be downloaded through Microsoft Store, and installation defaults to the C: drive.':
      '目前 Windows 中的 Codex 和 ChatGPT 都只能在微軟商店中下載，下載後也只能默認安裝在 C 盤中',
    'Enable global proxy mode: the installer downloads ChatGPT through Microsoft Store, and the Chinese version of the store checks download eligibility based on your location.':
      '必須開啓全局代理，因為安裝器是通過微軟應用商城下載ChatGPT，而中國版本的微軟應用商城是會根據你的使用地點檢測你是否有下載權限的',
    'If ChatGPT later reports that the client is unavailable or disabled, Microsoft Store may have detected your real location. Enabling global proxy mode may resolve this.':
      '如果後續遇到“ChatGPT客戶端不可用、禁用”等問題，很大概率是ChatGPT通過微軟應用商城檢測到了你的真實使用地址，這時候開啓全局代理就能解決。',
    'If the problem persists, use the second method below to install the Codex package directly.':
      '如果實在無法解決，請使用第二個下載安裝方法——直接安裝Codex安裝包',
    'The download has started. Wait for the progress bar to finish.':
      '此時已經開始下載，等待進度條結束就成功了',
    'Download a Microsoft installation package': '微軟安裝包下載',
    'The Windows Codex app normally installs through Microsoft Store. Differences in computer settings and networks can cause errors, repeated retries, or failed installations. This method installs Codex directly without using the store.':
      'Codex APP Windows版本默認只能通過微軟商店進行安裝，由於不同電腦的設置、網絡等差異，可能會導致大家在使用微軟商店的時候遇到各種各樣的錯誤，要麼報錯要麼一直重試然後安裝失敗，今天就教大家怎麼直接安裝Codex，不經過微軟商店。',
    'Open <a0>https://store.rg-adguard.net</a0>, select ProductId, enter the Microsoft product code <code>9PLM9XGG6VKS</code>, select RP on the right, and click the check mark.':
      '打開網站：<a0>https://store.rg-adguard.net</a0> ，下拉框選擇Productld，輸入微軟產品代碼 <code>9PLM9XGG6VKS</code>，右邊選擇RP，點擊勾',
    'Download <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0>. The downloaded file is a standalone installer; open it to install.':
      '下載 <a0>OpenAI.Codex_26.1013.3656.0_x64__2p2nqsd0c76g0.msix</a0> 這個版本()，下載下來之後就是msi直接安裝包，直接安裝即可',
    'Desktop shortcut': '桌面圖標',
    'Why is there no ChatGPT desktop shortcut on Windows?':
      '為什麼Windows版ChatGPT沒有桌面圖標？',
    'The icon is in Microsoft Store, so you need to search for it when opening the app.':
      '因為圖標在微軟應用市場裡面！平時使用需要額外搜索',
    'Codex and ChatGPT on Windows currently download through Microsoft Store and normally open from there.':
      '目前 Windows 中的 Codex 和 ChatGPT 都只能在微軟商店中下載，默認也就只能從微軟應用商城打開！',
    'Press Win+R to open the Run dialog.': '同時按住Win+R，打開資源管理器',
    'Enter <code>shell:AppsFolder</code>, as shown below, and press Enter.':
      '輸入<code>shell:AppsFolder</code>（如下圖），然後按回車',
    'Find ChatGPT in the window that opens, right-click it, and choose Create shortcut.':
      '在打開的頁面中，找到ChatGPT圖標，右鍵單單擊，點擊創建快捷方式',
    'macOS version': 'Mac版',
    'Downloading the macOS version is simpler.': 'Mac版本的下載就簡單多了',
    'After downloading, double-click the file and drag ChatGPT/Codex into Applications to finish installation.':
      '下載完成後，雙擊文件，將ChatGPT/Codex拖入Applications裡面，即可完成安裝',
    'ChatGPT sign-in': 'ChatGPT登錄',
    '1. Sign in with an official ChatGPT account': '1.ChatGPT官方賬號登錄',
    'Click Continue to sign in.': '直接點擊繼續登錄即可',
    '2. Sign in with the official ChatGPT API': '2.ChatGPT官方API登錄',
    'Click Use another sign-in method below, then enter your official OpenAI API key.':
      '點擊下方使用其他方式登錄，然後輸入OpenAI官方API密鑰即可',
    '3. Sign in with a third-party API or gateway': '3.第三方API、中轉站登錄',
    'Click Use another sign-in method and enter an arbitrary value.':
      '點擊使用其他方式登錄，然後胡亂輸入一些亂七八糟的',
    'Download CC Switch, a configuration manager for popular agents, or edit config.toml yourself. See the configuration section below.':
      '下載CC Switch（一款管理各種主流智能體配置的插件）或者自行修改config.toml，詳情看下文《ChatGPT配置》',
    'Configure a third-party API in ChatGPT': 'ChatGPT配置第三方API',
    'Use CC Switch for one-click setup, or edit <code>auth.json</code> and <code>config.toml</code> manually to connect Codex.':
      '可選擇使用 CC Switch 一鍵配置，或手動編輯 <code>auth.json</code> 和 <code>config.toml</code> 接入 Codex。',
    '1. One-click import with CC Switch': '1. 使用 CC Switch 一鍵導入',
    'CC Switch imports your API key, selected model, and Codex service URL without requiring manual TOML editing.':
      'CC Switch 可以導入 API Key、所選模型和 Codex 服務地址，無需手動編輯 TOML。',
    '1.1 Install CC Switch': '1.1 安裝cc switch',
    'Download and install the app from the <a0>CC Switch releases page</a0>.':
      '從 <a0>CC Switch 發佈頁</a0> 下載並安裝應用',
    '1.2 Create an API key': '1.2 創建密鑰',
    'Open your model provider or gateway API key page, create a key, and import it with one click. Most gateways support one-click import; others require entering the base URL and key in CC Switch manually. For manual import, see the <a0>official CC Switch quick-start guide</a0>.':
      '打開模型服務商/中轉站的API key頁面，創建密鑰，並一鍵導入。（大部分中轉站可以一鍵導入，少部分需要手動把密鑰的base url 和 key自己放入cc switch的配置，手動導入參考<a0>cc switch官方快速上手文檔</a0>）',
    'The third-party API must support the Responses protocol. Most gateways and model providers support it; the official Claude API may not.':
      '第三方API需要支持Response協議，一般中轉站/模型廠商都會支持，只有Claude官方可能不支持',
    '1.3 Configure before importing': '1.3 導入前配置',
    'Select Codex, choose a primary model available to this API key, and keep the automatically generated <code>/v1</code> URL.':
      '選擇 Codex，再選擇當前 API Key 可用的主模型，並保留自動生成的 <code>/v1</code> 地址。',
    'Name: the configuration name.': '名稱：配置名',
    'Primary model: the default model for new windows. You can change it later.':
      '主模型：新開窗口默認的模型，可以後期自己調整',
    '1.4 Confirm the configuration in CC Switch': '1.4 CC Switch側確認配置',
    'Confirm the browser prompt to open the link, then review, save, and enable the imported provider in CC Switch.':
      '確認瀏覽器打開鏈接的提示，然後在 CC Switch 中檢查、保存並啓用導入的服務商。',
    '1.5 Restart the client': '1.5 重啓客戶端',
    'The import includes the selected API key, primary model, and a service URL ending in <code>/v1</code>. Confirm the import only on your own device.':
      '導入內容包含所選 API Key、主模型和以 <code>/v1</code> 結尾的服務地址。請只在自己的設備上確認導入。',
    'Manual import in CC Switch': '手動導入CC Switch',
    '2.1 Add a provider': '2.1 添加供應商',
    'Click Add provider.': '點擊添加供應商',
    'Then choose Custom configuration.': '然後點擊自定義配置',
    '2.2 Custom configuration': '2.2 自定義配置',
    'Fill in the fields as prompted. Make sure the API key and API request URL are correct.':
      '根據提示進行填寫，重點是API Key和API請求地址不要填寫錯誤',
    '3. Edit auth.json and config.toml':
      '3. 編輯 <code>auth.json</code> 和 <code>config.toml</code>',
    'Codex reads credentials from <code>auth.json</code> in your user directory, and provider, model, and API settings from <code>config.toml</code>.':
      'Codex 從用戶目錄中的 <code>auth.json</code> 讀取認證憑據，從 <code>config.toml</code> 讀取服務商、模型和接口設置。',
    '3.1 Edit auth.json': '3.1編輯 <code>auth.json</code>',
    'Enter your API key in the user-level <code>auth.json</code>. If it already contains other authentication fields, update only <code>OPENAI_API_KEY</code>; do not overwrite the entire file.':
      '在用戶級 <code>auth.json</code> 中寫入 API Key。如果文件中已有其他認證字段，只更新 <code>OPENAI_API_KEY</code>，不要覆蓋整個文件。',
    'In file mode, <code>auth.json</code> contains a plaintext API key. Keep it only on your own device, never commit it to Git or share it. Store the key in <code>auth.json</code>; use <code>config.toml</code> only for provider, model, and API settings, not temporary terminal settings.':
      '文件模式下的 <code>auth.json</code> 包含明文密鑰，請只保存在自己的設備上，不要提交到 Git 或發送給他人。密鑰保存在 <code>auth.json</code>；<code>config.toml</code> 只用於服務商、模型和接口設置，不要依賴當前終端的臨時設置。',
    '3.2 Edit config.toml': '3.2編輯 <code>config.toml</code>',
    'Preserve your existing Codex settings and merge the provider block below. If <code>cli_auth_credentials_store</code> already exists, change it to <code>"file"</code> instead of adding a duplicate. Replace the example model with one available to your API key.':
      '保留現有 Codex 配置，並合併下面的服務商配置塊。已有 <code>cli_auth_credentials_store</code> 時將它改為 <code>"file"</code>，不要重復添加。按 API Key 的可用模型替換示例模型。',
    'Conversation history issues': '會話歷史相關問題',
    'If earlier conversations are missing, check that <code>model_provider</code> in <code>config.toml</code> exactly matches the value used to create them, including capitalization. Restore that value, restart Codex, and use <code>codex resume</code> to find the history. The records are usually not deleted.':
      '如果原有對話歷史沒有顯示，檢查當前 <code>config.toml</code> 中的 <code>model_provider</code> 是否與創建對話時使用的值完全一致（包括大小寫）。改回原值並重啓 Codex 後，可運行 <code>codex resume</code> 查找歷史；這通常不代表記錄已被刪除。',
    '3.3 Restart and verify': '3.3重啓並驗證',
    'Save <code>auth.json</code> and <code>config.toml</code>, then restart any running Codex app or IDE extension.':
      '保存 <code>auth.json</code> 和 <code>config.toml</code>，重啓正在運行的 Codex 應用或 IDE 擴展。',
    'Run <code>codex</code> in a terminal and start a test task. Credentials are provided by <code>auth.json</code>.':
      '在終端運行 <code>codex</code> 併發起一個測試任務；認證信息由 <code>auth.json</code> 提供。',
    'If startup fails, run <code>codex doctor</code> to check whether the credentials file is being read, then check the model name and Base URL.':
      '如啓動失敗，運行 <code>codex doctor</code> 檢查認證文件是否被讀取，再檢查模型名稱和 Base URL。',
    'Keep <code>wire_api = "responses"</code> and a Base URL ending in <code>/v1</code>. <code>cli_auth_credentials_store = "file"</code> makes Codex save and read credentials in <code>auth.json</code>; <code>requires_openai_auth = true</code> makes the custom provider use those credentials.':
      '請保持 <code>wire_api = "responses"</code>，並確保 Base URL 以 <code>/v1</code> 結尾。<code>cli_auth_credentials_store = "file"</code> 讓 Codex 使用 <code>auth.json</code> 保存和讀取憑據；<code>requires_openai_auth = true</code> 讓自定義服務商使用該憑據。',
    'References: <a0>Codex authentication</a0> · <a1>Codex configuration reference</a1>':
      '參考：<a0>Codex 認證說明</a0> · <a1>Codex 配置參考</a1>',
    'ChatGPT beginner installation guide': 'ChatGPT保姆級安裝教程',
    'Download, install, sign in, and configure third-party APIs with this illustrated guide.':
      '下載、安裝、登錄和第三方 API 配置完整圖文教程。',
    'The Codex website': 'Codex 官網',
    'The ChatGPT desktop website': 'ChatGPT 桌面客戶端官網',
    'Run the Windows installer': '運行 Windows 安裝器',
    'Wait for the installation to finish': '等待安裝完成',
    'Microsoft Store package download': '微軟商店安裝包下載',
    'Open shell:AppsFolder': '打開 shell:AppsFolder',
    'Create a desktop shortcut': '創建桌面快捷方式',
    'Install ChatGPT on macOS': '在 macOS 中安裝 ChatGPT',
    'ChatGPT sign-in screen': 'ChatGPT 登錄界面',
    'API key sign-in screen': 'API Key 登錄界面',
    'CC Switch release downloads': 'CC Switch 發佈頁安裝包列表',
    'Create and import an API key': '創建並導入 API Key',
    'Configure the Codex import': '配置 Codex 導入信息',
    'Confirm the provider in CC Switch': '在 CC Switch 中確認服務商',
    'Enable the provider and restart the client': '啓用服務商並重啓客戶端',
    'Add a provider in CC Switch': '在 CC Switch 中添加供應商',
    'Choose a custom provider': '選擇自定義供應商',
    'Enter the API key and request URL': '填寫 API Key 和 API 請求地址',
  },
}

for (const [locale, translations] of Object.entries(
  codexTutorialTranslations
)) {
  Object.assign(newKeys[locale], translations)
}

const retiredKeys = new Set([
  'Use any provider in Codex or Claude Code',
  'Use DeepSeek or Claude in Codex, and GPT or Gemini in Claude Code. Choose a model available on this site; the gateway handles protocol conversion.',
  'Connect Codex or Claude Code to DeepSeek, Claude, OpenAI, Gemini, and other providers through one gateway.',
  'Connect models from leading providers to the agent tools you already use.',
  // Remove malformed locale-key entries from an earlier homepage pricing pass.
  '各種價格一目了然',
  '前往模型廣場查看更多',
  'Official pricing',
  'Official pricing for {{model}}',
  'Official USD prices × site exchange rate: 1 USD = {{rate}} CNY',
  'Exchange rate unavailable; prices are shown in USD.',
  'Standard API prices; input excludes cache hits. Batch, Fast and tool fees are not included.',
  'Official pricing checked on {{date}}; not synced live.',
  'Models can change while your product interface stays stable.',
  'One gateway accepts compatible endpoints and routes each request through the current configuration.',
  'Usage and observability',
  'Every request leaves a clear record.',
  'Review usage, cost, and response timing by model, token, and time.',
  'Available placeholders: {{system_name}}, {{code}}, {{valid_minutes}}, {{reset_link}}.',
  // Error self-check guide no longer has a separate directory.
  'Common error directory',
  'Error index',
  'Common errors',

  // API keys no longer carry per-group retry overrides.
  'A request uses the selected groups in order; after a group reaches its retry count, it can continue to the next group.',
  'Default retries',
  'Inherit group policy',
  'Max Retries',
  'Number of additional attempts in this group before moving to the next one.',
  'Retries for {{group}}',
  'Select one or more groups and set the default retry count for each group.',
  'footer.columns.about.links.aboutProject',
  'footer.columns.about.links.contact',
  'footer.columns.about.links.features',
  'footer.columns.about.title',
  'footer.columns.docs.links.apiDocs',
  'footer.columns.docs.links.installation',
  'footer.columns.docs.links.quickStart',
  'footer.columns.docs.title',
  'footer.columns.related.links.midjourney',
  'footer.columns.related.links.newApiKeyTool',
  'footer.columns.related.links.oneApi',
  'footer.columns.related.title',
  'footer.defaultCopyright',
  'footer.newapi.projectAttributionSuffix',
  'Powerful API Management Platform',
  // Superseded API-key setup guidance now stores credentials in each
  // tool's persistent configuration file instead of a shell environment.
  'Claude Code environment variables',
  'Claude Code reads Anthropic-compatible settings from environment variables.',
  'Confirm provider is custom, base_url ends in /v1, and api_key still contains an environment-variable reference.',
  'Connect Claude Code through CC Switch or Anthropic-compatible environment variables.',
  'Connect Claude Code with Anthropic-compatible environment variables.',
  'Connect Gemini CLI through CC Switch or Gemini API key environment variables.',
  'Custom endpoint support can vary by tool version, so check the current provider options if these environment variables are unavailable.',
  'For authentication errors, confirm ANTHROPIC_AUTH_TOKEN is visible in the same terminal.',
  'Hermes expands ${VAR} and ${env:VAR} references when it loads config.yaml. LLM_MODEL is no longer used for custom endpoints.',
  'Hermes stores its main configuration at ~/.hermes/config.yaml. Store the API key in an environment variable, then merge the model block into the existing file.',
  'If authentication fails, confirm ALLTOKEN_API_KEY is available to the process that starts Hermes.',
  'If authentication fails, start OpenCode from a terminal that contains ALLTOKEN_API_KEY.',
  'If startup fails, confirm the environment variable, model name, and base URL before retrying.',
  'If the API key is empty, ensure the gateway service inherits ALLTOKEN_API_KEY.',
  'If OpenClaw runs as a service, add the same variable to the service environment so the gateway process can read it.',
  'On Windows, open a new terminal after setting the persistent user variable. On macOS and Linux, export applies only to the current shell unless you add it to your shell profile.',
  'Open a new terminal after saving persistent variables; keep using the current terminal when you used export.',
  'Restart the terminal after saving persistent environment variables.',
  'Restart the terminal so persistent variables are available.',
  'Set ALLTOKEN_API_KEY before starting OpenCode.',
  'Set ALLTOKEN_API_KEY in the environment used to start the OpenClaw gateway.',
  'Set the Anthropic authentication token, service root, and exact model ID in the terminal that starts Claude Code.',
  'Set the API key environment variable',
  'Set the API key, service root, and exact Gemini model ID before starting Gemini CLI.',
  'Store the API key in a dedicated environment variable instead of writing the secret into config.toml.',
  'Save config.toml, then restart the terminal and any running Codex app or IDE extension.',
  // Retired Codex locale strings from the previous config.toml-only guide.
  '2. Configure config.toml manually',
  'Choose CC Switch one-click import or edit config.toml manually to connect Codex.',
  'Codex reads user settings from ~/.codex/config.toml. On Windows, this is usually %USERPROFILE%\\.codex\\config.toml.',
  'Edit config.toml',

  // Removed package comparison dashboard.
  'Channel hit rate',
  'Package',
  'Package comparison',
  'Packages to compare',
  'Plan quota',
  'Select at least one plan to compare usage',
  'Select plans',
  'The same key is used for each package comparison',
  'Each package was tested sequentially with the same API key',
  'Selected packages run one by one with the same key',
  'Unknown plan',
  'Usage cost',
  'Usage is attributed to the plan selected when each request is billed.',

  // Superseded by the subscription-aware compliance copy.
  'Payment and redemption codes are locked until the root administrator confirms the compliance terms.',
  'This confirmation unlocks payment and redemption code features. Please read the statements carefully.',
  'Add billing group route',
  'Add channel',
  'Add error mapping',
  'All channels',
  'Attempts on this channel',
  'Billing group routes',
  'Channel monitoring',
  'Channel routing saved',
  'Channel switches',
  'Configure ordered channels for each billing group',
  'Error mappings',
  'Error rate',
  'Failure scope',
  'In flight',
  'Live channel routing health and failover metrics',
  'Maximum total attempts',
  'Move down',
  'Move up',
  'Request RPS',
  'Stable code',
  'Total timeout (ms)',
  'Upstream error code',

  'Upgrade Group',
  'Downgrade Group',
  'Downgrade to pre-purchase group',
  'No Upgrade',
  'Account and system routing options must be used on their own',
  'Add a group identifier to the auto assignment list.',
  'Add auto group',
  'Auto assignment order',
  'Auto group behavior',
  'Auto Group Chain',
  'Default to auto groups',
  'Follow the group order maintained by the administrator',
  'Group Pricing',
  'If default auto group is enabled, newly created tokens start with auto instead of an empty group.',
  'Inter-group overrides',
  'Inter-group ratio overrides',
  'JSON array of group identifiers. When enabled below, new tokens rotate through this list.',
  'Look for a special ratio rule matching this user group and this billing group. If one exists, use its ratio. Otherwise use the billing group base ratio from the pricing table.',
  'Nested JSON: source group \u2192',
  'Nested JSON defining per-group rules for adding (+:), removing (-:), or appending usable groups.',
  'Priority order for automatic group assignment. New tokens rotate through this list.',
  'Priority order for tokens in the auto group. The system tries groups from top to bottom.',
  'Special group',
  'Special ratio rules',
  'Special ratios override the token group ratio for specific user group and token group combinations.',
  'Special usable group rules',
  'Special usable group rules can add, remove, or append selectable token groups for a specific user group.',
  'Special usable group rules make extra token groups visible to, or hide default ones from, users of a specific user group.',
  'Special visibility rules',
  'System-managed routing',
  'Select at least one group',
  'Select one or more groups',
  'Select one or more groups; an API key can use models from multiple groups. When model names match, groups listed first have higher priority.',
  'System routing must be used on its own',
  'The admin configured three groups and one special ratio rule:',
  'The admin wants vip users to pay even less when they use premium. That needs an override rule: in the override matrix, set the cell at row vip, column premium to 0.3.',
  'Understand how user groups, token groups, ratios, and special rules work together.',
  'Use the group set on the token. If the token has no group, use the user group. The auto group tries the auto assignment order from top to bottom.',
  'When a token uses the auto group, the system tries groups from top to bottom until it finds an available group.',
  'When enabled, newly created tokens start in the first auto group.',
  'to override billing when a user in one group uses a token of another group.',
  '(instead of {{ratio}})',
  'Billing group = default (the token has a group, so use it)',
  'Billing group = premium (the token has a group, so use it)',
  'Billing group = vip (the token has no group, so use the user group)',
  'Call 1: the token group is premium',
  'Call 2: the token group is default',
  'Call 3: the token has no group',
  'Charge.',
  'Common pitfall: the user group base ratio is NOT a personal discount. It only applies when the user group itself is the billing group.',
  'Cost = 10 × 0.3 = 3',
  'Cost = 10 × 0.8 = 8',
  'Cost = 10 × 1.0 = 10',
  'Cost = model price × that one ratio. Nothing else from the group settings enters the formula.',
  'Every group name in the pricing table can be used in two places: on a user (the user group, assigned by admins) and on a token (the token group, chosen when creating the token). Same name pool, two different jobs.',
  'Find the billing group.',
  'Find the ratio.',
  'How a call is priced',
  'In JSON, the user group is the outer key and the billing group is the inner key. The example below means: vip users pay 0.8 when billed as standard, and 0.3 when billed as premium.',
  'In the visual editor these appear as Extra visible and Hidden. In JSON, +: (or no prefix) adds a group and -: removes one.',
  'No rule for vip billed as default → use the base ratio of default, 1.0 (the 0.8 of vip is not used)',
  'No rule for vip billed as vip → use the base ratio of vip, 0.8',
  'Only configured combinations are overridden. All other calls keep the billing group base ratio.',
  'There is a rule for vip billed as premium → use its ratio 0.3',
  'Three calls made by the same vip user. Assume the base price of one call is 10.',
  'Users of vip, when billed as premium, pay ratio',
  'Worked example',
  'decides the top-up ratio, which groups the user can pick for tokens, and whether an override ratio applies.',
  '(billed as vip itself, so base ratio of vip)',
  '(falls back to billing as vip, so base ratio of vip)',
  '(hits the override rule above)',
  '(no matrix cell, so base ratio of default; the vip 0.8 is irrelevant)',
  '(no override rule, so base ratio of default; the 0.8 of vip plays no part)',
  'All group names live here. Ratio applies when calls are billed as this group; top-up ratio applies to users whose account is in this group.',
  'All groups share one pool of names managed in the pricing group table, but a name plays two different roles: as a user group it describes the user, as a token group it decides routing and billing.',
  'Billed as default. No cell for this combination, so the base ratio of default applies — the 0.8 of vip plays no part.',
  'Billed as premium. The highlighted cell matches, so the override 0.3 applies.',
  'Billing rule: each call is billed as the token group (falling back to the user group when the token has none). The base ratio always comes from that billing group, not from the user group. To give a user group a special price on another billing group, add an entry in the override matrix.',
  'Check the override matrix at row = user group, column = billing group. If that cell is set, use it. Otherwise use the billing group base ratio from the pricing table.',
  'Each matrix cell is one rule: users of this row group pay this ratio when billed as this column group. In JSON the row is the outer key and the column is the inner key.',
  'Each rule reads as a sentence: users of one group pay a special ratio when billed as another group. Without a rule, the billing group base ratio applies.',
  'Imagine the pricing table has three groups: default (ratio 1.0), premium (ratio 0.5), and vip (ratio 0.8). Users whose account is in the vip group get user-level perks, and premium is a cheaper channel pool that users can pick for their tokens.',
  'Now a user whose user group is vip creates tokens with different groups and makes one call with each:',
  'Only configured combinations are overridden. All other calls keep the token group base ratio.',
  'Override rule: when a vip user is billed as premium, the ratio is 0.3 instead of 0.5',
  'Rows are user groups, columns are billing groups. Empty cells fall back to the base ratio shown in gray.',
  'Setup: three groups and one override matrix cell.',
  'Setup: three groups and one override rule.',
  'The token has no group, so it is billed as the user group vip, using the base ratio of vip.',
  'Three calls made by the same user, whose user group is vip:',
  'Three groups; the override matrix has exactly one cell filled in (highlighted).',
  'Group pricing usage guide',
  'JSON map of group → ratio applied when the user selects the group explicitly.',
  'Pricing group example',
  'The two roles of a group',
  'Token group',
  'Usage guide',
  'decides which channels are used and which base ratio applies.',
])

const onlyKeysArgIndex = process.argv.indexOf('--only')
const onlyKeys =
  onlyKeysArgIndex >= 0
    ? new Set(process.argv[onlyKeysArgIndex + 1]?.split('|') ?? [])
    : undefined

const localeFiles = (await fs.readdir(LOCALES_DIR, { withFileTypes: true }))
  .filter((entry) => entry.isFile() && entry.name.endsWith('.json'))
  .map((entry) => entry.name)

for (const filename of localeFiles) {
  const locale = filename.replace(/\.json$/, '')
  const file = path.join(LOCALES_DIR, `${locale}.json`)
  const json = JSON.parse(await fs.readFile(file, 'utf8'))
  let changed = false
  for (const key of retiredKeys) {
    if (onlyKeys && !onlyKeys.has(key)) continue
    if (Object.hasOwn(json.translation, key)) {
      delete json.translation[key]
      changed = true
    }
  }
  for (const [key, value] of Object.entries(newKeys[locale] ?? {})) {
    if (onlyKeys && !onlyKeys.has(key)) continue
    if (json.translation[key] !== value) {
      json.translation[key] = value
      changed = true
    }
  }
  const sortedTranslations = Object.fromEntries(
    Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
  )
  if (JSON.stringify(json.translation) !== JSON.stringify(sortedTranslations)) {
    json.translation = sortedTranslations
    changed = true
  }
  if (changed) {
    await fs.writeFile(file, `${JSON.stringify(json, null, 2)}\n`, 'utf8')
  }
}
