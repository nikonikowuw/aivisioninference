export const permissions = {
  title: '權限管理',
  button: {
    create: '新增權限',
  },
  table: {
    columns: {
      name: '權限名稱',
      code: '權限代碼',
      type: '類型',
    },
  },
  form: {
    name: {
      label: '權限名稱',
      placeholder: '請輸入權限名稱',
    },
    code: {
      label: '權限代碼',
      placeholder: '例如：user:create',
    },
    type: {
      label: '類型',
      menu: '選單',
      button: '按鈕',
      api: 'API',
    },
    parentId: {
      label: '父層ID',
      placeholder: '留空為頂層',
    },
  },
  modal: {
    createTitle: '新增權限',
  },
  message: {
    createSuccess: '建立成功',
    createFailed: '建立失敗',
    emptyData: '暫無權限資料',
  },
} as const;

export default permissions;
