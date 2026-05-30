export const brandConfig = {
  title: "品牌設定",
  fields: {
    systemName: "系統名稱",
    logo: "品牌 Logo",
  },
  actions: {
    save: "儲存設定",
    uploadLogo: "上傳 Logo",
    remove: "移除",
  },
  message: {
    loadFailed: "載入品牌設定失敗",
    saved: "品牌設定已儲存",
    saveFailed: "儲存品牌設定失敗",
    logoUploaded: "Logo 上傳成功",
    logoUploadFailed: "Logo 上傳失敗",
    logoHint: "支援 JPG、PNG、GIF、WebP，最大 2MB",
  },
} as const;

export default brandConfig;
