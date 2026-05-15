import {
    List,
    Datagrid,
    TextField,
    Edit,
    SimpleForm,
    TextInput,
    ChipField,
    DateField,
} from 'react-admin';

export const TaskList = () => (
    <List>
        <Datagrid rowClick="edit">
            <TextField source="id" />
            <TextField source="type" />
            <ChipField source="status" />
            <TextField source="retry_count" />
            <TextField source="error_message" />
            <DateField source="created_at" />
        </Datagrid>
    </List>
);

export const TaskEdit = () => (
    <Edit>
        <SimpleForm>
            <TextInput source="type" disabled />
            <TextInput source="status" disabled />
            <TextInput source="payload" disabled multiline />
            <TextInput source="result" disabled multiline />
        </SimpleForm>
    </Edit>
);
