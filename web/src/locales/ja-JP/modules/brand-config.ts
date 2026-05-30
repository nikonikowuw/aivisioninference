export const brandConfig = {
  title: 'ブランド設定',
  fields: {
    systemName: 'システム名',
    logo: "ブランド Logo",
  },
  actions: {
    save: '保存',
    uploadLogo: 'Logo をアップロード',
    remove: '削除',
  },
  message: {
    loadFailed: 'ブランド設定の読み込みに失敗しました',
    saved: 'ブランド設定を保存しました',
    saveFailed: 'ブランド設定の保存に失敗しました',
    logoUploaded: 'Logo をアップロードしました',
    logoUploadFailed: 'Logo のアップロードに失敗しました',
    logoHint: 'JPG、PNG、GIF、WebP に対応。最大 2MB。',
  },
} as const;

export default brandConfig;
