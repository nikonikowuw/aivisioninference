export const users = {
  title: "ユーザー管理",
  table: {
    columns: {
      username: "ユーザー名",
      displayName: "表示名",
      email: "メールアドレス",
      roles: "ロール",
    },
  },
  form: {
    username: {
      label: "ユーザー名",
      placeholder: "ユーザー名を入力",
    },
    displayName: {
      label: "表示名",
      placeholder: "表示名を入力",
    },
    email: {
      label: "メールアドレス",
      placeholder: "メールアドレスを入力",
    },
    password: {
      label: "パスワード",
      placeholder: "パスワードを入力",
      hint: "変更しない場合は空白のままにしてください",
    },
    status: {
      label: "ステータス",
      active: "有効",
      inactive: "無効",
      readOnlyHint:
        "ユーザーの有効/無効の切り替えは、一覧画面のステータススイッチを使用してください。",
    },
  },
  modal: {
    createTitle: "ユーザーを追加",
    editTitle: "ユーザーを編集",
  },
  message: {
    resetPasswordSuccess: "パスワードをリセットしました",
    deleteConfirm: "このユーザーを削除してもよろしいですか？",
    disableConfirm: "このユーザーを無効にしてもよろしいですか？",
    enableConfirm: "このユーザーを有効にしてもよろしいですか？",
    batchDeleteConfirm: "選択した {{count}} 件のユーザーを削除しますか？",
    batchEnableConfirm: "選択した {{count}} 件のユーザーを有効化しますか？",
    batchDisableConfirm: "選択した {{count}} 件のユーザーを無効化しますか？",
  },
  batch: {
    selected: "{{count}} 件のユーザーを選択中",
    selectPage: "このページのユーザーを選択",
    selectRow: "ユーザー {{username}} を選択",
    delete: "一括削除",
    enable: "一括有効化",
    disable: "一括無効化",
  },
} as const;

export default users;
