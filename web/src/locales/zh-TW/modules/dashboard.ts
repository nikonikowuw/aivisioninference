export const dashboard = {
  title: "儀表板",
  stats: {
    totalUsers: "使用者總數",
    totalFiles: "檔案總數",
    activeTasks: "活躍任務",
  },
  charts: {
    userStats: "使用者增長與活躍",
    userNew: "新增使用者",
    userActive: "活躍使用者",
  },
  auditLog: {
    title: "最近活動",
    empty: "暫無最近活動",
    columns: {
      username: "使用者",
      action: "操作",
      method: "方式",
      time: "時間",
    },
  },
  message: {
    loadFailed: "載入儀表板資料失敗",
  },
} as const;

export default dashboard;
