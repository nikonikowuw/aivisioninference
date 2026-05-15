import {
    List,
    Datagrid,
    TextField,
    NumberField,
    DeleteButton,
    DateField,
} from 'react-admin';

export const FileList = () => (
    <List>
        <Datagrid>
            <TextField source="id" />
            <TextField source="original_name" />
            <TextField source="mime_type" />
            <NumberField source="size" />
            <TextField source="storage_type" />
            <DateField source="created_at" />
            <DeleteButton />
        </Datagrid>
    </List>
);
