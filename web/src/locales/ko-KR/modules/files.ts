export const files = {
  title: "파일 관리",
  filter: {
    storageTypes: {
      local: "로컬",
      oss: "OSS",
      pg: "PostgreSQL",
    },
  },
  table: {
    columns: {
      name: "파일명",
      type: "유형",
      size: "크기",
      storageType: "저장 유형",
      uploadTime: "업로드 시간",
    },
  },
  upload: {
    uploading: "업로드 중...",
    progress: "업로드 진행률",
  },
  message: {
    deleteConfirm: "이 파일을 삭제하시겠습니까?",
    batchDeleteConfirm: "선택한 {{count}}개 파일을 삭제하시겠습니까?",
    exportFailed: "내보내기 실패",
  },
  batch: {
    delete: "일괄 삭제",
  },
  actions: {
    export: "내보내기",
    upload: "업로드",
  },
  size: {
    bytes: "B",
    kilobytes: "KB",
    megabytes: "MB",
    gigabytes: "GB",
  },
} as const;

export default files;
