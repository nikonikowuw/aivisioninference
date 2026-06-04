export const feedback = {
  title: "用户反馈",
  source: {
    user: "站内提交",
    email: "邮件同步",
  },
  status: {
    open: "待处理",
    processing: "处理中",
    resolved: "已解决",
    closed: "已关闭",
  },
  table: {
    source: "来源",
    category: "分类",
    title: "标题",
    content: "内容",
    status: "状态",
    createdAt: "提交时间",
    actions: "操作",
  },
  actions: {
    batchUpdateStatus: "批量修改状态",
    viewDetails: "查看详情",
    copy: "复制",
  },
  detail: {
    title: "反馈详情",
    email: "联系人邮箱",
    updatedAt: "更新时间",
    noEmail: "未提供邮箱",
    copySuccess: "邮箱已复制",
    copyFailed: "复制邮箱失败",
  },
  batch: {
    selected: "已选中 {{count}} 条反馈",
  },
  message: {
    updated: "反馈状态已更新",
    batchDone: "批量更新完成：成功 {{success}} 条，失败 {{failed}} 条",
    batchUpdateConfirm: "确定修改选中的 {{count}} 条反馈的状态？",
    operationFailed: "操作失败",
  },
} as const;

export default feedback;
