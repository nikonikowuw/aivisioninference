export const tasks = {
  title: "タスク管理",
  filter: {
    taskTypes: {
      email: "メール",
      export: "エクスポート",
      import: "インポート",
      backup: "バックアップ",
    },
  },
  table: {
    columns: {
      type: "タイプ",
      error: "エラー",
    },
    status: {
      pending: "待機中",
      running: "実行中",
      completed: "完了",
      failed: "失敗",
      cancelled: "キャンセル済み",
    },
  },
  message: {
    cancelled: "タスクをキャンセルしました",
    cancelFailed: "タスクのキャンセルに失敗しました",
    cancelConfirm: "このタスクをキャンセルしてもよろしいですか？",
    batchCancelConfirm: "選択した {{count}} 個のタスクをキャンセルしますか？",
    confirmCancel: "はい、キャンセルします",
  },
  actions: {
    cancel: "タスクをキャンセル",
  },
} as const;

export default tasks;
