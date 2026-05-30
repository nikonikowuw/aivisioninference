export const feedback = {
  title: '用户反馈',
  source: {
    user: '站内提交',
    email: '邮件同步',
  },
  status: {
    open: '待处理',
    processing: '处理中',
    resolved: '已解决',
    closed: '已关闭',
  },
  table: {
    source: '来源',
    category: '分类',
    title: '标题',
    content: '内容',
    status: '状态',
    createdAt: '提交时间',
    actions: '操作',
  },
  actions: {
    refresh: '更新',
    batchUpdateStatus: '批量修改状态',
  },
  message: {
    updated: '反馈状态已更新',
    updateFailed: '更新反馈状态失败',
    batchUpdateConfirm: '确定修改选中的 {{count}} 条反馈的状态？',
  },
} as const;

export default feedback;
