import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

const newKeys = {
  en: {
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
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?',
  },
  zh: {
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
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      '这会根据所有渠道配置重建渠道路由索引，包括支持的模型、分组和权重。重建期间路由可能短暂不完整。是否继续？',
  },
  fr: {
    'Authentication email templates': 'Modèles d’e-mails d’authentification',
    'Available placeholders': 'Variables disponibles',
    'Verification email subject': 'Objet de l’e-mail de vérification',
    'Verification email body': 'Corps de l’e-mail de vérification',
    'Password reset email subject': 'Objet de l’e-mail de réinitialisation du mot de passe',
    'Password reset email body': 'Corps de l’e-mail de réinitialisation du mot de passe',
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
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Cette action reconstruit l’index de routage des canaux à partir de toutes les configurations, notamment les modèles pris en charge, les groupes et les poids. Le routage peut être brièvement incomplet pendant la reconstruction. Continuer ?',
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
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'すべてのチャネル設定からルーティングインデックスを再構築します。対応モデル、グループ、重みが含まれます。再構築中はルーティングが一時的に不完全になる可能性があります。続行しますか？',
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
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Индекс маршрутизации каналов будет перестроен на основе всех конфигураций каналов, включая поддерживаемые модели, группы и веса. Во время перестроения маршрутизация может быть временно неполной. Продолжить?',
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
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Thao tác này sẽ xây dựng lại chỉ mục định tuyến kênh từ mọi cấu hình kênh, bao gồm mô hình được hỗ trợ, nhóm và trọng số. Định tuyến có thể tạm thời chưa đầy đủ trong quá trình xây dựng lại. Tiếp tục?',
  },
  'zh-TW': {
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
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      '這會根據所有渠道設定重建渠道路由索引，包括支援模型、分組和權重。重建期間路由可能短暫不完整。是否繼續？',
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
    'Search error keywords...': 'Search error keywords...',
    'Search error keywords': 'Search error keywords',
    'No matching errors found.': 'No matching errors found.',
    'Model capacity insufficient': 'Model capacity insufficient',
    'Authentication failed': 'Authentication failed',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI and other official providers temporarily lack enough compute capacity.',
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
    'Search error keywords...': '搜索错误关键词...',
    'Search error keywords': '搜索错误关键词',
    'No matching errors found.': '未找到匹配的错误。',
    'Model capacity insufficient': '模型容量不足',
    'Authentication failed': '认证失败',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI 及其他官方厂商暂时缺乏足够的算力。',
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
    'Search error keywords...': '搜尋錯誤關鍵字...',
    'Search error keywords': '搜尋錯誤關鍵字',
    'No matching errors found.': '找不到符合的錯誤。',
    'Model capacity insufficient': '模型容量不足',
    'Authentication failed': '驗證失敗',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI 及其他官方廠商暫時缺乏足夠的算力。',
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
    'Search error keywords...': 'Rechercher des mots-clés d’erreur...',
    'Search error keywords': 'Rechercher des mots-clés d’erreur',
    'No matching errors found.': 'Aucune erreur correspondante trouvée.',
    'Model capacity insufficient': 'Capacité du modèle insuffisante',
    'Authentication failed': 'Échec de l’authentification',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI et les autres fournisseurs officiels ne disposent temporairement pas d’une capacité de calcul suffisante.',
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
    'Search error keywords...': 'エラーキーワードを検索...',
    'Search error keywords': 'エラーキーワードを検索',
    'No matching errors found.': '一致するエラーが見つかりません。',
    'Model capacity insufficient': 'モデルの容量不足',
    'Authentication failed': '認証に失敗しました',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI などの公式プロバイダーで、選択したモデルに割り当てる計算リソースが一時的に不足しています。',
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
    'Search error keywords...': 'Поиск по ключевым словам ошибок...',
    'Search error keywords': 'Поиск по ключевым словам ошибок',
    'No matching errors found.': 'Подходящие ошибки не найдены.',
    'Model capacity insufficient': 'Недостаточная ёмкость модели',
    'Authentication failed': 'Ошибка аутентификации',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'У OpenAI и других официальных поставщиков временно недостаточно вычислительных ресурсов.',
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
    'Search error keywords...': 'Tìm kiếm từ khóa lỗi...',
    'Search error keywords': 'Tìm kiếm từ khóa lỗi',
    'No matching errors found.': 'Không tìm thấy lỗi phù hợp.',
    'Model capacity insufficient': 'Mô hình không đủ năng lực',
    'Authentication failed': 'Xác thực không thành công',
    'OpenAI and other official providers temporarily lack enough compute capacity.':
      'OpenAI và các nhà cung cấp chính thức khác tạm thời không có đủ năng lực tính toán.',
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

const retiredKeys = new Set([
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
  'Open Grafana',
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

const localeFiles = (await fs.readdir(LOCALES_DIR, { withFileTypes: true }))
  .filter((entry) => entry.isFile() && entry.name.endsWith('.json'))
  .map((entry) => entry.name)

for (const filename of localeFiles) {
  const locale = filename.replace(/\.json$/, '')
  const file = path.join(LOCALES_DIR, `${locale}.json`)
  const json = JSON.parse(await fs.readFile(file, 'utf8'))
  let changed = false
  for (const key of retiredKeys) {
    if (Object.hasOwn(json.translation, key)) {
      delete json.translation[key]
      changed = true
    }
  }
  for (const [key, value] of Object.entries(newKeys[locale] ?? {})) {
    if (json.translation[key] !== value) {
      json.translation[key] = value
      changed = true
    }
  }
  if (changed) {
    await fs.writeFile(file, `${JSON.stringify(json, null, 2)}\n`, 'utf8')
  }
}
