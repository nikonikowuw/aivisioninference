export const brandConfig = {
  title: "Konfigurasi Merek",
  fields: {
    systemName: "Nama Sistem",
    logo: "Logo Merek",
  },
  actions: {
    save: "Simpan",
    uploadLogo: "Unggah Logo",
    remove: "Hapus",
  },
  message: {
    loadFailed: "Gagal memuat konfigurasi merek",
    saved: "Konfigurasi merek disimpan",
    saveFailed: "Gagal menyimpan konfigurasi merek",
    logoUploaded: "Logo berhasil diunggah",
    logoUploadFailed: "Gagal mengunggah logo",
    logoHint: "Mendukung JPG, PNG, GIF, WebP. Maks 2MB.",
  },
} as const;

export default brandConfig;
