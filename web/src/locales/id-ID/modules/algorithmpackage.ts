export const algorithmpackage = {
  "title": "Paket Algoritma",
  "uploadZone": {
    "title": "Klik atau seret paket algoritma ZIP ke area ini untuk mengunggah",
    "hint": "Mendukung paket hingga 1GB. Sistem akan secara otomatis mengemas ulang (ZIP -> TAR) dan menjalani uji mandiri.",
    "onlyZip": "Hanya file ZIP yang didukung.",
    "dropHere": "Letakkan di sini..."
  },
  "uploading": "Mengunggah...",
  "checking": "Uji mandiri...",
  "card": {
    "version": "Versi",
    "domain": "Domain",
    "hardware": "Platform Perangkat Keras",
    "checkStatus": "Status Uji Mandiri",
    "packageSize": "Ukuran Paket",
    "md5": "MD5 Checksum",
    "capabilities": "Kemampuan",
    "details": "Detail",
    "delete": "Hapus Paket",
    "deleteConfirm": "Apakah Anda yakin ingin menghapus paket algoritma ini? Tindakan ini akan menghapus file secara permanen dan tidak dapat dibatalkan.",
    "selfCheckBtn": "Jalankan Uji Mandiri",
    "selfCheckSuccess": "Uji Mandiri Berhasil",
    "selfCheckFailed": "Uji Mandiri Gagal",
    "selfCheckRunning": "Uji Mandiri Berjalan",
    "selfCheckPending": "Uji Mandiri Menunggu",
    "schema": "Skema Hasil",
    "paramsSchema": "Skema Parameter",
    "copied": "Disalin ke clipboard",
    "noSchema": "Tidak ada skema",
    "errorMsg": "Kesalahan Uji Mandiri",
    "empty": "Belum ada paket algoritma. Unggah satu di atas."
  },
  "message": {
    "uploadSuccess": "Paket algoritma berhasil diunggah, uji mandiri dipicu",
    "uploadFailed": "Gagal mengunggah paket",
    "deleteSuccess": "Paket algoritma berhasil dihapus",
    "deleteFailed": "Gagal menghapus paket",
    "selfCheckTriggered": "Perintah uji mandiri berhasil dikirim",
    "selfCheckFailed": "Gagal mengirim perintah uji mandiri",
    "operationFailed": "Operasi gagal"
  },
  "search": {
    "placeholder": "Cari nama algoritma, alias, atau domain..."
  },
  "stats": {
    "total": "Total Paket",
    "passed": "Uji Mandiri Berhasil",
    "failed": "Uji Mandiri Gagal",
    "active": "Aktif"
  }
} as const;

export default algorithmpackage;
