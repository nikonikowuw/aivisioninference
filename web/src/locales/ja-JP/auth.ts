export const auth = {
  signIn: {
    title: 'Niko Admin',
    titlePart1: 'Niko',
    titlePart2: 'Admin',
    subtitle: 'ユーザー名とパスワードを入力してサインインしてください',
    divider: 'アカウントでサインイン',
    username: {
      label: 'ユーザー名',
      placeholder: 'ユーザー名を入力',
      required: 'ユーザー名は必須です',
    },
    password: {
      label: 'パスワード',
      placeholder: 'パスワードを入力',
      required: 'パスワードは必須です',
    },
    rememberMe: 'ログイン状態を保持する',
    forgotPassword: 'パスワードをお忘れですか？',
    submit: 'サインイン',
    loading: 'サインイン中...',
  },
  message: {
    signInSuccess: 'サインインしました',
    signInFailed: 'サインインに失敗しました',
    signOutSuccess: 'サインアウトしました',
    invalidCredentials: 'ユーザー名またはパスワードが正しくありません',
    sessionExpired: 'セッションの期限が切れました。再度サインインしてください。',
  },
} as const;

export default auth;
