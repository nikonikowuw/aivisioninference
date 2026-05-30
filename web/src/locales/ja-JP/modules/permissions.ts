export const permissions = {
  title: '権限管理',
  button: {
    create: '権限を追加',
  },
  actions: {
    edit: '編集',
    delete: '削除',
  },
  table: {
    columns: {
      name: '権限名',
      code: '権限コード',
      type: 'タイプ',
    },
  },
  form: {
    name: {
      label: '権限名',
      placeholder: '権限名を入力してください',
    },
    code: {
      label: '権限コード',
      placeholder: '例：user:create',
    },
    type: {
      label: 'タイプ',
      menu: 'メニュー',
      button: 'ボタン',
    },
    parentId: {
      label: '親権限',
      placeholder: '親権限を選択してください',
      none: 'なし（トップレベル）',
    },
  },
  modal: {
    createTitle: '権限を追加',
    editTitle: '権限を編集',
    deleteTitle: '権限を削除',
  },
  message: {
    createSuccess: '作成に成功しました',
    createFailed: '作成に失敗しました',
    updateSuccess: '更新に成功しました',
    updateFailed: '更新に失敗しました',
    deleteSuccess: '削除に成功しました',
    deleteFailed: '削除に失敗しました',
    deleteConfirm: 'この権限とそのすべての子権限を削除してもよろしいですか？',
    emptyData: '権限データがありません',
  },
  codes: {
    dashboard: {
      view: 'ダッシュボードを表示',
    },
    user: {
      list: 'ユーザー一覧',
      create: 'ユーザーを作成',
      edit: 'ユーザーを編集',
      delete: 'ユーザーを削除',
      view: 'ユーザー詳細を表示',
      'reset-password': 'パスワードをリセット',
      'upload-avatar': 'アバターをアップロード',
    },
    role: {
      list: 'ロール一覧',
      create: 'ロールを作成',
      edit: 'ロールを編集',
      delete: 'ロールを削除',
      'assign-permissions': '権限を割り当て',
      'view-permissions': 'ロール権限を表示',
      view: 'ロール詳細を表示',
    },
    permission: {
      list: '権限一覧',
      create: '権限を作成',
      edit: '権限を編集',
      delete: '権限を削除',
      view: '権限詳細を表示',
    },
    file: {
      list: 'ファイル一覧',
      upload: 'ファイルをアップロード',
      check: 'ファイルをチェック',
      'upload-progress': 'アップロード進捗',
      delete: 'ファイルを削除',
      view: 'ファイルを表示',
      download: 'ファイルをダウンロード',
    },
    audit: {
      view: '操作ログを表示',
    },
    'audit-log': {
      view: '操作ログを表示',
    },
    task: {
      list: 'タスク一覧',
      create: 'タスクを作成',
      cancel: 'タスクをキャンセル',
      view: 'タスク詳細を表示',
      update: 'タスクを更新',
      delete: 'タスクを削除',
    },
    'brand-config': {
      view: 'ブランド設定を表示',
      edit: 'ブランド設定を編集',
      'upload-logo': 'ブランド Logo をアップロード',
    },
    'mail-config': {
      view: 'メール設定を表示',
      edit: 'メール設定を編集',
      'test-smtp': 'SMTPテスト',
      'test-imap': 'IMAPテスト',
      'sync-imap': 'フィードバックメールを同期',
      update: 'メール設定を更新',
      test: 'メール設定テスト',
      sync: 'メール同期',
    },
    feedback: {
      view: 'フィードバックを表示',
      'update-status': 'フィードバックステータスを更新',
      update: 'フィードバックを編集',
    },
  },
} as const;

export default permissions;
