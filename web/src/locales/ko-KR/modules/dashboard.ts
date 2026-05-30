export const dashboard = {
  title: "대시보드",
  stats: {
    totalUsers: "총 사용자",
    totalFiles: "총 파일",
    activeTasks: "활성 작업",
  },
  charts: {
    userStats: "사용자 성장 및 활동",
    userNew: "신규 사용자",
    userActive: "활성 사용자",
  },
  auditLog: {
    title: "최근 활동",
    empty: "최근 활동이 없습니다",
    columns: {
      username: "사용자",
      action: "작업",
      method: "방법",
      time: "시간",
    },
  },
  message: {
    loadFailed: "대시보드 데이터를 불러오지 못했습니다",
  },
} as const;

export default dashboard;
