export const users = {
  title: "用户管理",
  table: {
    columns: {
      username: "用户名",
      displayName: "显示名称",
      email: "邮箱",
      roles: "角色",
    },
  },
  form: {
    username: {
      label: "用户名",
      placeholder: "请输入用户名",
    },
    displayName: {
      label: "显示名称",
      placeholder: "请输入显示名称",
    },
    email: {
      label: "邮箱",
      placeholder: "请输入邮箱",
    },
    password: {
      label: "密码",
      placeholder: "请输入密码",
      hint: "留空不修改",
    },
    status: {
      label: "状态",
      active: "正常",
      inactive: "禁用",
      readOnlyHint: "编辑时请在列表状态开关中启用或禁用用户。",
    },
  },
  modal: {
    createTitle: "新增用户",
    editTitle: "编辑用户",
  },
  message: {
    resetPasswordSuccess: "密码重置成功",
    deleteConfirm: "确定删除该用户？",
    disableConfirm: "确定禁用该用户？",
    enableConfirm: "确定启用该用户？",
    batchDeleteConfirm: "确定删除选中的 {{count}} 个用户？",
    batchEnableConfirm: "确定启用选中的 {{count}} 个用户？",
    batchDisableConfirm: "确定禁用选中的 {{count}} 个用户？",
  },
  actions: {
    create: '新增用户',
    edit: '编辑',
    delete: '删除',
    enable: '启用',
    disable: '禁用',
  },
  batch: {
    selected: "已选择 {{count}} 个用户",
    selectPage: "选择当前页用户",
    selectRow: "选择用户 {{username}}",
    delete: "批量删除",
    enable: "批量启用",
    disable: "批量禁用",
  },
} as const;

export default users;
