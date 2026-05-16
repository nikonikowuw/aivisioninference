export const layout = {
  sidebar: {
    dashboard: 'Dashboard',
    userManagement: 'User Management',
    roleManagement: 'Role Management',
    permissionManagement: 'Permission Management',
    fileManagement: 'File Management',
    auditLogs: 'Audit Logs',
    taskManagement: 'Task Management',
  },
  navbar: {
    profile: 'Profile',
    settings: 'Settings',
    logout: 'Logout',
    notifications: 'Notifications',
  },
  footer: {
    copyright: '© {{year}} Niko Admin. All rights reserved.',
  },
} as const;

export default layout;
