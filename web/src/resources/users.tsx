import {
    List,
    Datagrid,
    TextField,
    EmailField,
    Edit,
    Create,
    SimpleForm,
    TextInput,
    SelectInput,
    FunctionField,
} from 'react-admin';

export const UserList = () => (
    <List>
        <Datagrid rowClick="edit">
            <TextField source="id" />
            <TextField source="username" />
            <EmailField source="email" />
            <TextField source="display_name" />
            <FunctionField
                label="Status"
                render={(record: Record<string, unknown>) =>
                    (record as { status?: number }).status === 1 ? 'Active' : 'Disabled'
                }
            />
            <TextField source="created_at" />
        </Datagrid>
    </List>
);

export const UserEdit = () => (
    <Edit>
        <SimpleForm>
            <TextInput source="username" disabled />
            <TextInput source="email" />
            <TextInput source="display_name" />
            <TextInput source="avatar_url" />
            <SelectInput
                source="status"
                choices={[
                    { id: 1, name: 'Active' },
                    { id: 0, name: 'Disabled' },
                ]}
            />
        </SimpleForm>
    </Edit>
);

export const UserCreate = () => (
    <Create>
        <SimpleForm>
            <TextInput source="username" />
            <TextInput source="password" type="password" />
            <TextInput source="email" />
            <TextInput source="display_name" />
        </SimpleForm>
    </Create>
);
