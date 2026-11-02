package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

// ==================== Константы ====================

const (
	sessionCookie = "session_id"
	csrfCookie    = "csrf_token"
	sessionTTL    = 24 * time.Hour
	pbkdf2Iter    = 100_000
	saltLen       = 16
	keyLen        = 32
)

// ==================== Модель пользователя ====================

type User struct {
	Username     string
	PasswordHash string // формат: iterations$salt_hex$hash_hex
	CreatedAt    time.Time
}

type UserStore struct {
	mu    sync.RWMutex
	users map[string]*User // username -> User
}

var users = &UserStore{users: make(map[string]*User)}

func (s *UserStore) Create(username, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(username)
	if _, exists := s.users[key]; exists {
		return fmt.Errorf("пользователь уже существует")
	}
	s.users[key] = &User{
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	return nil
}

func (s *UserStore) Get(username string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[strings.ToLower(username)]
	return u, ok
}

// ==================== Хеширование паролей (stdlib) ====================

// PBKDF2 на базе crypto/sha256 — без внешних зависимостей
func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	prf := sha256.New
	hashLen := prf().Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var dk []byte
	var block [4]byte

	for i := 1; i <= numBlocks; i++ {
		block[0] = byte(i >> 24)
		block[1] = byte(i >> 16)
		block[2] = byte(i >> 8)
		block[3] = byte(i)

		mac := prf()
		mac.Write(password)
		mac.Write(salt)
		mac.Write(block[:])
		u := mac.Sum(nil)

		t := make([]byte, len(u))
		copy(t, u)

		for j := 1; j < iter; j++ {
			mac := prf()
			mac.Write(u)
			u = mac.Sum(nil)
			for k := range t {
				t[k] ^= u[k]
			}
		}
		dk = append(dk, t...)
	}
	return dk[:keyLen]
}

func hashPassword(password string) string {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		log.Fatalf("Ошибка соли: %v", err)
	}
	dk := pbkdf2SHA256([]byte(password), salt, pbkdf2Iter, keyLen)
	return fmt.Sprintf("%d$%s$%s", pbkdf2Iter, hex.EncodeToString(salt), hex.EncodeToString(dk))
}

func verifyPassword(password, stored string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 3 {
		return false
	}
	var iter int
	if _, err := fmt.Sscanf(parts[0], "%d", &iter); err != nil {
		return false
	}
	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	expected, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}
	actual := pbkdf2SHA256([]byte(password), salt, iter, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// ==================== Сессии ====================

type Session struct {
	Username string
	Expires  time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

var sessions = &SessionStore{sessions: make(map[string]*Session)}

func (s *SessionStore) Create(username string) string {
	token := randomToken(32)
	s.mu.Lock()
	s.sessions[token] = &Session{Username: username, Expires: time.Now().Add(sessionTTL)}
	s.mu.Unlock()
	return token
}

func (s *SessionStore) Get(token string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.Expires) {
		return nil, false
	}
	return sess, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *SessionStore) Cleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			s.mu.Lock()
			for k, v := range s.sessions {
				if time.Now().After(v.Expires) {
					delete(s.sessions, k)
				}
			}
			s.mu.Unlock()
		}
	}()
}

// ==================== Утилиты ====================

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Ошибка токена: %v", err)
	}
	return hex.EncodeToString(b)
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ==================== Валидация ====================

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

func validateRegistration(username, password, password2 string) string {
	if !usernameRe.MatchString(username) {
		return "Логин: 3-20 символов, только латиница, цифры и _"
	}
	if len(password) < 8 {
		return "Пароль должен быть не короче 8 символов"
	}
	if len(password) > 128 {
		return "Пароль слишком длинный"
	}
	if password != password2 {
		return "Пароли не совпадают"
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "Пароль должен содержать буквы и цифры"
	}
	return ""
}

// ==================== Шаблоны ====================

const registerTemplate = `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Регистрация</title>
    <style>
        body { font-family: sans-serif; background: #f4f4f4; display: flex;
               justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
        .card { background: white; padding: 32px; border-radius: 10px;
                box-shadow: 0 2px 12px rgba(0,0,0,0.1); width: 340px; }
        h1 { margin: 0 0 20px; font-size: 22px; text-align: center; }
        label { display: block; margin-bottom: 6px; font-size: 14px; }
        input { width: 100%; padding: 10px; box-sizing: border-box;
                border: 1px solid #ccc; border-radius: 6px; margin-bottom: 16px; font-size: 14px; }
        button { width: 100%; padding: 10px; background: #16a34a; color: white;
                 border: none; border-radius: 6px; font-size: 15px; cursor: pointer; }
        button:hover { background: #15803d; }
        .error { color: #dc2626; font-size: 14px; margin-bottom: 14px; text-align: center; }
        .link { text-align: center; margin-top: 16px; font-size: 14px; }
        .link a { color: #2563eb; text-decoration: none; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Регистрация</h1>
        {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
        <form method="POST" action="/register">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <label for="username">Логин</label>
            <input id="username" name="username" type="text" required autofocus value="{{.Username}}">
            <label for="password">Пароль</label>
            <input id="password" name="password" type="password" required>
            <label for="password2">Повторите пароль</label>
            <input id="password2" name="password2" type="password" required>
            <button type="submit">Зарегистрироваться</button>
        </form>
        <div class="link">Уже есть аккаунт? <a href="/login">Войти</a></div>
    </div>
</body>
</html>
`

const loginTemplate = `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Вход</title>
    <style>
        body { font-family: sans-serif; background: #f4f4f4; display: flex;
               justify-content: center; align-items: center; height: 100vh; margin: 0; }
        .card { background: white; padding: 32px; border-radius: 10px;
                box-shadow: 0 2px 12px rgba(0,0,0,0.1); width: 320px; }
        h1 { margin: 0 0 20px; font-size: 22px; text-align: center; }
        label { display: block; margin-bottom: 6px; font-size: 14px; }
        input { width: 100%; padding: 10px; box-sizing: border-box;
                border: 1px solid #ccc; border-radius: 6px; margin-bottom: 16px; font-size: 14px; }
        button { width: 100%; padding: 10px; background: #2563eb; color: white;
                 border: none; border-radius: 6px; font-size: 15px; cursor: pointer; }
        button:hover { background: #1d4ed8; }
        .error { color: #dc2626; font-size: 14px; margin-bottom: 14px; text-align: center; }
        .link { text-align: center; margin-top: 16px; font-size: 14px; }
        .link a { color: #2563eb; text-decoration: none; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Вход</h1>
        {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
        <form method="POST" action="/login">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
            <label for="username">Логин</label>
            <input id="username" name="username" type="text" required autofocus>
            <label for="password">Пароль</label>
            <input id="password" name="password" type="password" required>
            <button type="submit">Войти</button>
        </form>
        <div class="link">Нет аккаунта? <a href="/register">Зарегистрироваться</a></div>
    </div>
</body>
</html>
`

const homeTemplate = `
<!DOCTYPE html>
<html lang="ru">
<head><meta charset="UTF-8"><title>Главная</title></head>
<body style="font-family: sans-serif; padding: 40px;">
    <h1>Привет, {{.Username}}!</h1>
    <p>Вы успешно авторизованы.</p>
    <form method="POST" action="/logout">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <button type="submit">Выйти</button>
    </form>
</body>
</html>
`

var (
	tmplLogin    = template.Must(template.New("login").Parse(loginTemplate))
	tmplRegister = template.Must(template.New("register").Parse(registerTemplate))
	tmplHome     = template.Must(template.New("home").Parse(homeTemplate))
)

// ==================== Хендлеры ====================

func registerPageHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if _, ok := sessions.Get(c.Value); ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	renderRegister(w, "", "")
}

func registerSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}

	formCSRF := r.PostFormValue("csrf_token")
	cookieCSRF, err := r.Cookie(csrfCookie)
	if err != nil || !constantTimeEqual(formCSRF, cookieCSRF.Value) {
		http.Error(w, "CSRF-токен недействителен", http.StatusForbidden)
		return
	}

	username := strings.TrimSpace(r.PostFormValue("username"))
	password := r.PostFormValue("password")
	password2 := r.PostFormValue("password2")

	if errMsg := validateRegistration(username, password, password2); errMsg != "" {
		renderRegister(w, errMsg, username)
		return
	}

	// Проверка на существующего пользователя
	if _, exists := users.Get(username); exists {
		renderRegister(w, "Пользователь с таким логином уже существует", username)
		return
	}

	// Хешируем пароль (PBKDF2 со солью, 100k итераций)
	pwHash := hashPassword(password)

	if err := users.Create(username, pwHash); err != nil {
		renderRegister(w, "Ошибка регистрации. Попробуйте ещё раз.", username)
		return
	}

	// Автоматический вход после регистрации
	token := sessions.Create(username)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if _, ok := sessions.Get(c.Value); ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	renderLogin(w, "")
}

func loginSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}

	formCSRF := r.PostFormValue("csrf_token")
	cookieCSRF, err := r.Cookie(csrfCookie)
	if err != nil || !constantTimeEqual(formCSRF, cookieCSRF.Value) {
		http.Error(w, "CSRF-токен недействителен", http.StatusForbidden)
		return
	}

	username := strings.TrimSpace(r.PostFormValue("username"))
	password := r.PostFormValue("password")

	u, ok := users.Get(username)
	// Важно: даже если пользователь не найден, всё равно запускаем verifyPassword
	// с фиктивным хешем, чтобы не выдать существование логина по времени ответа.
	valid := false
	if ok {
		valid = verifyPassword(password, u.PasswordHash)
	} else {
		// Фиктивная проверка
		verifyPassword(password, "100000$00000000000000000000000000000000$0000000000000000000000000000000000000000000000000000000000000000")
	}

	if !valid {
		renderLogin(w, "Неверный логин или пароль")
		return
	}

	token := sessions.Create(u.Username)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	sess := getSession(r)
	if sess == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	csrf := ensureCSRFCookie(w, r)
	tmplHome.Execute(w, map[string]string{
		"Username":  sess.Username,
		"CSRFToken": csrf,
	})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}
	formCSRF := r.PostFormValue("csrf_token")
	cookieCSRF, err := r.Cookie(csrfCookie)
	if err != nil || !constantTimeEqual(formCSRF, cookieCSRF.Value) {
		http.Error(w, "CSRF-токен недействителен", http.StatusForbidden)
		return
	}
	if c, err := r.Cookie(sessionCookie); err == nil {
		sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ==================== Вспомогательные ====================

func renderLogin(w http.ResponseWriter, errMsg string) {
	token := ensureCSRFCookie(w, nil)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmplLogin.Execute(w, map[string]string{
		"Error":     errMsg,
		"CSRFToken": token,
	})
}

func renderRegister(w http.ResponseWriter, errMsg, username string) {
	token := ensureCSRFCookie(w, nil)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmplRegister.Execute(w, map[string]string{
		"Error":     errMsg,
		"CSRFToken": token,
		"Username":  username,
	})
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if r != nil {
		if c, err := r.Cookie(csrfCookie); err == nil && c.Value != "" {
			return c.Value
		}
	}
	token := randomToken(32)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    token,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})
	return token
}

func getSession(r *http.Request) *Session {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	sess, ok := sessions.Get(c.Value)
	if !ok {
		return nil
	}
	return sess
}

// ==================== main ====================

func main() {
	sessions.Cleanup(10 * time.Minute)

	mux := http.NewServeMux()

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			registerPageHandler(w, r)
		case http.MethodPost:
			registerSubmitHandler(w, r)
		default:
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			loginPageHandler(w, r)
		case http.MethodPost:
			loginSubmitHandler(w, r)
		default:
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/logout", logoutHandler)
	mux.HandleFunc("/", homeHandler)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}
