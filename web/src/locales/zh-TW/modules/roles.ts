export const roles = {
  title: '角色管理',
  button: {
    create: '新增角色',
    assignPermissions: '分配權限',
  },
  table: {
    columns: {
      id: 'ID',
      name: '角色名稱',
      description: '描述',
      permissionCount: '權限數',
      actions: '操作',
    },
  },
  form: {
    name: {
      label: '角色名稱',
      placeholder: '請輸入角色名稱',
    },
    description: {
      label: '描述',
      placeholder: '請輸入描述',
    },
    level: {
      label: '等級',
      placeholder: '等級數值越小權限越高',
    },
  },
  modal: {
    createTitle: '新增角色',
    editTitle: '編輯角色',
    assignPermissionsTitle: '分配權限',
  },
  permissions: {
    empty: '暫無權限資料',
  },
  message: {
    createSuccess: '建立成功',
    updateSuccess: '更新成功',
    deleteSuccess: '刪除成功',
    assignPermissionsSuccess: '權限分配成功',
    loadPermissionsFailed: '取得權限失敗',
    assignPermissionsFailed: '分配失敗',
    deleteConfirm: '確定刪除該角色？',
    operationFailed: '操作失敗',
    deleteFailed: '刪除失敗',
    levelInvalidTitle: '等級輸入無效',
    levelInvalidDescription: '請輸入 {{min}} 到 {{max}} 之間的整數等級',
  },
  actions: {
    edit: '編輯',
    delete: '刪除',
  },
} as const;

export default roles;
