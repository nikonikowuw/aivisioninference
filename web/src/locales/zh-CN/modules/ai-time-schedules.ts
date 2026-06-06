export const aiTimeSchedules = {
  title: '时间配置',
  fields: {
    name: '配置名称',
    description: '描述',
    dateRange: '有效日期',
    timeWindows: '时间窗',
    updatedAt: '更新时间',
  },
  actions: {
    create: '新建配置',
    edit: '编辑',
    delete: '删除',
    save: '保存',
  },
  form: {
    namePlaceholder: '例如：工作日白天、全天候监控',
    descriptionPlaceholder: '可选，配置用途说明',
    startDate: '生效起始日期',
    endDate: '生效结束日期',
    timeWindows: '每日时间窗',
    addTimeWindow: '添加时间段',
  },
  message: {
    deleteConfirm: '确认要删除此时间配置吗？已引用此配置的推理任务不受影响。',
    deleteSuccess: '删除成功',
    deleteFailed: '删除失败',
    updateSuccess: '更新成功',
    createSuccess: '创建成功',
    nameRequired: '请输入配置名称',
    startDateRequired: '请选择开始日期',
    endDateRequired: '请选择结束日期',
    dateInvalid: '结束日期不能早于开始日期',
  },
  empty: '暂无时间配置',
} as const;

export default aiTimeSchedules;
