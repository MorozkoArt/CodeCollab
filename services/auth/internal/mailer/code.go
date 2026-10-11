package mailer

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
	"time"

	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

var funcs = map[string]any{"inc": func(i int) int { return i + 1 }}

var (
	htmlTmpl = htmltemplate.Must(htmltemplate.New("code.html.tmpl").Funcs(funcs).ParseFS(templatesFS, "templates/code.html.tmpl"))
	textTmpl = texttemplate.Must(texttemplate.New("code.txt.tmpl").Funcs(funcs).ParseFS(templatesFS, "templates/code.txt.tmpl"))
)

type codeParams struct {
	AppName string
	Purpose string
	Code    string
	TTL     time.Duration
	Now     time.Time
}

type codeContent struct {
	Subject   string
	Preheader string
	Emoji     string
	Heading   string
	Intro     string
	Purpose   string
	Steps     []string
	Ignore    string
}

type codeView struct {
	codeContent
	AppName     string
	Code        string
	TTL         string
	RequestedAt string
	ExpiresAt   string
	Year        int
}

func contentFor(appName, purpose string) codeContent {
	if purpose == enum.PurposeRegister {
		return codeContent{
			Subject:   "Подтверждение регистрации в " + appName,
			Preheader: "Ваш код подтверждения регистрации",
			Emoji:     "✉️",
			Heading:   "Добро пожаловать в " + appName + "!",
			Intro: "Вы начали регистрацию в " + appName + ". Осталось подтвердить, что этот адрес " +
				"электронной почты принадлежит вам, и аккаунт будет активирован.",
			Purpose: "Подтверждение регистрации",
			Steps: []string{
				"Вернитесь на страницу регистрации " + appName + ".",
				"Введите шестизначный код из этого письма.",
				"После подтверждения войдите в аккаунт с вашим паролем: для входа мы отправим ещё один код.",
			},
			Ignore: "Если вы не регистрировались в " + appName + ", просто удалите это письмо. " +
				"Без подтверждения аккаунт не будет активирован.",
		}
	}
	return codeContent{
		Subject:   "Код для входа в " + appName,
		Preheader: "Ваш одноразовый код для входа",
		Emoji:     "🔐",
		Heading:   "Подтвердите вход",
		Intro: "Кто-то ввёл правильный пароль от вашего аккаунта в " + appName + ". " +
			"Чтобы завершить вход, подтвердите его одноразовым кодом.",
		Purpose: "Вход в аккаунт",
		Steps: []string{
			"Вернитесь на страницу входа в " + appName + ".",
			"Введите шестизначный код из этого письма.",
			"Готово: вы войдёте в аккаунт.",
		},
		Ignore: "Если это были не вы, значит, кто-то знает ваш пароль. Код без вас не сработает, " +
			"но пароль лучше сменить как можно скорее.",
	}
}

func renderCode(p codeParams) (subject, text, html string, err error) {
	v := codeView{
		codeContent: contentFor(p.AppName, p.Purpose),
		AppName:     p.AppName,
		Code:        p.Code,
		TTL:         minutes(p.TTL),
		RequestedAt: formatTime(p.Now),
		ExpiresAt:   formatTime(p.Now.Add(p.TTL)),
		Year:        p.Now.Year(),
	}

	var tb, hb bytes.Buffer
	if err = textTmpl.Execute(&tb, v); err != nil {
		return "", "", "", fmt.Errorf("render text: %w", err)
	}
	if err = htmlTmpl.Execute(&hb, v); err != nil {
		return "", "", "", fmt.Errorf("render html: %w", err)
	}
	return v.Subject, tb.String(), hb.String(), nil
}
