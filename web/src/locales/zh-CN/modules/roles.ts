export const roles = {
  title: "角色管理",
  button: {
    assignPermissions: "分配权限",
  },
  table: {
    columns: {
      name: "角色名",
      description: "描述",
      permissionCount: "权限数",
    },
  },
  form: {
    name: {
      label: "角色名",
      placeholder: "请输入角色名",
    },
    description: {
      label: "描述",
      placeholder: "请输入描述",
    },
    level: {
      label: "等级",
      placeholder: "等级数值越小权限越高",
      helper: "等级数值越小权限越高；最终权限校验以后端规则为准",
    },
  },
  modal: {
    createTitle: "新增角色",
    editTitle: "编辑角色",
    assignPermissionsTitle: "分配权限",
  },
  permissions: {
    empty: "暂无权限数据",
  },
  message: {
    assignPermissionsSuccess: "权限分配成功",
    loadPermissionsFailed: "获取权限失败",
    assignPermissionsFailed: "分配失败",
    deleteConfirm: "确定删除该角色？",
    levelInvalidTitle: "等级输入无效",
    levelInvalidDescription: "请输入 {{min}} 到 {{max}} 之间的整数等级",
    batchDeleteConfirm: "确定删除选中的 {{count}} 个角色？",
  },
  batch: {
    selected: "已选中 {{count}} 个角色",
    delete: "批量删除",
  },
} as const;

export default roles;
