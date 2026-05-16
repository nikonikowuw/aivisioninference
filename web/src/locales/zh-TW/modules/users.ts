export const users = {
  title: '使用者管理',
  button: {
    create: '新增使用者',
  },
  table: {
    columns: {
      id: 'ID',
      username: '使用者名稱',
      displayName: '顯示名稱',
      email: '電子郵件',
      status: '狀態',
      roles: '角色',
      actions: '操作',
    },
    status: {
      active: '正常',
      inactive: '停用',
    },
  },
  form: {
    username: {
      label: '使用者名稱',
      placeholder: '請輸入使用者名稱',
    },
    displayName: {
      label: '顯示名稱',
      placeholder: '請輸入顯示名稱',
    },
    email: {
      label: '電子郵件',
      placeholder: '請輸入電子郵件',
    },
    password: {
      label: '密碼',
      placeholder: '請輸入密碼',
      hint: '留空不修改',
    },
    status: {
      label: '狀態',
      active: '正常',
      inactive: '停用',
    },
  },
  modal: {
    createTitle: '新增使用者',
    editTitle: '編輯使用者',
  },
  message: {
    createSuccess: '建立成功',
    updateSuccess: '更新成功',
    deleteSuccess: '刪除成功',
    deleteConfirm: '確定刪除該使用者？',
    operationFailed: '操作失敗',
    deleteFailed: '刪除失敗',
  },
  actions: {
    edit: '編輯',
    delete: '刪除',
  },
} as const;

export default users;
