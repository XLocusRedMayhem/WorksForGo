package main

import (
	"html/template"
	"log"
	"net/http"
)

var tmpl = template.Must(template.New("page").Parse(`
<!DOCTYPE html>
<html>
<body>
    {{ if .Name }}
        <p>Привет, {{ .Name }}!</p>
        <p>Ваш email: {{ .Email }}</p>
        <form action="/" method="get">
            <button type="submit">Назад</button>
        </form>
    {{ else }}
        <form action="/submit" method="post">
            <p>
                <label for="username">Имя:</label>
                <input type="text" id="username" name="username" required>
            </p>
            <p>
                <label for="email">Email:</label>
                <input type="email" id="email" name="email" required>
            </p>
            <button type="submit">Отправить</button>
        </form>
    {{ end }}
</body>
</html>
`))

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	})

	mux.HandleFunc("POST /submit", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Ошибка разбора формы", http.StatusBadRequest)
			return
		}
		name := r.FormValue("username")
		if name == "" {
			http.Error(w, "Имя не может быть пустым", http.StatusBadRequest)
			return
		}
		password := r.FormValue("")
		if password == "" {
			http.Error(w, "Пароль не может быть пустым", http.StatusBadRequest)
		}
		tmpl.Execute(w, map[string]string{"Name": name, "Password": password})
	})

	log.Println("http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
