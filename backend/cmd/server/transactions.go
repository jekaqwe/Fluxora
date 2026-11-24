package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type transactionPayload struct {
	Title             string `json:"title"`
	Type              string `json:"type"`
	Amount            string `json:"amount"`
	Category          string `json:"category"`
	Date              string `json:"date"`
	AccountID         int64  `json:"accountId"`
	TransferAccountID int64  `json:"transferAccountId"`
	Comment           string `json:"comment"`
	IdempotencyKey    string `json:"idempotencyKey"`
}

type transactionResponse struct {
	ID                int64     `json:"id"`
	Title             string    `json:"title"`
	Type              string    `json:"type"`
	Amount            string    `json:"amount"`
	Category          string    `json:"category"`
	Date              string    `json:"date"`
	AccountID         int64     `json:"accountId"`
	TransferAccountID *int64    `json:"transferAccountId,omitempty"`
	Comment           string    `json:"comment"`
	CreatedAt         time.Time `json:"createdAt"`
}

func scanTransaction(row interface{ Scan(...any) error }) (transactionResponse, error) {
	var item transactionResponse
	var transferID pgtype.Int8
	var date time.Time
	err := row.Scan(&item.ID, &item.Title, &item.Type, &item.Amount, &item.Category, &date,
		&item.AccountID, &transferID, &item.Comment, &item.CreatedAt)
	item.Date = date.Format("2006-01-02")
	if transferID.Valid {
		value := transferID.Int64
		item.TransferAccountID = &value
	}
	return item, err
}

func validateTransactionPayload(input *transactionPayload) (time.Time, *int64, string) {
	input.Title = strings.TrimSpace(input.Title)
	input.Type = strings.TrimSpace(input.Type)
	input.Amount = strings.TrimSpace(input.Amount)
	input.Category = strings.TrimSpace(input.Category)
	input.Comment = strings.TrimSpace(input.Comment)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.Title == "" || len([]rune(input.Title)) > 160 {
		return time.Time{}, nil, "Название операции должно содержать от 1 до 160 символов"
	}
	if !accountAmountPattern.MatchString(input.Amount) || strings.HasPrefix(input.Amount, "-") || strings.Trim(input.Amount, "0.") == "" {
		return time.Time{}, nil, "Сумма должна быть положительным числом с точностью до двух знаков"
	}
	if input.Category == "" || len([]rune(input.Category)) > 80 || len([]rune(input.Comment)) > 1000 {
		return time.Time{}, nil, "Проверьте категорию и комментарий"
	}
	if input.AccountID <= 0 {
		return time.Time{}, nil, "Выберите счёт"
	}
	parsedDate, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return time.Time{}, nil, "Укажите корректную дату операции"
	}
	var transferID *int64
	switch input.Type {
	case "income", "expense":
		if input.TransferAccountID != 0 {
			return time.Time{}, nil, "Для дохода или расхода нельзя указывать второй счёт"
		}
	case "transfer":
		if input.TransferAccountID <= 0 || input.TransferAccountID == input.AccountID {
			return time.Time{}, nil, "Для перевода выберите другой счёт"
		}
		value := input.TransferAccountID
		transferID = &value
	default:
		return time.Time{}, nil, "Неизвестный тип операции"
	}
	if input.IdempotencyKey != "" && len(input.IdempotencyKey) > 100 {
		return time.Time{}, nil, "Некорректный ключ операции"
	}
	return parsedDate, transferID, ""
}

func transactionsHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	if r.URL.Path != "/api/transactions" {
		transactionItemHandler(w, r, database)
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
			SELECT id, title, operation_type, amount::text, category, operation_date,
			       account_id, transfer_account_id, comment, created_at
			FROM transactions WHERE user_id = $1
			ORDER BY operation_date DESC, id DESC LIMIT 500
		`, userID)
		if err != nil {
			http.Error(w, `{"error":"Не удалось загрузить операции"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		items := make([]transactionResponse, 0)
		for rows.Next() {
			item, err := scanTransaction(rows)
			if err != nil {
				http.Error(w, `{"error":"Не удалось прочитать операции"}`, http.StatusInternalServerError)
				return
			}
			items = append(items, item)
		}
		if rows.Err() != nil {
			http.Error(w, `{"error":"Ошибка чтения операций"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
		var input transactionPayload
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error":"Некорректные данные"}`, http.StatusBadRequest)
			return
		}
		date, transferID, validationError := validateTransactionPayload(&input)
		if validationError != "" || input.IdempotencyKey == "" {
			if validationError == "" { validationError = "Отсутствует ключ идемпотентности" }
			http.Error(w, `{"error":"`+validationError+`"}`, http.StatusBadRequest)
			return
		}
		tx, err := database.Begin(r.Context())
		if err != nil {
			http.Error(w, `{"error":"Не удалось начать создание операции"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())
		var exists bool
		if err := tx.QueryRow(r.Context(), "SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND user_id = $2)", input.AccountID, userID).Scan(&exists); err != nil || !exists {
			http.Error(w, `{"error":"Исходный счёт не найден"}`, http.StatusBadRequest)
			return
		}
		if transferID != nil {
			if err := tx.QueryRow(r.Context(), "SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND user_id = $2)", *transferID, userID).Scan(&exists); err != nil || !exists {
				http.Error(w, `{"error":"Счёт получения не найден"}`, http.StatusBadRequest)
				return
			}
		}
		item, err := scanTransaction(tx.QueryRow(r.Context(), `
			INSERT INTO transactions (user_id, account_id, transfer_account_id, title, operation_type, amount, category, operation_date, comment, idempotency_key)
			VALUES ($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9,$10)
			ON CONFLICT (user_id, idempotency_key) DO NOTHING
			RETURNING id, title, operation_type, amount::text, category, operation_date,
			          account_id, transfer_account_id, comment, created_at
		`, userID, input.AccountID, transferID, input.Title, input.Type, input.Amount, input.Category, date, input.Comment, input.IdempotencyKey))
		created := true
		if err == pgx.ErrNoRows {
			created = false
			item, err = scanTransaction(tx.QueryRow(r.Context(), `
				SELECT id, title, operation_type, amount::text, category, operation_date,
				       account_id, transfer_account_id, comment, created_at
				FROM transactions WHERE user_id = $1 AND idempotency_key = $2
			`, userID, input.IdempotencyKey))
		}
		if err != nil {
			http.Error(w, `{"error":"Не удалось сохранить операцию"}`, http.StatusInternalServerError)
			return
		}
		if created {
			_, _ = tx.Exec(r.Context(), "INSERT INTO notifications(user_id,notification_type,title,message,related_type,related_id) VALUES($1,'transaction','Операция добавлена', $2, 'transaction', $3)", userID, input.Title, item.ID)
		}
		if err := tx.Commit(r.Context()); err != nil {
			http.Error(w, `{"error":"Не удалось завершить создание операции"}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(item)

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
	}
}

func transactionItemHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}
	rawID := strings.TrimPrefix(r.URL.Path, "/api/transactions/")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 || strings.Contains(rawID, "/") {
		http.Error(w, `{"error":"Некорректный идентификатор операции"}`, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		tag, err := database.Exec(r.Context(), "DELETE FROM transactions WHERE id = $1 AND user_id = $2", id, userID)
		if err != nil {
			http.Error(w, `{"error":"Не удалось удалить операцию"}`, http.StatusInternalServerError)
			return
		}
		if tag.RowsAffected() == 0 {
			http.Error(w, `{"error":"Операция не найдена"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})

	case http.MethodPatch:
		r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
		var input transactionPayload
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error":"Некорректные данные"}`, http.StatusBadRequest)
			return
		}
		date, transferID, validationError := validateTransactionPayload(&input)
		if validationError != "" {
			http.Error(w, `{"error":"`+validationError+`"}`, http.StatusBadRequest)
			return
		}
		tx, err := database.Begin(r.Context())
		if err != nil {
			http.Error(w, `{"error":"Не удалось начать изменение операции"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())
		var exists bool
		if err := tx.QueryRow(r.Context(), "SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND user_id = $2)", input.AccountID, userID).Scan(&exists); err != nil || !exists {
			http.Error(w, `{"error":"Исходный счёт не найден"}`, http.StatusBadRequest)
			return
		}
		if transferID != nil {
			if err := tx.QueryRow(r.Context(), "SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1 AND user_id = $2)", *transferID, userID).Scan(&exists); err != nil || !exists {
				http.Error(w, `{"error":"Счёт получения не найден"}`, http.StatusBadRequest)
				return
			}
		}
		item, err := scanTransaction(tx.QueryRow(r.Context(), `
			UPDATE transactions SET account_id=$3, transfer_account_id=$4, title=$5, operation_type=$6,
				amount=$7::numeric, category=$8, operation_date=$9, comment=$10, updated_at=NOW()
			WHERE id=$1 AND user_id=$2
			RETURNING id, title, operation_type, amount::text, category, operation_date,
			          account_id, transfer_account_id, comment, created_at
		`, id, userID, input.AccountID, transferID, input.Title, input.Type, input.Amount, input.Category, date, input.Comment))
		if err != nil {
			http.Error(w, `{"error":"Операция не найдена или не удалось сохранить изменения"}`, http.StatusNotFound)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			http.Error(w, `{"error":"Не удалось завершить изменение операции"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(item)

	default:
		w.Header().Set("Allow", "PATCH, DELETE")
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
	}
}
