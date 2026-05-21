export const auth = {
  hero: {
    welcome: '欢迎来到',
    title: 'Niko Admin',
    subtitle: '一个为开发者和企业构建的现代化、强大且可扩展的企业管理系统。',
    features: {
      modern: {
        title: '现代界面',
        desc: '干净直观的控制台',
      },
      secure: {
        title: '安全可靠',
        desc: '企业级基于角色的访问控制',
      },
      fast: {
        title: '极致快速',
        desc: '为高性能进行深度优化',
      },
    },
  },
  signIn: {
    title: 'Niko Admin',
    subtitle: '输入用户名和密码登录系统',
    divider: '账号登录',
    username: {
      label: '用户名',
      placeholder: '请输入用户名',
      required: '请输入用户名',
    },
    password: {
      label: '密码',
      placeholder: '请输入密码',
      required: '请输入密码',
    },
    rememberMe: '记住我',
    forgotPassword: '忘记密码？',
    submit: '登录',
    loading: '登录中...',
  },
  message: {
    signInSuccess: '登录成功',
    signInFailed: '登录失败',
    signOutSuccess: '退出成功',
    invalidCredentials: '用户名或密码错误',
    sessionExpired: '会话已过期，请重新登录',
  },
} as const;

export default auth;
