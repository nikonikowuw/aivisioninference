export const auditLogs = {
  title: "稽核日誌",
  filter: {
    results: {
      success: "成功",
      failed: "失敗",
    },
    resourceTypes: {
      user: "使用者",
      role: "角色",
      permission: "權限",
      file: "檔案",
      task: "任務",
      system: "系統",
      feedback: "回饋",
    },
  },
  table: {
    columns: {
      username: "操作者",
      actionType: "操作類型",
      resourceType: "資源類型",
      method: "方法",
      path: "路徑",
      ip: "IP",
      duration: "耗時",
      result: "結果",
    },
    durationMs: "{{value}} ms",
  },
  actionTypes: {
    action: {
      view: {
        users: "檢視使用者",
        roles: "檢視角色",
        permissions: "檢視權限",
        files: "檢視檔案",
        "audit-logs": "檢視稽核日誌",
        tasks: "檢視任務",
        system: "檢視系統配置",
        dashboard: "檢視儀表板",
        feedback: "檢視回饋",
      },
      create: {
        users: "建立使用者",
        roles: "建立角色",
        permissions: "建立權限",
        files: "上傳檔案",
        tasks: "建立任務",
        system: "建立系統配置",
        feedback: "提交回饋",
      },
      update: {
        users: "更新使用者",
        roles: "更新角色",
        permissions: "更新權限",
        files: "更新檔案",
        tasks: "更新任務",
        system: "更新系統配置",
        feedback: "更新回饋狀態",
      },
      delete: {
        users: "刪除使用者",
        roles: "刪除角色",
        permissions: "刪除權限",
        files: "刪除檔案",
        tasks: "刪除任務",
      },
      login: "使用者登入",
      logout: "使用者登出",
      auth: "認證操作",
    },
  },
} as const;

export default auditLogs;
