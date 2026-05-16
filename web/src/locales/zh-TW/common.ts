export const common = {
  button: {
    submit: '提交',
    cancel: '取消',
    save: '儲存',
    create: '建立',
    delete: '刪除',
    edit: '編輯',
    confirm: '確認',
    close: '關閉',
    back: '返回',
    next: '下一步',
    search: '搜尋',
    refresh: '重新整理',
    export: '匯出',
    import: '匯入',
  },
  status: {
    loading: '載入中...',
    success: '成功',
    error: '錯誤',
    warning: '警告',
    info: '提示',
  },
  message: {
    confirmDelete: '確定要刪除嗎？',
    confirmCancel: '確定要取消嗎？',
    operationSuccess: '操作成功',
    operationFailed: '操作失敗',
    networkError: '網路錯誤，請稍後重試',
    unauthorized: '未授權，請重新登入',
    forbidden: '無權限執行此操作',
    notFound: '請求的資源不存在',
    serverError: '伺服器錯誤',
    loadFailed: '載入失敗',
    requiredFields: '請填寫必填欄位',
  },
  user: {
    defaultName: '使用者',
    logout: '登出',
  },
  pagination: {
    total: '共 {{total}} 筆',
    page: '第 {{page}} 頁',
    pageSize: '每頁 {{size}} 筆',
  },
} as const;

export default common;
