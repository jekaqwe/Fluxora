package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var accountAmountPattern = regexp.MustCompile("^-?(?:0|[1-9][0-9]{0,17})(?:\\.[0-9]{1,2})?$")

type accountPayload struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	InitialBalance string `json:"initialBalance"`
}

type accountResponse struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	InitialBalance string    `json:"initialBalance"`
	Currency       string    `json:"currency"`
	CreatedAt      time.Time `json:"createdAt"`
}

func accountsHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := database.Query(r.Context(), `
			SELECT id, name, account_type, initial_balance::text, currency, created_at
			FROM accounts WHERE user_id = $1 ORDER BY created_at, id
		`, userID)
		if err != nil {
			http.Error(w, `{"error":"Не удалось загрузить счета"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		result := make([]accountResponse, 0)
		for rows.Next() {
			var item accountResponse
			if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.InitialBalance, &item.Currency, &item.CreatedAt); err != nil {
				http.Error(w, `{"error":"Не удалось прочитать счета"}`, http.StatusInternalServerError)
				return
			}
			result = append(result, item)
		}
		if rows.Err() != nil {
			http.Error(w, `{"error":"Ошибка чтения счетов"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(result)

	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		var input accountPayload
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error":"Некорректные данные"}`, http.StatusBadRequest)
			return
		}
		input.Name = strings.TrimSpace(input.Name)
		input.Type = strings.TrimSpace(input.Type)
		input.InitialBalance = strings.TrimSpace(input.InitialBalance)
		if input.Name == "" || len([]rune(input.Name)) > 100 {
			http.Error(w, `{"error":"Название счёта должно содержать от 1 до 100 символов"}`, http.StatusBadRequest)
			return
		}
		switch input.Type {
		case "cash", "bank", "card", "savings", "other":
		default:
			http.Error(w, `{"error":"Неизвестный тип счёта"}`, http.StatusBadRequest)
			return
		}
		if input.InitialBalance == "" {
			input.InitialBalance = "0"
		}
		if !accountAmountPattern.MatchString(input.InitialBalance) {
			http.Error(w, `{"error":"Баланс должен быть числом с точностью до двух знаков после запятой"}`, http.StatusBadRequest)
			return
		}
		var item accountResponse
		err := database.QueryRow(r.Context(), `
			INSERT INTO accounts (user_id, name, account_type, initial_balance)
			VALUES ($1, $2, $3, $4::numeric)
			RETURNING id, name, account_type, initial_balance::text, currency, created_at
		`, userID, input.Name, input.Type, input.InitialBalance).Scan(
			&item.ID, &item.Name, &item.Type, &item.InitialBalance, &item.Currency, &item.CreatedAt,
		)
		if err != nil {
			http.Error(w, `{"error":"Не удалось создать счёт"}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(item)

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
	}
}
