export const dashboard = {
  title: 'ダッシュボード',
  stats: {
    totalUsers: '総ユーザー数',
    totalFiles: '総ファイル数',
    activeTasks: '実行中のタスク',
  },
  charts: {
    userStats: 'ユーザー成長とアクティビティ',
    userNew: '新規ユーザー',
    userActive: 'アクティブユーザー',
  },
  auditLog: {
    title: '最近のアクティビティ',
    empty: '最近のアクティビティはありません',
    columns: {
      username: 'ユーザー',
      action: '操作',
      method: 'メソッド',
      time: '時間',
    },
  },
  message: {
    loadFailed: 'ダッシュボードデータの読み込みに失敗しました',
  },
} as const;

export default dashboard;
