import { Admin, Resource, Layout, AppBar, Title, Login } from 'react-admin';
import { authProvider } from './authProvider';
import { dataProvider } from './dataProvider';
import { UserList, UserEdit, UserCreate } from './resources/users';
import { RoleList, RoleEdit, RoleCreate } from './resources/roles';
import { PermissionList, PermissionCreate } from './resources/permissions';
import { FileList } from './resources/files';
import { TaskList, TaskEdit } from './resources/tasks';
import { AuditLogList } from './resources/auditLogs';
import { Dashboard } from './resources/dashboard';

const CustomAppBar = () => (
    <AppBar>
        <Title title="Niko Admin" />
    </AppBar>
);

const CustomLayout = (props: React.ComponentProps<typeof Layout>) => (
    <Layout {...props} appBar={CustomAppBar} />
);

const LoginPage = () => <Login title="Niko Admin" />;

export const App = () => (
    <Admin
        dataProvider={dataProvider}
        authProvider={authProvider}
        dashboard={Dashboard}
        layout={CustomLayout}
        loginPage={LoginPage}
    >
        <Resource
            name="users"
            list={UserList}
            edit={UserEdit}
            create={UserCreate}
            options={{ label: 'Users' }}
        />
        <Resource
            name="roles"
            list={RoleList}
            edit={RoleEdit}
            create={RoleCreate}
            options={{ label: 'Roles' }}
        />
        <Resource
            name="permissions"
            list={PermissionList}
            create={PermissionCreate}
            options={{ label: 'Permissions' }}
        />
        <Resource name="files" list={FileList} options={{ label: 'Files' }} />
        <Resource
            name="tasks"
            list={TaskList}
            edit={TaskEdit}
            options={{ label: 'Tasks' }}
        />
        <Resource
            name="audit-logs"
            list={AuditLogList}
            options={{ label: 'Audit Logs' }}
        />
    </Admin>
);
