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
*/
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

const newKeys = {
  en: {
    'Channels are attempted in order; each channel can simulate failures and latency.':
      'Channels are attempted in order; each channel can simulate failures and latency.',
    'Mock channel latency is invalid': 'Mock channel latency is invalid',
    'No capacity limit': 'No capacity limit',
    'Local load-test agent': 'Local load-test agent',
    'Run high-volume tests outside the browser and keep results linked to this account.':
      'Run high-volume tests outside the browser and keep results linked to this account.',
    'Pair agent': 'Pair agent',
    'Load generator': 'Load generator',
    'No online agents': 'No online agents',
    'Start with agent': 'Start with agent',
    Offline: 'Offline',
    'Remove agent': 'Remove agent',
    'Agent load test queued': 'Agent load test queued',
    'Pair local agent': 'Pair local agent',
    'Run this command on the computer that will generate load.':
      'Run this command on the computer that will generate load.',
    'Pairing code expires in 5 minutes.': 'Pairing code expires in 5 minutes.',
    'Copy command': 'Copy command',
    'Start in browser': 'Start in browser',
    queued: 'Queued',
    dispatched: 'Dispatched',
    cancel_requested: 'Stopping',
    completed: 'Completed',
    cancelled: 'Cancelled',
    'Clear history': 'Clear history',
    'Completed at': 'Completed at',
    'Each completed test is saved with its own Run ID.':
      'Each completed test is saved with its own Run ID.',
    'Duration must be between {{min}} and {{max}} seconds.':
      'Duration must be between {{min}} and {{max}} seconds.',
    'Load test history': 'Load test history',
    'Load test history cleared': 'Load test history cleared',
    'No previous load tests': 'No previous load tests',
    'Billing group ratio': 'Billing group ratio',
    'Channel cost factor': 'Channel cost factor',
    'Current run metrics': 'Current run metrics',
    'Actual channel cost = official model price × channel cost factor; user charge = official model price × billing group ratio.':
      'Actual channel cost = official model price × channel cost factor; user charge = official model price × billing group ratio.',
    'Requests per second must be between {{min}} and {{max}}.':
      'Requests per second must be between {{min}} and {{max}}.',
    'Failed to load load-test limits': 'Failed to load load-test limits',
    'Maximum concurrency must be between {{min}} and {{max}}.':
      'Maximum concurrency must be between {{min}} and {{max}}.',
    'Allowed range: {{min}}-{{max}} concurrent requests':
      'Allowed range: {{min}}-{{max}} concurrent requests',
    'Load Test Limits': 'Load Test Limits',
    'These limits apply to every load-test demo run.':
      'These limits apply to every load-test demo run.',
    'Planned requests for this run': 'Planned requests for this run',
    'Maximum duration (seconds)': 'Maximum duration (seconds)',
    'Maximum requests per second': 'Maximum requests per second',
    'Allowed range: 5-3600 seconds': 'Allowed range: 5-3600 seconds',
    'Allowed range: 1-10000 RPS': 'Allowed range: 1-10000 RPS',
    'Allowed range: 1-10000 concurrent requests':
      'Allowed range: 1-10000 concurrent requests',
    'User Type': 'User Type',
    ToB: 'ToB',
    ToC: 'ToC',
    'ToB users can access the load test demo.':
      'ToB users can access the load test demo.',
    'Add billing group route': 'Add billing group route',
    'Add channel': 'Add channel',
    'Add error mapping': 'Add error mapping',
    'All channels': 'All channels',
    'Attempts on this channel': 'Attempts on this channel',
    Balanced: 'Balanced',
    'Billing group routes': 'Billing group routes',
    'Channel monitoring': 'Channel monitoring',
    'Live channel routing health and failover metrics':
      'Live channel routing health and failover metrics',
    'Channel routing saved': 'Channel routing saved',
    'Channel switches': 'Channel switches',
    'Circuit cooldown (seconds)': 'Circuit cooldown (seconds)',
    'Circuit failure threshold': 'Circuit failure threshold',
    'Circuit window (seconds)': 'Circuit window (seconds)',
    'Circuit protection': 'Circuit protection',
    'Tune when a channel is temporarily removed after repeated failures.':
      'Tune when a channel is temporarily removed after repeated failures.',
    'Quick presets': 'Quick presets',
    Sensitive: 'Sensitive',
    Standard: 'Standard',
    Relaxed: 'Relaxed',
    'Configure ordered channels for each billing group':
      'Configure ordered channels for each billing group',
    'Cost factor': 'Cost factor',
    'Cost first': 'Cost first',
    'In flight': 'In flight',
    'Maximum total attempts': 'Maximum total attempts',
    Order: 'Order',
    'Request RPS': 'Request RPS',
    'Routing strategy': 'Routing strategy',
    'Stability first': 'Stability first',
    'Total timeout (ms)': 'Total timeout (ms)',
    'Upstream error code': 'Upstream error code',
    '24 hours': '24 hours',
    '7 days': '7 days',
    '30 days': '30 days',
    'Actual cost': 'Actual cost',
    'Actual cost (USD)': 'Actual cost (USD)',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.',
    'Average latency': 'Average latency',
    'Average token price': 'Average token price',
    'User charge = official model price × billing group ratio.':
      'User charge = official model price × billing group ratio.',
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
  },
  zh: {
    'Channels are attempted in order; each channel can simulate failures and latency.':
      '渠道将按顺序尝试；每个渠道可以模拟失败和延迟。',
    'Mock channel latency is invalid': 'Mock 渠道延迟配置无效',
    'No capacity limit': '不限制容量',
    'Local load-test agent': '本地压测 Agent',
    'Run high-volume tests outside the browser and keep results linked to this account.':
      '在浏览器外执行大规模压测，并将结果保存到当前账户。',
    'Pair agent': '配对 Agent',
    'Load generator': '压测机',
    'No online agents': '没有在线 Agent',
    'Start with agent': '使用 Agent 开始',
    Offline: '离线',
    'Remove agent': '移除 Agent',
    'Agent load test queued': 'Agent 压测任务已排队',
    'Pair local agent': '配对本地 Agent',
    'Run this command on the computer that will generate load.':
      '请在用于发起压测的电脑上运行此命令。',
    'Pairing code expires in 5 minutes.': '配对码将在 5 分钟后过期。',
    'Copy command': '复制命令',
    'Start in browser': '在浏览器中开始',
    queued: '排队中',
    dispatched: '已下发',
    cancel_requested: '正在停止',
    completed: '已完成',
    cancelled: '已取消',
    'Clear history': '清理历史记录',
    'Completed at': '完成时间',
    'Each completed test is saved with its own Run ID.':
      '每次完成的压测都会按独立 Run ID 保存。',
    'Duration must be between {{min}} and {{max}} seconds.':
      '压测时长必须在 {{min}} 到 {{max}} 秒之间。',
    'Load test history': '压测历史',
    'Load test history cleared': '压测历史已清理',
    'No previous load tests': '暂无历史压测记录',
    'Billing group ratio': '分组倍率',
    'Channel cost factor': '渠道成本系数',
    'Current run metrics': '当前运行指标',
    'Actual channel cost = official model price × channel cost factor; user charge = official model price × billing group ratio.':
      '实际渠道成本 = 模型官方价格 × 渠道成本系数；用户收费 = 模型官方价格 × 分组倍率。',
    'Requests per second must be between {{min}} and {{max}}.':
      '每秒请求数必须在 {{min}} 到 {{max}} 之间。',
    'Failed to load load-test limits': '加载压测限制失败',
    'Maximum concurrency must be between {{min}} and {{max}}.':
      '最大并发数必须在 {{min}} 到 {{max}} 之间。',
    'Allowed range: {{min}}-{{max}} concurrent requests':
      '允许范围：{{min}}-{{max}} 个并发请求',
    'Load Test Limits': '压测限制',
    'These limits apply to every load-test demo run.':
      '这些限制适用于所有压测 Demo。',
    'Planned requests for this run': '本次运行计划请求数',
    'Maximum duration (seconds)': '最大时长（秒）',
    'Maximum requests per second': '最大每秒请求数',
    'Allowed range: 5-3600 seconds': '允许范围：5-3600 秒',
    'Allowed range: 1-10000 RPS': '允许范围：1-10000 RPS',
    'Allowed range: 1-10000 concurrent requests':
      '允许范围：1-10000 个并发请求',
    'User Type': '用户类型',
    ToB: 'ToB',
    ToC: 'ToC',
    'ToB users can access the load test demo.': 'ToB 用户可以使用压测 Demo。',
    'Add billing group route': '添加计费分组路由',
    'Add channel': '添加渠道',
    'Add error mapping': '添加错误映射',
    'All channels': '全部渠道',
    'Attempts on this channel': '本渠道尝试次数',
    Balanced: '均衡',
    'Billing group routes': '计费分组路由',
    'Channel monitoring': '渠道监控',
    'Live channel routing health and failover metrics':
      '实时渠道路由健康度和切流指标',
    'Channel routing saved': '渠道路由已保存',
    'Channel switches': '渠道切换次数',
    'Circuit cooldown (seconds)': '熔断冷却时间（秒）',
    'Circuit failure threshold': '熔断失败阈值',
    'Circuit window (seconds)': '熔断统计窗口（秒）',
    'Circuit protection': '熔断保护',
    'Tune when a channel is temporarily removed after repeated failures.':
      '调整渠道在连续失败后暂时下线的触发条件。',
    'Quick presets': '快速预设',
    Sensitive: '敏感',
    Standard: '标准',
    Relaxed: '宽松',
    'Configure ordered channels for each billing group':
      '为每个计费分组配置有序渠道',
    'Cost factor': '成本系数',
    'Cost first': '成本优先',
    'In flight': '进行中请求',
    'Maximum total attempts': '最大总尝试次数',
    Order: '顺序',
    'Request RPS': '请求 RPS',
    'Routing strategy': '路由策略',
    'Stability first': '稳定优先',
    'Total timeout (ms)': '总超时（毫秒）',
    'Upstream error code': '上游错误码',
    '24 hours': '24 小时',
    '7 days': '7 天',
    '30 days': '30 天',
    'Actual cost': '实际成本',
    'Actual cost (USD)': '实际成本（USD）',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      '实际成本由管理员手工录入，并按所选时间段分摊。估算成本由网关根据计费快照计算。',
    'Average latency': '平均延迟',
    'Average token price': '平均每个 token 单价',
    'User charge = official model price × billing group ratio.':
      '用户扣费 = 模型官方价格 × 计费分组倍率。',
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
  },
  fr: {
    'Local load-test agent': 'Agent de test de charge local',
    'Run high-volume tests outside the browser and keep results linked to this account.':
      'Exécutez les tests intensifs hors du navigateur et associez les résultats à ce compte.',
    'Pair agent': 'Associer l’agent',
    'Load generator': 'Générateur de charge',
    'No online agents': 'Aucun agent en ligne',
    'Start with agent': 'Démarrer avec l’agent',
    Offline: 'Hors ligne',
    'Remove agent': 'Supprimer l’agent',
    'Agent load test queued': 'Test de charge mis en attente',
    'Pair local agent': 'Associer un agent local',
    'Run this command on the computer that will generate load.':
      'Exécutez cette commande sur l’ordinateur qui générera la charge.',
    'Pairing code expires in 5 minutes.': 'Le code expire dans 5 minutes.',
    'Copy command': 'Copier la commande',
    'Start in browser': 'Démarrer dans le navigateur',
    queued: 'En attente',
    dispatched: 'Envoyé',
    cancel_requested: 'Arrêt en cours',
    completed: 'Terminé',
    cancelled: 'Annulé',
    'User Type': "Type d'utilisateur",
    ToB: 'ToB',
    ToC: 'ToC',
    'ToB users can access the load test demo.':
      'Les utilisateurs ToB peuvent accéder à la démo de test de charge.',
    'Add billing group route': 'Ajouter une route de facturation',
    'Add channel': 'Ajouter un canal',
    'Add error mapping': 'Ajouter une correspondance',
    'All channels': 'Tous les canaux',
    'Attempts on this channel': 'Tentatives sur ce canal',
    Balanced: 'Équilibré',
    'Billing group routes': 'Routes des groupes de facturation',
    'Channel monitoring': 'Surveillance des canaux',
    'Live channel routing health and failover metrics':
      'État en direct du routage des canaux et indicateurs de bascule',
    'Channel routing saved': 'Routage des canaux enregistré',
    'Channel switches': 'Changements de canal',
    'Circuit cooldown (seconds)': 'Délai du circuit (secondes)',
    'Circuit failure threshold': 'Seuil d’échec du circuit',
    'Circuit window (seconds)': 'Fenêtre du circuit (secondes)',
    'Circuit protection': 'Protection du circuit',
    'Tune when a channel is temporarily removed after repeated failures.':
      'Réglez la mise à l’écart temporaire d’un canal après plusieurs échecs.',
    'Quick presets': 'Préréglages rapides',
    Sensitive: 'Sensible',
    Standard: 'Standard',
    Relaxed: 'Souple',
    'Configure ordered channels for each billing group':
      'Configurer les canaux ordonnés de chaque groupe de facturation',
    'Cost factor': 'Facteur de coût',
    'Cost first': 'Coût prioritaire',
    'In flight': 'En cours',
    'Maximum total attempts': 'Nombre maximal de tentatives',
    Order: 'Ordre',
    'Request RPS': 'RPS des requêtes',
    'Routing strategy': 'Stratégie de routage',
    'Stability first': 'Stabilité prioritaire',
    'Total timeout (ms)': 'Délai total (ms)',
    'Upstream error code': 'Code d’erreur amont',
    '24 hours': '24 heures',
    '7 days': '7 jours',
    '30 days': '30 jours',
    'Actual cost': 'Coût réel',
    'Actual cost (USD)': 'Coût réel (USD)',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      'Les coûts réels sont saisis manuellement et répartis sur la période sélectionnée. Les coûts estimés sont calculés par la passerelle à partir des instantanés de facturation.',
    'Average latency': 'Latence moyenne',
    'Average token price': 'Prix moyen par token',
    'User charge = official model price × billing group ratio.':
      'Facturation utilisateur = prix officiel du modèle × ratio du groupe de facturation.',
    'Failed to load load-test limits':
      'Échec du chargement des limites de test de charge',
    'Maximum concurrency must be between {{min}} and {{max}}.':
      'La concurrence maximale doit être comprise entre {{min}} et {{max}}.',
    'Allowed range: {{min}}-{{max}} concurrent requests':
      'Plage autorisée : {{min}}-{{max}} requêtes simultanées',
    'Load Test Limits': 'Limites du test de charge',
    'These limits apply to every load-test demo run.':
      'Ces limites s’appliquent à chaque test de charge.',
    'Planned requests for this run': 'Requêtes prévues pour cette exécution',
    'Maximum duration (seconds)': 'Durée maximale (secondes)',
    'Maximum requests per second': 'Nombre maximal de requêtes par seconde',
    'Allowed range: 5-3600 seconds': 'Plage autorisée : 5-3600 secondes',
    'Allowed range: 1-10000 RPS': 'Plage autorisée : 1-10000 RPS',
    'Allowed range: 1-10000 concurrent requests':
      'Plage autorisée : 1-10000 requêtes simultanées',
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
  },
  ja: {
    'Local load-test agent': 'ローカル負荷テスト Agent',
    'Run high-volume tests outside the browser and keep results linked to this account.':
      'ブラウザ外で大規模テストを実行し、結果をこのアカウントに保存します。',
    'Pair agent': 'Agent をペアリング',
    'Load generator': '負荷生成マシン',
    'No online agents': 'オンラインの Agent がありません',
    'Start with agent': 'Agent で開始',
    Offline: 'オフライン',
    'Remove agent': 'Agent を削除',
    'Agent load test queued': 'Agent 負荷テストをキューに追加しました',
    'Pair local agent': 'ローカル Agent をペアリング',
    'Run this command on the computer that will generate load.':
      '負荷を生成するコンピューターでこのコマンドを実行してください。',
    'Pairing code expires in 5 minutes.':
      'ペアリングコードは 5 分で期限切れになります。',
    'Copy command': 'コマンドをコピー',
    'Start in browser': 'ブラウザで開始',
    queued: '待機中',
    dispatched: '送信済み',
    cancel_requested: '停止中',
    completed: '完了',
    cancelled: 'キャンセル済み',
    'User Type': 'ユーザー種別',
    ToB: 'ToB',
    ToC: 'ToC',
    'ToB users can access the load test demo.':
      'ToB ユーザーは負荷テストデモを利用できます。',
    'Add billing group route': '課金グループルートを追加',
    'Add channel': 'チャネルを追加',
    'Add error mapping': 'エラーマッピングを追加',
    'All channels': 'すべてのチャネル',
    'Attempts on this channel': 'このチャネルでの試行回数',
    Balanced: 'バランス',
    'Billing group routes': '課金グループルート',
    'Channel monitoring': 'チャネル監視',
    'Live channel routing health and failover metrics':
      'チャネルルーティングの稼働状況とフェイルオーバー指標',
    'Channel routing saved': 'チャネルルーティングを保存しました',
    'Channel switches': 'チャネル切替回数',
    'Circuit cooldown (seconds)': 'サーキット待機時間（秒）',
    'Circuit failure threshold': 'サーキット失敗しきい値',
    'Circuit window (seconds)': 'サーキット集計期間（秒）',
    'Circuit protection': 'サーキット保護',
    'Tune when a channel is temporarily removed after repeated failures.':
      '連続失敗後にチャネルを一時停止する条件を調整します。',
    'Quick presets': 'クイックプリセット',
    Sensitive: '厳格',
    Standard: '標準',
    Relaxed: '緩やか',
    'Configure ordered channels for each billing group':
      '課金グループごとに順序付きチャネルを設定します',
    'Cost factor': 'コスト係数',
    'Cost first': 'コスト優先',
    'In flight': '処理中',
    'Maximum total attempts': '最大総試行回数',
    Order: '順序',
    'Request RPS': 'リクエスト RPS',
    'Routing strategy': 'ルーティング戦略',
    'Stability first': '安定性優先',
    'Total timeout (ms)': '合計タイムアウト（ms）',
    'Upstream error code': '上流エラーコード',
    '24 hours': '24時間',
    '7 days': '7日間',
    '30 days': '30日間',
    'Actual cost': '実コスト',
    'Actual cost (USD)': '実コスト（USD）',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      '実コストは手動入力され、選択期間に按分されます。推定コストはゲートウェイの課金スナップショットから計算されます。',
    'Average latency': '平均レイテンシ',
    'Average token price': 'トークン平均単価',
    'User charge = official model price × billing group ratio.':
      'ユーザー請求額 = モデル公式価格 × 課金グループ倍率。',
    'Failed to load load-test limits': '負荷テスト制限の読み込みに失敗しました',
    'Maximum concurrency must be between {{min}} and {{max}}.':
      '最大同時実行数は {{min}} から {{max}} の間で指定してください。',
    'Allowed range: {{min}}-{{max}} concurrent requests':
      '許可範囲: {{min}}-{{max}} 件の同時リクエスト',
    'Load Test Limits': '負荷テスト制限',
    'These limits apply to every load-test demo run.':
      'これらの制限はすべての負荷テストに適用されます。',
    'Planned requests for this run': '今回の実行の予定リクエスト数',
    'Maximum duration (seconds)': '最大実行時間（秒）',
    'Maximum requests per second': '最大リクエスト毎秒数',
    'Allowed range: 5-3600 seconds': '許可範囲: 5-3600 秒',
    'Allowed range: 1-10000 RPS': '許可範囲: 1-10000 RPS',
    'Allowed range: 1-10000 concurrent requests':
      '許可範囲: 1-10000 件の同時リクエスト',
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
  },
  ru: {
    'Local load-test agent': 'Локальный агент нагрузочного теста',
    'Run high-volume tests outside the browser and keep results linked to this account.':
      'Запускайте высокую нагрузку вне браузера и сохраняйте результаты в этой учётной записи.',
    'Pair agent': 'Подключить агент',
    'Load generator': 'Генератор нагрузки',
    'No online agents': 'Нет агентов в сети',
    'Start with agent': 'Запустить через агент',
    Offline: 'Не в сети',
    'Remove agent': 'Удалить агент',
    'Agent load test queued': 'Нагрузочный тест поставлен в очередь',
    'Pair local agent': 'Подключить локальный агент',
    'Run this command on the computer that will generate load.':
      'Выполните эту команду на компьютере, который будет создавать нагрузку.',
    'Pairing code expires in 5 minutes.': 'Код подключения действует 5 минут.',
    'Copy command': 'Копировать команду',
    'Start in browser': 'Запустить в браузере',
    queued: 'В очереди',
    dispatched: 'Отправлено',
    cancel_requested: 'Останавливается',
    completed: 'Завершено',
    cancelled: 'Отменено',
    'User Type': 'Тип пользователя',
    ToB: 'ToB',
    ToC: 'ToC',
    'ToB users can access the load test demo.':
      'Пользователи ToB могут использовать демо нагрузочного тестирования.',
    'Add billing group route': 'Добавить маршрут группы тарификации',
    'Add channel': 'Добавить канал',
    'Add error mapping': 'Добавить сопоставление ошибки',
    'All channels': 'Все каналы',
    'Attempts on this channel': 'Попытки на этом канале',
    Balanced: 'Сбалансированный',
    'Billing group routes': 'Маршруты групп тарификации',
    'Channel monitoring': 'Мониторинг каналов',
    'Live channel routing health and failover metrics':
      'Текущее состояние маршрутизации каналов и показатели переключения',
    'Channel routing saved': 'Маршрутизация каналов сохранена',
    'Channel switches': 'Переключения каналов',
    'Circuit cooldown (seconds)': 'Пауза автомата (секунды)',
    'Circuit failure threshold': 'Порог ошибок автомата',
    'Circuit window (seconds)': 'Окно автомата (секунды)',
    'Circuit protection': 'Защита автомата',
    'Tune when a channel is temporarily removed after repeated failures.':
      'Настройте временное исключение канала после повторяющихся сбоев.',
    'Quick presets': 'Быстрые пресеты',
    Sensitive: 'Чувствительный',
    Standard: 'Стандартный',
    Relaxed: 'Мягкий',
    'Configure ordered channels for each billing group':
      'Настройте порядок каналов для каждой группы тарификации',
    'Cost factor': 'Коэффициент стоимости',
    'Cost first': 'Приоритет стоимости',
    'In flight': 'В обработке',
    'Maximum total attempts': 'Максимум попыток',
    Order: 'Порядок',
    'Request RPS': 'RPS запросов',
    'Routing strategy': 'Стратегия маршрутизации',
    'Stability first': 'Приоритет стабильности',
    'Total timeout (ms)': 'Общий тайм-аут (мс)',
    'Upstream error code': 'Код ошибки провайдера',
    '24 hours': '24 часа',
    '7 days': '7 дней',
    '30 days': '30 дней',
    'Actual cost': 'Фактическая стоимость',
    'Actual cost (USD)': 'Фактическая стоимость (USD)',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      'Фактические затраты вводятся вручную и распределяются по выбранному периоду. Расчётная стоимость вычисляется шлюзом по снимкам биллинга.',
    'Average latency': 'Средняя задержка',
    'Average token price': 'Средняя цена токена',
    'User charge = official model price × billing group ratio.':
      'Списание с пользователя = официальная цена модели × коэффициент группы биллинга.',
    'Failed to load load-test limits':
      'Не удалось загрузить ограничения нагрузочного теста',
    'Maximum concurrency must be between {{min}} and {{max}}.':
      'Максимальная параллельность должна быть от {{min}} до {{max}}.',
    'Allowed range: {{min}}-{{max}} concurrent requests':
      'Допустимый диапазон: {{min}}-{{max}} параллельных запросов',
    'Load Test Limits': 'Ограничения нагрузочного теста',
    'These limits apply to every load-test demo run.':
      'Эти ограничения применяются к каждому нагрузочному тесту.',
    'Planned requests for this run': 'Запланированные запросы для запуска',
    'Maximum duration (seconds)': 'Максимальная длительность (секунды)',
    'Maximum requests per second': 'Максимальное число запросов в секунду',
    'Allowed range: 5-3600 seconds': 'Допустимый диапазон: 5-3600 секунд',
    'Allowed range: 1-10000 RPS': 'Допустимый диапазон: 1-10000 RPS',
    'Allowed range: 1-10000 concurrent requests':
      'Допустимый диапазон: 1-10000 параллельных запросов',
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
  },
  vi: {
    'Local load-test agent': 'Agent kiểm thử tải cục bộ',
    'Run high-volume tests outside the browser and keep results linked to this account.':
      'Chạy kiểm thử tải lớn ngoài trình duyệt và lưu kết quả vào tài khoản này.',
    'Pair agent': 'Ghép nối Agent',
    'Load generator': 'Máy tạo tải',
    'No online agents': 'Không có Agent trực tuyến',
    'Start with agent': 'Bắt đầu bằng Agent',
    Offline: 'Ngoại tuyến',
    'Remove agent': 'Xóa Agent',
    'Agent load test queued': 'Đã xếp hàng kiểm thử tải Agent',
    'Pair local agent': 'Ghép nối Agent cục bộ',
    'Run this command on the computer that will generate load.':
      'Chạy lệnh này trên máy tính sẽ tạo tải.',
    'Pairing code expires in 5 minutes.': 'Mã ghép nối hết hạn sau 5 phút.',
    'Copy command': 'Sao chép lệnh',
    'Start in browser': 'Bắt đầu trong trình duyệt',
    queued: 'Đang chờ',
    dispatched: 'Đã gửi',
    cancel_requested: 'Đang dừng',
    completed: 'Đã hoàn tất',
    cancelled: 'Đã hủy',
    'User Type': 'Loại người dùng',
    ToB: 'ToB',
    ToC: 'ToC',
    'ToB users can access the load test demo.':
      'Người dùng ToB có thể truy cập bản demo kiểm thử tải.',
    'Add billing group route': 'Thêm tuyến nhóm tính phí',
    'Add channel': 'Thêm kênh',
    'Add error mapping': 'Thêm ánh xạ lỗi',
    'All channels': 'Tất cả kênh',
    'Attempts on this channel': 'Số lần thử trên kênh này',
    Balanced: 'Cân bằng',
    'Billing group routes': 'Tuyến nhóm tính phí',
    'Channel monitoring': 'Giám sát kênh',
    'Live channel routing health and failover metrics':
      'Tình trạng định tuyến kênh trực tiếp và chỉ số chuyển tuyến',
    'Channel routing saved': 'Đã lưu định tuyến kênh',
    'Channel switches': 'Số lần chuyển kênh',
    'Circuit cooldown (seconds)': 'Thời gian chờ ngắt mạch (giây)',
    'Circuit failure threshold': 'Ngưỡng lỗi ngắt mạch',
    'Circuit window (seconds)': 'Cửa sổ ngắt mạch (giây)',
    'Circuit protection': 'Bảo vệ ngắt mạch',
    'Tune when a channel is temporarily removed after repeated failures.':
      'Điều chỉnh thời điểm tạm loại kênh sau nhiều lần thất bại.',
    'Quick presets': 'Thiết lập nhanh',
    Sensitive: 'Nhạy',
    Standard: 'Tiêu chuẩn',
    Relaxed: 'Thoáng',
    'Configure ordered channels for each billing group':
      'Cấu hình thứ tự kênh cho từng nhóm tính phí',
    'Cost factor': 'Hệ số chi phí',
    'Cost first': 'Ưu tiên chi phí',
    'In flight': 'Đang xử lý',
    'Maximum total attempts': 'Tổng số lần thử tối đa',
    Order: 'Thứ tự',
    'Request RPS': 'RPS yêu cầu',
    'Routing strategy': 'Chiến lược định tuyến',
    'Stability first': 'Ưu tiên ổn định',
    'Total timeout (ms)': 'Tổng thời gian chờ (ms)',
    'Upstream error code': 'Mã lỗi thượng nguồn',
    '24 hours': '24 giờ',
    '7 days': '7 ngày',
    '30 days': '30 ngày',
    'Actual cost': 'Chi phí thực tế',
    'Actual cost (USD)': 'Chi phí thực tế (USD)',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      'Chi phí thực tế được nhập thủ công và phân bổ theo khoảng thời gian đã chọn. Chi phí ước tính được cổng tính từ ảnh chụp dữ liệu thanh toán.',
    'Average latency': 'Độ trễ trung bình',
    'Average token price': 'Giá trung bình mỗi token',
    'User charge = official model price × billing group ratio.':
      'Phí người dùng = giá chính thức của mô hình × hệ số nhóm thanh toán.',
    'Failed to load load-test limits': 'Không thể tải giới hạn kiểm thử tải',
    'Maximum concurrency must be between {{min}} and {{max}}.':
      'Mức đồng thời tối đa phải từ {{min}} đến {{max}}.',
    'Allowed range: {{min}}-{{max}} concurrent requests':
      'Phạm vi cho phép: {{min}}-{{max}} yêu cầu đồng thời',
    'Load Test Limits': 'Giới hạn kiểm thử tải',
    'These limits apply to every load-test demo run.':
      'Các giới hạn này áp dụng cho mọi lần chạy kiểm thử tải.',
    'Planned requests for this run': 'Số yêu cầu dự kiến cho lần chạy này',
    'Maximum duration (seconds)': 'Thời lượng tối đa (giây)',
    'Maximum requests per second': 'Số yêu cầu tối đa mỗi giây',
    'Allowed range: 5-3600 seconds': 'Phạm vi cho phép: 5-3600 giây',
    'Allowed range: 1-10000 RPS': 'Phạm vi cho phép: 1-10000 RPS',
    'Allowed range: 1-10000 concurrent requests':
      'Phạm vi cho phép: 1-10000 yêu cầu đồng thời',
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
  },
  'zh-TW': {
    'Channels are attempted in order; each channel can simulate failures and latency.':
      '頻道將按順序嘗試；每個頻道可以模擬失敗和延遲。',
    'Mock channel latency is invalid': 'Mock 頻道延遲設定無效',
    'No capacity limit': '不限制容量',
    'User Type': '使用者類型',
    ToB: 'ToB',
    ToC: 'ToC',
    'ToB users can access the load test demo.': 'ToB 使用者可以使用壓測 Demo。',
    'Add billing group route': '新增計費分組路由',
    'Add channel': '新增渠道',
    'Add error mapping': '新增錯誤映射',
    'All channels': '全部渠道',
    'Attempts on this channel': '本渠道嘗試次數',
    Balanced: '均衡',
    'Billing group routes': '計費分組路由',
    'Channel monitoring': '渠道監控',
    'Live channel routing health and failover metrics':
      '即時渠道路由健康狀態與切換指標',
    'Channel routing saved': '渠道路由已儲存',
    'Channel switches': '渠道切換次數',
    'Circuit cooldown (seconds)': '熔斷冷卻時間（秒）',
    'Circuit failure threshold': '熔斷失敗閾值',
    'Circuit window (seconds)': '熔斷統計窗口（秒）',
    'Circuit protection': '熔斷保護',
    'Tune when a channel is temporarily removed after repeated failures.':
      '調整渠道連續失敗後暫時停用的條件。',
    'Quick presets': '快速預設',
    Sensitive: '敏感',
    Standard: '標準',
    Relaxed: '寬鬆',
    'Configure ordered channels for each billing group':
      '為每個計費分組設定有序渠道',
    'Cost factor': '成本係數',
    'Cost first': '成本優先',
    'In flight': '進行中請求',
    'Maximum total attempts': '最大總嘗試次數',
    Order: '順序',
    'Request RPS': '請求 RPS',
    'Routing strategy': '路由策略',
    'Stability first': '穩定優先',
    'Total timeout (ms)': '總逾時（毫秒）',
    'Upstream error code': '上游錯誤碼',
    '24 hours': '24 小時',
    '7 days': '7 天',
    '30 days': '30 天',
    'Actual cost': '實際成本',
    'Actual cost (USD)': '實際成本（USD）',
    'Actual costs are manually recorded and prorated across the selected period. Estimated costs are calculated by the gateway from billing snapshots.':
      '實際成本由管理員手動輸入，並按所選期間分攤。估算成本由閘道根據計費快照計算。',
    'Average latency': '平均延遲',
    'Average token price': '平均每個 token 單價',
    'User charge = official model price × billing group ratio.':
      '使用者扣費 = 模型官方價格 × 計費分組倍率。',
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
  },
}

const groupPricingWorkspaceKeys = {
  en: {
    'Current billing group ratio: {{ratio}}x':
      'Current billing group ratio: {{ratio}}x',
    Enforce: 'Enforce',
    'Minimum profit margin (%)': 'Minimum profit margin (%)',
    'Monitor records pricing risk without blocking traffic. Enforce stops attempts that fall below the minimum margin.':
      'Monitor records pricing risk without blocking traffic. Enforce stops attempts that fall below the minimum margin.',
    Off: 'Off',
    'Pricing risk detected': 'Pricing risk detected',
    'Profit protection': 'Profit protection',
    'Protection mode': 'Protection mode',
    'The configured attempt path falls below the minimum margin from attempt {{position}} onward.':
      'The configured attempt path falls below the minimum margin from attempt {{position}} onward.',
    'Clear history': 'Clear history',
    'Completed at': 'Completed at',
    'Each completed test is saved with its own Run ID.':
      'Each completed test is saved with its own Run ID.',
    'Duration must be between {{min}} and {{max}} seconds.':
      'Duration must be between {{min}} and {{max}} seconds.',
    'Basic information': 'Basic information',
    'Channel priority': 'Channel priority',
    'Channel priority and weight are read-only and managed on the Channels page.':
      'Channel priority and weight are read-only and managed on the Channels page.',
    'Error mappings saved': 'Error mappings saved',
    'Group pricing': 'Group pricing',
    'Group type': 'Group type',
    'Groups with channel routing are ToB; other billing groups are ToC.':
      'Groups with channel routing are ToB; other billing groups are ToC.',
    'No billing groups configured': 'No billing groups configured',
    'Manage channel error mappings and monitor channel health.':
      'Manage channel error mappings and monitor channel health.',
    'No ToB groups configured': 'No ToB groups configured',
    'No ToC groups configured': 'No ToC groups configured',
    'Load test history': 'Load test history',
    'Load test history cleared': 'Load test history cleared',
    'No previous load tests': 'No previous load tests',
    'No channels configured for this group':
      'No channels configured for this group',
    'ToC pricing is read-only here and is managed in Basic information.':
      'ToC pricing is read-only here and is managed in Basic information.',
    'Requests per second must be between {{min}} and {{max}}.':
      'Requests per second must be between {{min}} and {{max}}.',
  },
  zh: {
    'Current billing group ratio: {{ratio}}x': '当前计费分组倍率：{{ratio}}x',
    Enforce: '强制保护',
    'Minimum profit margin (%)': '最低利润率（%）',
    'Monitor records pricing risk without blocking traffic. Enforce stops attempts that fall below the minimum margin.':
      '仅监控会记录定价风险但不会拦截流量。强制保护会停止低于最低利润率的尝试。',
    Off: '关闭',
    'Pricing risk detected': '检测到定价风险',
    'Profit protection': '利润率保护',
    'Protection mode': '保护模式',
    'The configured attempt path falls below the minimum margin from attempt {{position}} onward.':
      '从第 {{position}} 次尝试开始，当前尝试路径将低于最低利润率。',
    'Clear history': '清理历史记录',
    'Completed at': '完成时间',
    'Each completed test is saved with its own Run ID.':
      '每次完成的压测都会按独立 Run ID 保存。',
    'Duration must be between {{min}} and {{max}} seconds.':
      '压测时长必须在 {{min}} 到 {{max}} 秒之间。',
    'Basic information': '基础信息',
    'Channel priority': '渠道优先级',
    'Channel priority and weight are read-only and managed on the Channels page.':
      '渠道优先级和权重只读，请在渠道页面管理。',
    'Error mappings saved': '错误映射已保存',
    'Group pricing': '分组定价',
    'Group type': '分组类型',
    'Groups with channel routing are ToB; other billing groups are ToC.':
      '包含渠道路由的分组为 ToB，其余计费分组为 ToC。',
    'No billing groups configured': '暂无计费分组配置',
    'Manage channel error mappings and monitor channel health.':
      '管理渠道错误映射并监控渠道健康状态。',
    'No ToB groups configured': '暂无 ToB 分组配置',
    'No ToC groups configured': '暂无 ToC 分组配置',
    'Load test history': '压测历史',
    'Load test history cleared': '压测历史已清理',
    'No previous load tests': '暂无历史压测记录',
    'No channels configured for this group': '该分组暂无渠道配置',
    'ToC pricing is read-only here and is managed in Basic information.':
      'ToC 定价在此处只读，请在基础信息中管理。',
    'Requests per second must be between {{min}} and {{max}}.':
      '每秒请求数必须在 {{min}} 到 {{max}} 之间。',
  },
  'zh-TW': {
    'Current billing group ratio: {{ratio}}x': '目前計費分組倍率：{{ratio}}x',
    Enforce: '強制保護',
    'Minimum profit margin (%)': '最低利潤率（%）',
    'Monitor records pricing risk without blocking traffic. Enforce stops attempts that fall below the minimum margin.':
      '僅監控會記錄定價風險但不攔截流量。強制保護會停止低於最低利潤率的嘗試。',
    Off: '關閉',
    'Pricing risk detected': '偵測到定價風險',
    'Profit protection': '利潤率保護',
    'Protection mode': '保護模式',
    'The configured attempt path falls below the minimum margin from attempt {{position}} onward.':
      '從第 {{position}} 次嘗試開始，目前嘗試路徑將低於最低利潤率。',
    'Clear history': '清除歷史記錄',
    'Completed at': '完成時間',
    'Each completed test is saved with its own Run ID.':
      '每次完成的壓測都會以獨立 Run ID 儲存。',
    'Duration must be between {{min}} and {{max}} seconds.':
      '壓測時間必須介於 {{min}} 到 {{max}} 秒之間。',
    'Basic information': '基本資訊',
    'Channel priority': '渠道優先級',
    'Channel priority and weight are read-only and managed on the Channels page.':
      '渠道優先級和權重僅供檢視，請在渠道頁面管理。',
    'Error mappings saved': '錯誤映射已儲存',
    'Group pricing': '分組定價',
    'Group type': '分組類型',
    'Groups with channel routing are ToB; other billing groups are ToC.':
      '包含渠道路由的分組為 ToB，其餘計費分組為 ToC。',
    'No billing groups configured': '尚未設定計費分組',
    'Manage channel error mappings and monitor channel health.':
      '管理渠道錯誤映射並監控渠道健康狀態。',
    'No ToB groups configured': '尚未設定 ToB 分組',
    'No ToC groups configured': '尚未設定 ToC 分組',
    'Load test history': '壓測歷史',
    'Load test history cleared': '壓測歷史已清除',
    'No previous load tests': '尚無歷史壓測記錄',
    'Billing group ratio': '分組倍率',
    'Channel cost factor': '渠道成本係數',
    'Current run metrics': '目前執行指標',
    'Actual channel cost = official model price × channel cost factor; user charge = official model price × billing group ratio.':
      '實際渠道成本 = 模型官方價格 × 渠道成本係數；使用者收費 = 模型官方價格 × 分組倍率。',
    'No channels configured for this group': '此分組尚未設定渠道',
    'ToC pricing is read-only here and is managed in Basic information.':
      'ToC 定價在此處僅供檢視，請在基本資訊中管理。',
    'Requests per second must be between {{min}} and {{max}}.':
      '每秒請求數必須介於 {{min}} 到 {{max}} 之間。',
  },
  fr: {
    'Channels are attempted in order; each channel can simulate failures and latency.':
      'Les canaux sont essayés dans l’ordre ; chacun peut simuler des échecs et une latence.',
    'Mock channel latency is invalid': 'La latence du canal simulé est invalide.',
    'No capacity limit': 'Aucune limite de capacité',
    'Current billing group ratio: {{ratio}}x':
      'Ratio actuel du groupe de facturation : {{ratio}}x',
    Enforce: 'Appliquer',
    'Minimum profit margin (%)': 'Marge bénéficiaire minimale (%)',
    'Monitor records pricing risk without blocking traffic. Enforce stops attempts that fall below the minimum margin.':
      'La surveillance consigne le risque tarifaire sans bloquer le trafic. L’application arrête les tentatives sous la marge minimale.',
    Off: 'Désactivé',
    'Pricing risk detected': 'Risque tarifaire détecté',
    'Profit protection': 'Protection de la marge',
    'Protection mode': 'Mode de protection',
    'The configured attempt path falls below the minimum margin from attempt {{position}} onward.':
      'Le chemin configuré passe sous la marge minimale à partir de la tentative {{position}}.',
    'Clear history': 'Effacer l’historique',
    'Completed at': 'Terminé le',
    'Each completed test is saved with its own Run ID.':
      'Chaque test terminé est enregistré avec son propre identifiant de session.',
    'Duration must be between {{min}} and {{max}} seconds.':
      'La durée doit être comprise entre {{min}} et {{max}} secondes.',
    'Basic information': 'Informations de base',
    'Channel priority': 'Priorité du canal',
    'Channel priority and weight are read-only and managed on the Channels page.':
      'La priorité et le poids sont en lecture seule et se gèrent dans la page des canaux.',
    'Error mappings saved': 'Correspondances enregistrées',
    'Group pricing': 'Tarification du groupe',
    'Group type': 'Type de groupe',
    'Groups with channel routing are ToB; other billing groups are ToC.':
      'Les groupes avec routage de canaux sont ToB ; les autres groupes de facturation sont ToC.',
    'No billing groups configured': 'Aucun groupe de facturation configuré',
    'Manage channel error mappings and monitor channel health.':
      'Gérez les erreurs des canaux et surveillez leur état.',
    'No ToB groups configured': 'Aucun groupe ToB configuré',
    'No ToC groups configured': 'Aucun groupe ToC configuré',
    'Load test history': 'Historique des tests de charge',
    'Load test history cleared': 'Historique des tests de charge effacé',
    'No previous load tests': 'Aucun test de charge précédent',
    'Billing group ratio': 'Ratio du groupe de facturation',
    'Channel cost factor': 'Facteur de coût du canal',
    'Current run metrics': 'Métriques du test en cours',
    'Actual channel cost = official model price × channel cost factor; user charge = official model price × billing group ratio.':
      'Coût réel du canal = prix officiel du modèle × facteur de coût du canal ; facturation utilisateur = prix officiel du modèle × ratio du groupe de facturation.',
    'No channels configured for this group':
      'Aucun canal configuré pour ce groupe',
    'ToC pricing is read-only here and is managed in Basic information.':
      'La tarification ToC est en lecture seule ici et se gère dans les informations de base.',
    'Requests per second must be between {{min}} and {{max}}.':
      'Le nombre de requêtes par seconde doit être compris entre {{min}} et {{max}}.',
  },
  ja: {
    'Channels are attempted in order; each channel can simulate failures and latency.':
      'チャネルは順番に試行され、それぞれで失敗と遅延をシミュレートできます。',
    'Mock channel latency is invalid': 'Mock チャネルの遅延が無効です。',
    'No capacity limit': '容量制限なし',
    'Current billing group ratio: {{ratio}}x':
      '現在の課金グループ倍率：{{ratio}}x',
    Enforce: '強制',
    'Minimum profit margin (%)': '最低利益率（%）',
    'Monitor records pricing risk without blocking traffic. Enforce stops attempts that fall below the minimum margin.':
      'モニタリングはトラフィックを遮断せず料金リスクを記録します。強制モードは最低利益率を下回る試行を停止します。',
    Off: 'オフ',
    'Pricing risk detected': '料金リスクを検出',
    'Profit protection': '利益率保護',
    'Protection mode': '保護モード',
    'The configured attempt path falls below the minimum margin from attempt {{position}} onward.':
      '設定された試行経路は {{position}} 回目以降、最低利益率を下回ります。',
    'Clear history': '履歴を消去',
    'Completed at': '完了日時',
    'Each completed test is saved with its own Run ID.':
      '完了した各テストは固有の Run ID で保存されます。',
    'Duration must be between {{min}} and {{max}} seconds.':
      'テスト時間は {{min}} 秒から {{max}} 秒の間で指定してください。',
    'Basic information': '基本情報',
    'Channel priority': 'チャネル優先度',
    'Channel priority and weight are read-only and managed on the Channels page.':
      'チャネルの優先度と重みは読み取り専用です。チャネルページで管理します。',
    'Error mappings saved': 'エラーマッピングを保存しました',
    'Group pricing': 'グループ料金',
    'Group type': 'グループ種別',
    'Groups with channel routing are ToB; other billing groups are ToC.':
      'チャネルルーティングがあるグループは ToB、それ以外の課金グループは ToC です。',
    'No billing groups configured': '課金グループが設定されていません',
    'Manage channel error mappings and monitor channel health.':
      'チャネルのエラーマッピングと稼働状態を管理します。',
    'No ToB groups configured': 'ToB グループが設定されていません',
    'No ToC groups configured': 'ToC グループが設定されていません',
    'Load test history': '負荷テスト履歴',
    'Load test history cleared': '負荷テスト履歴を消去しました',
    'No previous load tests': '過去の負荷テストはありません',
    'Billing group ratio': '課金グループ倍率',
    'Channel cost factor': 'チャネルコスト係数',
    'Current run metrics': '現在の実行指標',
    'Actual channel cost = official model price × channel cost factor; user charge = official model price × billing group ratio.':
      '実際のチャネルコスト = モデル公式価格 × チャネルコスト係数、ユーザー請求額 = モデル公式価格 × 課金グループ倍率。',
    'No channels configured for this group':
      'このグループに設定されたチャネルはありません',
    'ToC pricing is read-only here and is managed in Basic information.':
      'ToC 料金はここでは読み取り専用です。基本情報で管理します。',
    'Requests per second must be between {{min}} and {{max}}.':
      '1 秒あたりのリクエスト数は {{min}} から {{max}} の間で指定してください。',
  },
  ru: {
    'Channels are attempted in order; each channel can simulate failures and latency.':
      'Каналы проверяются по порядку; для каждого можно смоделировать ошибки и задержку.',
    'Mock channel latency is invalid': 'Недопустимая задержка Mock-канала.',
    'No capacity limit': 'Без ограничения ёмкости',
    'Current billing group ratio: {{ratio}}x':
      'Текущий коэффициент группы биллинга: {{ratio}}x',
    Enforce: 'Применять',
    'Minimum profit margin (%)': 'Минимальная рентабельность (%)',
    'Monitor records pricing risk without blocking traffic. Enforce stops attempts that fall below the minimum margin.':
      'Мониторинг фиксирует ценовой риск без блокировки трафика. Принудительный режим останавливает попытки ниже минимальной рентабельности.',
    Off: 'Выкл.',
    'Pricing risk detected': 'Обнаружен ценовой риск',
    'Profit protection': 'Защита рентабельности',
    'Protection mode': 'Режим защиты',
    'The configured attempt path falls below the minimum margin from attempt {{position}} onward.':
      'Настроенный путь опускается ниже минимальной рентабельности начиная с попытки {{position}}.',
    'Clear history': 'Очистить историю',
    'Completed at': 'Время завершения',
    'Each completed test is saved with its own Run ID.':
      'Каждый завершённый тест сохраняется с отдельным идентификатором запуска.',
    'Duration must be between {{min}} and {{max}} seconds.':
      'Длительность должна быть от {{min}} до {{max}} секунд.',
    'Basic information': 'Основная информация',
    'Channel priority': 'Приоритет канала',
    'Channel priority and weight are read-only and managed on the Channels page.':
      'Приоритет и вес канала доступны только для чтения и управляются на странице каналов.',
    'Error mappings saved': 'Сопоставления ошибок сохранены',
    'Group pricing': 'Тарификация группы',
    'Group type': 'Тип группы',
    'Groups with channel routing are ToB; other billing groups are ToC.':
      'Группы с маршрутизацией каналов относятся к ToB, остальные группы биллинга — к ToC.',
    'No billing groups configured': 'Группы биллинга не настроены',
    'Manage channel error mappings and monitor channel health.':
      'Управляйте сопоставлениями ошибок и состоянием каналов.',
    'No ToB groups configured': 'Группы ToB не настроены',
    'No ToC groups configured': 'Группы ToC не настроены',
    'Load test history': 'История нагрузочных тестов',
    'Load test history cleared': 'История нагрузочных тестов очищена',
    'No previous load tests': 'Предыдущих нагрузочных тестов нет',
    'Billing group ratio': 'Коэффициент группы биллинга',
    'Channel cost factor': 'Коэффициент стоимости канала',
    'Current run metrics': 'Показатели текущего запуска',
    'Actual channel cost = official model price × channel cost factor; user charge = official model price × billing group ratio.':
      'Фактическая стоимость канала = официальная цена модели × коэффициент стоимости канала; списание с пользователя = официальная цена модели × коэффициент группы биллинга.',
    'No channels configured for this group':
      'Для этой группы каналы не настроены',
    'ToC pricing is read-only here and is managed in Basic information.':
      'Тарификация ToC доступна только для чтения и управляется в основной информации.',
    'Requests per second must be between {{min}} and {{max}}.':
      'Число запросов в секунду должно быть от {{min}} до {{max}}.',
  },
  vi: {
    'Channels are attempted in order; each channel can simulate failures and latency.':
      'Các kênh được thử theo thứ tự; mỗi kênh có thể mô phỏng lỗi và độ trễ.',
    'Mock channel latency is invalid': 'Độ trễ của kênh mô phỏng không hợp lệ.',
    'No capacity limit': 'Không giới hạn công suất',
    'Current billing group ratio: {{ratio}}x':
      'Hệ số nhóm thanh toán hiện tại: {{ratio}}x',
    Enforce: 'Bắt buộc',
    'Minimum profit margin (%)': 'Biên lợi nhuận tối thiểu (%)',
    'Monitor records pricing risk without blocking traffic. Enforce stops attempts that fall below the minimum margin.':
      'Chế độ giám sát ghi nhận rủi ro giá mà không chặn lưu lượng. Chế độ bắt buộc dừng các lần thử dưới biên tối thiểu.',
    Off: 'Tắt',
    'Pricing risk detected': 'Đã phát hiện rủi ro giá',
    'Profit protection': 'Bảo vệ lợi nhuận',
    'Protection mode': 'Chế độ bảo vệ',
    'The configured attempt path falls below the minimum margin from attempt {{position}} onward.':
      'Lộ trình đã cấu hình thấp hơn biên tối thiểu từ lần thử thứ {{position}}.',
    'Clear history': 'Xóa lịch sử',
    'Completed at': 'Hoàn tất lúc',
    'Each completed test is saved with its own Run ID.':
      'Mỗi bài kiểm thử hoàn tất được lưu bằng Run ID riêng.',
    'Duration must be between {{min}} and {{max}} seconds.':
      'Thời lượng phải nằm trong khoảng {{min}} đến {{max}} giây.',
    'Basic information': 'Thông tin cơ bản',
    'Channel priority': 'Độ ưu tiên kênh',
    'Channel priority and weight are read-only and managed on the Channels page.':
      'Độ ưu tiên và trọng số kênh chỉ được xem, quản lý tại trang Kênh.',
    'Error mappings saved': 'Đã lưu ánh xạ lỗi',
    'Group pricing': 'Định giá nhóm',
    'Group type': 'Loại nhóm',
    'Groups with channel routing are ToB; other billing groups are ToC.':
      'Nhóm có định tuyến kênh là ToB; các nhóm thanh toán khác là ToC.',
    'No billing groups configured': 'Chưa cấu hình nhóm thanh toán',
    'Manage channel error mappings and monitor channel health.':
      'Quản lý ánh xạ lỗi và theo dõi tình trạng kênh.',
    'No ToB groups configured': 'Chưa cấu hình nhóm ToB',
    'No ToC groups configured': 'Chưa cấu hình nhóm ToC',
    'Load test history': 'Lịch sử kiểm thử tải',
    'Load test history cleared': 'Đã xóa lịch sử kiểm thử tải',
    'No previous load tests': 'Chưa có kiểm thử tải trước đó',
    'Billing group ratio': 'Hệ số nhóm thanh toán',
    'Channel cost factor': 'Hệ số chi phí kênh',
    'Current run metrics': 'Chỉ số lượt chạy hiện tại',
    'Actual channel cost = official model price × channel cost factor; user charge = official model price × billing group ratio.':
      'Chi phí thực tế của kênh = giá chính thức của mô hình × hệ số chi phí kênh; phí người dùng = giá chính thức của mô hình × hệ số nhóm thanh toán.',
    'No channels configured for this group': 'Chưa cấu hình kênh cho nhóm này',
    'ToC pricing is read-only here and is managed in Basic information.':
      'Định giá ToC chỉ xem tại đây và được quản lý trong Thông tin cơ bản.',
    'Requests per second must be between {{min}} and {{max}}.':
      'Số yêu cầu mỗi giây phải nằm trong khoảng {{min}} đến {{max}}.',
  },
}

for (const [locale, translations] of Object.entries(
  groupPricingWorkspaceKeys
)) {
  Object.assign(newKeys[locale], translations)
}

const loadTestTabsKeys = {
  en: {
    'Add server agent': 'Add server agent',
    'Browser test': 'Browser test',
    'Local Agent': 'Local Agent',
    'No local agent is paired.': 'No local agent is paired.',
    'No server load generator is available.':
      'No server load generator is available.',
    'Pair server agent': 'Pair server agent',
    'Run the test on a shared load generator managed by the platform.':
      'Run the test on a shared load generator managed by the platform.',
    'Server load test': 'Server load test',
    'Edit server agent capacity': 'Edit server agent capacity',
    'These limits apply to tasks submitted to this server agent.':
      'These limits apply to tasks submitted to this server agent.',
    'Agent capacity must be a positive integer':
      'Agent capacity must be a positive integer',
    'Agent capacity must not exceed the load-test limits':
      'Agent capacity must not exceed the load-test limits',
    'Agent capacity updated': 'Agent capacity updated',
  },
  zh: {
    'Add server agent': '添加服务器 Agent',
    'Browser test': '浏览器测试',
    'Local Agent': '本地 Agent',
    'No local agent is paired.': '尚未配对本地 Agent。',
    'No server load generator is available.': '暂无可用的服务器压测机。',
    'Pair server agent': '配对服务器 Agent',
    'Run the test on a shared load generator managed by the platform.':
      '使用平台管理的共享压测机执行测试。',
    'Server load test': '服务器压测',
    'Edit server agent capacity': '编辑服务器 Agent 容量',
    'These limits apply to tasks submitted to this server agent.':
      '这些限制适用于提交到此服务器 Agent 的任务。',
    'Agent capacity must be a positive integer': 'Agent 容量必须是正整数',
    'Agent capacity must not exceed the load-test limits':
      'Agent 容量不能超过压测限制',
    'Agent capacity updated': 'Agent 容量已更新',
  },
  'zh-TW': {
    'Add server agent': '新增伺服器 Agent',
    'Browser test': '瀏覽器測試',
    'Local Agent': '本機 Agent',
    'No local agent is paired.': '尚未配對本機 Agent。',
    'No server load generator is available.': '目前沒有可用的伺服器壓測機。',
    'Pair server agent': '配對伺服器 Agent',
    'Run the test on a shared load generator managed by the platform.':
      '使用平台管理的共享壓測機執行測試。',
    'Server load test': '伺服器壓測',
    'Edit server agent capacity': '編輯伺服器 Agent 容量',
    'These limits apply to tasks submitted to this server agent.':
      '這些限制套用於提交至此伺服器 Agent 的工作。',
    'Agent capacity must be a positive integer': 'Agent 容量必須是正整數',
    'Agent capacity must not exceed the load-test limits':
      'Agent 容量不得超過壓測限制',
    'Agent capacity updated': 'Agent 容量已更新',
  },
  fr: {
    'Add server agent': 'Ajouter un agent serveur',
    'Browser test': 'Test navigateur',
    'Local Agent': 'Agent local',
    'No local agent is paired.': "Aucun agent local n'est associé.",
    'No server load generator is available.':
      "Aucun générateur de charge serveur n'est disponible.",
    'Pair server agent': 'Associer un agent serveur',
    'Run the test on a shared load generator managed by the platform.':
      'Exécuter le test sur un générateur de charge partagé géré par la plateforme.',
    'Server load test': 'Test de charge serveur',
    'Edit server agent capacity': 'Modifier la capacité de l’agent serveur',
    'These limits apply to tasks submitted to this server agent.':
      'Ces limites s’appliquent aux tâches envoyées à cet agent serveur.',
    'Agent capacity must be a positive integer':
      'La capacité de l’agent doit être un entier positif',
    'Agent capacity must not exceed the load-test limits':
      'La capacité de l’agent ne doit pas dépasser les limites du test de charge',
    'Agent capacity updated': 'Capacité de l’agent mise à jour',
  },
  ja: {
    'Add server agent': 'サーバー Agent を追加',
    'Browser test': 'ブラウザテスト',
    'Local Agent': 'ローカル Agent',
    'No local agent is paired.': 'ローカル Agent はペアリングされていません。',
    'No server load generator is available.':
      '利用可能なサーバー負荷生成機がありません。',
    'Pair server agent': 'サーバー Agent をペアリング',
    'Run the test on a shared load generator managed by the platform.':
      'プラットフォーム管理の共有負荷生成機でテストを実行します。',
    'Server load test': 'サーバー負荷テスト',
    'Edit server agent capacity': 'サーバー Agent 容量を編集',
    'These limits apply to tasks submitted to this server agent.':
      'これらの制限は、このサーバー Agent に送信されるタスクに適用されます。',
    'Agent capacity must be a positive integer':
      'Agent 容量は正の整数で入力してください',
    'Agent capacity must not exceed the load-test limits':
      'Agent 容量は負荷テストの上限を超えられません',
    'Agent capacity updated': 'Agent 容量を更新しました',
  },
  ru: {
    'Add server agent': 'Добавить серверный агент',
    'Browser test': 'Тест в браузере',
    'Local Agent': 'Локальный агент',
    'No local agent is paired.': 'Локальный агент не подключен.',
    'No server load generator is available.':
      'Нет доступного серверного генератора нагрузки.',
    'Pair server agent': 'Подключить серверный агент',
    'Run the test on a shared load generator managed by the platform.':
      'Запустить тест на общем генераторе нагрузки под управлением платформы.',
    'Server load test': 'Серверный нагрузочный тест',
    'Edit server agent capacity': 'Изменить емкость серверного агента',
    'These limits apply to tasks submitted to this server agent.':
      'Эти ограничения применяются к задачам, отправленным этому серверному агенту.',
    'Agent capacity must be a positive integer':
      'Емкость агента должна быть положительным целым числом',
    'Agent capacity must not exceed the load-test limits':
      'Емкость агента не может превышать ограничения нагрузочного теста',
    'Agent capacity updated': 'Емкость агента обновлена',
  },
  vi: {
    'Add server agent': 'Thêm Agent máy chủ',
    'Browser test': 'Kiểm thử trên trình duyệt',
    'Local Agent': 'Agent cục bộ',
    'No local agent is paired.': 'Chưa ghép đôi Agent cục bộ.',
    'No server load generator is available.':
      'Không có máy tạo tải máy chủ khả dụng.',
    'Pair server agent': 'Ghép đôi Agent máy chủ',
    'Run the test on a shared load generator managed by the platform.':
      'Chạy kiểm thử trên máy tạo tải dùng chung do nền tảng quản lý.',
    'Server load test': 'Kiểm thử tải máy chủ',
    'Edit server agent capacity': 'Chỉnh sửa dung lượng Agent máy chủ',
    'These limits apply to tasks submitted to this server agent.':
      'Các giới hạn này áp dụng cho tác vụ gửi đến Agent máy chủ này.',
    'Agent capacity must be a positive integer':
      'Dung lượng Agent phải là số nguyên dương',
    'Agent capacity must not exceed the load-test limits':
      'Dung lượng Agent không được vượt quá giới hạn kiểm thử tải',
    'Agent capacity updated': 'Đã cập nhật dung lượng Agent',
  },
}

for (const [locale, translations] of Object.entries(loadTestTabsKeys)) {
  Object.assign(newKeys[locale], translations)
}

const mockLoadTestKeys = {
  en: {
    'Test mode': 'Test mode',
    'Real channels': 'Real channels',
    'Mock channels': 'Mock channels',
    'Mock mode uses dedicated channels and does not consume the real account pool.':
      'Mock mode uses dedicated channels and does not consume the real account pool.',
    'Real mode uses the API key configured account pool.':
      'Real mode uses the API key configured account pool.',
    'Consume account pool': 'Consume account pool',
    'Do not consume account pool': 'Do not consume account pool',
    'This mode uses dedicated test channels and does not consume the account pool.':
      'This mode uses dedicated test channels and does not consume the account pool.',
    'This mode consumes the API key account pool.':
      'This mode consumes the API key account pool.',
    'Random failure rate': 'Random failure rate',
    'Failure status': 'Failure status',
    'Mixed distribution': 'Mixed distribution',
    'Randomly distribute injected failures across 429, 500, 502, 503, and 504.':
      'Randomly distribute injected failures across 429, 500, 502, 503, and 504.',
    'Additional latency (ms)': 'Additional latency (ms)',
    Mock: 'Mock',
    Real: 'Real',
  },
  zh: {
    'Test mode': '测试模式',
    'Real channels': '真实渠道',
    'Mock channels': 'Mock 渠道',
    'Mock mode uses dedicated channels and does not consume the real account pool.':
      'Mock 模式使用专用渠道，不消耗真实号池。',
    'Real mode uses the API key configured account pool.':
      '真实模式使用 API Key 配置的号池。',
    'Consume account pool': '消费号池',
    'Do not consume account pool': '不消费号池',
    'This mode uses dedicated test channels and does not consume the account pool.':
      '此模式使用专用测试渠道，不消费号池。',
    'This mode consumes the API key account pool.':
      '此模式会消费 API Key 配置的号池。',
    'Random failure rate': '随机失败率',
    'Failure status': '失败状态',
    'Mixed distribution': '混合分布',
    'Randomly distribute injected failures across 429, 500, 502, 503, and 504.':
      '在 429、500、502、503 和 504 之间随机分布注入的失败。',
    'Additional latency (ms)': '额外延迟（毫秒）',
    Mock: 'Mock',
    Real: '真实',
  },
  'zh-TW': {
    'Test mode': '測試模式',
    'Real channels': '真實渠道',
    'Mock channels': 'Mock 渠道',
    'Mock mode uses dedicated channels and does not consume the real account pool.':
      'Mock 模式使用專用渠道，不消耗真實號池。',
    'Real mode uses the API key configured account pool.':
      '真實模式使用 API Key 設定的號池。',
    'Consume account pool': '消費號池',
    'Do not consume account pool': '不消費號池',
    'This mode uses dedicated test channels and does not consume the account pool.':
      '此模式使用專用測試渠道，不消費號池。',
    'This mode consumes the API key account pool.':
      '此模式會消費 API Key 設定的號池。',
    'Random failure rate': '隨機失敗率',
    'Failure status': '失敗狀態',
    'Mixed distribution': '混合分佈',
    'Randomly distribute injected failures across 429, 500, 502, 503, and 504.':
      '在 429、500、502、503 和 504 之間隨機分布注入的失敗。',
    'Additional latency (ms)': '額外延遲（毫秒）',
    Mock: 'Mock',
    Real: '真實',
  },
  fr: {
    'Test mode': 'Mode de test',
    'Real channels': 'Canaux réels',
    'Mock channels': 'Canaux simulés',
    'Mock mode uses dedicated channels and does not consume the real account pool.':
      'Le mode simulé utilise des canaux dédiés et ne consomme pas le pool réel.',
    'Real mode uses the API key configured account pool.':
      'Le mode réel utilise le pool configuré par la clé API.',
    'Consume account pool': 'Utiliser le pool de comptes',
    'Do not consume account pool': 'Ne pas utiliser le pool de comptes',
    'This mode uses dedicated test channels and does not consume the account pool.':
      'Ce mode utilise des canaux de test dédiés et ne consomme pas le pool de comptes.',
    'This mode consumes the API key account pool.':
      'Ce mode utilise le pool de comptes de la clé API.',
    'Random failure rate': 'Taux d’échec aléatoire',
    'Failure status': 'Statut d’échec',
    'Mixed distribution': 'Distribution mixte',
    'Randomly distribute injected failures across 429, 500, 502, 503, and 504.':
      'Répartit aléatoirement les échecs injectés entre 429, 500, 502, 503 et 504.',
    'Additional latency (ms)': 'Latence supplémentaire (ms)',
    Mock: 'Simulé',
    Real: 'Réel',
  },
  ja: {
    'Test mode': 'テストモード',
    'Real channels': '実チャネル',
    'Mock channels': 'Mock チャネル',
    'Mock mode uses dedicated channels and does not consume the real account pool.':
      'Mock モードは専用チャネルを使用し、実際のアカウントプールを消費しません。',
    'Real mode uses the API key configured account pool.':
      '実モードは API キーで設定されたアカウントプールを使用します。',
    'Consume account pool': 'アカウントプールを消費',
    'Do not consume account pool': 'アカウントプールを消費しない',
    'This mode uses dedicated test channels and does not consume the account pool.':
      'このモードは専用テストチャネルを使用し、アカウントプールを消費しません。',
    'This mode consumes the API key account pool.':
      'このモードは API キーのアカウントプールを消費します。',
    'Random failure rate': 'ランダム失敗率',
    'Failure status': '失敗ステータス',
    'Mixed distribution': '混合分布',
    'Randomly distribute injected failures across 429, 500, 502, 503, and 504.':
      '注入する失敗を 429、500、502、503、504 に均等にランダム分散します。',
    'Additional latency (ms)': '追加レイテンシ（ms）',
    Mock: 'Mock',
    Real: '実',
  },
  ru: {
    'Test mode': 'Режим тестирования',
    'Real channels': 'Реальные каналы',
    'Mock channels': 'Mock-каналы',
    'Mock mode uses dedicated channels and does not consume the real account pool.':
      'В Mock-режиме используются выделенные каналы, реальный пул аккаунтов не расходуется.',
    'Real mode uses the API key configured account pool.':
      'Реальный режим использует пул, настроенный для API-ключа.',
    'Consume account pool': 'Использовать пул аккаунтов',
    'Do not consume account pool': 'Не использовать пул аккаунтов',
    'This mode uses dedicated test channels and does not consume the account pool.':
      'Этот режим использует выделенные тестовые каналы и не расходует пул аккаунтов.',
    'This mode consumes the API key account pool.':
      'Этот режим расходует пул аккаунтов, настроенный для API-ключа.',
    'Random failure rate': 'Случайная доля ошибок',
    'Failure status': 'Статус ошибки',
    'Mixed distribution': 'Смешанное распределение',
    'Randomly distribute injected failures across 429, 500, 502, 503, and 504.':
      'Случайно распределяет внедрённые ошибки между кодами 429, 500, 502, 503 и 504.',
    'Additional latency (ms)': 'Дополнительная задержка (мс)',
    Mock: 'Mock',
    Real: 'Реальный',
  },
  vi: {
    'Test mode': 'Chế độ kiểm thử',
    'Real channels': 'Kênh thật',
    'Mock channels': 'Kênh Mock',
    'Mock mode uses dedicated channels and does not consume the real account pool.':
      'Chế độ Mock dùng các kênh riêng và không tiêu tốn pool tài khoản thật.',
    'Real mode uses the API key configured account pool.':
      'Chế độ thật dùng pool được cấu hình cho API key.',
    'Consume account pool': 'Tiêu tốn pool tài khoản',
    'Do not consume account pool': 'Không tiêu tốn pool tài khoản',
    'This mode uses dedicated test channels and does not consume the account pool.':
      'Chế độ này dùng các kênh kiểm thử riêng và không tiêu tốn pool tài khoản.',
    'This mode consumes the API key account pool.':
      'Chế độ này tiêu tốn pool tài khoản được cấu hình cho API key.',
    'Random failure rate': 'Tỷ lệ lỗi ngẫu nhiên',
    'Failure status': 'Trạng thái lỗi',
    'Mixed distribution': 'Phân phối hỗn hợp',
    'Randomly distribute injected failures across 429, 500, 502, 503, and 504.':
      'Phân bổ ngẫu nhiên các lỗi được chèn vào các mã 429, 500, 502, 503 và 504.',
    'Additional latency (ms)': 'Độ trễ bổ sung (ms)',
    Mock: 'Mock',
    Real: 'Thật',
  },
}

for (const [locale, translations] of Object.entries(mockLoadTestKeys)) {
  Object.assign(newKeys[locale], translations)
}

const mockChannelSettingKeys = {
  en: {
    'Allow account-pool-free load tests': 'Allow account-pool-free load tests',
    'Allow this channel to be selected by managed load tests that do not consume the account pool.':
      'Allow this channel to be selected by managed load tests that do not consume the account pool.',
  },
  zh: {
    'Allow account-pool-free load tests': '允许不消费号池压测',
    'Allow this channel to be selected by managed load tests that do not consume the account pool.':
      '允许“不消费号池”的服务器压测选择此渠道。',
  },
  'zh-TW': {
    'Allow account-pool-free load tests': '允許不消耗號池壓測',
    'Allow this channel to be selected by managed load tests that do not consume the account pool.':
      '允許「不消耗號池」的伺服器壓測選擇此渠道。',
  },
  fr: {
    'Allow account-pool-free load tests':
      'Autoriser les tests de charge sans utiliser le pool de comptes',
    'Allow this channel to be selected by managed load tests that do not consume the account pool.':
      'Autorise les tests de charge gérés qui ne consomment pas le pool de comptes à sélectionner ce canal.',
  },
  ja: {
    'Allow account-pool-free load tests':
      'アカウントプールを消費しない負荷テストを許可',
    'Allow this channel to be selected by managed load tests that do not consume the account pool.':
      'アカウントプールを消費しないサーバー負荷テストで、このチャネルを選択できるようにします。',
  },
  ru: {
    'Allow account-pool-free load tests':
      'Разрешить нагрузочные тесты без расходования пула аккаунтов',
    'Allow this channel to be selected by managed load tests that do not consume the account pool.':
      'Разрешает серверным нагрузочным тестам, не расходующим пул аккаунтов, выбирать этот канал.',
  },
  vi: {
    'Allow account-pool-free load tests':
      'Cho phép kiểm thử tải không tiêu tốn pool tài khoản',
    'Allow this channel to be selected by managed load tests that do not consume the account pool.':
      'Cho phép các bài kiểm thử tải trên máy chủ không tiêu tốn pool tài khoản chọn kênh này.',
  },
}

for (const [locale, translations] of Object.entries(mockChannelSettingKeys)) {
  Object.assign(newKeys[locale], translations)
}

const mockChannelProfileKeys = {
  en: {
    'Fallback channel profiles': 'Fallback channel profiles',
    'Fallback channel {{index}}': 'Fallback channel {{index}}',
  },
  zh: {
    'Fallback channel profiles': '降级渠道配置',
    'Fallback channel {{index}}': '降级渠道 {{index}}',
  },
  'zh-TW': {
    'Fallback channel profiles': '降級渠道設定',
    'Fallback channel {{index}}': '降級渠道 {{index}}',
  },
  fr: {
    'Fallback channel profiles': 'Profils des canaux de secours',
    'Fallback channel {{index}}': 'Canal de secours {{index}}',
  },
  ja: {
    'Fallback channel profiles': 'フォールバックチャネル設定',
    'Fallback channel {{index}}': 'フォールバックチャネル {{index}}',
  },
  ru: {
    'Fallback channel profiles': 'Профили резервных каналов',
    'Fallback channel {{index}}': 'Резервный канал {{index}}',
  },
  vi: {
    'Fallback channel profiles': 'Cấu hình kênh dự phòng',
    'Fallback channel {{index}}': 'Kênh dự phòng {{index}}',
  },
}

for (const [locale, translations] of Object.entries(mockChannelProfileKeys)) {
  Object.assign(newKeys[locale], translations)
}

const unifiedLoadTestLimitKeys = {
  en: {
    'Run limits': 'Run limits',
    'The stricter value between the system and selected agent applies.':
      'The stricter value between the system and selected agent applies.',
    'Selected agent: {{name}} · up to {{rps}} RPS · {{concurrency}} concurrent requests.':
      'Selected agent: {{name}} · up to {{rps}} RPS · {{concurrency}} concurrent requests.',
    'Select a server or local agent to see its capacity here.':
      'Select a server or local agent to see its capacity here.',
    'System maximum': 'System maximum',
    'Effective maximum': 'Effective maximum',
    'concurrent requests': 'concurrent requests',
  },
  zh: {
    'Run limits': '本次运行限制',
    'The stricter value between the system and selected agent applies.':
      '系统限制与所选 Agent 限制取较小值。',
    'Selected agent: {{name}} · up to {{rps}} RPS · {{concurrency}} concurrent requests.':
      '当前 Agent：{{name}} · 最多 {{rps}} RPS · {{concurrency}} 个并发请求。',
    'Select a server or local agent to see its capacity here.':
      '选择服务器或本地 Agent 后，这里会显示它的容量限制。',
    'System maximum': '系统上限',
    'Effective maximum': '当前有效上限',
    'concurrent requests': '个并发请求',
  },
  'zh-TW': {
    'Run limits': '本次執行限制',
    'The stricter value between the system and selected agent applies.':
      '系統限制與所選 Agent 限制取較小值。',
    'Selected agent: {{name}} · up to {{rps}} RPS · {{concurrency}} concurrent requests.':
      '目前 Agent：{{name}} · 最多 {{rps}} RPS · {{concurrency}} 個並發請求。',
    'Select a server or local agent to see its capacity here.':
      '選擇伺服器或本機 Agent 後，這裡會顯示其容量限制。',
    'System maximum': '系統上限',
    'Effective maximum': '目前有效上限',
    'concurrent requests': '個並發請求',
  },
  fr: {
    'Run limits': 'Limites du test',
    'The stricter value between the system and selected agent applies.':
      'La limite la plus stricte entre le système et l’agent sélectionné s’applique.',
    'Selected agent: {{name}} · up to {{rps}} RPS · {{concurrency}} concurrent requests.':
      'Agent sélectionné : {{name}} · jusqu’à {{rps}} RPS · {{concurrency}} requêtes simultanées.',
    'Select a server or local agent to see its capacity here.':
      'Sélectionnez un agent serveur ou local pour voir sa capacité ici.',
    'System maximum': 'Limite système',
    'Effective maximum': 'Limite effective',
    'concurrent requests': 'requêtes simultanées',
  },
  ja: {
    'Run limits': '実行制限',
    'The stricter value between the system and selected agent applies.':
      'システムと選択したエージェントのうち、厳しい方の制限が適用されます。',
    'Selected agent: {{name}} · up to {{rps}} RPS · {{concurrency}} concurrent requests.':
      '選択中のエージェント：{{name}} · 最大 {{rps}} RPS · 同時実行 {{concurrency}} 件。',
    'Select a server or local agent to see its capacity here.':
      'サーバーまたはローカルエージェントを選択すると、ここに容量制限が表示されます。',
    'System maximum': 'システム上限',
    'Effective maximum': '有効上限',
    'concurrent requests': '同時実行リクエスト',
  },
  ru: {
    'Run limits': 'Ограничения запуска',
    'The stricter value between the system and selected agent applies.':
      'Применяется меньшее ограничение из системного и ограничения выбранного агента.',
    'Selected agent: {{name}} · up to {{rps}} RPS · {{concurrency}} concurrent requests.':
      'Выбранный агент: {{name}} · до {{rps}} RPS · {{concurrency}} одновременных запросов.',
    'Select a server or local agent to see its capacity here.':
      'Выберите серверный или локальный агент, чтобы увидеть его ограничения.',
    'System maximum': 'Системный предел',
    'Effective maximum': 'Фактический предел',
    'concurrent requests': 'одновременных запросов',
  },
  vi: {
    'Run limits': 'Giới hạn chạy',
    'The stricter value between the system and selected agent applies.':
      'Áp dụng giới hạn nhỏ hơn giữa hệ thống và agent đã chọn.',
    'Selected agent: {{name}} · up to {{rps}} RPS · {{concurrency}} concurrent requests.':
      'Agent đã chọn: {{name}} · tối đa {{rps}} RPS · {{concurrency}} yêu cầu đồng thời.',
    'Select a server or local agent to see its capacity here.':
      'Chọn agent máy chủ hoặc cục bộ để xem giới hạn công suất tại đây.',
    'System maximum': 'Giới hạn hệ thống',
    'Effective maximum': 'Giới hạn hiệu dụng',
    'concurrent requests': 'yêu cầu đồng thời',
  },
}

const sharedAgentKeys = {
  en: {
    'Execution mode': 'Execution mode',
    'Single Agent': 'Single Agent',
    'Shared Agent': 'Shared Agent',
    'Expected workers': 'Expected workers',
    'Use one paired Agent across multiple machines; the total load is split between workers.':
      'Use one paired Agent across multiple machines; the total load is split between workers.',
    'Run the load from one paired Agent.': 'Run the load from one paired Agent.',
    'All RPS and concurrency values on this page are the total across workers.':
      'All RPS and concurrency values on this page are the total across workers.',
    'Estimated aggregate capacity: {{rps}} RPS · {{concurrency}} concurrent requests':
      'Estimated aggregate capacity: {{rps}} RPS · {{concurrency}} concurrent requests',
    'Enter 2-256 workers to estimate aggregate capacity.':
      'Enter 2-256 workers to estimate aggregate capacity.',
    'Shared mode requires 2-256 workers': 'Shared mode requires 2-256 workers',
    'Workers joined': 'Workers joined',
  },
  zh: {
    'Execution mode': '执行模式',
    'Single Agent': '单机 Agent',
    'Shared Agent': '共享 Agent',
    'Expected workers': '预计 Worker 数量',
    'Use one paired Agent across multiple machines; the total load is split between workers.':
      '将同一配对 Agent 复制到多台机器，页面配置的总压力会分摊到各 Worker。',
    'Run the load from one paired Agent.': '由一台已配对 Agent 发起压测。',
    'All RPS and concurrency values on this page are the total across workers.':
      '本页面的 RPS 和并发数均表示所有 Worker 的总和。',
    'Estimated aggregate capacity: {{rps}} RPS · {{concurrency}} concurrent requests':
      '预计总容量：{{rps}} RPS · {{concurrency}} 个并发请求',
    'Enter 2-256 workers to estimate aggregate capacity.':
      '请输入 2-256 个 Worker 以估算总容量。',
    'Shared mode requires 2-256 workers': '共享模式需要 2-256 个 Worker',
    'Workers joined': '已加入 Worker',
  },
  fr: {
    'Execution mode': 'Mode d’exécution',
    'Single Agent': 'Agent unique',
    'Shared Agent': 'Agent partagé',
    'Expected workers': 'Workers attendus',
    'Use one paired Agent across multiple machines; the total load is split between workers.':
      'Utilisez un Agent appairé sur plusieurs machines ; la charge totale est répartie entre les workers.',
    'Run the load from one paired Agent.': 'Lancer la charge depuis un Agent appairé.',
    'All RPS and concurrency values on this page are the total across workers.':
      'Les valeurs RPS et de concurrence indiquées sont le total de tous les workers.',
    'Estimated aggregate capacity: {{rps}} RPS · {{concurrency}} concurrent requests':
      'Capacité agrégée estimée : {{rps}} RPS · {{concurrency}} requêtes simultanées',
    'Enter 2-256 workers to estimate aggregate capacity.':
      'Saisissez 2 à 256 workers pour estimer la capacité agrégée.',
    'Shared mode requires 2-256 workers': 'Le mode partagé nécessite 2 à 256 workers',
    'Workers joined': 'Workers connectés',
  },
  ja: {
    'Execution mode': '実行モード',
    'Single Agent': '単一 Agent',
    'Shared Agent': '共有 Agent',
    'Expected workers': '想定 Worker 数',
    'Use one paired Agent across multiple machines; the total load is split between workers.':
      '1 つのペア済み Agent を複数マシンで使い、総負荷を Worker 間で分割します。',
    'Run the load from one paired Agent.': '1 台のペア済み Agent から負荷を生成します。',
    'All RPS and concurrency values on this page are the total across workers.':
      'このページの RPS と同時実行数は全 Worker の合計です。',
    'Estimated aggregate capacity: {{rps}} RPS · {{concurrency}} concurrent requests':
      '推定合計容量：{{rps}} RPS · 同時実行 {{concurrency}} 件',
    'Enter 2-256 workers to estimate aggregate capacity.':
      '合計容量を推定するには 2〜256 の Worker を入力してください。',
    'Shared mode requires 2-256 workers': '共有モードでは 2〜256 Worker が必要です',
    'Workers joined': '参加 Worker',
  },
  ru: {
    'Execution mode': 'Режим выполнения',
    'Single Agent': 'Один агент',
    'Shared Agent': 'Общий агент',
    'Expected workers': 'Ожидаемое число workers',
    'Use one paired Agent across multiple machines; the total load is split between workers.':
      'Используйте одного сопряжённого агента на нескольких машинах; общая нагрузка распределяется между workers.',
    'Run the load from one paired Agent.': 'Запускать нагрузку с одного сопряжённого агента.',
    'All RPS and concurrency values on this page are the total across workers.':
      'Значения RPS и параллельности на этой странице являются суммой по всем workers.',
    'Estimated aggregate capacity: {{rps}} RPS · {{concurrency}} concurrent requests':
      'Расчётная общая ёмкость: {{rps}} RPS · {{concurrency}} одновременных запросов',
    'Enter 2-256 workers to estimate aggregate capacity.':
      'Введите от 2 до 256 workers для оценки общей ёмкости.',
    'Shared mode requires 2-256 workers': 'Общий режим требует от 2 до 256 workers',
    'Workers joined': 'Подключено workers',
  },
  vi: {
    'Execution mode': 'Chế độ thực thi',
    'Single Agent': 'Agent đơn',
    'Shared Agent': 'Agent dùng chung',
    'Expected workers': 'Số worker dự kiến',
    'Use one paired Agent across multiple machines; the total load is split between workers.':
      'Dùng một Agent đã ghép nối trên nhiều máy; tổng tải được chia cho các worker.',
    'Run the load from one paired Agent.': 'Tạo tải từ một Agent đã ghép nối.',
    'All RPS and concurrency values on this page are the total across workers.':
      'Các giá trị RPS và đồng thời trên trang này là tổng của tất cả worker.',
    'Estimated aggregate capacity: {{rps}} RPS · {{concurrency}} concurrent requests':
      'Công suất tổng ước tính: {{rps}} RPS · {{concurrency}} yêu cầu đồng thời',
    'Enter 2-256 workers to estimate aggregate capacity.':
      'Nhập 2-256 worker để ước tính công suất tổng.',
    'Shared mode requires 2-256 workers': 'Chế độ dùng chung cần 2-256 worker',
    'Workers joined': 'Worker đã tham gia',
  },
  'zh-TW': {
    'Execution mode': '執行模式',
    'Single Agent': '單機 Agent',
    'Shared Agent': '共享 Agent',
    'Expected workers': '預計 Worker 數量',
    'Use one paired Agent across multiple machines; the total load is split between workers.':
      '將同一個配對 Agent 複製到多台機器，頁面設定的總壓力會分攤到各 Worker。',
    'Run the load from one paired Agent.': '由一台已配對 Agent 發起壓測。',
    'All RPS and concurrency values on this page are the total across workers.':
      '本頁面的 RPS 和並發數均表示所有 Worker 的總和。',
    'Estimated aggregate capacity: {{rps}} RPS · {{concurrency}} concurrent requests':
      '預計總容量：{{rps}} RPS · {{concurrency}} 個並發請求',
    'Enter 2-256 workers to estimate aggregate capacity.':
      '請輸入 2-256 個 Worker 以估算總容量。',
    'Shared mode requires 2-256 workers': '共享模式需要 2-256 個 Worker',
    'Workers joined': '已加入 Worker',
  },
}

for (const [locale, translations] of Object.entries(unifiedLoadTestLimitKeys)) {
  Object.assign(newKeys[locale], translations)
}

for (const [locale, translations] of Object.entries(sharedAgentKeys)) {
  Object.assign(newKeys[locale], translations)
}

for (const locale of Object.keys(newKeys)) {
  const file = path.join(LOCALES_DIR, `${locale}.json`)
  const json = JSON.parse(await fs.readFile(file, 'utf8'))
  delete json.translation[
    'These limits apply to every load-test demo run. The request cap is fixed at 10,000 requests.'
  ]
  for (const [key, value] of Object.entries(newKeys[locale])) {
    if (json.translation[key] !== value) json.translation[key] = value
  }
  await fs.writeFile(file, `${JSON.stringify(json, null, 2)}\n`, 'utf8')
}
