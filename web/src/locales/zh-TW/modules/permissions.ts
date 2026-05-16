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
      label: '父層權限',
      placeholder: '選擇父層權限',
      none: '無（頂層節點）',
    },
  },
  modal: {
    createTitle: '新增權限',
    editTitle: '編輯權限',
    deleteTitle: '刪除權限',
  },
  message: {
    createSuccess: '建立成功',
    createFailed: '建立失敗',
    updateSuccess: '更新成功',
    updateFailed: '更新失敗',
    deleteSuccess: '刪除成功',
    deleteFailed: '刪除失敗',
    deleteConfirm: '確定要刪除此權限及其所有子權限嗎？',
    emptyData: '暫無權限資料',
  },
} as const;

export default permissions;
