export const dashboard = {
  title: 'Dashboard',
  stats: {
    totalUsers: 'Total Users',
    totalFiles: 'Total Files',
    activeTasks: 'Active Tasks',
  },
  charts: {
    userStats: 'User Growth & Activity',
    userNew: 'New Users',
    userActive: 'Active Users',
  },
  auditLog: {
    title: 'Recent Activity',
    empty: 'No recent activity',
    columns: {
      username: 'USER',
      action: 'ACTION',
      method: 'METHOD',
      time: 'TIME',
    },
  },
  message: {
    loadFailed: 'Failed to load dashboard data',
  },
} as const;

export default dashboard;
