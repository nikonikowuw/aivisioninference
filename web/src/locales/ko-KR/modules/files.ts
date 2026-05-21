export const files = {
  title: '파일 관리',
  button: {
    upload: '파일 업로드',
  },
  filter: {
    storageTypes: {
      local: '로컬',
      oss: 'OSS',
      pg: 'PostgreSQL',
    },
  },
  table: {
    columns: {
      id: 'ID',
      name: '파일명',
      type: '유형',
      size: '크기',
      storageType: '저장 유형',
      uploadTime: '업로드 시간',
      actions: '작업',
    },
  },
  upload: {
    uploading: '업로드 중...',
    progress: '업로드 진행률',
  },
  message: {
    uploadSuccess: '파일이 성공적으로 업로드되었습니다',
    uploadFailed: '업로드 실패',
    deleteSuccess: '파일이 성공적으로 삭제되었습니다',
    deleteFailed: '삭제 실패',
    deleteConfirm: '이 파일을 삭제하시겠습니까?',
  },
  actions: {
    delete: '삭제',
  },
  size: {
    bytes: 'B',
    kilobytes: 'KB',
    megabytes: 'MB',
    gigabytes: 'GB',
  },
} as const;

export default files;
