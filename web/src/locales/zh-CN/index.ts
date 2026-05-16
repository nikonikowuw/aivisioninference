import { common } from './common';
import { layout } from './layout';
import { auth } from './auth';
import { dashboard } from './modules/dashboard';
import { users } from './modules/users';
import { roles } from './modules/roles';
import { permissions } from './modules/permissions';
import { files } from './modules/files';
import { auditLogs } from './modules/audit-logs';
import { tasks } from './modules/tasks';

export default {
  common,
  layout,
  auth,
  'modules/dashboard': dashboard,
  'modules/users': users,
  'modules/roles': roles,
  'modules/permissions': permissions,
  'modules/files': files,
  'modules/audit-logs': auditLogs,
  'modules/tasks': tasks,
};
