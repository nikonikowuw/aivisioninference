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
    noAccess: 'No access',
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
