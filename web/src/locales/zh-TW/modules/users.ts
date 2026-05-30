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
      readOnlyHint: '編輯時請在列表狀態開關中啟用或停用使用者。',
    },
  },
  modal: {
    createTitle: '新增使用者',
    editTitle: '編輯使用者',
  },
  message: {
    resetPasswordSuccess: '密碼重置成功',
    deleteConfirm: '確定刪除該使用者？',
    disableConfirm: '確定停用該使用者？',
    enableConfirm: '確定啟用該使用者？',
  },
  batch: {
    selected: '已選取 {{count}} 個使用者',
    selectPage: '選取目前頁面使用者',
    selectRow: '選取使用者 {{username}}',
    delete: '批次刪除',
    enable: '批次啟用',
    disable: '批次停用',
  },
  actions: {
    edit: '編輯',
    delete: '刪除',
    enable: '啟用',
    disable: '停用',
  },
} as const;

export default users;
