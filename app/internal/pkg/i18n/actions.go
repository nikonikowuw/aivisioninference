package i18n

// 审计日志操作类型常量
const (
	ActionViewUsers         = "action.view.users"
	ActionCreateUsers       = "action.create.users"
	ActionUpdateUsers       = "action.update.users"
	ActionDeleteUsers       = "action.delete.users"
	ActionViewRoles         = "action.view.roles"
	ActionCreateRoles       = "action.create.roles"
	ActionUpdateRoles       = "action.update.roles"
	ActionDeleteRoles       = "action.delete.roles"
	ActionViewPermissions   = "action.view.permissions"
	ActionCreatePermissions = "action.create.permissions"
	ActionUpdatePermissions = "action.update.permissions"
	ActionDeletePermissions = "action.delete.permissions"
	ActionViewFiles         = "action.view.files"
	ActionCreateFiles       = "action.create.files"
	ActionUpdateFiles       = "action.update.files"
	ActionDeleteFiles       = "action.delete.files"
	ActionViewAuditLogs     = "action.view.audit-logs"
	ActionViewTasks         = "action.view.tasks"
	ActionCreateTasks       = "action.create.tasks"
	ActionUpdateTasks       = "action.update.tasks"
	ActionDeleteTasks       = "action.delete.tasks"
	ActionViewDashboard     = "action.view.dashboard"
	ActionViewMailConfig    = "action.view.mail-config"
	ActionUpdateMailConfig  = "action.update.mail-config"
	ActionCreateMailConfig  = "action.create.mail-config"
	ActionViewFeedback      = "action.view.feedback"
	ActionUpdateFeedback    = "action.update.feedback"
	ActionLogin             = "action.login"
	ActionLogout            = "action.logout"
	ActionAuth              = "action.auth"
)

// 审计日志操作类型翻译
var actionStorage = map[string]map[string]string{
	"en": {
		"action.view.users":         "View users",
		"action.create.users":       "Create user",
		"action.update.users":       "Update user",
		"action.delete.users":       "Delete user",
		"action.view.roles":         "View roles",
		"action.create.roles":       "Create role",
		"action.update.roles":       "Update role",
		"action.delete.roles":       "Delete role",
		"action.view.permissions":   "View permissions",
		"action.create.permissions": "Create permission",
		"action.update.permissions": "Update permission",
		"action.delete.permissions": "Delete permission",
		"action.view.files":         "View files",
		"action.create.files":       "Upload file",
		"action.update.files":       "Update file",
		"action.delete.files":       "Delete file",
		"action.view.audit-logs":    "View audit logs",
		"action.view.tasks":         "View tasks",
		"action.create.tasks":       "Create task",
		"action.update.tasks":       "Update task",
		"action.delete.tasks":       "Delete task",
		"action.view.dashboard":     "View dashboard",
		"action.view.mail-config":   "View mail config",
		"action.update.mail-config": "Update mail config",
		"action.create.mail-config": "Operate mail config",
		"action.view.feedback":      "View feedback",
		"action.update.feedback":    "Update feedback",
		"action.login":              "User login",
		"action.logout":             "User logout",
		"action.auth":               "Auth operation",
	},
	"zh": {
		"action.view.users":         "查看用户",
		"action.create.users":       "创建用户",
		"action.update.users":       "更新用户",
		"action.delete.users":       "删除用户",
		"action.view.roles":         "查看角色",
		"action.create.roles":       "创建角色",
		"action.update.roles":       "更新角色",
		"action.delete.roles":       "删除角色",
		"action.view.permissions":   "查看权限",
		"action.create.permissions": "创建权限",
		"action.update.permissions": "更新权限",
		"action.delete.permissions": "删除权限",
		"action.view.files":         "查看文件",
		"action.create.files":       "上传文件",
		"action.update.files":       "更新文件",
		"action.delete.files":       "删除文件",
		"action.view.audit-logs":    "查看审计日志",
		"action.view.tasks":         "查看任务",
		"action.create.tasks":       "创建任务",
		"action.update.tasks":       "更新任务",
		"action.delete.tasks":       "删除任务",
		"action.view.dashboard":     "查看仪表盘",
		"action.view.mail-config":   "查看邮件配置",
		"action.update.mail-config": "更新邮件配置",
		"action.create.mail-config": "邮件配置操作",
		"action.view.feedback":      "查看反馈",
		"action.update.feedback":    "更新反馈状态",
		"action.login":              "用户登录",
		"action.logout":             "用户登出",
		"action.auth":               "认证操作",
	},
	"zh-tw": {
		"action.view.users":         "檢視使用者",
		"action.create.users":       "建立使用者",
		"action.update.users":       "更新使用者",
		"action.delete.users":       "刪除使用者",
		"action.view.roles":         "檢視角色",
		"action.create.roles":       "建立角色",
		"action.update.roles":       "更新角色",
		"action.delete.roles":       "刪除角色",
		"action.view.permissions":   "檢視權限",
		"action.create.permissions": "建立權限",
		"action.update.permissions": "更新權限",
		"action.delete.permissions": "刪除權限",
		"action.view.files":         "檢視檔案",
		"action.create.files":       "上傳檔案",
		"action.update.files":       "更新檔案",
		"action.delete.files":       "刪除檔案",
		"action.view.audit-logs":    "檢視稽核日誌",
		"action.view.tasks":         "檢視任務",
		"action.create.tasks":       "建立任務",
		"action.update.tasks":       "更新任務",
		"action.delete.tasks":       "刪除任務",
		"action.view.dashboard":     "檢視儀表板",
		"action.view.mail-config":   "檢視郵件設定",
		"action.update.mail-config": "更新郵件設定",
		"action.create.mail-config": "郵件設定操作",
		"action.view.feedback":      "檢視回饋",
		"action.update.feedback":    "更新回饋狀態",
		"action.login":              "使用者登入",
		"action.logout":             "使用者登出",
		"action.auth":               "認證操作",
	},
	"id": {
		"action.view.users":         "Lihat pengguna",
		"action.create.users":       "Buat pengguna",
		"action.update.users":       "Perbarui pengguna",
		"action.delete.users":       "Hapus pengguna",
		"action.view.roles":         "Lihat peran",
		"action.create.roles":       "Buat peran",
		"action.update.roles":       "Perbarui peran",
		"action.delete.roles":       "Hapus peran",
		"action.view.permissions":   "Lihat izin",
		"action.create.permissions": "Buat izin",
		"action.update.permissions": "Perbarui izin",
		"action.delete.permissions": "Hapus izin",
		"action.view.files":         "Lihat file",
		"action.create.files":       "Unggah file",
		"action.update.files":       "Perbarui file",
		"action.delete.files":       "Hapus file",
		"action.view.audit-logs":    "Lihat log audit",
		"action.view.tasks":         "Lihat tugas",
		"action.create.tasks":       "Buat tugas",
		"action.update.tasks":       "Perbarui tugas",
		"action.delete.tasks":       "Hapus tugas",
		"action.view.dashboard":     "Lihat dasbor",
		"action.view.mail-config":   "Lihat konfigurasi email",
		"action.update.mail-config": "Perbarui konfigurasi email",
		"action.create.mail-config": "Operasi konfigurasi email",
		"action.view.feedback":      "Lihat umpan balik",
		"action.update.feedback":    "Perbarui status umpan balik",
		"action.login":              "Login pengguna",
		"action.logout":             "Logout pengguna",
		"action.auth":               "Operasi autentikasi",
	},
	"ja": {
		"action.view.users":         "ユーザー一覧表示",
		"action.create.users":       "ユーザー作成",
		"action.update.users":       "ユーザー更新",
		"action.delete.users":       "ユーザー削除",
		"action.view.roles":         "ロール一覧表示",
		"action.create.roles":       "ロール作成",
		"action.update.roles":       "ロール更新",
		"action.delete.roles":       "ロール削除",
		"action.view.permissions":   "権限一覧表示",
		"action.create.permissions": "権限作成",
		"action.update.permissions": "権限更新",
		"action.delete.permissions": "権限削除",
		"action.view.files":         "ファイル一覧表示",
		"action.create.files":       "ファイルアップロード",
		"action.update.files":       "ファイル更新",
		"action.delete.files":       "ファイル削除",
		"action.view.audit-logs":    "監査ログ表示",
		"action.view.tasks":         "タスク一覧表示",
		"action.create.tasks":       "タスク作成",
		"action.update.tasks":       "タスク更新",
		"action.delete.tasks":       "タスク削除",
		"action.view.dashboard":     "ダッシュボード表示",
		"action.view.mail-config":   "メール設定表示",
		"action.update.mail-config": "メール設定更新",
		"action.create.mail-config": "メール設定操作",
		"action.view.feedback":      "フィードバック一覧表示",
		"action.update.feedback":    "フィードバックステータス更新",
		"action.login":              "ユーザーログイン",
		"action.logout":             "ユーザーログアウト",
		"action.auth":               "認証操作",
	},
	"ko": {
		"action.view.users":         "사용자 목록 보기",
		"action.create.users":       "사용자 생성",
		"action.update.users":       "사용자 수정",
		"action.delete.users":       "사용자 삭제",
		"action.view.roles":         "역할 목록 보기",
		"action.create.roles":       "역할 생성",
		"action.update.roles":       "역할 수정",
		"action.delete.roles":       "역할 삭제",
		"action.view.permissions":   "권한 목록 보기",
		"action.create.permissions": "권한 생성",
		"action.update.permissions": "권한 수정",
		"action.delete.permissions": "권한 삭제",
		"action.view.files":         "파일 목록 보기",
		"action.create.files":       "파일 업로드",
		"action.update.files":       "파일 수정",
		"action.delete.files":       "파일 삭제",
		"action.view.audit-logs":    "감사 로그 보기",
		"action.view.tasks":         "작업 목록 보기",
		"action.create.tasks":       "작업 생성",
		"action.update.tasks":       "작업 수정",
		"action.delete.tasks":       "작업 삭제",
		"action.view.dashboard":     "대시보드 보기",
		"action.view.mail-config":   "메일 설정 보기",
		"action.update.mail-config": "메일 설정 수정",
		"action.create.mail-config": "메일 설정 작업",
		"action.view.feedback":      "피드백 보기",
		"action.update.feedback":    "피드백 상태 수정",
		"action.login":              "사용자 로그인",
		"action.logout":             "사용자 로그아웃",
		"action.auth":               "인증 작업",
	},
}

// TranslateAction 将审计日志操作 key 翻译为本地化文本。
func TranslateAction(lang, key string) string {
	if lang == "" {
		lang = DefaultLanguage
	}

	mu.RLock()
	defer mu.RUnlock()

	if msgs, ok := actionStorage[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	// 回退到英文
	if msgs, ok := actionStorage[DefaultLanguage]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	return key
}
