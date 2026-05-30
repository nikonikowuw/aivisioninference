// Package i18n provides internationalization support for the niko-admin application.
// It maps business error codes to translated messages in multiple languages.
package i18n

import (
	"fmt"
	"sync"
)

// DefaultLanguage 是 i18n 翻译的默认回退语言。
const DefaultLanguage = "en"

var (
	mu      sync.RWMutex
	storage = map[string]map[int]string{
		"zh": {
			0:     "成功",
			10001: "请求参数错误",
			10002: "不能禁用自己",
			10003: "没有权限操作同级或更高级别的用户",
			10004: "没有权限操作同级或更高级别的角色，也不能设置高于自己权限的角色等级",
			10005: "邮箱已被使用",
			10006: "旧密码错误",
			10008: "开始时间格式错误",
			10009: "结束时间格式错误",
			10010: "开始时间不能晚于结束时间",
			10011: "文件大小超过限制",
			10012: "不支持的文件类型",
			10013: "邮件服务未启用",
			10014: "SMTP测试失败",
			10015: "IMAP测试失败",
			10016: "验证码无效或已过期",
			10017: "CSV内容无效",
			10018: "CSV行数超过限制",
			10019: "CSV至少需要 username、email、display_name、password 四列",
			10020: "状态必须为0或1",
			10021: "CSV表头必须为 username,email,display_name,password,status",
			10022: "CSV中存在重复用户名",
			10023: "密码长度不能少于6位",
			10024: "邮箱格式不正确",
			10042: "请求过于频繁，请稍后再试",
			20001: "未登录",
			20002: "Token已过期",
			20003: "Token无效",
			20004: "Token已被复用，所有设备已强制登出",
			30001: "无权限",
			30002: "请求来源不被允许",
			40001: "资源不存在",
			40002: "反馈不存在",
			50001: "服务器内部错误",
		},
		"en": {
			0:     "Success",
			10001: "Bad request parameters",
			10002: "Cannot disable yourself",
			10003: "No permission to operate on users at or above your level",
			10004: "No permission to operate on roles at or above your level",
			10005: "Email is already taken",
			10006: "Old password is incorrect",
			10008: "Invalid start time format",
			10009: "Invalid end time format",
			10010: "Start time cannot be later than end time",
			10011: "File size exceeds limit",
			10012: "Unsupported file type",
			10013: "Mail service is not enabled",
			10014: "SMTP test failed",
			10015: "IMAP test failed",
			10016: "Invalid or expired verification token",
			10017: "Invalid CSV content",
			10018: "CSV row limit exceeded",
			10019: "CSV requires at least username, email, display_name, and password columns",
			10020: "Status must be 0 or 1",
			10021: "CSV header must be username,email,display_name,password,status",
			10022: "Duplicate username in CSV",
			10023: "Password must be at least 6 characters",
			10024: "Invalid email format",
			10042: "Too many requests, please try again later",
			20001: "Unauthorized",
			20002: "Token expired",
			20003: "Token invalid",
			20004: "Token reused",
			30001: "Forbidden",
			30002: "Origin not allowed",
			40001: "Resource not found",
			40002: "Feedback not found",
			50001: "Internal server error",
		},
		"zh-tw": {
			0:     "成功",
			10001: "請求參數錯誤",
			10002: "不能停用自己",
			10003: "沒有權限操作同級或更高級別的使用者",
			10004: "沒有權限操作同級或更高級別的角色，也不能設定高於自己權限的角色等級",
			10005: "信箱已被使用",
			10006: "舊密碼錯誤",
			10008: "開始時間格式錯誤",
			10009: "結束時間格式錯誤",
			10010: "開始時間不能晚於結束時間",
			10011: "檔案大小超過限制",
			10012: "不支援的檔案類型",
			10013: "郵件服務未啟用",
			10014: "SMTP測試失敗",
			10015: "IMAP測試失敗",
			10016: "驗證碼無效或已過期",
			10017: "CSV內容無效",
			10018: "CSV列數超過限制",
			10019: "CSV至少需要 username、email、display_name、password 四欄",
			10020: "狀態必須為0或1",
			10021: "CSV表頭必須為 username,email,display_name,password,status",
			10022: "CSV中存在重複使用者名稱",
			10023: "密碼長度不能少於6位",
			10024: "信箱格式不正確",
			10042: "請求過於頻繁，請稍後再試",
			20001: "未登入",
			20002: "Token已過期",
			20003: "Token無效",
			20004: "Token已被重複使用，所有裝置已強制登出",
			30001: "無權限",
			30002: "請求來源不被允許",
			40001: "資源不存在",
			40002: "回饋不存在",
			50001: "伺服器內部錯誤",
		},
		"id": {
			0:     "Berhasil",
			10001: "Permintaan buruk",
			10002: "Tidak dapat menonaktifkan diri sendiri",
			10003: "Tidak ada izin untuk mengoperasi pengguna di tingkat yang sama atau lebih tinggi",
			10004: "Tidak ada izin untuk mengoperasi peran di tingkat yang sama atau lebih tinggi",
			10005: "Email sudah digunakan",
			10006: "Kata sandi lama salah",
			10008: "Format waktu mulai tidak valid",
			10009: "Format waktu selesai tidak valid",
			10010: "Waktu mulai tidak boleh setelah waktu selesai",
			10011: "Ukuran file melebihi batas",
			10012: "Tipe file tidak didukung",
			10013: "Layanan email dinonaktifkan",
			10014: "Tes SMTP gagal",
			10015: "Tes IMAP gagal",
			10016: "Token tidak valid atau kedaluwarsa",
			10017: "Konten CSV tidak valid",
			10018: "Jumlah baris CSV melebihi batas",
			10019: "CSV memerlukan minimal kolom username, email, display_name, dan password",
			10020: "Status harus 0 or 1",
			10021: "Header CSV harus username,email,display_name,password,status",
			10022: "Username duplikat di CSV",
			10023: "Kata sandi minimal 6 karakter",
			10024: "Format email tidak valid",
			10042: "Terlalu banyak permintaan, silakan coba lagi nanti",
			20001: "Tidak sah",
			20002: "Token kedaluwarsa",
			20003: "Token tidak valid",
			20004: "Penggunaan kembali token penyegaran terdeteksi",
			30001: "Terlarang",
			30002: "Asal permintaan tidak diizinkan",
			40001: "Sumber daya tidak ditemukan",
			40002: "Umpan balik tidak ditemukan",
			50001: "Kesalahan server internal",
		},
		"ja": {
			0:     "成功",
			10001: "不正なリクエスト",
			10002: "自分自身を无効にすることはできません",
			10003: "同等またはそれ以上のレベルのユーザーを操作する権限がありません",
			10004: "同等またはそれ以上のレベルのロールを操作する権限がありません",
			10005: "このメールアドレスは既に登録されています",
			10006: "旧パスワードが正しくありません",
			10008: "開始時間の形式が正しくありません",
			10009: "終了時間の形式が正しくありません",
			10010: "開始時間は終了時間より前である必要があります",
			10011: "ファイルサイズが制限を超えています",
			10012: "サポートされていないファイル形式です",
			10013: "メールサービスが無効です",
			10014: "SMTPテストに失敗しました",
			10015: "IMAPテストに失敗しました",
			10016: "トークンが無効または期限切れです",
			10017: "CSVの内容が無効です",
			10018: "CSVの行数が制限を超えています",
			10019: "CSVには少なくとも username、email、display_name、password 列が必要です",
			10020: "ステータスは0または1である必要があります",
			10021: "CSVヘッダーは username,email,display_name,password,status である必要があります",
			10022: "CSV内に重複したユーザー名があります",
			10023: "パスワードは6文字以上である必要があります",
			10024: "メールアドレスの形式が正しくありません",
			10042: "リクエストが多すぎます。後でもう一度お試しください",
			20001: "認証されていません",
			20002: "トークンの期限が切れています",
			20003: "トークンが無効です",
			20004: "リフレッシュトークンの再利用が検出されました",
			30001: "アクセスが拒否されました",
			30002: "リクエストのオリジンが許可されていません",
			40001: "リソースが見つかりません",
			40002: "フィードバックが見つかりません",
			50001: "サーバー内部エラー",
		},
		"ko": {
			0:     "성공",
			10001: "잘못된 요청",
			10002: "자기 자신을 비활성화할 수 없습니다",
			10003: "동일하거나 높은 레벨의 사용자를 조작할 권한이 없습니다",
			10004: "동일하거나 높은 레벨의 역할을 조작할 권한이 없거나, 자신의 레벨보다 높은 역할 레벨을 설정할 수 없습니다",
			10005: "이미 사용 중인 이메일입니다",
			10006: "기존 비밀번호가 올바르지 않습니다",
			10008: "시작 시간 형식이 올바르지 않습니다",
			10009: "종료 시간 형식이 올바르지 않습니다",
			10010: "시작 시간은 종료 시간보다 빨라야 합니다",
			10011: "파일 크기가 제한을 초과했습니다",
			10012: "지원하지 않는 파일 형식입니다",
			10013: "메일 서비스가 활성화되지 않았습니다",
			10014: "SMTP 테스트 실패",
			10015: "IMAP 테스트 실패",
			10016: "유효하지 않거나 만료된 토큰입니다",
			10017: "CSV 내용이 올바르지 않습니다",
			10018: "CSV 행 수가 제한을 초과했습니다",
			10019: "CSV에는 최소 username, email, display_name, password 열이 필요합니다",
			10020: "상태는 0 또는 1이어야 합니다",
			10021: "CSV 헤더는 username,email,display_name,password,status 여야 합니다",
			10022: "CSV에 중복된 사용자 이름이 있습니다",
			10023: "비밀번호는 6자 이상이어야 합니다",
			10024: "이메일 형식이 올바르지 않습니다",
			10042: "요청이 너무 많습니다. 나중에 다시 시도해 주세요",
			20001: "인증되지 않았습니다",
			20002: "토큰이 만료되었습니다",
			20003: "유효하지 않은 토큰입니다",
			20004: "리프레시 토큰 재사용이 감지되었습니다",
			30001: "접근 권한이 없습니다",
			30002: "허용되지 않은 요청 오리진입니다",
			40001: "리소스를 찾을 수 없습니다",
			40002: "피드백을 찾을 수 없습니다",
			50001: "서버 내부 오류",
		},
	}
)

// Translate returns the translated message for the given error code and language.
// If lang is empty, it falls back to "en". If the language or code is not found,
// it falls back to English. If still not found, returns "unknown error".
func Translate(lang string, code int) string {
	if lang == "" {
		lang = DefaultLanguage
	}

	mu.RLock()
	defer mu.RUnlock()

	// Try the requested language first.
	if msgs, ok := storage[lang]; ok {
		if msg, ok := msgs[code]; ok {
			return msg
		}
	}

	// Fallback to English.
	if msgs, ok := storage[DefaultLanguage]; ok {
		if msg, ok := msgs[code]; ok {
			return msg
		}
	}

	return fmt.Sprintf("unknown error (code=%d)", code)
}

// RegisterMessages adds or overwrites translations for the given language.
// It is safe to call concurrently.
func RegisterMessages(lang string, msgs map[int]string) {
	mu.Lock()
	defer mu.Unlock()

	if storage[lang] == nil {
		storage[lang] = make(map[int]string)
	}
	for k, v := range msgs {
		storage[lang][k] = v
	}
}

// Languages returns the list of currently registered languages.
func Languages() []string {
	mu.RLock()
	defer mu.RUnlock()

	langs := make([]string, 0, len(storage))
	for lang := range storage {
		langs = append(langs, lang)
	}
	return langs
}

// HasLanguage reports whether translations exist for the given language code.
func HasLanguage(lang string) bool {
	mu.RLock()
	defer mu.RUnlock()

	_, ok := storage[lang]
	return ok
}
