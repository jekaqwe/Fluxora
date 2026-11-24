package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func main() {
	// Получаем адрес базы данных
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL не задан")
	}

	// Создаём подключение к PostgreSQL
	var err error
	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal("Ошибка подключения к PostgreSQL: ", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("PostgreSQL недоступен: ", err)
	}

	log.Println("Подключение к PostgreSQL успешно!")

	if err := runMigrations(db); err != nil {
		log.Fatal("Ошибка применения миграций: ", err)
	}
	log.Println("Миграции базы данных применены")

	// Регистрация пользователя
	http.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		registerHandler(w, r, db)
	})

	// Авторизация пользователя
	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		loginHandler(w, r, db)
	})

	// Проверка авторизации
	http.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		meHandler(w, r, db)
	})

	// Профиль пользователя
	http.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
		profileHandler(w, r, db)
	})

	// Аватар пользователя
	http.HandleFunc("/api/profile/avatar", func(w http.ResponseWriter, r *http.Request) {
		avatarHandler(w, r, db)
	})

	// Личные счета пользователя
	http.HandleFunc("/api/accounts", func(w http.ResponseWriter, r *http.Request) {
		accountsHandler(w, r, db)
	})

	// Финансовые операции
	http.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
		transactionsHandler(w, r, db)
	})

	// Финансовая аналитика
	http.HandleFunc("/api/analytics", func(w http.ResponseWriter, r *http.Request) {
		analyticsHandler(w, r, db)
	})

	// Сводка главной страницы
	http.HandleFunc("/api/dashboard", func(w http.ResponseWriter, r *http.Request) {
		dashboardHandler(w, r, db)
	})

	// Финансовые цели
	http.HandleFunc("/api/goals", func(w http.ResponseWriter, r *http.Request) {
		goalsHandler(w, r, db)
	})

	// Семейный бюджет и общие операции
	http.HandleFunc("/api/family", func(w http.ResponseWriter, r *http.Request) {
		familyHandler(w, r, db)
	})
	http.HandleFunc("/api/family/", func(w http.ResponseWriter, r *http.Request) {
		familyHandler(w, r, db)
	})

	// Регулярные операции
	http.HandleFunc("/api/recurring", func(w http.ResponseWriter, r *http.Request) {
		recurringHandler(w, r, db)
	})
	http.HandleFunc("/api/recurring/", func(w http.ResponseWriter, r *http.Request) {
		recurringHandler(w, r, db)
	})

	// Долги и кредиты
	http.HandleFunc("/api/debts", func(w http.ResponseWriter, r *http.Request) {
		debtsHandler(w, r, db)
	})
	http.HandleFunc("/api/debts/", func(w http.ResponseWriter, r *http.Request) {
		debtsHandler(w, r, db)
	})

	// Смена пароля
	http.HandleFunc("/api/account/password", func(w http.ResponseWriter, r *http.Request) {
		passwordChangeHandler(w, r, db)
	})

	// Выход из аккаунта
	http.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		logoutHandler(w, r, db)
	})

	// Проверка Backend
	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := map[string]string{
			"status":  "ok",
			"message": "Hello from Fluxora Backend!",
		}

		json.NewEncoder(w).Encode(response)
	})

	startRecurringScheduler(db)
	log.Println("Fluxora Backend запущен на http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
