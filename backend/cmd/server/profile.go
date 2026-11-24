package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const profileMaxAvatarSize = 5 << 20

type profileResponse struct {
	ID                 int64      `json:"id"`
	FirstName          string     `json:"firstName"`
	LastName           string     `json:"lastName"`
	Username           string     `json:"username"`
	Email              string     `json:"email"`
	AvatarURL          *string    `json:"avatarUrl"`
	FirstNameAvailable *time.Time `json:"firstNameAvailableAt"`
	LastNameAvailable  *time.Time `json:"lastNameAvailableAt"`
	UsernameAvailable  *time.Time `json:"usernameAvailableAt"`
}

func authenticatedUserID(r *http.Request, database *pgxpool.Pool) (int64, error) {
	cookie, err := r.Cookie("fluxora_session")
	if err != nil {
		return 0, errors.New("unauthorized")
	}
	hash := sha256.Sum256([]byte(cookie.Value))
	hashString := hex.EncodeToString(hash[:])
	var userID int64
	err = database.QueryRow(r.Context(), `
		SELECT user_id FROM sessions
		WHERE token_hash = $1 AND expires_at > NOW()
	`, hashString).Scan(&userID)
	if err != nil {
		return 0, errors.New("unauthorized")
	}
	return userID, nil
}

func profileHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getProfileHandler(w, r, database, userID)
	case http.MethodPatch:
		updateProfileHandler(w, r, database, userID)
	default:
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
	}
}

func getProfileHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool, userID int64) {
	var profile profileResponse
	var avatarKey *string
	err := database.QueryRow(r.Context(), `
		SELECT id, first_name, last_name, username, email, avatar_key,
		       CASE WHEN first_name_changed_at IS NULL THEN NULL ELSE first_name_changed_at + INTERVAL '14 days' END,
		       CASE WHEN last_name_changed_at IS NULL THEN NULL ELSE last_name_changed_at + INTERVAL '14 days' END,
		       CASE WHEN username_changed_at IS NULL THEN NULL ELSE username_changed_at + INTERVAL '14 days' END
		FROM users WHERE id = $1
	`, userID).Scan(
		&profile.ID, &profile.FirstName, &profile.LastName, &profile.Username, &profile.Email, &avatarKey,
		&profile.FirstNameAvailable, &profile.LastNameAvailable, &profile.UsernameAvailable,
	)
	if err != nil {
		http.Error(w, `{"error":"Не удалось загрузить профиль"}`, http.StatusInternalServerError)
		return
	}

	if avatarKey != nil && *avatarKey != "" {
		store, storeErr := newS3Store()
		if storeErr == nil {
			if signedURL, signErr := store.presignGet(*avatarKey, 15*time.Minute); signErr == nil {
				profile.AvatarURL = &signedURL
			}
		}
	}
	_ = json.NewEncoder(w).Encode(profile)
}

type updateProfileRequest struct {
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Username  *string `json:"username"`
}

func updateProfileHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool, userID int64) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var input updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"Некорректные данные"}`, http.StatusBadRequest)
		return
	}
	if input.FirstName == nil && input.LastName == nil && input.Username == nil {
		http.Error(w, `{"error":"Нет данных для изменения"}`, http.StatusBadRequest)
		return
	}

	if input.FirstName != nil {
		*input.FirstName = strings.TrimSpace(*input.FirstName)
		if *input.FirstName == "" || len([]rune(*input.FirstName)) > 100 {
			http.Error(w, `{"error":"Имя должно содержать от 1 до 100 символов"}`, http.StatusBadRequest)
			return
		}
	}
	if input.LastName != nil {
		*input.LastName = strings.TrimSpace(*input.LastName)
		if *input.LastName == "" || len([]rune(*input.LastName)) > 100 {
			http.Error(w, `{"error":"Фамилия должна содержать от 1 до 100 символов"}`, http.StatusBadRequest)
			return
		}
	}
	if input.Username != nil {
		*input.Username = strings.TrimSpace(*input.Username)
		if len([]rune(*input.Username)) < 3 || len([]rune(*input.Username)) > 50 {
			http.Error(w, `{"error":"Логин должен содержать от 3 до 50 символов"}`, http.StatusBadRequest)
			return
		}
	}

	tx, err := database.Begin(r.Context())
	if err != nil {
		http.Error(w, `{"error":"Не удалось начать изменение профиля"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	var firstNameChanged, lastNameChanged, usernameChanged *time.Time
	err = tx.QueryRow(r.Context(), `
		SELECT first_name_changed_at, last_name_changed_at, username_changed_at
		FROM users WHERE id = $1 FOR UPDATE
	`, userID).Scan(&firstNameChanged, &lastNameChanged, &usernameChanged)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не найден"}`, http.StatusNotFound)
		return
	}

	now := time.Now().UTC()
	checkCooldown := func(changedAt *time.Time) bool {
		return changedAt != nil && now.Before(changedAt.Add(14*24*time.Hour))
	}
	if input.FirstName != nil && checkCooldown(firstNameChanged) {
		http.Error(w, `{"error":"Имя можно изменить только один раз в 14 дней"}`, http.StatusConflict)
		return
	}
	if input.LastName != nil && checkCooldown(lastNameChanged) {
		http.Error(w, `{"error":"Фамилию можно изменить только один раз в 14 дней"}`, http.StatusConflict)
		return
	}
	if input.Username != nil && checkCooldown(usernameChanged) {
		http.Error(w, `{"error":"Логин можно изменить только один раз в 14 дней"}`, http.StatusConflict)
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE users SET
			first_name = COALESCE($2, first_name),
			last_name = COALESCE($3, last_name),
			username = COALESCE($4, username),
			first_name_changed_at = CASE WHEN $2::text IS NULL THEN first_name_changed_at ELSE $5 END,
			last_name_changed_at = CASE WHEN $3::text IS NULL THEN last_name_changed_at ELSE $5 END,
			username_changed_at = CASE WHEN $4::text IS NULL THEN username_changed_at ELSE $5 END
		WHERE id = $1
	`, userID, input.FirstName, input.LastName, input.Username, now)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, `{"error":"Этот логин уже занят"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"Не удалось сохранить изменения"}`, http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, `{"error":"Не удалось сохранить изменения"}`, http.StatusInternalServerError)
		return
	}
	getProfileHandler(w, r, database, userID)
}

func avatarHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")
	userID, err := authenticatedUserID(r, database)
	if err != nil {
		http.Error(w, `{"error":"Пользователь не авторизован"}`, http.StatusUnauthorized)
		return
	}
	store, err := newS3Store()
	if err != nil {
		http.Error(w, `{"error":"Хранилище файлов пока не настроено"}`, http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodPut:
		uploadAvatarHandler(w, r, database, store, userID)
	case http.MethodDelete:
		deleteAvatarHandler(w, r, database, store, userID)
	default:
		http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
	}
}

func uploadAvatarHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool, store *s3Store, userID int64) {
	r.Body = http.MaxBytesReader(w, r.Body, profileMaxAvatarSize+(1<<20))
	if err := r.ParseMultipartForm(profileMaxAvatarSize); err != nil {
		http.Error(w, `{"error":"Файл слишком большой или некорректный"}`, http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, `{"error":"Выберите изображение"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, profileMaxAvatarSize+1))
	if err != nil || len(data) == 0 || len(data) > profileMaxAvatarSize {
		http.Error(w, `{"error":"Размер изображения должен быть не более 5 МБ"}`, http.StatusBadRequest)
		return
	}
	contentType := http.DetectContentType(data)
	extensions := map[string]string{
		"image/jpeg": ".jpg",
		"image/png": ".png",
		"image/webp": ".webp",
		"image/gif": ".gif",
	}
	extension, ok := extensions[contentType]
	if !ok {
		http.Error(w, `{"error":"Поддерживаются JPEG, PNG, WebP и GIF"}`, http.StatusBadRequest)
		return
	}

	objectID, err := randomObjectID()
	if err != nil {
		http.Error(w, `{"error":"Не удалось подготовить файл"}`, http.StatusInternalServerError)
		return
	}
	key := path.Join("avatars", fmt.Sprintf("%d", userID), objectID+extension)
	if err := store.put(r.Context(), key, contentType, data); err != nil {
		http.Error(w, `{"error":"Не удалось загрузить изображение в хранилище"}`, http.StatusBadGateway)
		return
	}

	var oldKey *string
	err = database.QueryRow(r.Context(), "SELECT avatar_key FROM users WHERE id = $1", userID).Scan(&oldKey)
	if err != nil {
		_ = store.delete(r.Context(), key)
		http.Error(w, `{"error":"Не удалось сохранить аватар"}`, http.StatusInternalServerError)
		return
	}
	_, err = database.Exec(r.Context(),
		"UPDATE users SET avatar_key = $2, avatar_content_type = $3 WHERE id = $1",
		userID, key, contentType,
	)
	if err != nil {
		_ = store.delete(r.Context(), key)
		http.Error(w, `{"error":"Не удалось сохранить аватар"}`, http.StatusInternalServerError)
		return
	}
	if oldKey != nil && *oldKey != "" {
		if err := store.delete(r.Context(), *oldKey); err != nil {
			// The new avatar is already active. A failed old-object cleanup is safe to retry operationally.
		}
	}
	signedURL, err := store.presignGet(key, 15*time.Minute)
	if err != nil {
		http.Error(w, `{"error":"Аватар сохранён, но не удалось создать ссылку"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"avatarUrl": signedURL})
}

func deleteAvatarHandler(w http.ResponseWriter, r *http.Request, database *pgxpool.Pool, store *s3Store, userID int64) {
	var key *string
	err := database.QueryRow(r.Context(), "SELECT avatar_key FROM users WHERE id = $1", userID).Scan(&key)
	if err != nil {
		http.Error(w, `{"error":"Не удалось загрузить профиль"}`, http.StatusInternalServerError)
		return
	}
	_, err = database.Exec(r.Context(),
		"UPDATE users SET avatar_key = NULL, avatar_content_type = NULL WHERE id = $1",
		userID,
	)
	if err != nil {
		http.Error(w, `{"error":"Не удалось удалить аватар"}`, http.StatusInternalServerError)
		return
	}
	if key != nil && *key != "" {
		if err := store.delete(r.Context(), *key); err != nil {
			http.Error(w, `{"error":"Аватар отвязан, но файл не удалось удалить из хранилища"}`, http.StatusBadGateway)
			return
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

