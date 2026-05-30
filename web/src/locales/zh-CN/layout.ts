export const layout = {
  sidebar: {
    dashboard: "仪表盘",
    userManagement: "用户管理",
    users: "用户",
    roles: "角色",
    permissions: "权限",
    systemManagement: "系统管理",
    files: "文件",
    auditLogs: "审计日志",
    tasks: "任务",
    brandConfig: "品牌配置",
    mailConfig: "邮件配置",
    feedback: "用户反馈",
    noAccess: "暂无访问权限",
  },
  navbar: {
    profile: "个人资料",
    settings: "设置",
    logout: "退出登录",
    notifications: "通知",
    pages: "页面",
    logoText: "Niko Admin",
  },
  footer: {
    copyright: "© {{year}} Niko Admin. All rights reserved.",
  },
} as const;

export default layout;
