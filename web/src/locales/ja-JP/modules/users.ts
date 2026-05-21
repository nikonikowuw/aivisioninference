export const users = {
  title: 'ユーザー管理',
  button: {
    create: 'ユーザーを追加',
  },
  table: {
    columns: {
      id: 'ID',
      username: 'ユーザー名',
      displayName: '表示名',
      email: 'メールアドレス',
      status: 'ステータス',
      roles: 'ロール',
      actions: '操作',
    },
    status: {
      active: '有効',
      inactive: '無効',
    },
  },
  form: {
    username: {
      label: 'ユーザー名',
      placeholder: 'ユーザー名を入力',
    },
    displayName: {
      label: '表示名',
      placeholder: '表示名を入力',
    },
    email: {
      label: 'メールアドレス',
      placeholder: 'メールアドレスを入力',
    },
    password: {
      label: 'パスワード',
      placeholder: 'パスワードを入力',
      hint: '変更しない場合は空白のままにしてください',
    },
    status: {
      label: 'ステータス',
      active: '有効',
      inactive: '無効',
      readOnlyHint: 'ユーザーの有効/無効の切り替えは、一覧画面のステータススイッチを使用してください。',
    },
  },
  modal: {
    createTitle: 'ユーザーを追加',
    editTitle: 'ユーザーを編集',
  },
  message: {
    createSuccess: 'ユーザーを作成しました',
    updateSuccess: 'ユーザーを更新しました',
    deleteSuccess: 'ユーザーを削除しました',
    resetPasswordSuccess: 'パスワードをリセットしました',
    deleteConfirm: 'このユーザーを削除してもよろしいですか？',
    disableConfirm: 'このユーザーを無効にしてもよろしいですか？',
    enableConfirm: 'このユーザーを有効にしてもよろしいですか？',
    operationFailed: '操作が失敗しました',
    deleteFailed: '削除に失敗しました',
  },
  actions: {
    edit: '編集',
    delete: '削除',
    enable: '有効化',
    disable: '無効化',
  },
} as const;

export default users;
