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
 if _, err := database.Exec(r.Context(), "UPDATE users SET password_hash = $2 WHERE id = $1", userID, string(nextHash)); err != nil { http.Error(w, `{"error":"Не удалось сохранить новый пароль"}`, http.StatusInternalServerError); return }
 _ = json.NewEncoder(w).Encode(map[string]string{"message":"Пароль успешно изменён"})
}