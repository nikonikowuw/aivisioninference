import { common } from "./common";
import { layout } from "./layout";
import { auth } from "./auth";
import { menu } from "./menu";
import { dashboard } from "./modules/dashboard";
import { users } from "./modules/users";
import { roles } from "./modules/roles";
import { permissions } from "./modules/permissions";
import { files } from "./modules/files";
import { auditLogs } from "./modules/audit-logs";
import { tasks } from "./modules/tasks";
import { brandConfig } from "./modules/brand-config";
import { mailConfig } from "./modules/mail-config";
import { feedback } from "./modules/feedback";
import { devices } from "./modules/devices";
import { deviceStaging } from "./modules/device-staging";
import media from "./modules/media";
import system from "./modules/system";
import { license } from "./modules/license";
import { persons } from "./modules/persons";

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
  "modules/license": license,
  "modules/persons": persons,
};
