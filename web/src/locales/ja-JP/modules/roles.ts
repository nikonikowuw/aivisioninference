export const roles = {
  title: 'ロール管理',
  button: {
    create: 'ロールを追加',
    assignPermissions: '権限を割り当て',
  },
  table: {
    columns: {
      id: 'ID',
      name: 'ロール名',
      description: '説明',
      permissionCount: '権限数',
      status: 'ステータス',
      actions: '操作',
    },
    status: {
      active: '有効',
      inactive: '無効',
    },
  },
  form: {
    name: {
      label: 'ロール名',
      placeholder: 'ロール名を入力',
    },
    description: {
      label: '説明',
      placeholder: '説明を入力',
    },
    level: {
      label: 'レベル',
      placeholder: '数値が小さいほど権限が高くなります',
      helper: '数値が小さいほど高い権限を付与します。最終的な権限チェックはバックエンドで行われます。',
    },
  },
  modal: {
    createTitle: 'ロールを追加',
    editTitle: 'ロールを編集',
    assignPermissionsTitle: '権限を割り当て',
  },
  permissions: {
    empty: '権限データがありません',
  },
  message: {
    createSuccess: 'ロールを作成しました',
    updateSuccess: 'ロールを更新しました',
    deleteSuccess: 'ロールを削除しました',
    assignPermissionsSuccess: '権限を割り当てました',
    loadPermissionsFailed: '権限の読み込みに失敗しました',
    assignPermissionsFailed: '権限の割り当てに失敗しました',
    deleteConfirm: 'このロールを削除してもよろしいですか？',
    operationFailed: '操作が失敗しました',
    deleteFailed: '削除に失敗しました',
    levelInvalidTitle: '無効なレベル入力',
    levelInvalidDescription: '{{min}} から {{max}} の間の整数を入力してください',
  },
  actions: {
    edit: '編集',
    delete: '削除',
  },
} as const;

export default roles;
