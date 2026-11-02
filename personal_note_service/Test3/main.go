package main

import (
	"html/template"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.Handler()

	srv := &http.Server{
		Addr:              "8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

const homeTemplate = `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Home — Главная страница</title>
    <style>
        /* ===== Сброс стилей ===== */
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: #f4f6f9;
            color: #333;
            line-height: 1.6;
        }

        /* ===== Шапка ===== */
        header {
            background: linear-gradient(135deg, #4a6cf7, #6a3ff7);
            color: #fff;
            padding: 20px 40px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        }

        .logo {
            font-size: 1.6rem;
            font-weight: bold;
            letter-spacing: 1px;
        }

        nav ul {
            list-style: none;
            display: flex;
            gap: 25px;
        }

        nav a {
            color: #fff;
            text-decoration: none;
            font-weight: 500;
            transition: opacity 0.3s;
        }

        nav a:hover {
            opacity: 0.75;
        }

        /* ===== Главный блок (hero) ===== */
        .hero {
            text-align: center;
            padding: 100px 20px;
            background: linear-gradient(135deg, #ffffff, #e8ecff);
        }

        .hero h1 {
            font-size: 3rem;
            color: #2c3e70;
            margin-bottom: 20px;
        }

        .hero p {
            font-size: 1.2rem;
            color: #555;
            max-width: 600px;
            margin: 0 auto 30px;
        }

        .btn {
            display: inline-block;
            padding: 12px 32px;
            background: #4a6cf7;
            color: #fff;
            text-decoration: none;
            border-radius: 30px;
            font-weight: 600;
            transition: background 0.3s, transform 0.2s;
        }

        .btn:hover {
            background: #3552d6;
            transform: translateY(-2px);
        }

        /* ===== Блок с карточками ===== */
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 25px;
            padding: 60px 40px;
            max-width: 1100px;
            margin: 0 auto;
        }

        .card {
            background: #fff;
            border-radius: 12px;
            padding: 30px 25px;
            text-align: center;
            box-shadow: 0 4px 15px rgba(0, 0, 0, 0.07);
            transition: transform 0.3s, box-shadow 0.3s;
        }

        .card:hover {
            transform: translateY(-5px);
            box-shadow: 0 8px 25px rgba(0, 0, 0, 0.12);
        }

        .card .icon {
            font-size: 2.5rem;
            margin-bottom: 15px;
        }

        .card h3 {
            color: #2c3e70;
            margin-bottom: 10px;
        }

        .card p {
            color: #666;
            font-size: 0.95rem;
        }

        /* ===== Подвал ===== */
        footer {
            background: #2c3e70;
            color: #cfd8ff;
            text-align: center;
            padding: 25px;
            margin-top: 40px;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>

    <!-- Шапка -->
    <header>
        <div class="logo">MySite</div>
        <nav>
            <ul>
                <li><a href="#">Главная</a></li>
                <li><a href="#">О нас</a></li>
                <li><a href="#">Услуги</a></li>
                <li><a href="#">Контакты</a></li>
            </ul>
        </nav>
    </header>

    <!-- Hero-блок -->
    <section class="hero">
        <h1>Добро пожаловать!</h1>
        <p>Мы создаём удобные и красивые веб-решения, которые помогают вашему бизнесу расти.</p>
        <a href="#" class="btn">Узнать больше</a>
    </section>

    <!-- Карточки -->
    <section class="features">
        <div class="card">
            <div class="icon">🚀</div>
            <h3>Быстрый старт</h3>
            <p>Начните работу за считанные минуты без лишних настроек.</p>
        </div>
        <div class="card">
            <div class="icon">🎨</div>
            <h3>Современный дизайн</h3>
            <p>Адаптивные интерфейсы, которые приятно использовать.</p>
        </div>
        <div class="card">
            <div class="icon">🔒</div>
            <h3>Надёжность</h3>
            <p>Безопасность и стабильность — в основе каждой нашей разработки.</p>
        </div>
    </section>

    <!-- Подвал -->
    <footer>
        &copy; 2026 MySite. Все права защищены.
    </footer>

</body>
</html>
`

var (
	tempHome = template.Must(template.New("Home").Parse(homeTemplate))
)
