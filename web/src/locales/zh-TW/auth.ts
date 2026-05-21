export const auth = {
  hero: {
    welcome: '歡迎來到',
    title: 'Niko Admin',
    subtitle: '一個為開發者和企業構建的現代化、強大且可擴展的企業管理系統。',
    features: {
      modern: {
        title: '現代介面',
        desc: '乾淨直觀的控制台',
      },
      secure: {
        title: '安全可靠',
        desc: '企業級基於角色的訪問控制',
      },
      fast: {
        title: '極致快速',
        desc: '為高性能進行深度優化',
      },
    },
  },
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
