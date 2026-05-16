export const auth = {
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
