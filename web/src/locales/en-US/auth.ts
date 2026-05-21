export const auth = {
  hero: {
    welcome: 'Welcome to',
    title: 'Niko Admin',
    subtitle: 'A modern, powerful, and scalable enterprise management system built for developers and businesses.',
    features: {
      modern: {
        title: 'Modern UI',
        desc: 'Clean and intuitive dashboard',
      },
      secure: {
        title: 'Secure',
        desc: 'Enterprise-grade RBAC',
      },
      fast: {
        title: 'Fast',
        desc: 'Optimized for high performance',
      },
    },
  },
  signIn: {
    title: 'Niko Admin',
    subtitle: 'Enter your username and password to sign in',
    divider: 'Sign In with Account',
    username: {
      label: 'Username',
      placeholder: 'Enter username',
      required: 'Username is required',
    },
    password: {
      label: 'Password',
      placeholder: 'Enter password',
      required: 'Password is required',
    },
    rememberMe: 'Remember me',
    forgotPassword: 'Forgot password?',
    submit: 'Sign In',
    loading: 'Signing in...',
  },
  message: {
    signInSuccess: 'Signed in successfully',
    signInFailed: 'Sign in failed',
    signOutSuccess: 'Signed out successfully',
    invalidCredentials: 'Invalid username or password',
    sessionExpired: 'Session expired, please sign in again',
  },
} as const;

export default auth;
