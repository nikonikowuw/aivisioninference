import {
    List,
    Datagrid,
    TextField,
    Edit,
    Create,
    SimpleForm,
    TextInput,
    NumberInput,
    SelectInput,
} from 'react-admin';

export const RoleList = () => (
    <List>
        <Datagrid rowClick="edit">
            <TextField source="id" />
            <TextField source="name" />
            <TextField source="description" />
            <TextField source="sort_order" />
            <TextField source="status" />
        </Datagrid>
    </List>
);

export const RoleEdit = () => (
    <Edit>
        <SimpleForm>
            <TextInput source="name" />
            <TextInput source="description" />
            <NumberInput source="sort_order" />
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

export const RoleCreate = () => (
    <Create>
        <SimpleForm>
            <TextInput source="name" />
            <TextInput source="description" />
            <NumberInput source="sort_order" />
        </SimpleForm>
    </Create>
);
