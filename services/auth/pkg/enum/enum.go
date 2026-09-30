package enum

const (
	// Форматы времени
	DateTimeFormat     = "2006-01-02 15:04:05"
	DateFormat         = "2006-01-02"
	DateTimeUserFormat = "02.01.2006 15:04"

	// HTTP
	BearerPrefix    = "Bearer "
	AuthHeaderKey   = "Authorization"
	ContentTypeJSON = "application/json"

	// Контекст
	ContextUserID = "user_id"
	ContextEmail  = "email"

	PurposeRegister = "register"
	PurposeLogin    = "login"

	// SMTP TLS modes
	SMTPTLSStartTLS = "starttls"
	SMTPTLSImplicit = "implicit"
	SMTPTLSNone     = "none"
)
