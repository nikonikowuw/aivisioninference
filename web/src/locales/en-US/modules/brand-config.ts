export const brandConfig = {
  title: 'Brand Configuration',
  fields: {
    systemName: 'System Name',
    logo: 'Brand Logo',
  },
  actions: {
    save: 'Save',
    uploadLogo: 'Upload Logo',
    remove: "Remove",
  },
  message: {
    loadFailed: 'Failed to load brand configuration',
    saved: 'Brand configuration saved',
    saveFailed: 'Failed to save brand configuration',
    logoUploaded: 'Logo uploaded',
    logoUploadFailed: 'Failed to upload logo',
    logoHint: 'PNG, JPG or SVG format. Max 2MB. Recommendation: 80x80px square image.',
  },
} as const;

export default brandConfig;
