export const layout = {
  sidebar: {
    dashboard: "Dasbor",
    userManagement: "Manajemen Pengguna",
    users: "Pengguna",
    roles: "Peran",
    permissions: "Izin",
    systemManagement: "Manajemen Sistem",
    files: "File",
    auditLogs: "Log Audit",
    tasks: "Tugas",
    brandConfig: "Konfigurasi Merek",
    mailConfig: "Konfigurasi Email",
    feedback: "Umpan Balik",
    noAccess: "Tidak ada akses",
  },
  navbar: {
    profile: "Profil",
    settings: "Pengaturan",
    logout: "Keluar",
    notifications: "Notifikasi",
  },
  footer: {
    copyright:
      "© {{year}} Niko Admin. Seluruh hak cipta dilindungi undang-undang.",
  },
} as const;

export default layout;
