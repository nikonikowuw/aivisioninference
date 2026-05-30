import { lazy, Suspense, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Navigate, Route, Routes } from 'react-router-dom';

// Chakra imports
import { Box, useColorModeValue } from '@chakra-ui/react';

// 懒加载认证页面
const SignIn = lazy(() => import('views/auth/signIn'));

// Custom Chakra theme
export default function Auth() {
  const authBg = useColorModeValue('white', 'navy.900');
  const { t } = useTranslation('common');

  useEffect(() => {
    document.documentElement.dir = 'ltr';
  }, []);

  return (
    <Box>
      <Box
          bg={authBg}
          float="right"
          minHeight="100vh"
          height="100%"
          position="relative"
          w="100%"
          transition="all 0.33s cubic-bezier(0.685, 0.0473, 0.346, 1)"
          transitionDuration=".2s, .2s, .35s"
          transitionProperty="top, bottom, width"
          transitionTimingFunction="linear, linear, ease"
        >
          <Box mx="auto" minH="100vh">
            <Routes>
              <Route
                path="/sign-in"
                element={
                  <Suspense fallback={<div>{t('status.loading')}</div>}>
                    <SignIn />
                  </Suspense>
                }
              />
              <Route
                path="/"
                element={<Navigate to="/auth/sign-in" replace />}
              />
            </Routes>
          </Box>
        </Box>
    </Box>
  );
}
