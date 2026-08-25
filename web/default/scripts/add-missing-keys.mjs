import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

const newKeys = {
  en: {
    'Please select at least one model billing group':
      'Please select at least one model billing group',
    'Add billing group route': 'Add billing group route',
    'Add channel': 'Add channel',
    'Add error mapping': 'Add error mapping',
    'All channels': 'All channels',
    'Attempts on this channel': 'Attempts on this channel',
    'Billing group routes': 'Billing group routes',
    'Channel monitoring': 'Channel monitoring',
    'Live channel routing health and failover metrics':
      'Live channel routing health and failover metrics',
    'Channel routing saved': 'Channel routing saved',
    'Channel switches': 'Channel switches',
    'Circuit cooldown (seconds)': 'Circuit cooldown (seconds)',
    'Circuit failure threshold': 'Circuit failure threshold',
    'Configure ordered channels for each billing group':
      'Configure ordered channels for each billing group',
    'Cost factor': 'Cost factor',
    'In flight': 'In flight',
    'Maximum total attempts': 'Maximum total attempts',
    Order: 'Order',
    'Request RPS': 'Request RPS',
    'Total timeout (ms)': 'Total timeout (ms)',
    'Upstream error code': 'Upstream error code',
    'Stable code': 'Stable code',
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
    'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.':
      'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.',
    'Gross margin': 'Gross margin',
    'Group Management': 'Group Management',
    'Inbound endpoints': 'Inbound endpoints',
    Loading: 'Loading',
    'Prompt Cache': 'Prompt Cache',
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
    'Price multiplier mode': 'Price multiplier mode',
    'USD-equivalent': 'USD-equivalent',
    CNY: 'CNY',
    'Probe failure auto-ban': 'Probe failure auto-ban',
    'Probe success auto-enable': 'Probe success auto-enable',
    'Force priority': 'Force priority',
    'Force priority scope': 'Force priority scope',
    'Current group only': 'Current group only',
    'Across selected groups': 'Across selected groups',
    'Previous-day probe success rate': 'Previous-day probe success rate',
    'Automatic probe': 'Automatic probe',
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
    'Open circuits': 'Open circuits',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?',
  },
  zh: {
    'Please select at least one model billing group':
      '请选择至少一个模型计费分组',
    'Add billing group route': '添加计费分组路由',
    'Add channel': '添加渠道',
    'Add error mapping': '添加错误映射',
    'All channels': '全部渠道',
    'Attempts on this channel': '本渠道尝试次数',
    'Billing group routes': '计费分组路由',
    'Channel monitoring': '渠道监控',
    'Live channel routing health and failover metrics':
      '实时渠道路由健康度和切流指标',
    'Channel routing saved': '渠道路由已保存',
    'Channel switches': '渠道切换次数',
    'Circuit cooldown (seconds)': '熔断冷却时间（秒）',
    'Circuit failure threshold': '熔断失败阈值',
    'Configure ordered channels for each billing group':
      '为每个计费分组配置有序渠道',
    'Cost factor': '成本系数',
    'In flight': '进行中请求',
    'Maximum total attempts': '最大总尝试次数',
    Order: '顺序',
    'Request RPS': '请求 RPS',
    'Total timeout (ms)': '总超时（毫秒）',
    'Upstream error code': '上游错误码',
    'Stable code': '稳定错误码',
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
    'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.':
      '估算成本 = 实际渠道 Token × 模型官方价格 × 计费分组倍率 × 渠道成本系数。',
    'Gross margin': '毛利差额',
    'Group Management': '分组管理',
    'Inbound endpoints': '入站端点',
    Loading: '加载中',
    'Prompt Cache': '提示词缓存',
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
    'Price multiplier mode': '价格倍率模式',
    'USD-equivalent': '美元等值',
    CNY: '人民币',
    'Probe failure auto-ban': '探测失败自动禁用',
    'Probe success auto-enable': '探测成功自动启用',
    'Force priority': '强制优先',
    'Force priority scope': '强制优先范围',
    'Current group only': '仅当前分组',
    'Across selected groups': '跨所选分组',
    'Previous-day probe success rate': '昨日成功率',
    'Automatic probe': '自动探测',
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
    'Open circuits': '已熔断渠道',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      '这会根据所有渠道配置重建渠道路由索引，包括支持的模型、分组和权重。重建期间路由可能短暂不完整。是否继续？',
  },
  fr: {
    'Please select at least one model billing group':
      'Veuillez sélectionner au moins un groupe de facturation de modèles',
    'Add billing group route': 'Ajouter une route de facturation',
    'Add channel': 'Ajouter un canal',
    'Add error mapping': 'Ajouter une correspondance',
    'All channels': 'Tous les canaux',
    'Attempts on this channel': 'Tentatives sur ce canal',
    'Billing group routes': 'Routes des groupes de facturation',
    'Channel monitoring': 'Surveillance des canaux',
    'Live channel routing health and failover metrics':
      'État en direct du routage des canaux et indicateurs de bascule',
    'Channel routing saved': 'Routage des canaux enregistré',
    'Channel switches': 'Changements de canal',
    'Circuit cooldown (seconds)': 'Délai du circuit (secondes)',
    'Circuit failure threshold': 'Seuil d’échec du circuit',
    'Configure ordered channels for each billing group':
      'Configurer les canaux ordonnés de chaque groupe de facturation',
    'Cost factor': 'Facteur de coût',
    'In flight': 'En cours',
    'Maximum total attempts': 'Nombre maximal de tentatives',
    Order: 'Ordre',
    'Request RPS': 'RPS des requêtes',
    'Total timeout (ms)': 'Délai total (ms)',
    'Upstream error code': 'Code d’erreur amont',
    'Stable code': 'Code stable',
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
    'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.':
      'Coût estimé = jetons réels du canal × prix officiel du modèle × coefficient du groupe de facturation × facteur de coût du canal.',
    'Gross margin': 'Marge brute',
    'Group Management': 'Gestion des groupes',
    'Inbound endpoints': 'Points d’entrée',
    Loading: 'Chargement',
    'Prompt Cache': 'Cache de prompt',
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
    'Price multiplier mode': 'Mode du multiplicateur de prix',
    'USD-equivalent': 'Équivalent USD',
    CNY: 'CNY',
    'Probe failure auto-ban': 'Désactivation automatique après échec de sonde',
    'Probe success auto-enable':
      'Activation automatique après réussite de sonde',
    'Force priority': 'Priorité forcée',
    'Force priority scope': 'Portée de la priorité forcée',
    'Current group only': 'Groupe actuel uniquement',
    'Across selected groups': 'Tous les groupes sélectionnés',
    'Previous-day probe success rate':
      'Taux de réussite des sondes du jour précédent',
    'Automatic probe': 'Sonde automatique',
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
    'Open circuits': 'Circuits ouverts',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Cette action reconstruit l’index de routage des canaux à partir de toutes les configurations, notamment les modèles pris en charge, les groupes et les poids. Le routage peut être brièvement incomplet pendant la reconstruction. Continuer ?',
  },
  ja: {
    'Please select at least one model billing group':
      'モデル課金グループを1つ以上選択してください',
    'Add billing group route': '課金グループルートを追加',
    'Add channel': 'チャネルを追加',
    'Add error mapping': 'エラーマッピングを追加',
    'All channels': 'すべてのチャネル',
    'Attempts on this channel': 'このチャネルでの試行回数',
    'Billing group routes': '課金グループルート',
    'Channel monitoring': 'チャネル監視',
    'Live channel routing health and failover metrics':
      'チャネルルーティングの稼働状況とフェイルオーバー指標',
    'Channel routing saved': 'チャネルルーティングを保存しました',
    'Channel switches': 'チャネル切替回数',
    'Circuit cooldown (seconds)': 'サーキット待機時間（秒）',
    'Circuit failure threshold': 'サーキット失敗しきい値',
    'Configure ordered channels for each billing group':
      '課金グループごとに順序付きチャネルを設定します',
    'Cost factor': 'コスト係数',
    'In flight': '処理中',
    'Maximum total attempts': '最大総試行回数',
    Order: '順序',
    'Request RPS': 'リクエスト RPS',
    'Total timeout (ms)': '合計タイムアウト（ms）',
    'Upstream error code': '上流エラーコード',
    'Stable code': '安定コード',
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
    'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.':
      '推定コスト = 実際のチャネルトークン数 × モデル公式価格 × 課金グループ倍率 × チャネルコスト係数。',
    'Gross margin': '粗利益差額',
    'Group Management': 'グループ管理',
    'Inbound endpoints': '受信エンドポイント',
    Loading: '読み込み中',
    'Prompt Cache': 'プロンプトキャッシュ',
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
    'Price multiplier mode': '価格倍率モード',
    'USD-equivalent': 'USD 相当',
    CNY: 'CNY',
    'Probe failure auto-ban': 'プローブ失敗時に自動無効化',
    'Probe success auto-enable': 'プローブ成功時に自動有効化',
    'Force priority': '優先を強制',
    'Force priority scope': '強制優先の範囲',
    'Current group only': '現在のグループのみ',
    'Across selected groups': '選択したグループ全体',
    'Previous-day probe success rate': '前日のプローブ成功率',
    'Automatic probe': '自動プローブ',
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
    'Open circuits': 'オープン中のサーキット',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'すべてのチャネル設定からルーティングインデックスを再構築します。対応モデル、グループ、重みが含まれます。再構築中はルーティングが一時的に不完全になる可能性があります。続行しますか？',
  },
  ru: {
    'Please select at least one model billing group':
      'Пожалуйста, выберите хотя бы одну группу тарификации моделей',
    'Add billing group route': 'Добавить маршрут группы тарификации',
    'Add channel': 'Добавить канал',
    'Add error mapping': 'Добавить сопоставление ошибки',
    'All channels': 'Все каналы',
    'Attempts on this channel': 'Попытки на этом канале',
    'Billing group routes': 'Маршруты групп тарификации',
    'Channel monitoring': 'Мониторинг каналов',
    'Live channel routing health and failover metrics':
      'Текущее состояние маршрутизации каналов и показатели переключения',
    'Channel routing saved': 'Маршрутизация каналов сохранена',
    'Channel switches': 'Переключения каналов',
    'Circuit cooldown (seconds)': 'Пауза автомата (секунды)',
    'Circuit failure threshold': 'Порог ошибок автомата',
    'Configure ordered channels for each billing group':
      'Настройте порядок каналов для каждой группы тарификации',
    'Cost factor': 'Коэффициент стоимости',
    'In flight': 'В обработке',
    'Maximum total attempts': 'Максимум попыток',
    Order: 'Порядок',
    'Request RPS': 'RPS запросов',
    'Total timeout (ms)': 'Общий тайм-аут (мс)',
    'Upstream error code': 'Код ошибки провайдера',
    'Stable code': 'Стабильный код',
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
    'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.':
      'Расчётная стоимость = фактические токены канала × официальная цена модели × коэффициент группы тарификации × коэффициент стоимости канала.',
    'Gross margin': 'Валовая маржа',
    'Group Management': 'Управление группами',
    'Inbound endpoints': 'Входные точки',
    Loading: 'Загрузка',
    'Prompt Cache': 'Кэш промпта',
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
    'Price multiplier mode': 'Режим ценового множителя',
    'USD-equivalent': 'Эквивалент USD',
    CNY: 'CNY',
    'Probe failure auto-ban': 'Автоотключение после сбоя проверки',
    'Probe success auto-enable': 'Автовключение после успешной проверки',
    'Force priority': 'Принудительный приоритет',
    'Force priority scope': 'Область принудительного приоритета',
    'Current group only': 'Только текущая группа',
    'Across selected groups': 'Во всех выбранных группах',
    'Previous-day probe success rate': 'Успешность проверок за предыдущий день',
    'Automatic probe': 'Автоматическая проверка',
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
    'Open circuits': 'Разомкнутые контуры',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Индекс маршрутизации каналов будет перестроен на основе всех конфигураций каналов, включая поддерживаемые модели, группы и веса. Во время перестроения маршрутизация может быть временно неполной. Продолжить?',
  },
  vi: {
    'Please select at least one model billing group':
      'Vui lòng chọn ít nhất một nhóm tính phí mô hình',
    'Add billing group route': 'Thêm tuyến nhóm tính phí',
    'Add channel': 'Thêm kênh',
    'Add error mapping': 'Thêm ánh xạ lỗi',
    'All channels': 'Tất cả kênh',
    'Attempts on this channel': 'Số lần thử trên kênh này',
    'Billing group routes': 'Tuyến nhóm tính phí',
    'Channel monitoring': 'Giám sát kênh',
    'Live channel routing health and failover metrics':
      'Tình trạng định tuyến kênh trực tiếp và chỉ số chuyển tuyến',
    'Channel routing saved': 'Đã lưu định tuyến kênh',
    'Channel switches': 'Số lần chuyển kênh',
    'Circuit cooldown (seconds)': 'Thời gian chờ ngắt mạch (giây)',
    'Circuit failure threshold': 'Ngưỡng lỗi ngắt mạch',
    'Configure ordered channels for each billing group':
      'Cấu hình thứ tự kênh cho từng nhóm tính phí',
    'Cost factor': 'Hệ số chi phí',
    'In flight': 'Đang xử lý',
    'Maximum total attempts': 'Tổng số lần thử tối đa',
    Order: 'Thứ tự',
    'Request RPS': 'RPS yêu cầu',
    'Total timeout (ms)': 'Tổng thời gian chờ (ms)',
    'Upstream error code': 'Mã lỗi thượng nguồn',
    'Stable code': 'Mã ổn định',
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
    'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.':
      'Chi phí ước tính = token thực tế của kênh × giá chính thức của mô hình × hệ số nhóm tính phí × hệ số chi phí kênh.',
    'Gross margin': 'Chênh lệch lợi nhuận gộp',
    'Group Management': 'Quản lý nhóm',
    'Inbound endpoints': 'Điểm cuối đầu vào',
    Loading: 'Đang tải',
    'Prompt Cache': 'Bộ nhớ đệm prompt',
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
    'Price multiplier mode': 'Chế độ hệ số giá',
    'USD-equivalent': 'Tương đương USD',
    CNY: 'CNY',
    'Probe failure auto-ban': 'Tự động tắt khi kiểm tra thất bại',
    'Probe success auto-enable': 'Tự động bật khi kiểm tra thành công',
    'Force priority': 'Ưu tiên bắt buộc',
    'Force priority scope': 'Phạm vi ưu tiên bắt buộc',
    'Current group only': 'Chỉ nhóm hiện tại',
    'Across selected groups': 'Trên các nhóm đã chọn',
    'Previous-day probe success rate': 'Tỷ lệ kiểm tra thành công ngày trước',
    'Automatic probe': 'Kiểm tra tự động',
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
    'Open circuits': 'Mạch đang mở',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      'Thao tác này sẽ xây dựng lại chỉ mục định tuyến kênh từ mọi cấu hình kênh, bao gồm mô hình được hỗ trợ, nhóm và trọng số. Định tuyến có thể tạm thời chưa đầy đủ trong quá trình xây dựng lại. Tiếp tục?',
  },
  'zh-TW': {
    'Please select at least one model billing group':
      '請至少選擇一個模型計費分組',
    'Add billing group route': '新增計費分組路由',
    'Add channel': '新增渠道',
    'Add error mapping': '新增錯誤映射',
    'All channels': '全部渠道',
    'Attempts on this channel': '本渠道嘗試次數',
    'Billing group routes': '計費分組路由',
    'Channel monitoring': '渠道監控',
    'Live channel routing health and failover metrics':
      '即時渠道路由健康狀態與切換指標',
    'Channel routing saved': '渠道路由已儲存',
    'Channel switches': '渠道切換次數',
    'Circuit cooldown (seconds)': '熔斷冷卻時間（秒）',
    'Circuit failure threshold': '熔斷失敗閾值',
    'Configure ordered channels for each billing group':
      '為每個計費分組設定有序渠道',
    'Cost factor': '成本係數',
    'In flight': '進行中請求',
    'Maximum total attempts': '最大總嘗試次數',
    Order: '順序',
    'Request RPS': '請求 RPS',
    'Total timeout (ms)': '總逾時（毫秒）',
    'Upstream error code': '上游錯誤碼',
    'Stable code': '穩定錯誤碼',
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
    'Estimated cost = actual channel tokens × official model price × billing group ratio × channel cost factor.':
      '估算成本 = 實際渠道 Token × 模型官方價格 × 計費分組倍率 × 渠道成本係數。',
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
    'Automatic probe': '自動探測',
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
    'Open circuits': '已熔斷渠道',
    'This will rebuild the channel routing index from every channel configuration, including supported models, groups, and weights. Routing may be briefly incomplete while the rebuild is running. Continue?':
      '這會根據所有渠道設定重建渠道路由索引，包括支援模型、分組和權重。重建期間路由可能短暫不完整。是否繼續？',
  },
}

const retiredKeys = new Set([
  'Account and system routing options must be used on their own',
  'Add a group identifier to the auto assignment list.',
  'Add auto group',
  'Auto (Circuit Breaker)',
  'Auto assignment order',
  'Auto group behavior',
  'Auto Group Chain',
  'Automatically selects the best available group with circuit breaker mechanism',
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
