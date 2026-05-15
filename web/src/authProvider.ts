import type { AuthProvider } from 'react-admin';

const API_URL = 'http://localhost:8080/api/v1';

export const authProvider: AuthProvider = {
    login: async ({ username, password }) => {
        const response = await fetch(`${API_URL}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });
        const data = await response.json();
        if (data.code !== 0) throw new Error(data.message);
        localStorage.setItem('access_token', data.data.access_token);
        localStorage.setItem('user', JSON.stringify(data.data.user));
        return Promise.resolve();
    },
    logout: async () => {
        const token = localStorage.getItem('access_token');
        if (token) {
            await fetch(`${API_URL}/auth/logout`, {
                method: 'POST',
                headers: { Authorization: `Bearer ${token}` },
            });
        }
        localStorage.removeItem('access_token');
        localStorage.removeItem('user');
        return Promise.resolve();
    },
    checkAuth: () => {
        const token = localStorage.getItem('access_token');
        return token ? Promise.resolve() : Promise.reject();
    },
    checkError: async (error) => {
        const status = error?.status;
        if (status === 401 || status === 403) {
            const token = localStorage.getItem('access_token');
            if (token) {
                try {
                    const response = await fetch(`${API_URL}/auth/refresh`, {
                        method: 'POST',
                        credentials: 'include',
                    });
                    const data = await response.json();
                    if (data.code === 0) {
                        localStorage.setItem('access_token', data.data.access_token);
                        return Promise.resolve();
                    }
                } catch {
                    // refresh failed
                }
            }
            localStorage.removeItem('access_token');
            localStorage.removeItem('user');
            return Promise.reject();
        }
        return Promise.resolve();
    },
    getIdentity: () => {
        const user = localStorage.getItem('user');
        if (user) {
            const parsed = JSON.parse(user);
            return Promise.resolve({ id: parsed.id, fullName: parsed.display_name });
        }
        return Promise.reject();
    },
    getPermissions: () => Promise.resolve(undefined),
};
