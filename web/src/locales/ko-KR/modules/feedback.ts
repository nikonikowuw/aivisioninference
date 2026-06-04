export const feedback = {
  title: "사용자 피드백",
  source: {
    user: "인앱",
    email: "이메일",
  },
  status: {
    open: "대기 중",
    processing: "처리 중",
    resolved: "해결됨",
    closed: "닫힘",
  },
  table: {
    source: "출처",
    category: "카테고리",
    title: "제목",
    content: "내용",
    status: "상태",
    createdAt: "생성일",
    actions: "작업",
  },
  actions: {
    batchUpdateStatus: "일괄 상태 업데이트",
    viewDetails: "상세 보기",
    copy: "복사",
  },
  detail: {
    title: "피드백 상세",
    email: "연락처 이메일",
    updatedAt: "업데이트 시간",
    noEmail: "이메일 미등록",
    copySuccess: "이메일 주소가 복사되었습니다",
    copyFailed: "이메일 주소 복사에 실패했습니다",
  },
  batch: {
    selected: "{{count}}개 선택됨",
  },
  message: {
    updated: "피드백 상태가 업데이트되었습니다",
    batchDone: "일괄 업데이트 완료: {{success}}건 성공, {{failed}}건 실패",
    batchUpdateConfirm: "선택한 {{count}}개 피드백의 상태를 업데이트하시겠습니까?",
    operationFailed: "작업에 실패했습니다",
  },
} as const;

export default feedback;
