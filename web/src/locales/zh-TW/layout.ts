export const layout = {
  sidebar: {
    dashboard: '儀表板',
    userManagement: '使用者管理',
    roleManagement: '角色管理',
    permissionManagement: '權限管理',
    fileManagement: '檔案管理',
    auditLogs: '稽核日誌',
    taskManagement: '任務管理',
  },
  navbar: {
    profile: '個人資料',
    settings: '設定',
    logout: '登出',
    notifications: '通知',
  },
  footer: {
    copyright: '© {{year}} Niko Admin. All rights reserved.',
  },
} as const;

export default layout;
