export const auth = {
  signIn: {
    title: 'Niko Admin',
    titlePart1: 'Niko',
    titlePart2: 'Admin',
    subtitle: 'Masukkan nama pengguna dan kata sandi Anda untuk masuk',
    divider: 'Masuk dengan Akun',
    username: {
      label: 'Nama Pengguna',
      placeholder: 'Masukkan nama pengguna',
      required: 'Nama pengguna wajib diisi',
    },
    password: {
      label: 'Kata Sandi',
      placeholder: 'Masukkan kata sandi',
      required: 'Kata sandi wajib diisi',
    },
    rememberMe: 'Ingat saya',
    forgotPassword: 'Lupa kata sandi?',
    submit: 'Masuk',
    loading: 'Masuk...',
  },
  message: {
    signInSuccess: 'Berhasil masuk',
    signInFailed: 'Gagal masuk',
    signOutSuccess: 'Berhasil keluar',
    invalidCredentials: 'Nama pengguna atau kata sandi tidak valid',
    sessionExpired: 'Sesi berakhir, silakan masuk lagi',
  },
} as const;

export default auth;
