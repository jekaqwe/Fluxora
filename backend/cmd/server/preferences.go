package main

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userPreferences struct {
	Theme         string          `json:"theme"`
	Notifications map[string]bool `json:"notifications"`
	Privacy       map[string]bool `json:"privacy"`
}

type preferencesPatch struct {
	Theme         *string          `json:"theme"`
	Notifications *map[string]bool `json:"notifications"`
	Privacy       *map[string]bool `json:"privacy"`
}

func defaultUserPreferences() userPreferences {
	return userPreferences{
		Theme: "light",
		Notifications: map[string]bool{
			"operations": true,
			"goals":      true,
			"family":     true,
		},
		Privacy: map[string]bool{
			"profile":      true,
			"achievements": true,
			"activity":     true,
		},
	}
}

func mergePreferenceFlags(current map[string]bool, patch map[string]bool, allowed map[string]bool) (map[string]bool, bool) {
	for key := range patch {
		if !allowed[key] {
			return current, false
		}
	}
	if current == nil {
		current = make(map[string]bool)
	}
	for key, value := range patch {
		current[key] = value
	}
	return current, true
}

func preferencesHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		result := defaultUserPreferences()
		var raw []byte
		err := database.QueryRow(r.Context(), `SELECT preferences FROM user_preferences WHERE user_id=$1`, userID).Scan(&raw)
		if err == nil {
			if err := json.Unmarshal(raw, &result); err != nil {
				http.Error(w, `{"error":"Не удалось прочитать настройки"}`, http.StatusInternalServerError)
				return
			}
			defaults := defaultUserPreferences()
			if result.Notifications == nil { result.Notifications = make(map[string]bool) }
			if result.Privacy == nil { result.Privacy = make(map[string]bool) }
			if result.Theme == "" {
				result.Theme = defaults.Theme
			}
			for key, value := range defaults.Notifications {
				if _, ok := result.Notifications[key]; !ok {
					result.Notifications[key] = value
				}
			}
			for key, value := range defaults.Privacy {
				if _, ok := result.Privacy[key]; !ok {
					result.Privacy[key] = value
				}
			}
		} else if err != pgx.ErrNoRows {
			http.Error(w, `{"error":"Не удалось загрузить настройки"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(result)

	case http.MethodPatch:
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		var patch preferencesPatch
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&patch); err != nil {
			http.Error(w, `{"error":"Некорректные настройки"}`, http.StatusBadRequest)
			return
		}
		if patch.Theme != nil && *patch.Theme != "light" && *patch.Theme != "dark" && *patch.Theme != "auto" {
			http.Error(w, `{"error":"Неизвестная тема оформления"}`, http.StatusBadRequest)
			return
		}

		tx, err := database.Begin(r.Context())
		if err != nil {
			http.Error(w, `{"error":"Не удалось сохранить настройки"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())

		_, err = tx.Exec(r.Context(), `INSERT INTO user_preferences(user_id) VALUES($1) ON CONFLICT(user_id) DO NOTHING`, userID)
		if err != nil {
			http.Error(w, `{"error":"Не удалось подготовить настройки"}`, http.StatusInternalServerError)
			return
		}
		var raw []byte
		if err := tx.QueryRow(r.Context(), `SELECT preferences FROM user_preferences WHERE user_id=$1 FOR UPDATE`, userID).Scan(&raw); err != nil {
			http.Error(w, `{"error":"Не удалось загрузить текущие настройки"}`, http.StatusInternalServerError)
			return
		}
		current := defaultUserPreferences()
		if err := json.Unmarshal(raw, &current); err != nil {
			http.Error(w, `{"error":"Не удалось прочитать текущие настройки"}`, http.StatusInternalServerError)
			return
		}
		if current.Notifications == nil { current.Notifications = make(map[string]bool) }
		if current.Privacy == nil { current.Privacy = make(map[string]bool) }
		if patch.Theme != nil {
			current.Theme = *patch.Theme
		}
		if patch.Notifications != nil {
			var ok bool
			current.Notifications, ok = mergePreferenceFlags(current.Notifications, *patch.Notifications, map[string]bool{"operations": true, "goals": true, "family": true})
			if !ok {
				http.Error(w, `{"error":"Неизвестный параметр уведомлений"}`, http.StatusBadRequest)
				return
			}
		}
		if patch.Privacy != nil {
			var ok bool
			current.Privacy, ok = mergePreferenceFlags(current.Privacy, *patch.Privacy, map[string]bool{"profile": true, "achievements": true, "activity": true})
			if !ok {
				http.Error(w, `{"error":"Неизвестный параметр приватности"}`, http.StatusBadRequest)
				return
			}
		}
		encoded, err := json.Marshal(current)
		if err != nil {
			http.Error(w, `{"error":"Не удалось подготовить настройки"}`, http.StatusInternalServerError)
			return
		}
		if _, err := tx.Exec(r.Context(), `UPDATE user_preferences SET preferences=$2::jsonb,updated_at=NOW() WHERE user_id=$1`, userID, string(encoded)); err != nil {
			http.Error(w, `{"error":"Не удалось сохранить настройки"}`, http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			http.Error(w, `{"error":"Не удалось завершить сохранение настроек"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(current)

	default:
		w.Header().Set("Allow", "GET, PATCH")
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
	}
}
