package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ==================== CONFIG ====================
const (
	JWTSecret    = "change-me-in-prod-super-secret-key"
	JWTIssuer    = "notes-api"
	TokenTTL     = 15 * time.Minute
	PBKDF2Iter   = 120000
	PBKDF2KeyLen = 32
	SaltLen      = 16
	MaxBodyBytes = 1 << 20 // 1 MiB
)

// ==================== MODELS ====================
type User struct {
	ID           int64
	Email        string // хранится в нижнем регистре
	PasswordHash string // "iter$salt_hex$hash_hex"
}

type Note struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ==================== STORAGE (in-memory) ====================
type Store struct {
	mu      sync.RWMutex
	users   map[int64]*User
	emails  map[string]int64 // нормализованный email -> userID
	notes   map[int64]*Note
	nextUID int64
	nextNID int64
}

func NewStore() *Store {
	return &Store{
		users:  make(map[int64]*User),
		emails: make(map[string]int64),
		notes:  make(map[int64]*Note),
	}
}

func (s *Store) CreateUser(email, passHash string) (*User, error) {
	key := strings.ToLower(strings.TrimSpace(email))
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.emails[key]; ok {
		return nil, errors.New("email already registered")
	}
	s.nextUID++
	u := &User{ID: s.nextUID, Email: key, PasswordHash: passHash}
	s.users[u.ID] = u
	s.emails[key] = u.ID
	return u, nil
}

func (s *Store) FindUserByEmail(email string) *User {
	key := strings.ToLower(strings.TrimSpace(email))
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.emails[key]
	if !ok {
		return nil
	}
	return s.users[id]
}

func (s *Store) CreateNote(userID int64, title, content string) *Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextNID++
	n := &Note{
		ID:        s.nextNID,
		UserID:    userID,
		Title:     title,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
	s.notes[n.ID] = n
	return n
}

func (s *Store) ListNotes(userID int64) []*Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Note, 0)
	for _, n := range s.notes {
		if n.UserID == userID {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

func (s *Store) GetNote(userID, noteID int64) *Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[noteID]
	if !ok || n.UserID != userID {
		return nil
	}
	return n
}

func (s *Store) DeleteNote(userID, noteID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notes[noteID]
	if !ok || n.UserID != userID {
		return false
	}
	delete(s.notes, noteID)
	return true
}

var store = NewStore()

// ==================== PASSWORD HASHING (PBKDF2-HMAC-SHA256) ====================
func pbkdf2Key(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	dk := make([]byte, 0, numBlocks*hashLen)
	var blockBuf [4]byte
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		blockBuf[0] = byte(block >> 24)
		blockBuf[1] = byte(block >> 16)
		blockBuf[2] = byte(block >> 8)
		blockBuf[3] = byte(block)
		prf.Write(blockBuf[:])
		u := prf.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iter; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		dk = append(dk, t...)
	}
	return dk[:keyLen]
}

func HashPassword(password string) string {
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}
	dk := pbkdf2Key([]byte(password), salt, PBKDF2Iter, PBKDF2KeyLen)
	return fmt.Sprintf("%d$%s$%s", PBKDF2Iter, hex.EncodeToString(salt), hex.EncodeToString(dk))
}

func VerifyPassword(password, stored string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 3 {
		return false
	}
	iter, err := strconv.Atoi(parts[0])
	if err != nil || iter <= 0 {
		return false
	}
	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	expected, err := hex.DecodeString(parts[2])
	if err != nil || len(expected) == 0 {
		return false
	}
	dk := pbkdf2Key([]byte(password), salt, iter, len(expected))
	return subtle.ConstantTimeCompare(dk, expected) == 1
}

// ==================== JWT (HS256) ====================
type JWTClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
	Iss   string `json:"iss"`
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func signJWT(claims JWTClaims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	hJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := b64url(hJSON) + "." + b64url(pJSON)
	mac := hmac.New(sha256.New, []byte(JWTSecret))
	mac.Write([]byte(signingInput))
	return signingInput + "." + b64url(mac.Sum(nil)), nil
}

func parseJWT(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	// Проверяем alg в заголовке — только HS256.
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("invalid token header")
	}
	var hdr struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &hdr); err != nil || hdr.Alg != "HS256" {
		return nil, errors.New("unsupported token algorithm")
	}

	// Проверяем подпись.
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(JWTSecret))
	mac.Write([]byte(signingInput))
	expected := b64url(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return nil, errors.New("invalid signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}
	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.New("invalid claims")
	}
	if claims.Iss != JWTIssuer {
		return nil, errors.New("invalid issuer")
	}
	if claims.Exp == 0 || time.Now().Unix() >= claims.Exp {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

// ==================== HTTP HELPERS ====================
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func limitBody(r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, MaxBodyBytes)
}

type ctxKey string

const ctxUserID ctxKey = "userID"

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeErr(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		claims, err := parseJWT(strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		uid, err := strconv.ParseInt(claims.Sub, 10, 64)
		if err != nil || uid <= 0 {
			writeErr(w, http.StatusUnauthorized, "invalid sub claim")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, uid)
		next(w, r.WithContext(ctx))
	}
}

func userIDFromCtx(r *http.Request) int64 {
	if v, ok := r.Context().Value(ctxUserID).(int64); ok {
		return v
	}
	return 0
}

// ==================== HANDLERS ====================
type credentialsReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limitBody(r)
	var req credentialsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(req.Email, "@") || len(req.Email) < 3 {
		writeErr(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(req.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "password must be at least 6 chars")
		return
	}
	u, err := store.CreateUser(req.Email, HashPassword(req.Password))
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id": u.ID, "email": u.Email,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limitBody(r)
	var req credentialsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	u := store.FindUserByEmail(req.Email)
	if u == nil || !VerifyPassword(req.Password, u.PasswordHash) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	now := time.Now()
	claims := JWTClaims{
		Sub:   strconv.FormatInt(u.ID, 10),
		Email: u.Email,
		Iat:   now.Unix(),
		Exp:   now.Add(TokenTTL).Unix(),
		Iss:   JWTIssuer,
	}
	token, err := signJWT(claims)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to sign token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"token_type": "Bearer",
		"expires_in": int(TokenTTL.Seconds()),
	})
}

type noteReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func handleNotes(w http.ResponseWriter, r *http.Request) {
	uid := userIDFromCtx(r)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, store.ListNotes(uid))
	case http.MethodPost:
		limitBody(r)
		var req noteReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		if req.Title == "" {
			writeErr(w, http.StatusBadRequest, "title is required")
			return
		}
		n := store.CreateNote(uid, req.Title, req.Content)
		writeJSON(w, http.StatusCreated, n)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleNoteByID(w http.ResponseWriter, r *http.Request) {
	uid := userIDFromCtx(r)
	idStr := strings.TrimPrefix(r.URL.Path, "/notes/")
	if idStr == "" || strings.Contains(idStr, "/") {
		writeErr(w, http.StatusBadRequest, "invalid note id")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "invalid note id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		n := store.GetNote(uid, id)
		if n == nil {
			writeErr(w, http.StatusNotFound, "note not found")
			return
		}
		writeJSON(w, http.StatusOK, n)
	case http.MethodDelete:
		if !store.DeleteNote(uid, id) {
			writeErr(w, http.StatusNotFound, "note not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ==================== HTML (SPA) ====================
const indexHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>Заметки</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: system-ui, -apple-system, Segoe UI, Roboto, sans-serif;
         margin: 0; background: #f3f4f6; color: #111827; }
  .container { max-width: 720px; margin: 40px auto; padding: 0 16px; }
  .card { background: #fff; border-radius: 12px; padding: 24px;
          box-shadow: 0 4px 20px rgba(0,0,0,.06); }
  h1 { margin: 0 0 16px; font-size: 22px; }
  h2 { margin: 0 0 16px; font-size: 18px; }
  input, textarea { width: 100%; padding: 10px 12px; margin: 6px 0 14px;
          border: 1px solid #d1d5db; border-radius: 8px; font: inherit; }
  textarea { min-height: 100px; resize: vertical; }
  button { padding: 10px 16px; border-radius: 8px; border: 0; cursor: pointer;
           font: inherit; background: #2563eb; color: white; }
  button.secondary { background: #e5e7eb; color: #111827; }
  button.danger { background: #dc2626; }
  button:hover { opacity: .92; }
  button:disabled { opacity: .5; cursor: default; }
  .row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  .muted { color: #6b7280; font-size: 13px; }
  .error { color: #dc2626; font-size: 14px; margin: 8px 0; min-height: 18px; }
  .ok { color: #059669; font-size: 14px; margin: 8px 0; min-height: 18px; }
  .note { border: 1px solid #e5e7eb; border-radius: 10px; padding: 14px;
          margin-bottom: 10px; background: #fff; }
  .note h3 { margin: 0 0 6px; font-size: 16px; word-break: break-word; }
  .note p { margin: 0 0 8px; white-space: pre-wrap; word-break: break-word; }
  .top { display: flex; justify-content: space-between; align-items: center;
         margin-bottom: 16px; gap: 8px; flex-wrap: wrap; }
  .hidden { display: none !important; }
</style>
</head>
<body>
<div class="container">

  <!-- Экран логина -->
  <div id="login-screen" class="card hidden">
    <h1>Вход</h1>
    <input id="login-email" type="email" placeholder="Email" autocomplete="username"/>
    <input id="login-password" type="password" placeholder="Пароль" autocomplete="current-password"/>
    <div class="error" id="login-error"></div>
    <div class="row">
      <button id="btn-login">Войти</button>
      <button id="btn-go-register" class="secondary">Регистрация</button>
    </div>
  </div>

  <!-- Экран регистрации -->
  <div id="register-screen" class="card hidden">
    <h1>Регистрация</h1>
    <input id="reg-email" type="email" placeholder="Email" autocomplete="username"/>
    <input id="reg-password" type="password" placeholder="Пароль (мин. 6 символов)" autocomplete="new-password"/>
    <div class="error" id="reg-error"></div>
    <div class="row">
      <button id="btn-register">Зарегистрироваться</button>
      <button id="btn-back-login" class="secondary">Назад</button>
    </div>
  </div>

  <!-- Экран заметок -->
  <div id="notes-screen" class="hidden">
    <div class="card" style="margin-bottom:16px;">
      <div class="top">
        <h1>Мои заметки</h1>
        <div class="row">
          <span class="muted" id="user-email"></span>
          <button id="btn-logout" class="secondary">Выйти</button>
        </div>
      </div>
      <h2>Новая заметка</h2>
      <input id="note-title" placeholder="Заголовок"/>
      <textarea id="note-content" placeholder="Текст заметки"></textarea>
      <div class="error" id="note-error"></div>
      <button id="btn-create">Создать заметку</button>
    </div>
    <div id="notes-list"></div>
  </div>

</div>

<script>
const $ = (id) => document.getElementById(id);
let token = localStorage.getItem('token') || '';
let userEmail = localStorage.getItem('email') || '';

function show(id) {
  ['login-screen','register-screen','notes-screen'].forEach(x => $(x).classList.add('hidden'));
  $(id).classList.remove('hidden');
}

// Возвращает {data, status}. Не бросает на не-2xx — статус проверяем сами.
async function api(path, opts={}) {
  const headers = Object.assign({}, opts.headers || {});
  headers['Content-Type'] = 'application/json';
  if (token) headers['Authorization'] = 'Bearer ' + token;
  const res = await fetch(path, Object.assign({}, opts, {headers}));
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch(e) { data = text; }
  return { data, status: res.status };
}

function clearSession() {
  token = ''; userEmail = '';
  localStorage.removeItem('token');
  localStorage.removeItem('email');
}

function goLogin(msg) {
  clearSession();
  $('login-error').textContent = msg || '';
  show('login-screen');
}

// ---- Логин ----
$('btn-login').onclick = async () => {
  $('login-error').textContent = '';
  const email = $('login-email').value.trim();
  const password = $('login-password').value;
  const btn = $('btn-login');
  btn.disabled = true;
  try {
    const {data, status} = await api('/login', {
      method: 'POST',
      body: JSON.stringify({email, password})
    });
    if (status !== 200) {
      $('login-error').textContent = (data && data.error) || ('HTTP ' + status);
      return;
    }
    token = data.token;
    userEmail = email.toLowerCase();
    localStorage.setItem('token', token);
    localStorage.setItem('email', userEmail);
    $('login-password').value = '';
    enterNotes();
  } catch(e) {
    $('login-error').textContent = e.message || 'сеть недоступна';
  } finally {
    btn.disabled = false;
  }
};

$('btn-go-register').onclick = () => {
  $('reg-error').textContent = '';
  $('reg-email').value = $('login-email').value.trim();
  show('register-screen');
};
$('btn-back-login').onclick = () => goLogin('');

// ---- Регистрация ----
$('btn-register').onclick = async () => {
  $('reg-error').textContent = '';
  const email = $('reg-email').value.trim();
  const password = $('reg-password').value;
  const btn = $('btn-register');
  btn.disabled = true;
  try {
    const {data, status} = await api('/register', {
      method: 'POST',
      body: JSON.stringify({email, password})
    });
    if (status !== 201) {
      $('reg-error').textContent = (data && data.error) || ('HTTP ' + status);
      return;
    }
    // Сразу на экран логина, email подставлен.
    $('login-email').value = email;
    $('login-password').value = '';
    $('login-error').textContent = 'Аккаунт создан. Войдите.';
    $('reg-password').value = '';
    show('login-screen');
  } catch(e) {
    $('reg-error').textContent = e.message || 'сеть недоступна';
  } finally {
    btn.disabled = false;
  }
};

// ---- Заметки ----
function enterNotes() {
  $('user-email').textContent = userEmail;
  show('notes-screen');
  loadNotes();
}

$('btn-logout').onclick = () => goLogin('');

$('btn-create').onclick = async () => {
  $('note-error').textContent = '';
  const title = $('note-title').value.trim();
  const content = $('note-content').value;
  if (!title) { $('note-error').textContent = 'Заголовок обязателен'; return; }
  const btn = $('btn-create');
  btn.disabled = true;
  try {
    const {data, status} = await api('/notes', {
      method: 'POST',
      body: JSON.stringify({title, content})
    });
    if (status === 401) { goLogin('Сессия истекла, войдите снова'); return; }
    if (status !== 201) {
      $('note-error').textContent = (data && data.error) || ('HTTP ' + status);
      return;
    }
    $('note-title').value = '';
    $('note-content').value = '';
    loadNotes();
  } catch(e) {
    $('note-error').textContent = e.message || 'сеть недоступна';
  } finally {
    btn.disabled = false;
  }
};

async function loadNotes() {
  const list = $('notes-list');
  list.innerHTML = '<div class="card muted">Загрузка…</div>';
  try {
    const {data, status} = await api('/notes');
    if (status === 401) { goLogin('Сессия истекла, войдите снова'); return; }
    if (status !== 200) {
      list.innerHTML = '<div class="card error">Ошибка: ' + ((data && data.error) || status) + '</div>';
      return;
    }
    const notes = data || [];
    list.innerHTML = '';
    if (!notes.length) {
      list.innerHTML = '<div class="card muted">Заметок пока нет.</div>';
      return;
    }
    notes.forEach(n => {
      const div = document.createElement('div');
      div.className = 'note';
      const h = document.createElement('h3'); h.textContent = n.title;
      const p = document.createElement('p'); p.textContent = n.content || '';
      const row = document.createElement('div'); row.className = 'row';
      const del = document.createElement('button');
      del.className = 'danger'; del.textContent = 'Удалить';
      del.onclick = async () => {
        if (!confirm('Удалить заметку "' + n.title + '"?')) return;
        del.disabled = true;
        try {
          const {status} = await api('/notes/' + n.id, {method:'DELETE'});
          if (status === 401) { goLogin('Сессия истекла, войдите снова'); return; }
          if (status !== 204) { alert('Ошибка удаления (HTTP ' + status + ')'); del.disabled = false; return; }
          loadNotes();
        } catch(e) {
          alert(e.message || 'сеть недоступна');
          del.disabled = false;
        }
      };
      row.appendChild(del);
      div.appendChild(h); div.appendChild(p); div.appendChild(row);
      list.appendChild(div);
    });
  } catch(e) {
    list.innerHTML = '<div class="card error">' + (e.message || 'сеть недоступна') + '</div>';
  }
}

// ---- Старт: проверяем сохранённый токен реальным запросом ----
(async () => {
  if (!token) { show('login-screen'); return; }
  try {
    const {status} = await api('/notes');
    if (status === 200) {
      enterNotes();
    } else {
      goLogin('');
    }
  } catch(e) {
    goLogin('');
  }
})();
</script>
</body>
</html>`

// ==================== ROUTER ====================
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(indexHTML))
		return
	}
	http.NotFound(w, r)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/register", handleRegister)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/notes", authMiddleware(handleNotes))
	mux.HandleFunc("/notes/", authMiddleware(handleNoteByID))
	mux.HandleFunc("/", rootHandler)

	addr := ":8080"
	log.Printf("Notes API listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
