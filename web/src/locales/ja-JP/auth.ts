export const auth = {
  hero: {
    welcome: 'ようこそ',
    title: 'Niko Admin',
    subtitle: '開発者や企業向けに構築された、モダンで強力かつスケーラブルな企業管理システム。',
    features: {
      modern: {
        title: 'モダン UI',
        desc: 'クリーンで直感的なダッシュボード',
      },
      secure: {
        title: 'セキュア',
        desc: 'エンタープライズ対応のRBAC',
      },
      fast: {
        title: '高速',
        desc: 'ハイパフォーマンス向けに最適化',
      },
    },
  },
  signIn: {
    title: 'Niko Admin',
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
