import { Card, CardContent, Typography, Grid } from '@mui/material';
import { useGetList, Title } from 'react-admin';

const StatCard = ({
    title,
    resource,
}: {
    title: string;
    resource: string;
}) => {
    const { total, isLoading } = useGetList(resource, {
        pagination: { page: 1, perPage: 1 },
        sort: { field: 'id', order: 'ASC' },
    });
    return (
        <Card>
            <CardContent>
                <Typography variant="h6">{title}</Typography>
                <Typography variant="h3">
                    {isLoading ? '...' : total || 0}
                </Typography>
            </CardContent>
        </Card>
    );
};

export const Dashboard = () => (
    <>
        <Title title="Dashboard" />
        <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid size={{ xs: 12, md: 3 }}>
                <StatCard title="Users" resource="users" />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
                <StatCard title="Roles" resource="roles" />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
                <StatCard title="Files" resource="files" />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
                <StatCard title="Tasks" resource="tasks" />
            </Grid>
        </Grid>
    </>
);
