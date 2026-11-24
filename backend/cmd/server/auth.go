package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Данные, которые приходят от frontend
type RegisterRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// Данные, которые приходят от frontend при входе
type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// Обработка регистрации
func registerHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")

	// Разрешаем только POST-запросы
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	// Ограничиваем размер тела запроса до разбора JSON.
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var user RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, `{"error":"Некорректные данные"}`, http.StatusBadRequest)
		return
	}

	// Убираем лишние пробелы
	user.FirstName = strings.TrimSpace(user.FirstName)
	user.LastName = strings.TrimSpace(user.LastName)
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(user.Email)

	// Проверяем обязательные поля
	if user.FirstName == "" || user.LastName == "" ||
		user.Username == "" || user.Email == "" || user.Password == "" {
		http.Error(w, `{"error":"Заполните все поля"}`, http.StatusBadRequest)
		return
	}
	if len([]rune(user.FirstName)) > 100 || len([]rune(user.LastName)) > 100 || len(user.Username) > 50 || len(user.Email) > 255 {
		http.Error(w, `{"error":"Проверьте длину имени, фамилии, логина и email"}`, http.StatusBadRequest)
		return
	}
	parsedEmail, emailErr := mail.ParseAddress(user.Email)
	if emailErr != nil || parsedEmail.Address != user.Email {
		http.Error(w, `{"error":"Укажите корректный email"}`, http.StatusBadRequest)
		return
	}

	// bcrypt поддерживает не более 72 байт пароля.
	if len(user.Password) < 8 || len(user.Password) > 72 {
		http.Error(w, `{"error":"Пароль должен содержать от 8 до 72 байт"}`, http.StatusBadRequest)
		return
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		http.Error(w, `{"error":"Ошибка обработки пароля"}`, http.StatusInternalServerError)
		return
	}

	// Сохраняем пользователя в PostgreSQL
	_, err = database.Exec(r.Context(),
		`INSERT INTO users (first_name, last_name, username, email, password_hash)
		 VALUES ($1, $2, $3, $4, $5)`,
		user.FirstName,
		user.LastName,
		user.Username,
		user.Email,
		string(hashedPassword),
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, `{"error":"Логин или email уже используются"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"Ошибка при регистрации пользователя"}`, http.StatusInternalServerError)
		return
	}

	// Отправляем успешный ответ
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Регистрация успешна",
	})
}

// Обработка входа
func loginHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var user LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, `{"error":"Некорректные данные"}`, http.StatusBadRequest)
		return
	}

	user.Identifier = strings.TrimSpace(user.Identifier)

	if user.Identifier == "" || user.Password == "" || len(user.Password) > 72 {
		http.Error(w, `{"error":"Заполните все поля"}`, http.StatusBadRequest)
		return
	}

	// Ищем пользователя по логину или email
	var userID int64
	var passwordHash string

	err := database.QueryRow(r.Context(),
		`SELECT id, password_hash
         FROM users
         WHERE LOWER(username) = LOWER($1) OR LOWER(email) = LOWER($1)
         LIMIT 1`,
		user.Identifier,
	).Scan(&userID, &passwordHash)

	if err != nil {
		http.Error(w, `{"error":"Неверный логин или пароль"}`, http.StatusUnauthorized)
		return
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(user.Password),
	); err != nil {
		http.Error(w, `{"error":"Неверный логин или пароль"}`, http.StatusUnauthorized)
		return
	}

	// Создаём случайный секретный токен
	tokenBytes := make([]byte, 32)

	if _, err := rand.Read(tokenBytes); err != nil {
		http.Error(w, `{"error":"Ошибка создания сессии"}`, http.StatusInternalServerError)
		return
	}

	token := hex.EncodeToString(tokenBytes)

	// Сохраняем в базе только SHA-256 хеш токена
	tokenHash := sha256.Sum256([]byte(token))
	tokenHashString := hex.EncodeToString(tokenHash[:])

	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err = database.Exec(r.Context(),
		`INSERT INTO sessions (user_id, token_hash, expires_at)
         VALUES ($1, $2, $3)`,
		userID,
		tokenHashString,
		expiresAt,
	)

	if err != nil {
		http.Error(w, `{"error":"Не удалось создать сессию"}`, http.StatusInternalServerError)
		return
	}

	// Устанавливаем Cookie в браузере
	http.SetCookie(w, &http.Cookie{
		Name:     "fluxora_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   30 * 24 * 60 * 60,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Вход выполнен успешно",
	})
}

// Проверка авторизованного пользователя
func meHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	// Получаем Cookie с токеном
	cookie, err := r.Cookie("fluxora_session")
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}

	// Вычисляем SHA-256 хеш токена
	tokenHash := sha256.Sum256([]byte(cookie.Value))
	tokenHashString := hex.EncodeToString(tokenHash[:])

	// Ищем действующую сессию и данные пользователя
	var user struct {
		ID        int    `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Username  string `json:"username"`
		Email     string `json:"email"`
	}

	err = database.QueryRow(r.Context(),
		`SELECT u.id, u.first_name, u.last_name, u.username, u.email
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1 AND s.expires_at > NOW()`,
		tokenHashString,
	).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Email,
	)

	if err != nil {
		http.Error(w, `{"error":"Сессия недействительна или истекла"}`, http.StatusUnauthorized)
		return
	}

	// Возвращаем данные авторизованного пользователя
	json.NewEncoder(w).Encode(user)
}

// Выход из аккаунта
func logoutHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}

	// Получаем Cookie
	cookie, err := r.Cookie("fluxora_session")

	if err == nil {
		// Вычисляем хеш токена
		tokenHash := sha256.Sum256([]byte(cookie.Value))
		tokenHashString := hex.EncodeToString(tokenHash[:])

		// Удаляем сессию из PostgreSQL
		_, err = database.Exec(r.Context(),
			`DELETE FROM sessions WHERE token_hash = $1`,
			tokenHashString,
		)

		if err != nil {
			http.Error(w, `{"error":"Не удалось завершить сессию"}`, http.StatusInternalServerError)
			return
		}
	}

	// Удаляем Cookie из браузера
	http.SetCookie(w, &http.Cookie{
		Name:     "fluxora_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Выход выполнен успешно",
	})
}
