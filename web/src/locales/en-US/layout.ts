export const layout = {
  sidebar: {
    dashboard: 'Dashboard',
    userManagement: 'User Management',
    users: 'Users',
    roles: 'Roles',
    permissions: 'Permissions',
    systemManagement: 'System Management',
    files: 'Files',
    auditLogs: 'Audit Logs',
    tasks: 'Tasks',
    brandConfig: 'Brand Config',
    mailConfig: 'Mail Config',
    feedback: 'Feedback',
    noAccess: 'No access',
  },
  navbar: {
    profile: 'Profile',
    settings: 'Settings',
    logout: 'Logout',
    notifications: 'Notifications',
    pages: 'Pages',
    logoText: 'Niko Admin',
  },
  footer: {
    copyright: '© {{year}} Niko Admin. All rights reserved.',
  },
} as const;

export default layout;
