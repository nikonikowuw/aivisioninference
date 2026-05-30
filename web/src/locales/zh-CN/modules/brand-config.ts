export const brandConfig = {
  title: '品牌配置',
  fields: {
    systemName: '系统名称',
    logo: '品牌 Logo',
  },
  actions: {
    save: '保存配置',
    uploadLogo: '上传 Logo',
    remove: "移除",
  },
  message: {
    loadFailed: '加载品牌配置失败',
    saved: '品牌配置已保存',
    saveFailed: '保存品牌配置失败',
    logoUploaded: 'Logo 上传成功',
    logoUploadFailed: 'Logo 上传失败',
    logoHint: 'PNG、JPG 或 SVG 格式，最大 2MB。建议：80x80px 正方形图片。',
  },
} as const;

export default brandConfig;
