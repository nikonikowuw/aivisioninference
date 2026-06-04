export const tasks = {
  title: "작업 관리",
  filter: {
    taskTypes: {
      email: "이메일",
      export: "내보내기",
      import: "가져오기",
      backup: "백업",
    },
  },
  table: {
    columns: {
      type: "유형",
      error: "오류",
    },
    status: {
      pending: "대기 중",
      running: "실행 중",
      completed: "완료",
      failed: "실패",
      cancelled: "취소됨",
    },
  },
  message: {
    cancelled: "작업이 취소되었습니다",
    cancelFailed: "작업 취소에 실패했습니다",
    cancelConfirm: "이 작업을 취소하시겠습니까?",
    batchCancelConfirm: "선택한 {{count}}개 작업을 취소하시겠습니까?",
    confirmCancel: "예, 취소합니다",
  },
  actions: {
    cancel: "작업 취소",
  },
} as const;

export default tasks;
