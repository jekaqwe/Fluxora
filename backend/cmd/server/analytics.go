package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type analyticsCategory struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
	Count  int64  `json:"count"`
}

type analyticsPoint struct {
	Date    string `json:"date"`
	Income  string `json:"income"`
	Expense string `json:"expense"`
}

type analyticsTransaction struct {
	Title  string `json:"title"`
	Amount string `json:"amount"`
	Date   string `json:"date"`
}

type analyticsResponse struct {
	Period        string                 `json:"period"`
	From          string                 `json:"from"`
	To            string                 `json:"to"`
	Income        string                 `json:"income"`
	Expense       string                 `json:"expense"`
	Net           string                 `json:"net"`
	AverageExpense string                `json:"averageExpense"`
	Categories    []analyticsCategory    `json:"categories"`
	Trend         []analyticsPoint       `json:"trend"`
	Largest       []analyticsTransaction `json:"largest"`
}

func analyticsRange(r *http.Request) (time.Time, time.Time, string, bool) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	period := r.URL.Query().Get("period")
	switch period {
	case "", "month":
		start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0), "month", true
	case "week":
		offset := (int(today.Weekday()) + 6) % 7
		start := today.AddDate(0, 0, -offset)
		return start, start.AddDate(0, 0, 7), "week", true
	case "year":
		start := time.Date(today.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(1, 0, 0), "year", true
	case "custom":
		from, err1 := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		to, err2 := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if err1 != nil || err2 != nil || to.Before(from) || to.Sub(from) > 366*24*time.Hour {
			return time.Time{}, time.Time{}, period, false
		}
		return from, to.AddDate(0, 0, 1), period, true
	default:
		return time.Time{}, time.Time{}, period, false
	}
}

func analyticsHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
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
	from, to, period, ok := analyticsRange(r)
	if !ok {
		http.Error(w, `{"error":"Укажите корректный период аналитики"}`, http.StatusBadRequest)
		return
	}

	result := analyticsResponse{Period: period, From: from.Format("2006-01-02"), To: to.AddDate(0, 0, -1).Format("2006-01-02"),
		Categories: make([]analyticsCategory, 0), Trend: make([]analyticsPoint, 0), Largest: make([]analyticsTransaction, 0)}
	err = database.QueryRow(r.Context(), `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE operation_type = 'income'), 0)::text,
			COALESCE(SUM(amount) FILTER (WHERE operation_type = 'expense'), 0)::text,
			(COALESCE(SUM(amount) FILTER (WHERE operation_type = 'income'), 0) -
			 COALESCE(SUM(amount) FILTER (WHERE operation_type = 'expense'), 0))::text,
			COALESCE(AVG(amount) FILTER (WHERE operation_type = 'expense'), 0)::numeric(20,2)::text
		FROM transactions
		WHERE user_id = $1 AND operation_date >= $2::date AND operation_date < $3::date
	`, userID, from, to).Scan(&result.Income, &result.Expense, &result.Net, &result.AverageExpense)
	if err != nil {
		http.Error(w, `{"error":"Не удалось рассчитать аналитику"}`, http.StatusInternalServerError)
		return
	}

	categoryRows, err := database.Query(r.Context(), `
		SELECT category, SUM(amount)::text, COUNT(*)
		FROM transactions
		WHERE user_id = $1 AND operation_type = 'expense'
		  AND operation_date >= $2::date AND operation_date < $3::date
		GROUP BY category ORDER BY SUM(amount) DESC
	`, userID, from, to)
	if err != nil {
		http.Error(w, `{"error":"Не удалось рассчитать категории"}`, http.StatusInternalServerError)
		return
	}
	for categoryRows.Next() {
		var item analyticsCategory
		if err := categoryRows.Scan(&item.Name, &item.Amount, &item.Count); err != nil {
			categoryRows.Close()
			http.Error(w, `{"error":"Ошибка чтения категорий"}`, http.StatusInternalServerError)
			return
		}
		result.Categories = append(result.Categories, item)
	}
	categoryRows.Close()

	bucket := "day"
	if period == "month" || period == "custom" {
		bucket = "day"
	} else if period == "year" {
		bucket = "month"
	}
	trendRows, err := database.Query(r.Context(), `
		SELECT TO_CHAR(date_trunc($4, operation_date::timestamp), 'YYYY-MM-DD'),
		       COALESCE(SUM(amount) FILTER (WHERE operation_type = 'income'), 0)::text,
		       COALESCE(SUM(amount) FILTER (WHERE operation_type = 'expense'), 0)::text
		FROM transactions
		WHERE user_id = $1 AND operation_date >= $2::date AND operation_date < $3::date
		GROUP BY date_trunc($4, operation_date::timestamp)
		ORDER BY date_trunc($4, operation_date::timestamp)
	`, userID, from, to, bucket)
	if err != nil {
		http.Error(w, `{"error":"Не удалось рассчитать динамику"}`, http.StatusInternalServerError)
		return
	}
	for trendRows.Next() {
		var item analyticsPoint
		if err := trendRows.Scan(&item.Date, &item.Income, &item.Expense); err != nil {
			trendRows.Close()
			http.Error(w, `{"error":"Ошибка чтения динамики"}`, http.StatusInternalServerError)
			return
		}
		result.Trend = append(result.Trend, item)
	}
	trendRows.Close()

	largestRows, err := database.Query(r.Context(), `
		SELECT title, amount::text, operation_date::text
		FROM transactions
		WHERE user_id = $1 AND operation_type = 'expense'
		  AND operation_date >= $2::date AND operation_date < $3::date
		ORDER BY amount DESC, operation_date DESC LIMIT 5
	`, userID, from, to)
	if err != nil {
		http.Error(w, `{"error":"Не удалось получить крупнейшие расходы"}`, http.StatusInternalServerError)
		return
	}
	for largestRows.Next() {
		var item analyticsTransaction
		if err := largestRows.Scan(&item.Title, &item.Amount, &item.Date); err != nil {
			largestRows.Close()
			http.Error(w, `{"error":"Ошибка чтения крупнейших расходов"}`, http.StatusInternalServerError)
			return
		}
		result.Largest = append(result.Largest, item)
	}
	largestRows.Close()
	_ = json.NewEncoder(w).Encode(result)
}
