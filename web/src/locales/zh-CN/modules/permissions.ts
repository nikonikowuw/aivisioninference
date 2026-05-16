export const permissions = {
  title: '权限管理',
  button: {
    create: '新增权限',
  },
  table: {
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
      api: 'API',
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
} as const;

export default permissions;
