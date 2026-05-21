export const permissions = {
  title: '権限管理',
  button: {
    create: '権限を追加',
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
      placeholder: '権限名を入力',
    },
    code: {
      label: '権限コード',
      placeholder: '例: user:create',
    },
    type: {
      label: 'タイプ',
      menu: 'メニュー',
      button: 'ボタン',
    },
    parentId: {
      label: '親権限',
      placeholder: '親権限を選択',
      none: 'なし (トップレベル)',
    },
  },
  modal: {
    createTitle: '権限を追加',
    editTitle: '権限を編集',
    deleteTitle: '権限を削除',
  },
  message: {
    createSuccess: '権限を作成しました',
    createFailed: '権限の作成に失敗しました',
    updateSuccess: '権限を更新しました',
    updateFailed: '権限の更新に失敗しました',
    deleteSuccess: '権限を削除しました',
    deleteFailed: '権限の削除に失敗しました',
    deleteConfirm: 'この権限とすべての子権限を削除してもよろしいですか？',
    emptyData: '権限データがありません',
  },
} as const;

export default permissions;
