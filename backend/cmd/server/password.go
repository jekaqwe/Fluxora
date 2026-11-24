package main

import ("encoding/json"; "net/http"; "strings";
"github.com/jackc/pgx/v5/pgxpool"; "golang.org/x/crypto/bcrypt")

type passwordChangePayload struct { Current string `json:"currentPassword"`; Next string `json:"newPassword"` }

func passwordChangeHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
 w.Header().Set("Content-Type", "application/json")
 if r.Method != http.MethodPost { http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed); return }
 userID, err := authenticatedUserID(r, database)
 if err != nil { http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized); return }
 r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
 var input passwordChangePayload
 if err := json.NewDecoder(r.Body).Decode(&input); err != nil { http.Error(w, `{"error":"Некорректные данные"}`, http.StatusBadRequest); return }
 input.Current = strings.TrimSpace(input.Current)
 if input.Current == "" || len(input.Next) < 8 || len(input.Next) > 72 { http.Error(w, `{"error":"Новый пароль должен содержать от 8 до 72 байт"}`, http.StatusBadRequest); return }
 var oldHash string
 if err := database.QueryRow(r.Context(), "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&oldHash); err != nil { http.Error(w, `{"error":"Не удалось проверить текущий пароль"}`, http.StatusInternalServerError); return }
 if bcrypt.CompareHashAndPassword([]byte(oldHash), []byte(input.Current)) != nil { http.Error(w, `{"error":"Текущий пароль указан неверно"}`, http.StatusUnauthorized); return }
 nextHash, err := bcrypt.GenerateFromPassword([]byte(input.Next), bcrypt.DefaultCost)
 if err != nil { http.Error(w, `{"error":"Не удалось обработать новый пароль"}`, http.StatusInternalServerError); return }
 sessionHash, err := currentSessionHash(r)
 if err != nil { http.Error(w, `{"error":"Текущая сессия не найдена"}`, http.StatusUnauthorized); return }
 tx, err := database.Begin(r.Context())
 if err != nil { http.Error(w, `{"error":"Не удалось начать смену пароля"}`, http.StatusInternalServerError); return }
 defer tx.Rollback(r.Context())
 if _, err := tx.Exec(r.Context(), "UPDATE users SET password_hash = $2 WHERE id = $1", userID, string(nextHash)); err != nil { http.Error(w, `{"error":"Не удалось сохранить новый пароль"}`, http.StatusInternalServerError); return }
 if _, err := tx.Exec(r.Context(), "DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2", userID, sessionHash); err != nil { http.Error(w, `{"error":"Не удалось завершить остальные сессии"}`, http.StatusInternalServerError); return }
 _, _ = tx.Exec(r.Context(), "INSERT INTO notifications(user_id,notification_type,title,message,related_type) VALUES($1,'account_password','Пароль изменён','Пароль аккаунта успешно обновлён. Остальные сессии завершены.','account')", userID)
 if err := tx.Commit(r.Context()); err != nil { http.Error(w, `{"error":"Не удалось завершить смену пароля"}`, http.StatusInternalServerError); return }
 _ = json.NewEncoder(w).Encode(map[string]string{"message":"Пароль успешно изменён. Остальные сессии завершены."})
}