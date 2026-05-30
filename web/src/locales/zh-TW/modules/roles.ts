export const roles = {
  title: "角色管理",
  button: {
    assignPermissions: "分配權限",
  },
  table: {
    columns: {
      name: "角色名稱",
      description: "描述",
      permissionCount: "權限數",
    },
  },
  form: {
    name: {
      label: "角色名稱",
      placeholder: "請輸入角色名稱",
    },
    description: {
      label: "描述",
      placeholder: "請輸入描述",
    },
    level: {
      label: "等級",
      placeholder: "等級數值越小權限越高",
      helper: "等級數值越小權限越高；最終權限校驗以後端規則為準",
    },
  },
  modal: {
    createTitle: "新增角色",
    editTitle: "編輯角色",
    assignPermissionsTitle: "分配權限",
  },
  permissions: {
    empty: "暫無權限資料",
  },
  message: {
    assignPermissionsSuccess: "權限分配成功",
    loadPermissionsFailed: "取得權限失敗",
    assignPermissionsFailed: "分配失敗",
    deleteConfirm: "確定刪除該角色？",
    levelInvalidTitle: "等級輸入無效",
    levelInvalidDescription: "請輸入 {{min}} 到 {{max}} 之間的整數等級",
    batchDeleteConfirm: "確定刪除選中的 {{count}} 個角色？",
  },
  batch: {
    delete: "批量刪除",
  },
} as const;

export default roles;
