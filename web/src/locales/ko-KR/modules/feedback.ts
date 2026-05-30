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
    refresh: "업데이트",
  },
  message: {
    updated: "피드백 상태가 업데이트되었습니다",
  },
} as const;

export default feedback;
