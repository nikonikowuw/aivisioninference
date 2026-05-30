// Package errors provides unified business error codes and error types
// for the niko-admin application.
package errors

import "fmt"

// Standard business error codes.
const (
	Success = 0

	// Client errors (1xxxx).
	ErrBadRequest            = 10001
	ErrCannotDisableSelf     = 10002
	ErrHierarchyLevelUser    = 10003
	ErrHierarchyLevelRole    = 10004
	ErrEmailTaken            = 10005
	ErrOldPasswordWrong      = 10006
	ErrStartTimeFormat       = 10008
	ErrEndTimeFormat         = 10009
	ErrTimeRangeOrder        = 10010
	ErrFileTooLarge          = 10011
	ErrFileInvalidType       = 10012
	ErrMailNotEnabled        = 10013
	ErrSMTPTestFailed        = 10014
	ErrIMAPTestFailed        = 10015
	ErrTokenInvalidOrExpired = 10016
	ErrCSVInvalidContent     = 10017
	ErrCSVRowLimitExceeded   = 10018
	ErrCSVColumnRequired     = 10019
	ErrCSVStatusInvalid      = 10020
	ErrCSVHeaderInvalid      = 10021
	ErrCSVDuplicateUsername  = 10022
	ErrCSVWeakPassword       = 10023
	ErrCSVInvalidEmail       = 10024

	// Auth errors (2xxxx).
	ErrUnauthorized      = 20001
	ErrTokenExpired      = 20002
	ErrTokenInvalid      = 20003
	ErrRefreshTokenReuse = 20004

	// Forbidden / CORS (3xxxx).
	ErrForbidden        = 30001
	ErrOriginNotAllowed = 30002

	// Not found (4xxxx).
	ErrNotFound         = 40001
	ErrFeedbackNotFound = 40002

	// Server errors (5xxxx).
	ErrInternal = 50001
)

// Standard error messages keyed by language and code.
var i18nMessages = map[string]map[int]string{
	"zh": {
		Success:                  "成功",
		ErrBadRequest:            "请求参数错误",
		ErrCannotDisableSelf:     "不能禁用自己",
		ErrHierarchyLevelUser:    "没有权限操作同级或更高级别的用户",
		ErrHierarchyLevelRole:    "没有权限操作同级或更高级别的角色，也不能设置高于自己权限的角色等级",
		ErrEmailTaken:            "邮箱已被使用",
		ErrOldPasswordWrong:      "旧密码错误",
		ErrStartTimeFormat:       "开始时间格式错误",
		ErrEndTimeFormat:         "结束时间格式错误",
		ErrTimeRangeOrder:        "开始时间不能晚于结束时间",
		ErrFileTooLarge:          "文件大小超过限制",
		ErrFileInvalidType:       "不支持的文件类型",
		ErrMailNotEnabled:        "邮件服务未启用",
		ErrSMTPTestFailed:        "SMTP测试失败",
		ErrIMAPTestFailed:        "IMAP测试失败",
		ErrTokenInvalidOrExpired: "验证码无效或已过期",
		ErrCSVInvalidContent:     "CSV内容无效",
		ErrCSVRowLimitExceeded:   "CSV行数超过限制",
		ErrCSVColumnRequired:     "CSV至少需要 username、email、display_name、password 四列",
		ErrCSVStatusInvalid:      "状态必须为0或1",
		ErrCSVHeaderInvalid:      "CSV表头必须为 username,email,display_name,password,status",
		ErrCSVDuplicateUsername:  "CSV中存在重复用户名",
		ErrCSVWeakPassword:       "密码长度不能少于6位",
		ErrCSVInvalidEmail:       "邮箱格式不正确",
		ErrUnauthorized:          "未登录",
		ErrTokenExpired:          "Token已过期",
		ErrTokenInvalid:          "Token无效",
		ErrRefreshTokenReuse:     "Token已被复用",
		ErrForbidden:             "无权限",
		ErrOriginNotAllowed:      "请求来源不被允许",
		ErrNotFound:              "资源不存在",
		ErrFeedbackNotFound:      "反馈不存在",
		ErrInternal:              "服务器内部错误",
	},
	"en": {
		Success:                  "Success",
		ErrBadRequest:            "Bad request parameters",
		ErrCannotDisableSelf:     "Cannot disable yourself",
		ErrHierarchyLevelUser:    "No permission to operate on users at or above your level",
		ErrHierarchyLevelRole:    "No permission to operate on roles at or above your level",
		ErrEmailTaken:            "Email is already taken",
		ErrOldPasswordWrong:      "Old password is incorrect",
		ErrStartTimeFormat:       "Invalid start time format",
		ErrEndTimeFormat:         "Invalid end time format",
		ErrTimeRangeOrder:        "Start time cannot be later than end time",
		ErrFileTooLarge:          "File size exceeds limit",
		ErrFileInvalidType:       "Unsupported file type",
		ErrMailNotEnabled:        "Mail service is not enabled",
		ErrSMTPTestFailed:        "SMTP test failed",
		ErrIMAPTestFailed:        "IMAP test failed",
		ErrTokenInvalidOrExpired: "Invalid or expired verification token",
		ErrCSVInvalidContent:     "Invalid CSV content",
		ErrCSVRowLimitExceeded:   "CSV row limit exceeded",
		ErrCSVColumnRequired:     "CSV requires at least username, email, display_name, and password columns",
		ErrCSVStatusInvalid:      "Status must be 0 or 1",
		ErrCSVHeaderInvalid:      "CSV header must be username,email,display_name,password,status",
		ErrCSVDuplicateUsername:  "Duplicate username in CSV",
		ErrCSVWeakPassword:       "Password must be at least 6 characters",
		ErrCSVInvalidEmail:       "Invalid email format",
		ErrUnauthorized:          "Unauthorized",
		ErrTokenExpired:          "Token expired",
		ErrTokenInvalid:          "Token invalid",
		ErrRefreshTokenReuse:     "Token reused",
		ErrForbidden:             "Forbidden",
		ErrOriginNotAllowed:      "Origin not allowed",
		ErrNotFound:              "Resource not found",
		ErrFeedbackNotFound:      "Feedback not found",
		ErrInternal:              "Internal server error",
	},
	"zh-tw": {
		Success:                  "成功",
		ErrBadRequest:            "請求參數錯誤",
		ErrCannotDisableSelf:     "不能停用自己",
		ErrHierarchyLevelUser:    "沒有權限操作同級或更高級別的使用者",
		ErrHierarchyLevelRole:    "沒有權限操作同級或更高級別的角色，也不能設定高於自己權限的角色等級",
		ErrEmailTaken:            "信箱已被使用",
		ErrOldPasswordWrong:      "舊密碼錯誤",
		ErrStartTimeFormat:       "開始時間格式錯誤",
		ErrEndTimeFormat:         "結束時間格式錯誤",
		ErrTimeRangeOrder:        "開始時間不能晚於結束時間",
		ErrFileTooLarge:          "檔案大小超過限制",
		ErrFileInvalidType:       "不支援的檔案類型",
		ErrMailNotEnabled:        "郵件服務未啟用",
		ErrSMTPTestFailed:        "SMTP測試失敗",
		ErrIMAPTestFailed:        "IMAP測試失敗",
		ErrTokenInvalidOrExpired: "驗證碼無效或已過期",
		ErrCSVInvalidContent:     "CSV內容無效",
		ErrCSVRowLimitExceeded:   "CSV列數超過限制",
		ErrCSVColumnRequired:     "CSV至少需要 username、email、display_name、password 四欄",
		ErrCSVStatusInvalid:      "狀態必須為0或1",
		ErrCSVHeaderInvalid:      "CSV表頭必須為 username,email,display_name,password,status",
		ErrCSVDuplicateUsername:  "CSV中存在重複使用者名稱",
		ErrCSVWeakPassword:       "密碼長度不能少於6位",
		ErrCSVInvalidEmail:       "信箱格式不正確",
		ErrUnauthorized:          "未登入",
		ErrTokenExpired:          "Token已過期",
		ErrTokenInvalid:          "Token無效",
		ErrRefreshTokenReuse:     "Token已被重複使用",
		ErrForbidden:             "無權限",
		ErrOriginNotAllowed:      "請求來源不被允許",
		ErrNotFound:              "資源不存在",
		ErrFeedbackNotFound:      "回饋不存在",
		ErrInternal:              "伺服器內部錯誤",
	},
	"id": {
		Success:                  "Berhasil",
		ErrBadRequest:            "Permintaan buruk",
		ErrCannotDisableSelf:     "Tidak dapat menonaktifkan diri sendiri",
		ErrHierarchyLevelUser:    "Tidak ada izin untuk mengoperasi pengguna di tingkat yang sama atau lebih tinggi",
		ErrHierarchyLevelRole:    "Tidak ada izin untuk mengoperasi peran di tingkat yang sama atau lebih tinggi",
		ErrEmailTaken:            "Email sudah digunakan",
		ErrOldPasswordWrong:      "Kata sandi lama salah",
		ErrStartTimeFormat:       "Format waktu mulai tidak valid",
		ErrEndTimeFormat:         "Format waktu selesai tidak valid",
		ErrTimeRangeOrder:        "Waktu mulai tidak boleh setelah waktu selesai",
		ErrFileTooLarge:          "Ukuran file melebihi batas",
		ErrFileInvalidType:       "Tipe file tidak didukung",
		ErrMailNotEnabled:        "Layanan email dinonaktifkan",
		ErrSMTPTestFailed:        "Tes SMTP gagal",
		ErrIMAPTestFailed:        "Tes IMAP gagal",
		ErrTokenInvalidOrExpired: "Token tidak valid atau kedaluwarsa",
		ErrCSVInvalidContent:     "Konten CSV tidak valid",
		ErrCSVRowLimitExceeded:   "Jumlah baris CSV melebihi batas",
		ErrCSVColumnRequired:     "CSV memerlukan minimal kolom username, email, display_name, dan password",
		ErrCSVStatusInvalid:      "Status harus 0 atau 1",
		ErrCSVHeaderInvalid:      "Header CSV harus username,email,display_name,password,status",
		ErrCSVDuplicateUsername:  "Username duplikat di CSV",
		ErrCSVWeakPassword:       "Kata sandi minimal 6 karakter",
		ErrCSVInvalidEmail:       "Format email tidak valid",
		ErrUnauthorized:          "Tidak sah",
		ErrTokenExpired:          "Token kedaluwarsa",
		ErrTokenInvalid:          "Token tidak valid",
		ErrRefreshTokenReuse:     "Penggunaan kembali token penyegaran terdeteksi",
		ErrForbidden:             "Terlarang",
		ErrOriginNotAllowed:      "Asal permintaan tidak diizinkan",
		ErrNotFound:              "Sumber daya tidak ditemukan",
		ErrFeedbackNotFound:      "Umpan balik tidak ditemukan",
		ErrInternal:              "Kesalahan server internal",
	},
	"ja": {
		Success:                  "成功",
		ErrBadRequest:            "不正なリクエスト",
		ErrCannotDisableSelf:     "自分自身を無効にすることはできません",
		ErrHierarchyLevelUser:    "同等またはそれ以上のレベルのユーザーを操作する権限がありません",
		ErrHierarchyLevelRole:    "同等またはそれ以上のレベルのロールを操作する権限がありません",
		ErrEmailTaken:            "このメールアドレスは既に使用されています",
		ErrOldPasswordWrong:      "現在のパスワードが正しくありません",
		ErrStartTimeFormat:       "開始時間の形式が不正です",
		ErrEndTimeFormat:         "終了時間の形式が不正です",
		ErrTimeRangeOrder:        "開始時間は終了時間より前である必要があります",
		ErrFileTooLarge:          "ファイルサイズが制限を超えています",
		ErrFileInvalidType:       "サポートされていないファイル形式です",
		ErrMailNotEnabled:        "メールサービスが無効です",
		ErrSMTPTestFailed:        "SMTPテストに失敗しました",
		ErrIMAPTestFailed:        "IMAPテストに失敗しました",
		ErrTokenInvalidOrExpired: "トークンが無効または期限切れです",
		ErrCSVInvalidContent:     "CSVの内容が無効です",
		ErrCSVRowLimitExceeded:   "CSVの行数が制限を超えています",
		ErrCSVColumnRequired:     "CSVには少なくとも username、email、display_name、password 列が必要です",
		ErrCSVStatusInvalid:      "ステータスは0または1である必要があります",
		ErrCSVHeaderInvalid:      "CSVヘッダーは username,email,display_name,password,status である必要があります",
		ErrCSVDuplicateUsername:  "CSV内に重複したユーザー名があります",
		ErrCSVWeakPassword:       "パスワードは6文字以上である必要があります",
		ErrCSVInvalidEmail:       "メールアドレスの形式が正しくありません",
		ErrUnauthorized:          "認証されていません",
		ErrTokenExpired:          "トークンの期限が切れました",
		ErrTokenInvalid:          "無効なトークンです",
		ErrRefreshTokenReuse:     "リフレッシュトークンの再利用が検出されました",
		ErrForbidden:             "アクセス禁止",
		ErrOriginNotAllowed:      "許可されていないリクエスト元です",
		ErrNotFound:              "リソースが見つかりません",
		ErrFeedbackNotFound:      "フィードバックが見つかりません",
		ErrInternal:              "内部サーバーエラー",
	},
	"ko": {
		Success:                  "성공",
		ErrBadRequest:            "잘못된 요청",
		ErrCannotDisableSelf:     "자기 자신을 비활성화할 수 없습니다",
		ErrHierarchyLevelUser:    "동일하거나 더 높은 레벨의 사용자를 조작할 권한이 없습니다",
		ErrHierarchyLevelRole:    "동일하거나 더 높은 레벨의 역할을 조작할 권한이 없습니다",
		ErrEmailTaken:            "이미 사용 중인 이메일입니다",
		ErrOldPasswordWrong:      "현재 비밀번호가 일치하지 않습니다",
		ErrStartTimeFormat:       "잘못된 시작 시간 형식입니다",
		ErrEndTimeFormat:         "잘못된 종료 시간 형식입니다",
		ErrTimeRangeOrder:        "시작 시간은 종료 시간보다 빨라야 합니다",
		ErrFileTooLarge:          "파일 크기가 제한을 초과했습니다",
		ErrFileInvalidType:       "지원되지 않는 파일 형식입니다",
		ErrMailNotEnabled:        "메일 서비스가 활성화되지 않았습니다",
		ErrSMTPTestFailed:        "SMTP 테스트 실패",
		ErrIMAPTestFailed:        "IMAP 테스트 실패",
		ErrTokenInvalidOrExpired: "유효하지 않거나 만료된 토큰입니다",
		ErrCSVInvalidContent:     "CSV 내용이 올바르지 않습니다",
		ErrCSVRowLimitExceeded:   "CSV 행 수가 제한을 초과했습니다",
		ErrCSVColumnRequired:     "CSV에는 최소 username, email, display_name, password 열이 필요합니다",
		ErrCSVStatusInvalid:      "상태는 0 또는 1이어야 합니다",
		ErrCSVHeaderInvalid:      "CSV 헤더는 username,email,display_name,password,status 여야 합니다",
		ErrCSVDuplicateUsername:  "CSV에 중복된 사용자 이름이 있습니다",
		ErrCSVWeakPassword:       "비밀번호는 6자 이상이어야 합니다",
		ErrCSVInvalidEmail:       "이메일 형식이 올바르지 않습니다",
		ErrUnauthorized:          "인증되지 않았습니다",
		ErrTokenExpired:          "토큰이 만료되었습니다",
		ErrTokenInvalid:          "유효하지 않은 토큰입니다",
		ErrRefreshTokenReuse:     "리프레시 토큰 재사용이 감지되었습니다",
		ErrForbidden:             "접근 거부",
		ErrOriginNotAllowed:      "허용되지 않은 요청 출처입니다",
		ErrNotFound:              "리소스를 찾을 수 없습니다",
		ErrFeedbackNotFound:      "피드백을 찾을 수 없습니다",
		ErrInternal:              "내부 서버 오류",
	},
}

const defaultLanguage = "en"

// AppError represents a business-level error with a code and message.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`

	// defaultMessage 标记 Message 是否由错误码默认文案生成，避免响应层通过字符串比较判断。
	defaultMessage bool
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// New creates a new AppError with the given code and message.
// If the message is empty, the default message for the code is used.
func New(code int, msg string) *AppError {
	isDefault := msg == ""
	if isDefault {
		msg = DefaultMessage(code, defaultLanguage)
	}
	return &AppError{Code: code, Message: msg, defaultMessage: isDefault}
}

// IsDefaultMessage reports whether the message was generated from the default code mapping.
func (e *AppError) IsDefaultMessage() bool {
	return e != nil && e.defaultMessage
}

// Newf creates a new AppError with a formatted explicit message.
func Newf(code int, format string, args ...interface{}) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...), defaultMessage: false}
}

// DefaultMessage returns the default message for a business error code in the specified language.
func DefaultMessage(code int, lang string) string {
	if m, ok := i18nMessages[lang]; ok {
		if msg, ok := m[code]; ok {
			return msg
		}
	}
	if m, ok := i18nMessages[defaultLanguage]; ok {
		if msg, ok := m[code]; ok {
			return msg
		}
	}
	return "Unknown error"
}
