package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5/pgxpool"
)

type sessionResponse struct {
 ID int64 `json:"id"`
 CreatedAt time.Time `json:"createdAt"`
 ExpiresAt time.Time `json:"expiresAt"`
 Current bool `json:"current"`
}
func currentSessionHash(r *http.Request)(string,error){
 cookie,err:=r.Cookie("fluxora_session");if err!=nil{return "",err}
 sum:=sha256.Sum256([]byte(cookie.Value));return hex.EncodeToString(sum[:]),nil
}
func sessionsHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 currentHash,err:=currentSessionHash(r);if err!=nil{http.Error(w,`{"error":"Текущая сессия не найдена"}`,401);return}
 path:=strings.TrimSuffix(r.URL.Path,"/")
 if path=="/api/account/sessions"{
  if r.Method!=http.MethodGet{w.Header().Set("Allow","GET");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
  rows,err:=database.Query(r.Context(),`SELECT id,created_at,expires_at,(token_hash=$2) FROM sessions WHERE user_id=$1 AND expires_at>NOW() ORDER BY created_at DESC`,userID,currentHash)
  if err!=nil{http.Error(w,`{"error":"Не удалось загрузить сессии"}`,500);return};defer rows.Close()
  items:=make([]sessionResponse,0);for rows.Next(){var x sessionResponse;if err:=rows.Scan(&x.ID,&x.CreatedAt,&x.ExpiresAt,&x.Current);err!=nil{http.Error(w,`{"error":"Ошибка чтения сессий"}`,500);return};items=append(items,x)}
  _=json.NewEncoder(w).Encode(items);return
 }
 if path=="/api/account/sessions/revoke-others"{
  if r.Method!=http.MethodPost{w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
  tag,err:=database.Exec(r.Context(),`DELETE FROM sessions WHERE user_id=$1 AND token_hash<>$2`,userID,currentHash)
  if err!=nil{http.Error(w,`{"error":"Не удалось завершить остальные сессии"}`,500);return}
  _=json.NewEncoder(w).Encode(map[string]int64{"revoked":tag.RowsAffected()});return
 }
 raw:=strings.TrimPrefix(path,"/api/account/sessions/");id,err:=strconv.ParseInt(raw,10,64)
 if err!=nil||id<=0||r.Method!=http.MethodDelete{http.Error(w,`{"error":"Некорректная сессия"}`,400);return}
 tag,err:=database.Exec(r.Context(),`DELETE FROM sessions WHERE id=$1 AND user_id=$2 AND token_hash<>$3`,id,userID,currentHash)
 if err!=nil{http.Error(w,`{"error":"Не удалось завершить сессию"}`,500);return}
 if tag.RowsAffected()==0{http.Error(w,`{"error":"Сессия не найдена или это текущая сессия"}`,404);return}
 _=json.NewEncoder(w).Encode(map[string]bool{"success":true})
}
