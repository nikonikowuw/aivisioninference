export const layout = {
  sidebar: {
    dashboard: 'ダッシュボード',
    userManagement: 'ユーザー管理',
    users: 'ユーザー',
    roles: 'ロール',
    permissions: '権限',
    systemManagement: 'システム管理',
    files: 'ファイル',
    auditLogs: '操作ログ',
    tasks: 'タスク',
    brandConfig: 'ブランド設定',
    mailConfig: 'メール設定',
    feedback: 'フィードバック',
    noAccess: 'アクセス権限がありません',
  },
  navbar: {
    profile: 'プロフィール',
    settings: '設定',
    logout: 'ログアウト',
    notifications: '通知',
  },
  footer: {
    copyright: '© {{year}} Niko Admin. All rights reserved.',
  },
} as const;

export default layout;
