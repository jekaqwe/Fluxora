package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type dashboardResponse struct {
	TotalBalance string                `json:"totalBalance"`
	MonthIncome  string                `json:"monthIncome"`
	MonthExpense string                `json:"monthExpense"`
	MonthNet     string                `json:"monthNet"`
	Recent       []transactionResponse `json:"recent"`
}

func dashboardHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}
	var result dashboardResponse
	err = database.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(a.initial_balance + COALESCE((SELECT SUM(CASE
			WHEN t.operation_type = 'income' AND t.account_id = a.id THEN t.amount
			WHEN t.operation_type = 'expense' AND t.account_id = a.id THEN -t.amount
			WHEN t.operation_type = 'transfer' AND t.account_id = a.id THEN -t.amount
			WHEN t.operation_type = 'transfer' AND t.transfer_account_id = a.id THEN t.amount
			ELSE 0 END) FROM transactions t
			WHERE t.user_id = a.user_id AND (t.account_id = a.id OR t.transfer_account_id = a.id)), 0)), 0)::text
		FROM accounts a WHERE a.user_id = $1
	`, userID).Scan(&result.TotalBalance)
	if err != nil {
		http.Error(w, `{"error":"Не удалось рассчитать общий баланс"}`, http.StatusInternalServerError)
		return
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	err = database.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(amount) FILTER (WHERE operation_type = 'income'), 0)::text,
		       COALESCE(SUM(amount) FILTER (WHERE operation_type = 'expense'), 0)::text,
		       (COALESCE(SUM(amount) FILTER (WHERE operation_type = 'income'), 0) -
		        COALESCE(SUM(amount) FILTER (WHERE operation_type = 'expense'), 0))::text
		FROM transactions WHERE user_id = $1 AND operation_date >= $2::date AND operation_date < $3::date
	`, userID, start, end).Scan(&result.MonthIncome, &result.MonthExpense, &result.MonthNet)
	if err != nil {
		http.Error(w, `{"error":"Не удалось рассчитать показатели месяца"}`, http.StatusInternalServerError)
		return
	}
	rows, err := database.Query(r.Context(), `
		SELECT id, title, operation_type, amount::text, category, operation_date,
		       account_id, transfer_account_id, comment, created_at
		FROM transactions WHERE user_id = $1
		ORDER BY operation_date DESC, id DESC LIMIT 5
	`, userID)
	if err != nil {
		http.Error(w, `{"error":"Не удалось загрузить последние операции"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	result.Recent = make([]transactionResponse, 0)
	for rows.Next() {
		item, err := scanTransaction(rows)
		if err != nil {
			http.Error(w, `{"error":"Ошибка чтения последних операций"}`, http.StatusInternalServerError)
			return
		}
		result.Recent = append(result.Recent, item)
	}
	_ = json.NewEncoder(w).Encode(result)
}
