export const permissions = {
  title: '权限管理',
  button: {
    create: '新增权限',
  },
  actions: {
    edit: '编辑',
    delete: '删除',
  },
  table: {
    totalCount: '共 {{count}} 项',
    columns: {
      name: '权限名',
      code: '权限代码',
      type: '类型',
    },
  },
  form: {
    name: {
      label: '权限名',
      placeholder: '请输入权限名',
    },
    code: {
      label: '权限代码',
      placeholder: '例如：user:create',
    },
    type: {
      label: '类型',
      menu: '菜单',
      button: '按钮',
    },
    parentId: {
      label: '父级权限',
      placeholder: '选择父级权限',
      none: '无（顶级节点）',
    },
  },
  modal: {
    createTitle: '新增权限',
    editTitle: '编辑权限',
    deleteTitle: '删除权限',
  },
  message: {
    createSuccess: '创建成功',
    createFailed: '创建失败',
    updateSuccess: '更新成功',
    updateFailed: '更新失败',
    deleteSuccess: '删除成功',
    deleteFailed: '删除失败',
    deleteConfirm: '确定要删除该权限及其所有子权限吗？',
    emptyData: '暂无权限数据',
  },
  codes: {
    dashboard: {
      view: '查看仪表盘',
    },
    user: {
      list: '用户列表',
      create: '新增用户',
      edit: '编辑用户',
      delete: '删除用户',
      view: '查看用户',
      'reset-password': '重置密码',
      'upload-avatar': '上传头像',
    },
    role: {
      list: '角色列表',
      create: '新增角色',
      edit: '编辑角色',
      delete: '删除角色',
      'assign-permissions': '分配权限',
      'view-permissions': '查看角色权限',
      view: '查看角色',
    },
    permission: {
      list: '权限列表',
      create: '新增权限',
      edit: '编辑权限',
      delete: '删除权限',
      view: '查看权限',
    },
    file: {
      list: '文件列表',
      upload: '上传文件',
      check: '校验文件',
      'upload-progress': '上传进度',
      delete: '删除文件',
      view: '查看文件',
      download: '下载文件',
    },
    audit: {
      view: '查看审计日志',
    },
    'audit-log': {
      view: '查看审计日志',
    },
    task: {
      list: '任务列表',
      create: '新增任务',
      cancel: '取消任务',
      view: '查看任务',
      update: '编辑任务',
      delete: '删除任务',
    },
    'brand-config': {
      view: '查看品牌配置',
      edit: '编辑品牌配置',
      'upload-logo': '上传品牌Logo',
    },
    'mail-config': {
      view: '查看邮件配置',
      edit: '编辑邮件配置',
      'test-smtp': '测试SMTP',
      'test-imap': '测试IMAP',
      'sync-imap': '同步反馈邮件',
      update: '编辑邮件配置',
      test: '测试邮件配置',
      sync: '同步邮件',
    },
    feedback: {
      view: '查看反馈',
      'update-status': '更新反馈状态',
      update: '编辑反馈',
    },
  },
} as const;

export default permissions;
