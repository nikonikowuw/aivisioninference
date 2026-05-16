export const auth = {
  signIn: {
    title: 'Niko Admin',
    subtitle: '輸入使用者名稱和密碼登入系統',
    divider: '帳號登入',
    username: {
      label: '使用者名稱',
      placeholder: '請輸入使用者名稱',
      required: '請輸入使用者名稱',
    },
    password: {
      label: '密碼',
      placeholder: '請輸入密碼',
      required: '請輸入密碼',
    },
    rememberMe: '記住我',
    forgotPassword: '忘記密碼？',
    submit: '登入',
    loading: '登入中...',
  },
  message: {
    signInSuccess: '登入成功',
    signInFailed: '登入失敗',
    signOutSuccess: '登出成功',
    invalidCredentials: '使用者名稱或密碼錯誤',
    sessionExpired: '工作階段已過期，請重新登入',
  },
} as const;

export default auth;
