export const feedback = {
  title: '使用者回饋',
  source: {
    user: '站內提交',
    email: '郵件同步',
  },
  status: {
    open: '待處理',
    processing: '處理中',
    resolved: '已解決',
    closed: '已關閉',
  },
  table: {
    source: '來源',
    category: '分類',
    title: '標題',
    content: '內容',
    status: '狀態',
    createdAt: '提交時間',
    actions: '操作',
  },
  actions: {
    refresh: '更新',
  },
  message: {
    updated: '回饋狀態已更新',
    updateFailed: '更新回饋狀態失敗',
  },
} as const;

export default feedback;
