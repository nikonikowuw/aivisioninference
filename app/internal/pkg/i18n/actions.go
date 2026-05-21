package i18n

// 审计日志操作类型常量
const (
	ActionViewUsers       = "action.view.users"
	ActionCreateUsers     = "action.create.users"
	ActionUpdateUsers     = "action.update.users"
	ActionDeleteUsers     = "action.delete.users"
	ActionViewRoles       = "action.view.roles"
	ActionCreateRoles     = "action.create.roles"
	ActionUpdateRoles     = "action.update.roles"
	ActionDeleteRoles     = "action.delete.roles"
	ActionViewPermissions = "action.view.permissions"
	ActionCreatePermissions = "action.create.permissions"
	ActionUpdatePermissions = "action.update.permissions"
	ActionDeletePermissions = "action.delete.permissions"
	ActionViewFiles       = "action.view.files"
	ActionCreateFiles     = "action.create.files"
	ActionUpdateFiles     = "action.update.files"
	ActionDeleteFiles     = "action.delete.files"
	ActionViewAuditLogs   = "action.view.audit-logs"
	ActionViewTasks       = "action.view.tasks"
	ActionCreateTasks     = "action.create.tasks"
	ActionUpdateTasks     = "action.update.tasks"
	ActionDeleteTasks     = "action.delete.tasks"
	ActionViewDashboard   = "action.view.dashboard"
	ActionLogin           = "action.login"
	ActionLogout          = "action.logout"
	ActionAuth            = "action.auth"
)

// 审计日志操作类型翻译
var actionStorage = map[string]map[string]string{
	"en": {
		"action.view.users":       "View users",
		"action.create.users":     "Create user",
		"action.update.users":     "Update user",
		"action.delete.users":     "Delete user",
		"action.view.roles":       "View roles",
		"action.create.roles":     "Create role",
		"action.update.roles":     "Update role",
		"action.delete.roles":     "Delete role",
		"action.view.permissions": "View permissions",
		"action.create.permissions": "Create permission",
		"action.update.permissions": "Update permission",
		"action.delete.permissions": "Delete permission",
		"action.view.files":       "View files",
		"action.create.files":     "Upload file",
		"action.update.files":     "Update file",
		"action.delete.files":     "Delete file",
		"action.view.audit-logs":  "View audit logs",
		"action.view.tasks":       "View tasks",
		"action.create.tasks":     "Create task",
		"action.update.tasks":     "Update task",
		"action.delete.tasks":     "Delete task",
		"action.view.dashboard":   "View dashboard",
		"action.login":            "User login",
		"action.logout":           "User logout",
		"action.auth":             "Auth operation",
	},
	"zh": {
		"action.view.users":       "查看用户",
		"action.create.users":     "创建用户",
		"action.update.users":     "更新用户",
		"action.delete.users":     "删除用户",
		"action.view.roles":       "查看角色",
		"action.create.roles":     "创建角色",
		"action.update.roles":     "更新角色",
		"action.delete.roles":     "删除角色",
		"action.view.permissions": "查看权限",
		"action.create.permissions": "创建权限",
		"action.update.permissions": "更新权限",
		"action.delete.permissions": "删除权限",
		"action.view.files":       "查看文件",
		"action.create.files":     "上传文件",
		"action.update.files":     "更新文件",
		"action.delete.files":     "删除文件",
		"action.view.audit-logs":  "查看审计日志",
		"action.view.tasks":       "查看任务",
		"action.create.tasks":     "创建任务",
		"action.update.tasks":     "更新任务",
		"action.delete.tasks":     "删除任务",
		"action.view.dashboard":   "查看仪表盘",
		"action.login":            "用户登录",
		"action.logout":           "用户登出",
		"action.auth":             "认证操作",
	},
	"zh-tw": {
		"action.view.users":       "檢視使用者",
		"action.create.users":     "建立使用者",
		"action.update.users":     "更新使用者",
		"action.delete.users":     "刪除使用者",
		"action.view.roles":       "檢視角色",
		"action.create.roles":     "建立角色",
		"action.update.roles":     "更新角色",
		"action.delete.roles":     "刪除角色",
		"action.view.permissions": "檢視權限",
		"action.create.permissions": "建立權限",
		"action.update.permissions": "更新權限",
		"action.delete.permissions": "刪除權限",
		"action.view.files":       "檢視檔案",
		"action.create.files":     "上傳檔案",
		"action.update.files":     "更新檔案",
		"action.delete.files":     "刪除檔案",
		"action.view.audit-logs":  "檢視稽核日誌",
		"action.view.tasks":       "檢視任務",
		"action.create.tasks":     "建立任務",
		"action.update.tasks":     "更新任務",
		"action.delete.tasks":     "刪除任務",
		"action.view.dashboard":   "檢視儀表板",
		"action.login":            "使用者登入",
		"action.logout":           "使用者登出",
		"action.auth":             "認證操作",
	},
	"id": {
		"action.view.users":       "Lihat pengguna",
		"action.create.users":     "Buat pengguna",
		"action.update.users":     "Perbarui pengguna",
		"action.delete.users":     "Hapus pengguna",
		"action.view.roles":       "Lihat peran",
		"action.create.roles":     "Buat peran",
		"action.update.roles":     "Perbarui peran",
		"action.delete.roles":     "Hapus peran",
		"action.view.permissions": "Lihat izin",
		"action.create.permissions": "Buat izin",
		"action.update.permissions": "Perbarui izin",
		"action.delete.permissions": "Hapus izin",
		"action.view.files":       "Lihat file",
		"action.create.files":     "Unggah file",
		"action.update.files":     "Perbarui file",
		"action.delete.files":     "Hapus file",
		"action.view.audit-logs":  "Lihat log audit",
		"action.view.tasks":       "Lihat tugas",
		"action.create.tasks":     "Buat tugas",
		"action.update.tasks":     "Perbarui tugas",
		"action.delete.tasks":     "Hapus tugas",
		"action.view.dashboard":   "Lihat dasbor",
		"action.login":            "Login pengguna",
		"action.logout":           "Logout pengguna",
		"action.auth":             "Operasi autentikasi",
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
