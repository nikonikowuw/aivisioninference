export const layout = {
  sidebar: {
    dashboard: '대시보드',
    userManagement: '사용자 관리',
    users: '사용자',
    roles: '역할',
    permissions: '권한',
    systemManagement: '시스템 관리',
    files: '파일',
    auditLogs: '감사 로그',
    tasks: '작업',
    noAccess: '접근 권한 없음',
  },
  navbar: {
    profile: '프로필',
    settings: '설정',
    logout: '로그아웃',
    notifications: '알림',
  },
  footer: {
    copyright: '© {{year}} Niko Admin. All rights reserved.',
  },
} as const;

export default layout;
