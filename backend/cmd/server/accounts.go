package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
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
	CurrentBalance string    `json:"currentBalance"`
	Currency       string    `json:"currency"`
	CreatedAt      time.Time `json:"createdAt"`
}

func accountsHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	if r.URL.Path != "/api/accounts" {
		accountItemHandler(w, r, database)
		return
	}
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
			SELECT a.id, a.name, a.account_type, a.initial_balance::text,
			       (a.initial_balance + COALESCE((SELECT SUM(CASE
			           WHEN t.operation_type = 'income' AND t.account_id = a.id THEN t.amount
			           WHEN t.operation_type = 'expense' AND t.account_id = a.id THEN -t.amount
			           WHEN t.operation_type = 'transfer' AND t.account_id = a.id THEN -t.amount
			           WHEN t.operation_type = 'transfer' AND t.transfer_account_id = a.id THEN t.amount
			           ELSE 0 END) FROM transactions t
			           WHERE t.user_id = a.user_id AND (t.account_id = a.id OR t.transfer_account_id = a.id)), 0))::text,
			       a.currency, a.created_at
			FROM accounts a WHERE a.user_id = $1 ORDER BY a.created_at, a.id
		`, userID)
		if err != nil {
			http.Error(w, `{"error":"Не удалось загрузить счета"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		result := make([]accountResponse, 0)
		for rows.Next() {
			var item accountResponse
			if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.InitialBalance, &item.CurrentBalance, &item.Currency, &item.CreatedAt); err != nil {
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
			RETURNING id, name, account_type, initial_balance::text,
			(initial_balance + COALESCE((SELECT SUM(CASE
			    WHEN t.operation_type = 'income' AND t.account_id = accounts.id THEN t.amount
			    WHEN t.operation_type = 'expense' AND t.account_id = accounts.id THEN -t.amount
			    WHEN t.operation_type = 'transfer' AND t.account_id = accounts.id THEN -t.amount
			    WHEN t.operation_type = 'transfer' AND t.transfer_account_id = accounts.id THEN t.amount
			    ELSE 0 END) FROM transactions t
			    WHERE t.user_id = accounts.user_id AND (t.account_id = accounts.id OR t.transfer_account_id = accounts.id)), 0))::text,
			currency, created_at
		`, userID, input.Name, input.Type, input.InitialBalance).Scan(
			&item.ID, &item.Name, &item.Type, &item.InitialBalance, &item.CurrentBalance, &item.Currency, &item.CreatedAt,
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

func accountItemHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}
	rawID := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	accountID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || accountID <= 0 || strings.Contains(rawID, "/") {
		http.Error(w, `{"error":"Некорректный идентификатор счёта"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		var input accountPayload
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error":"Некорректные данные"}`, http.StatusBadRequest)
			return
		}
		input.Name = strings.TrimSpace(input.Name)
		input.Type = strings.TrimSpace(input.Type)
		input.InitialBalance = strings.TrimSpace(input.InitialBalance)
		if input.Name == "" || len([]rune(input.Name)) > 100 || !accountAmountPattern.MatchString(input.InitialBalance) {
			http.Error(w, `{"error":"Проверьте название и начальный баланс"}`, http.StatusBadRequest)
			return
		}
		switch input.Type {
		case "cash", "bank", "card", "savings", "other":
		default:
			http.Error(w, `{"error":"Неизвестный тип счёта"}`, http.StatusBadRequest)
			return
		}
		var item accountResponse
		err := database.QueryRow(r.Context(), `
			UPDATE accounts SET name = $3, account_type = $4, initial_balance = $5::numeric, updated_at = NOW()
			WHERE id = $1 AND user_id = $2
			RETURNING id, name, account_type, initial_balance::text,
			(initial_balance + COALESCE((SELECT SUM(CASE
			    WHEN t.operation_type = 'income' AND t.account_id = accounts.id THEN t.amount
			    WHEN t.operation_type = 'expense' AND t.account_id = accounts.id THEN -t.amount
			    WHEN t.operation_type = 'transfer' AND t.account_id = accounts.id THEN -t.amount
			    WHEN t.operation_type = 'transfer' AND t.transfer_account_id = accounts.id THEN t.amount
			    ELSE 0 END) FROM transactions t
			    WHERE t.user_id = accounts.user_id AND (t.account_id = accounts.id OR t.transfer_account_id = accounts.id)), 0))::text,
			currency, created_at
		`, accountID, userID, input.Name, input.Type, input.InitialBalance).Scan(
			&item.ID, &item.Name, &item.Type, &item.InitialBalance, &item.CurrentBalance, &item.Currency, &item.CreatedAt,
		)
		if err != nil {
			http.Error(w, `{"error":"Счёт не найден или не удалось сохранить изменения"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(item)

	case http.MethodDelete:
		tag, err := database.Exec(r.Context(), "DELETE FROM accounts WHERE id = $1 AND user_id = $2", accountID, userID)
		if err != nil {
			http.Error(w, `{"error":"Не удалось удалить счёт"}`, http.StatusInternalServerError)
			return
		}
		if tag.RowsAffected() == 0 {
			http.Error(w, `{"error":"Счёт не найден"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})

	default:
		w.Header().Set("Allow", "PATCH, DELETE")
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
	}
}
