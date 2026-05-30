export const roles = {
  title: "ロール管理",
  button: {
    assignPermissions: "権限を割り当て",
  },
  table: {
    columns: {
      name: "ロール名",
      description: "説明",
      permissionCount: "権限数",
    },
  },
  form: {
    name: {
      label: "ロール名",
      placeholder: "ロール名を入力",
    },
    description: {
      label: "説明",
      placeholder: "説明を入力",
    },
    level: {
      label: "レベル",
      placeholder: "数値が小さいほど権限が高くなります",
      helper:
        "数値が小さいほど高い権限を付与します。最終的な権限チェックはバックエンドで行われます。",
    },
  },
  modal: {
    createTitle: "ロールを追加",
    editTitle: "ロールを編集",
    assignPermissionsTitle: "権限を割り当て",
  },
  permissions: {
    empty: "権限データがありません",
  },
  message: {
    assignPermissionsSuccess: "権限を割り当てました",
    loadPermissionsFailed: "権限の読み込みに失敗しました",
    assignPermissionsFailed: "権限の割り当てに失敗しました",
    deleteConfirm: "このロールを削除してもよろしいですか？",
    levelInvalidTitle: "無効なレベル入力",
    levelInvalidDescription:
      "{{min}} から {{max}} の間の整数を入力してください",
    batchDeleteConfirm: "選択した {{count}} 件のロールを削除しますか？",
  },
  batch: {
    delete: "一括削除",
  },
} as const;

export default roles;
