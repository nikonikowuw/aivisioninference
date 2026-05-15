import type { DataProvider } from 'react-admin';
import { fetchUtils } from 'react-admin';

const API_URL = 'http://localhost:8080/api/v1';

const httpClient = (url: string, options: fetchUtils.Options = {}) => {
    const token = localStorage.getItem('access_token');
    if (!options.headers) options.headers = new Headers();
    if (token) {
        (options.headers as Headers).set('Authorization', `Bearer ${token}`);
    }
    return fetchUtils.fetchJson(url, options);
};

export const dataProvider: DataProvider = {
    getList: async (resource, params) => {
        const page = params.pagination?.page ?? 1;
        const perPage = params.pagination?.perPage ?? 20;
        const field = params.sort?.field ?? 'id';
        const order = params.sort?.order ?? 'DESC';
        const query = new URLSearchParams({
            page: String(page),
            page_size: String(perPage),
            sort: field,
            order,
        });
        Object.entries(params.filter).forEach(([key, value]) => {
            if (value !== undefined && value !== null && value !== '') {
                query.set(key, String(value));
            }
        });
        const url = `${API_URL}/${resource}?${query}`;
        const { json } = await httpClient(url);
        return {
            data: json.data?.list || [],
            total: json.data?.total || 0,
        };
    },

    getOne: async (resource, params) => {
        const { json } = await httpClient(`${API_URL}/${resource}/${params.id}`);
        return { data: json.data };
    },

    create: async (resource, params) => {
        const { json } = await httpClient(`${API_URL}/${resource}`, {
            method: 'POST',
            body: JSON.stringify(params.data),
        });
        return { data: json.data };
    },

    update: async (resource, params) => {
        const { json } = await httpClient(`${API_URL}/${resource}/${params.id}`, {
            method: 'PUT',
            body: JSON.stringify(params.data),
        });
        return { data: json.data };
    },

    delete: async (resource, params) => {
        await httpClient(`${API_URL}/${resource}/${params.id}`, {
            method: 'DELETE',
        });
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        return { data: params.previousData! };
    },

    getMany: async (resource, params) => {
        const results = await Promise.all(
            params.ids.map((id) =>
                httpClient(`${API_URL}/${resource}/${id}`).then((r) => r.json.data)
            )
        );
        return { data: results };
    },

    getManyReference: async (resource, params) => {
        const page = params.pagination?.page ?? 1;
        const perPage = params.pagination?.perPage ?? 20;
        const query = new URLSearchParams({
            page: String(page),
            page_size: String(perPage),
        });
        Object.entries(params.filter).forEach(([key, value]) => {
            if (value !== undefined && value !== null) query.set(key, String(value));
        });
        if (params.target) query.set(params.target, String(params.id));
        const { json } = await httpClient(`${API_URL}/${resource}?${query}`);
        return {
            data: json.data?.list || [],
            total: json.data?.total || 0,
        };
    },

    updateMany: async (resource, params) => {
        await Promise.all(
            params.ids.map((id) =>
                httpClient(`${API_URL}/${resource}/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify(params.data),
                })
            )
        );
        return { data: params.ids };
    },

    deleteMany: async (resource, params) => {
        await Promise.all(
            params.ids.map((id) =>
                httpClient(`${API_URL}/${resource}/${id}`, {
                    method: 'DELETE',
                })
            )
        );
        return { data: params.ids };
    },
};
