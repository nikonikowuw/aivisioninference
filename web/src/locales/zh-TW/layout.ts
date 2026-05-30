export const layout = {
  sidebar: {
    dashboard: "儀表板",
    userManagement: "使用者管理",
    users: "使用者",
    roles: "角色",
    permissions: "權限",
    systemManagement: "系統管理",
    files: "檔案",
    auditLogs: "稽核日誌",
    tasks: "任務",
    brandConfig: "品牌設定",
    mailConfig: "郵件設定",
    feedback: "使用者回饋",
    noAccess: "暫無存取權限",
  },
  navbar: {
    profile: "個人資料",
    settings: "設定",
    logout: "登出",
    notifications: "通知",
  },
  footer: {
    copyright: "© {{year}} Niko Admin. All rights reserved.",
  },
} as const;

export default layout;
