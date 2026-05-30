export const auditLogs = {
  title: '操作ログ',
  filter: {
    results: {
      success: '成功',
      failed: '失敗',
    },
    resourceTypes: {
      user: 'ユーザー',
      role: 'ロール',
      permission: '権限',
      file: 'ファイル',
      task: 'タスク',
      system: 'システム',
      feedback: 'フィードバック',
    },
  },
  table: {
    columns: {
      username: '操作者',
      actionType: '操作',
      resourceType: 'リソースタイプ',
      method: 'メソッド',
      path: 'パス',
      ip: 'IP',
      status: 'ステータス',
      duration: '所要時間',
      result: '結果',
      time: '日時',
    },
    durationMs: '{{value}} ms',
  },
  actionTypes: {
    action: {
      view: {
        users: 'ユーザー一覧を表示',
        roles: 'ロール一覧を表示',
        permissions: '権限一覧を表示',
        files: 'ファイル一覧を表示',
        'audit-logs': '操作ログを表示',
        tasks: 'タスク一覧を表示',
        system: 'システム設定を表示',
        dashboard: 'ダッシュボードを表示',
        feedback: 'フィードバックを表示',
      },
      create: {
        users: 'ユーザーを作成',
        roles: 'ロールを作成',
        permissions: '権限を作成',
        files: 'ファイルをアップロード',
        tasks: 'タスクを作成',
        system: 'システム設定を作成',
        feedback: 'フィードバックを送信',
      },
      update: {
        users: 'ユーザーを更新',
        roles: 'ロールを更新',
        permissions: '権限を更新',
        files: 'ファイルを更新',
        tasks: 'タスクを更新',
        system: 'システム設定を更新',
        feedback: 'フィードバック状態を更新',
      },
      delete: {
        users: 'ユーザーを削除',
        roles: 'ロールを削除',
        permissions: '権限を削除',
        files: 'ファイルを削除',
        tasks: 'タスクを削除',
      },
      login: 'ログイン',
      logout: 'ログアウト',
      auth: '認証操作',
    },
  },
} as const;

export default auditLogs;
