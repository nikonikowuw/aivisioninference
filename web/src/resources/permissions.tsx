import {
    List,
    Datagrid,
    TextField,
    Create,
    SimpleForm,
    TextInput,
    SelectInput,
    NumberInput,
} from 'react-admin';

export const PermissionList = () => (
    <List>
        <Datagrid>
            <TextField source="id" />
            <TextField source="name" />
            <TextField source="code" />
            <TextField source="path" />
            <TextField source="method" />
            <TextField source="type" />
            <TextField source="sort_order" />
        </Datagrid>
    </List>
);

export const PermissionCreate = () => (
    <Create>
        <SimpleForm>
            <TextInput source="name" />
            <TextInput source="code" />
            <TextInput source="path" />
            <TextInput source="method" />
            <SelectInput
                source="type"
                choices={[
                    { id: 'menu', name: 'Menu' },
                    { id: 'api', name: 'API' },
                ]}
            />
            <NumberInput source="sort_order" />
        </SimpleForm>
    </Create>
);
