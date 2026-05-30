export const feedback = {
  title: "ユーザーフィードバック",
  source: {
    user: "アプリ内",
    email: "メール",
  },
  status: {
    open: "未処理",
    processing: "処理中",
    resolved: "解決済み",
    closed: "終了",
  },
  table: {
    source: "ソース",
    category: "カテゴリ",
    title: "タイトル",
    content: "内容",
    status: "ステータス",
    createdAt: "作成日時",
    actions: "操作",
  },
  actions: {
    refresh: "更新",
  },
  message: {
    updated: "ステータスが更新されました",
  },
} as const;

export default feedback;
