export const layout = {
  sidebar: {
    dashboard: '仪表盘',
    userManagement: '用户管理',
    roleManagement: '角色管理',
    permissionManagement: '权限管理',
    fileManagement: '文件管理',
    auditLogs: '审计日志',
    taskManagement: '任务管理',
    noAccess: '暂无访问权限',
  },
  navbar: {
    profile: '个人资料',
    settings: '设置',
    logout: '退出登录',
    notifications: '通知',
  },
  footer: {
    copyright: '© {{year}} Niko Admin. All rights reserved.',
  },
} as const;

export default layout;
