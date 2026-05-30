import { ChakraProvider } from '@chakra-ui/react';
import { useState } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import './assets/css/App.css';
import { AuthProvider } from './contexts/AuthContext';
import { BrandProvider } from './contexts/BrandContext';
import AdminLayout from './layouts/admin';
import AuthLayout from './layouts/auth';
import initialTheme from './theme/theme';

export default function Main() {
  const [currentTheme, setCurrentTheme] = useState(initialTheme);
  return (
    <ChakraProvider theme={currentTheme}>
      <BrandProvider>
        <AuthProvider>
          <Routes>
            <Route path="auth/*" element={<AuthLayout />} />
            <Route
              path="admin/*"
              element={
                <AdminLayout theme={currentTheme} setTheme={setCurrentTheme} />
              }
            />
            <Route path="/" element={<Navigate to="/auth/sign-in" replace />} />
          </Routes>
        </AuthProvider>
      </BrandProvider>
    </ChakraProvider>
  );
}
