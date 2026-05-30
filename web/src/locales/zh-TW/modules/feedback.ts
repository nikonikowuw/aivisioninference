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
    batchUpdateStatus: '批量修改狀態',
    viewDetails: '查看詳情',
    copy: '複製',
  },
  detail: {
    title: '回饋詳情',
    email: '聯絡人郵箱',
    updatedAt: '更新時間',
    noEmail: '未提供郵箱',
    copySuccess: '郵箱已複製',
    copyFailed: '複製郵箱失敗',
  },
  batch: {
    selected: '已選擇 {{count}} 條',
  },
  message: {
    updated: '回饋狀態已更新',
    updateFailed: '更新回饋狀態失敗',
    batchDone: '批量更新完成：成功 {{success}} 條，失敗 {{failed}} 條',
    batchUpdateConfirm: '確定修改選中的 {{count}} 條回饋的狀態？',
    operationFailed: '操作失敗',
  },
} as const;

export default feedback;
