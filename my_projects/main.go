package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// ---------- Модель данных ----------

type Resource struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Store struct {
	mu        sync.Mutex
	filePath  string
	nextID    int
	Resources []Resource `json:"resources"`
}

// ---------- Работа с JSON-файлом ----------

func NewStore(path string) (*Store, error) {
	s := &Store{filePath: path, nextID: 1}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// создаём пустой файл
			if err := s.save(); err != nil {
				return nil, err
			}
			return s, nil
		}
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, s); err != nil {
			return nil, err
		}
	}
	// восстановим nextID
	for _, r := range s.Resources {
		if r.ID >= s.nextID {
			s.nextID = r.ID + 1
		}
	}
	return s, nil
}

// save вызывается при удерживаемом mu
func (s *Store) save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) List() []Resource {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Resource, len(s.Resources))
	copy(out, s.Resources)
	return out
}

func (s *Store) Add(name string) (Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := Resource{ID: s.nextID, Name: name}
	s.nextID++
	s.Resources = append(s.Resources, r)
	return r, s.save()
}

func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.Resources {
		if r.ID == id {
			s.Resources = append(s.Resources[:i], s.Resources[i+1:]...)
			return s.save()
		}
	}
	return os.ErrNotExist
}

// ---------- HTML-шаблоны ----------

const tmplIndex = `
<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8">
<title>Меню ресурсов</title>
<style>
  body { font-family: sans-serif; max-width: 720px; margin: 40px auto; padding: 0 16px; background:#f7f7f9; color:#222; }
  h1 { border-bottom: 2px solid #333; padding-bottom: 8px; }
  form { display:flex; gap:8px; margin-bottom: 24px; }
  input[type=text] { flex:1; padding:10px; font-size:15px; border:1px solid #ccc; border-radius:6px; }
  button { padding:10px 18px; font-size:15px; border:none; border-radius:6px; cursor:pointer; }
  .btn-add { background:#2e7d32; color:#fff; }
  .btn-add:hover { background:#256628; }
  .btn-del { background:#c62828; color:#fff; padding:6px 12px; font-size:13px; }
  .btn-del:hover { background:#a31f1f; }
  table { width:100%; border-collapse: collapse; background:#fff; border-radius:8px; overflow:hidden; box-shadow:0 1px 4px rgba(0,0,0,.08); }
  th, td { padding:12px 14px; text-align:left; border-bottom:1px solid #eee; }
  th { background:#f0f0f4; font-size:14px; text-transform:uppercase; letter-spacing:.04em; }
  tr:last-child td { border-bottom:none; }
  .empty { text-align:center; color:#888; padding:24px; }
</style>
</head>
<body>
  <h1>Меню ресурсов</h1>

  <form method="POST" action="/add">
    <input type="text" name="name" placeholder="Название ресурса…" required autofocus>
    <button type="submit" class="btn-add">Добавить</button>
  </form>

  <table>
    <thead>
      <tr><th style="width:70px">ID</th><th>Название</th><th style="width:110px">Действие</th></tr>
    </thead>
    <tbody>
      {{if .}}
        {{range .}}
          <tr>
            <td>{{.ID}}</td>
            <td>{{.Name}}</td>
            <td>
              <form method="POST" action="/delete" style="margin:0">
                <input type="hidden" name="id" value="{{.ID}}">
                <button type="submit" class="btn-del">Удалить</button>
              </form>
            </td>
          </tr>
        {{end}}
      {{else}}
        <tr><td colspan="3" class="empty">Список пуст. Добавьте первый ресурс.</td></tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>
`

var tmpl = template.Must(template.New("index").Parse(tmplIndex))

// ---------- HTTP-обработчики ----------

type App struct {
	store *Store
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, a.store.List()); err != nil {
		log.Println("template:", err)
	}
}

func (a *App) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	name := r.FormValue("name")
	if name != "" {
		if _, err := a.store.Add(name); err != nil {
			http.Error(w, "Ошибка сохранения: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}
	if err := a.store.Delete(id); err != nil && !os.IsNotExist(err) {
		http.Error(w, "Ошибка удаления: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---------- main ----------

func main() {
	store, err := NewStore("resources.json")
	if err != nil {
		log.Fatalf("не удалось открыть хранилище: %v", err)
	}

	app := &App{store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleIndex)
	mux.HandleFunc("/add", app.handleAdd)
	mux.HandleFunc("/delete", app.handleDelete)

	addr := ":8080"
	log.Printf("Сервер запущен: http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
