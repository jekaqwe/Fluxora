package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func integrationStatusHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := authenticatedUserID(r, database); err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		return
	}
	s3Configured := strings.TrimSpace(os.Getenv("S3_ENDPOINT")) != "" &&
		strings.TrimSpace(os.Getenv("S3_BUCKET")) != "" &&
		strings.TrimSpace(os.Getenv("S3_REGION")) != "" &&
		strings.TrimSpace(os.Getenv("S3_ACCESS_KEY_ID")) != "" &&
		strings.TrimSpace(os.Getenv("S3_SECRET_ACCESS_KEY")) != ""
	smtpConfigured := strings.TrimSpace(os.Getenv("SMTP_HOST")) != "" &&
		strings.TrimSpace(os.Getenv("SMTP_FROM")) != ""
	_ = json.NewEncoder(w).Encode(map[string]any{
		"s3": map[string]any{
			"configured": s3Configured,
			"note":       "Проверяется наличие переменных окружения, а не доступность бакета",
		},
		"email": map[string]any{
			"configured": smtpConfigured,
			"note":       "Для отправки требуется SMTP с поддержкой STARTTLS",
		},
	})
}
