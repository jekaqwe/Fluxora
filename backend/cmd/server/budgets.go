package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type categoryBudget struct {
	ID int64 `json:"id"`
	Category string `json:"category"`
	MonthlyLimit string `json:"monthlyLimit"`
	Spent string `json:"spent"`
	Remaining string `json:"remaining"`
}

func budgetsHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, "{\"error\":\"Пользователь не авторизован\"}", http.StatusUnauthorized)
		return
	}
	if r.URL.Path == "/api/budgets" {
		switch r.Method {
		case http.MethodGet:
			rows, err := database.Query(r.Context(), "SELECT b.id,b.category,b.monthly_limit::text,COALESCE(SUM(t.amount),0)::numeric(20,2)::text,(b.monthly_limit-COALESCE(SUM(t.amount),0))::numeric(20,2)::text FROM category_budgets b LEFT JOIN transactions t ON t.user_id=b.user_id AND t.operation_type='expense' AND lower(t.category)=lower(b.category) AND t.operation_date>=date_trunc('month',CURRENT_DATE)::date AND t.operation_date<CURRENT_DATE+1 WHERE b.user_id=$1 GROUP BY b.id ORDER BY lower(b.category),b.id", userID)
			if err != nil { http.Error(w, "{\"error\":\"Не удалось загрузить бюджеты категорий\"}", 500); return }
			defer rows.Close()
			items := make([]categoryBudget, 0)
			for rows.Next() {
				var item categoryBudget
				if err := rows.Scan(&item.ID,&item.Category,&item.MonthlyLimit,&item.Spent,&item.Remaining); err != nil { http.Error(w, "{\"error\":\"Ошибка чтения бюджетов\"}", 500); return }
				items = append(items,item)
			}
			if err := rows.Err(); err != nil { http.Error(w, "{\"error\":\"Ошибка чтения бюджетов\"}", 500); return }
			_ = json.NewEncoder(w).Encode(items)
		case http.MethodPost:
			r.Body = http.MaxBytesReader(w,r.Body,8<<10)
			var input struct {
				Category string `json:"category"`
				Amount string `json:"amount"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil { http.Error(w, "{\"error\":\"Некорректные данные бюджета\"}", 400); return }
			input.Category = strings.TrimSpace(input.Category)
			input.Amount = strings.TrimSpace(strings.ReplaceAll(input.Amount,",","."))
			if input.Category=="" || len([]rune(input.Category))>80 || !accountAmountPattern.MatchString(input.Amount) || strings.HasPrefix(input.Amount,"-") || strings.Trim(input.Amount,"0.")=="" {
				http.Error(w, "{\"error\":\"Укажите категорию и положительный месячный лимит\"}",400); return
			}
			var item categoryBudget
			err = database.QueryRow(r.Context(), "INSERT INTO category_budgets(user_id,category,monthly_limit) VALUES($1,$2,$3::numeric) ON CONFLICT (user_id, (lower(category))) DO UPDATE SET category=EXCLUDED.category,monthly_limit=EXCLUDED.monthly_limit,updated_at=NOW() RETURNING id,category,monthly_limit::text", userID,input.Category,input.Amount).Scan(&item.ID,&item.Category,&item.MonthlyLimit)
			if err != nil { http.Error(w, "{\"error\":\"Не удалось сохранить бюджет категории\"}",500); return }
			_ = database.QueryRow(r.Context(), "SELECT COALESCE(SUM(t.amount),0)::numeric(20,2)::text,(b.monthly_limit-COALESCE(SUM(t.amount),0))::numeric(20,2)::text FROM category_budgets b LEFT JOIN transactions t ON t.user_id=b.user_id AND t.operation_type='expense' AND lower(t.category)=lower(b.category) AND t.operation_date>=date_trunc('month',CURRENT_DATE)::date AND t.operation_date<CURRENT_DATE+1 WHERE b.id=$1 AND b.user_id=$2 GROUP BY b.id",item.ID,userID).Scan(&item.Spent,&item.Remaining)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(item)
		default:
			w.Header().Set("Allow","GET, POST")
			http.Error(w, "{\"error\":\"Метод не поддерживается\"}",405)
		}
		return
	}
	rawID := strings.Trim(strings.TrimPrefix(r.URL.Path,"/api/budgets/"),"/")
	id,err := strconv.ParseInt(rawID,10,64)
	if err != nil || id<=0 { http.Error(w, "{\"error\":\"Некорректный бюджет\"}",400); return }
	if r.Method != http.MethodDelete { w.Header().Set("Allow","DELETE"); http.Error(w, "{\"error\":\"Метод не поддерживается\"}",405); return }
	tag,err := database.Exec(r.Context(),"DELETE FROM category_budgets WHERE id=$1 AND user_id=$2",id,userID)
	if err != nil { http.Error(w, "{\"error\":\"Не удалось удалить бюджет\"}",500); return }
	if tag.RowsAffected()==0 { http.Error(w, "{\"error\":\"Бюджет не найден\"}",404); return }
	_ = json.NewEncoder(w).Encode(map[string]bool{"success":true})
}
