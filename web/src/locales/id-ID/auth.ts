export const auth = {
  hero: {
    welcome: 'Selamat datang di',
    title: 'Niko Admin',
    subtitle: 'Sistem manajemen perusahaan modern, kuat, dan terukur yang dibangun untuk pengembang dan bisnis.',
    features: {
      modern: {
        title: 'UI Modern',
        desc: 'Dasbor yang bersih dan intuitif',
      },
      secure: {
        title: 'Aman',
        desc: 'Kontrol Akses Berbasis Peran tingkat Perusahaan',
      },
      fast: {
        title: 'Cepat',
        desc: 'Dioptimalkan untuk kinerja tinggi',
      },
    },
  },
  signIn: {
    title: 'Niko Admin',
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
