export const auth = {
  signIn: {
    title: "Niko Admin",
    titlePart1: "Niko",
    titlePart2: "Admin",
    subtitle: "사용자 이름과 비밀번호를 입력하여 로그인하세요",
    divider: "계정으로 로그인",
    username: {
      label: "사용자 이름",
      placeholder: "사용자 이름 입력",
      required: "사용자 이름은 필수입니다",
    },
    password: {
      label: "비밀번호",
      placeholder: "비밀번호 입력",
      required: "비밀번호는 필수입니다",
    },
    rememberMe: "로그인 상태 유지",
    forgotPassword: "비밀번호를 잊으셨나요?",
    submit: "로그인",
    loading: "로그인 중...",
  },
  message: {
    signInSuccess: "성공적으로 로그인되었습니다",
    signInFailed: "로그인에 실패했습니다",
    signOutSuccess: "성공적으로 로그아웃되었습니다",
    invalidCredentials: "사용자 이름 또는 비밀번호가 올바르지 않습니다",
    sessionExpired: "세션이 만료되었습니다. 다시 로그인해 주세요.",
  },
} as const;

export default auth;
