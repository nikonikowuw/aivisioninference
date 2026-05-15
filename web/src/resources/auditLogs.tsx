import {
    List,
    Datagrid,
    TextField,
    Filter,
    TextInput,
    DateField,
} from 'react-admin';

const AuditLogFilter = () => (
    <Filter>
        <TextInput source="username" label="Username" />
        <TextInput source="action" label="Action" />
        <TextInput source="resource_type" label="Resource Type" />
        <TextInput
            source="start_time"
            label="Start Time"
            placeholder="2024-01-01T00:00:00Z"
        />
        <TextInput
            source="end_time"
            label="End Time"
            placeholder="2024-12-31T23:59:59Z"
        />
    </Filter>
);

export const AuditLogList = () => (
    <List filters={<AuditLogFilter />}>
        <Datagrid>
            <TextField source="username" />
            <TextField source="action" />
            <TextField source="resource_type" />
            <TextField source="resource_id" />
            <TextField source="request_path" />
            <TextField source="request_method" />
            <TextField source="request_ip" />
            <TextField source="duration_ms" />
            <DateField source="created_at" />
        </Datagrid>
    </List>
);
