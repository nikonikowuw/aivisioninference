export const brandConfig = {
  title: '브랜드 설정',
  fields: {
    systemName: '시스템 이름',
    logo: "브랜드 Logo",
  },
  actions: {
    save: '저장',
    uploadLogo: 'Logo 업로드',
    remove: '제거',
  },
  message: {
    loadFailed: '브랜드 설정을 불러오지 못했습니다',
    saved: '브랜드 설정이 저장되었습니다',
    saveFailed: '브랜드 설정 저장에 실패했습니다',
    logoUploaded: 'Logo가 업로드되었습니다',
    logoUploadFailed: 'Logo 업로드에 실패했습니다',
    logoHint: 'JPG, PNG, GIF, WebP 지원. 최대 2MB.',
  },
} as const;

export default brandConfig;
