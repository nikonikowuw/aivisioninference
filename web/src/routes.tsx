import { Icon } from '@chakra-ui/react';
import {
  MdBarChart,
  MdPerson,
  MdHome,
  MdSecurity,
  MdVpnKey,
  MdFolder,
  MdHistory,
  MdAssignment,
} from 'react-icons/md';

import Dashboard from 'views/admin/default';
import Users from 'views/admin/users';
import Roles from 'views/admin/roles';
import Permissions from 'views/admin/permissions';
import Files from 'views/admin/files';
import AuditLogs from 'views/admin/audit-logs';
import Tasks from 'views/admin/tasks';

const routes = [
  {
    name: '仪表盘',
    layout: '/admin',
    path: '/default',
    icon: <Icon as={MdHome} width="20px" height="20px" color="inherit" />,
    component: <Dashboard />,
  },
  {
    name: '用户管理',
    layout: '/admin',
    path: '/users',
    icon: <Icon as={MdPerson} width="20px" height="20px" color="inherit" />,
    component: <Users />,
  },
  {
    name: '角色管理',
    layout: '/admin',
    path: '/roles',
    icon: <Icon as={MdSecurity} width="20px" height="20px" color="inherit" />,
    component: <Roles />,
  },
  {
    name: '权限管理',
    layout: '/admin',
    path: '/permissions',
    icon: <Icon as={MdVpnKey} width="20px" height="20px" color="inherit" />,
    component: <Permissions />,
  },
  {
    name: '文件管理',
    layout: '/admin',
    path: '/files',
    icon: <Icon as={MdFolder} width="20px" height="20px" color="inherit" />,
    component: <Files />,
  },
  {
    name: '审计日志',
    layout: '/admin',
    path: '/audit-logs',
    icon: <Icon as={MdHistory} width="20px" height="20px" color="inherit" />,
    component: <AuditLogs />,
  },
  {
    name: '任务管理',
    layout: '/admin',
    path: '/tasks',
    icon: (
      <Icon as={MdAssignment} width="20px" height="20px" color="inherit" />
    ),
    component: <Tasks />,
  },
];

export default routes;
