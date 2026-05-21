export const tasks = {
  title: 'タスク管理',
  filter: {
    taskTypes: {
      email: 'メール',
      export: 'エクスポート',
      import: 'インポート',
      backup: 'バックアップ',
    },
  },
  table: {
    columns: {
      id: 'ID',
      type: 'タイプ',
      status: 'ステータス',
      error: 'エラー',
      createdAt: '作成日時',
      updatedAt: '更新日時',
      actions: '操作',
    },
    status: {
      pending: '待機中',
      running: '実行中',
      completed: '完了',
      failed: '失敗',
      cancelled: 'キャンセル済み',
    },
  },
  message: {
    cancelled: 'タスクをキャンセルしました',
    cancelFailed: 'タスクのキャンセルに失敗しました',
    cancelConfirm: 'このタスクをキャンセルしてもよろしいですか？',
  },
  actions: {
    cancel: 'タスクをキャンセル',
  },
} as const;

export default tasks;
