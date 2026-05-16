export const users = {
  title: '用户管理',
  button: {
    create: '新增用户',
  },
  table: {
    columns: {
      id: 'ID',
      username: '用户名',
      displayName: '显示名称',
      email: '邮箱',
      status: '状态',
      roles: '角色',
      actions: '操作',
    },
    status: {
      active: '正常',
      inactive: '禁用',
    },
  },
  form: {
    username: {
      label: '用户名',
      placeholder: '请输入用户名',
    },
    displayName: {
      label: '显示名称',
      placeholder: '请输入显示名称',
    },
    email: {
      label: '邮箱',
      placeholder: '请输入邮箱',
    },
    password: {
      label: '密码',
      placeholder: '请输入密码',
      hint: '留空不修改',
    },
    status: {
      label: '状态',
      active: '正常',
      inactive: '禁用',
    },
  },
  modal: {
    createTitle: '新增用户',
    editTitle: '编辑用户',
  },
  message: {
    createSuccess: '创建成功',
    updateSuccess: '更新成功',
    deleteSuccess: '删除成功',
    deleteConfirm: '确定删除该用户？',
    operationFailed: '操作失败',
    deleteFailed: '删除失败',
  },
  actions: {
    edit: '编辑',
    delete: '删除',
  },
} as const;

export default users;
