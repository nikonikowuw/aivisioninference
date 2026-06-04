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
    batchUpdateStatus: "一括ステータス更新",
    viewDetails: "詳細を見る",
    copy: "コピー",
  },
  detail: {
    title: "フィードバック詳細",
    email: "連絡先メール",
    updatedAt: "更新日時",
    noEmail: "メール未登録",
    copySuccess: "メールアドレスをコピーしました",
    copyFailed: "メールアドレスのコピーに失敗しました",
  },
  batch: {
    selected: "{{count}} 件選択中",
  },
  message: {
    updated: "フィードバック状態を更新しました",
    batchDone: "一括更新完了: {{success}} 件成功、{{failed} 件失敗",
    batchUpdateConfirm: "選択した {{count}} 件のフィードバックのステータスを更新しますか?",
    operationFailed: "操作に失敗しました",
  },
} as const;

export default feedback;
