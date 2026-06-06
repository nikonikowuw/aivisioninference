export const aiTimeSchedules = {
  title: '時間配置',
  fields: {
    name: '配置名稱',
    description: '描述',
    dateRange: '有效日期',
    timeWindows: '時間窗',
    updatedAt: '更新時間',
  },
  actions: {
    create: '新建配置',
    edit: '編輯',
    delete: '刪除',
    save: '儲存',
  },
  form: {
    namePlaceholder: '例如：工作日白天、全天候監控',
    descriptionPlaceholder: '可選，配置用途說明',
    startDate: '生效起始日期',
    endDate: '生效結束日期',
    timeWindows: '每日時間窗',
    addTimeWindow: '添加時間段',
  },
  message: {
    deleteConfirm: '確認要刪除此時間配置嗎？已引用此配置的推理任務不受影響。',
    deleteSuccess: '刪除成功',
    deleteFailed: '刪除失敗',
    updateSuccess: '更新成功',
    createSuccess: '建立成功',
    nameRequired: '請輸入配置名稱',
    startDateRequired: '請選擇開始日期',
    endDateRequired: '請選擇結束日期',
    dateInvalid: '結束日期不能早於開始日期',
  },
  empty: '暫無時間配置',
} as const;
export default aiTimeSchedules;
