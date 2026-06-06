import { auth } from "./auth";
import { common } from "./common";
import { layout } from "./layout";
import { menu } from "./menu";
import { auditLogs } from "./modules/audit-logs";
import { brandConfig } from "./modules/brand-config";
import { dashboard } from "./modules/dashboard";
import { feedback } from "./modules/feedback";
import { files } from "./modules/files";
import { mailConfig } from "./modules/mail-config";
import { permissions } from "./modules/permissions";
import { roles } from "./modules/roles";
import { tasks } from "./modules/tasks";
import { users } from "./modules/users";
import { devices } from "./modules/devices";
import { deviceStaging } from "./modules/device-staging";
import media from "./modules/media";
import system from "./modules/system";

export default {
  common,
  layout,
  auth,
  menu,
  "modules/dashboard": dashboard,
  "modules/users": users,
  "modules/roles": roles,
  "modules/permissions": permissions,
  "modules/files": files,
  "modules/audit-logs": auditLogs,
  "modules/tasks": tasks,
  "modules/brand-config": brandConfig,
  "modules/mail-config": mailConfig,
  "modules/feedback": feedback,
  "modules/devices": devices,
  "modules/device-staging": deviceStaging,
  "modules/media": media,
  "modules/system": system,
};
