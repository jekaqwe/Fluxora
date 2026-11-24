package main

import (
 "context"
 "encoding/json"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgxpool"
)

type achievementInfo struct {
 Key string `json:"key"`
 Title string `json:"title"`
 Description string `json:"description"`
 XPAwarded int `json:"xpAwarded"`
 AwardedAt time.Time `json:"awardedAt"`
}
type gamificationResponse struct {
 XP int64 `json:"xp"`
 Level int `json:"level"`
 Operations int64 `json:"operations"`
 CurrentStreak int `json:"currentStreak"`
 Achievements []achievementInfo `json:"achievements"`
}
type notificationResponse struct {
 ID int64 `json:"id"`
 Type string `json:"type"`
 Title string `json:"title"`
 Message string `json:"message"`
 RelatedType *string `json:"relatedType,omitempty"`
 RelatedID *int64 `json:"relatedId,omitempty"`
 ReadAt *time.Time `json:"readAt,omitempty"`
 CreatedAt time.Time `json:"createdAt"`
}
type achievementDefinition struct{key,title,description string;xp int}
var achievementDefinitions=[]achievementDefinition{
 {"first_operation","Первая операция","Вы добавили первую финансовую операцию",10},
 {"ten_operations","Десять операций","Вы добавили 10 финансовых операций",20},
 {"hundred_operations","Сто операций","Вы добавили 100 финансовых операций",50},
 {"first_goal_completed","Первая цель","Вы завершили первую финансовую цель",30},
 {"first_savings","Первые накопления","Вы впервые пополнили цель",20},
 {"first_family_budget","Семейный бюджет","Вы создали или приняли семейный бюджет",20},
 {"thirty_day_streak","30 дней подряд","Вы сохраняли финансовую активность 30 дней подряд",100},
}
func awardAchievement(ctx context.Context,tx pgx.Tx,userID int64,def achievementDefinition)error{
 tag,err:=tx.Exec(ctx,`INSERT INTO user_achievements(user_id,achievement_key,xp_awarded) VALUES($1,$2,$3) ON CONFLICT(user_id,achievement_key) DO NOTHING`,userID,def.key,def.xp)
 if err!=nil||tag.RowsAffected()==0{return err}
 _,err=tx.Exec(ctx,`UPDATE users SET xp=xp+$2,level=((xp+$2)/100)+1 WHERE id=$1`,userID,def.xp)
 if err!=nil{return err}
 _,err=tx.Exec(ctx,`INSERT INTO notifications(user_id,notification_type,title,message,related_type) VALUES($1,'achievement',$2,$3,'achievement')`,userID,"Новое достижение: "+def.title,def.description+" · +"+strconv.Itoa(def.xp)+" XP")
 return err
}
func gamificationHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 if r.Method!=http.MethodGet{w.Header().Set("Allow","GET");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 tx,err:=database.Begin(r.Context());if err!=nil{http.Error(w,`{"error":"Не удалось обновить достижения"}`,500);return};defer tx.Rollback(r.Context())
 var operations int64;var completedGoals,savings,family bool
 if err:=tx.QueryRow(r.Context(),`SELECT (SELECT COUNT(*) FROM transactions WHERE user_id=$1)+(SELECT COUNT(*) FROM family_transactions WHERE author_user_id=$1)`,userID).Scan(&operations);err!=nil{http.Error(w,`{"error":"Не удалось проверить операции"}`,500);return}
 if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM goals WHERE user_id=$1 AND status='completed' UNION SELECT 1 FROM family_goals fg JOIN family_members fm ON fm.family_id=fg.family_id WHERE fm.user_id=$1 AND fg.status='completed')`,userID).Scan(&completedGoals);err!=nil{http.Error(w,`{"error":"Не удалось проверить цели"}`,500);return}
 if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM goal_entries WHERE user_id=$1 AND entry_type='contribution' UNION SELECT 1 FROM family_goal_entries WHERE author_user_id=$1 AND entry_type='contribution')`,userID).Scan(&savings);err!=nil{http.Error(w,`{"error":"Не удалось проверить накопления"}`,500);return}
 if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM family_members WHERE user_id=$1)`,userID).Scan(&family);err!=nil{http.Error(w,`{"error":"Не удалось проверить семейный бюджет"}`,500);return}
 rows,err:=tx.Query(r.Context(),`SELECT activity_date FROM (
  SELECT operation_date AS activity_date FROM transactions WHERE user_id=$1
  UNION SELECT operation_date FROM family_transactions WHERE author_user_id=$1
  UNION SELECT created_at::date FROM goal_entries WHERE user_id=$1
  UNION SELECT created_at::date FROM family_goal_entries WHERE author_user_id=$1 AND entry_type='contribution'
  UNION SELECT joined_at::date FROM family_members WHERE user_id=$1
 ) activity WHERE activity_date<=CURRENT_DATE ORDER BY activity_date DESC`,userID)
 if err!=nil{http.Error(w,`{"error":"Не удалось рассчитать активность"}`,500);return}
 dates:=make([]time.Time,0);for rows.Next(){var d time.Time;if rows.Scan(&d)==nil{dates=append(dates,d)}};rows.Close()
 streak:=0
 if len(dates)>0{
  today:=time.Now().UTC().Truncate(24*time.Hour);expected:=dates[0].UTC().Truncate(24*time.Hour)
  if expected.Before(today.AddDate(0,0,-1)){expected=time.Time{}} 
  if !expected.IsZero(){for _,d:=range dates{day:=d.UTC().Truncate(24*time.Hour);if !day.Equal(expected){break};streak++;expected=expected.AddDate(0,0,-1)}}
 }
 checks:=map[string]bool{"first_operation":operations>=1,"ten_operations":operations>=10,"hundred_operations":operations>=100,"first_goal_completed":completedGoals,"first_savings":savings,"first_family_budget":family,"thirty_day_streak":streak>=30}
 for _,def:=range achievementDefinitions{if checks[def.key]{if err:=awardAchievement(r.Context(),tx,userID,def);err!=nil{http.Error(w,`{"error":"Не удалось начислить достижение"}`,500);return}}}
 if err:=tx.Commit(r.Context());err!=nil{http.Error(w,`{"error":"Не удалось сохранить достижения"}`,500);return}
 result:=gamificationResponse{Operations:operations,CurrentStreak:streak,Achievements:make([]achievementInfo,0)}
 if err:=database.QueryRow(r.Context(),`SELECT xp,level FROM users WHERE id=$1`,userID).Scan(&result.XP,&result.Level);err!=nil{http.Error(w,`{"error":"Не удалось загрузить уровень"}`,500);return}
 achievementRows,err:=database.Query(r.Context(),`SELECT a.achievement_key,d.title,d.description,a.xp_awarded,a.awarded_at FROM user_achievements a JOIN (VALUES
 ('first_operation','Первая операция','Вы добавили первую финансовую операцию'),
 ('ten_operations','Десять операций','Вы добавили 10 финансовых операций'),
 ('hundred_operations','Сто операций','Вы добавили 100 финансовых операций'),
 ('first_goal_completed','Первая цель','Вы завершили первую финансовую цель'),
 ('first_savings','Первые накопления','Вы впервые пополнили цель'),
 ('first_family_budget','Семейный бюджет','Вы создали или приняли семейный бюджет'),
 ('thirty_day_streak','30 дней подряд','Вы сохраняли финансовую активность 30 дней подряд')
 ) AS d(key,title,description) ON d.key=a.achievement_key WHERE a.user_id=$1 ORDER BY a.awarded_at`,userID)
 if err==nil{defer achievementRows.Close();for achievementRows.Next(){var a achievementInfo;if achievementRows.Scan(&a.Key,&a.Title,&a.Description,&a.XPAwarded,&a.AwardedAt)==nil{result.Achievements=append(result.Achievements,a)}}}
 _=json.NewEncoder(w).Encode(result)
}
func notificationsHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 if r.URL.Path=="/api/notifications"{
  if r.Method!=http.MethodGet{w.Header().Set("Allow","GET");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
  rows,err:=database.Query(r.Context(),`SELECT id,notification_type,title,message,related_type,related_id,read_at,created_at FROM notifications WHERE user_id=$1
   AND CASE
    WHEN notification_type IN ('family_goal','family_transaction','family_invitation','family_invite','family_activity') THEN COALESCE((SELECT (preferences->'notifications'->>'family')::boolean FROM user_preferences WHERE user_id=$1),true)
    WHEN notification_type IN ('goal','goal_entry','goal_completed','achievement','savings_goal') THEN COALESCE((SELECT (preferences->'notifications'->>'goals')::boolean FROM user_preferences WHERE user_id=$1),true)
    WHEN notification_type IN ('transaction','recurring_created','recurring_missed','debt_payment') THEN COALESCE((SELECT (preferences->'notifications'->>'operations')::boolean FROM user_preferences WHERE user_id=$1),true)
    ELSE true END
   ORDER BY created_at DESC,id DESC LIMIT 100`,userID)
  if err!=nil{http.Error(w,`{"error":"Не удалось загрузить уведомления"}`,500);return};defer rows.Close();items:=make([]notificationResponse,0)
  for rows.Next(){var x notificationResponse;if err:=rows.Scan(&x.ID,&x.Type,&x.Title,&x.Message,&x.RelatedType,&x.RelatedID,&x.ReadAt,&x.CreatedAt);err!=nil{http.Error(w,`{"error":"Ошибка чтения уведомлений"}`,500);return};items=append(items,x)}
  var unread int64;_=database.QueryRow(r.Context(),`SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND read_at IS NULL
   AND CASE
    WHEN notification_type IN ('family_goal','family_transaction','family_invitation','family_invite','family_activity') THEN COALESCE((SELECT (preferences->'notifications'->>'family')::boolean FROM user_preferences WHERE user_id=$1),true)
    WHEN notification_type IN ('goal','goal_entry','goal_completed','achievement','savings_goal') THEN COALESCE((SELECT (preferences->'notifications'->>'goals')::boolean FROM user_preferences WHERE user_id=$1),true)
    WHEN notification_type IN ('transaction','recurring_created','recurring_missed','debt_payment') THEN COALESCE((SELECT (preferences->'notifications'->>'operations')::boolean FROM user_preferences WHERE user_id=$1),true)
    ELSE true END`,userID).Scan(&unread)
  _=json.NewEncoder(w).Encode(map[string]any{"items":items,"unread":unread});return
 }
 if r.URL.Path=="/api/notifications/read-all"&&r.Method==http.MethodPost{_,err:=database.Exec(r.Context(),`UPDATE notifications SET read_at=NOW() WHERE user_id=$1 AND read_at IS NULL`,userID);if err!=nil{http.Error(w,`{"error":"Не удалось отметить уведомления"}`,500);return};_=json.NewEncoder(w).Encode(map[string]bool{"success":true});return}
 raw:=strings.TrimPrefix(r.URL.Path,"/api/notifications/");id,err:=strconv.ParseInt(raw,10,64);if err!=nil||id<=0||r.Method!=http.MethodPatch{http.Error(w,`{"error":"Некорректный адрес уведомления"}`,400);return}
 tag,err:=database.Exec(r.Context(),`UPDATE notifications SET read_at=COALESCE(read_at,NOW()) WHERE id=$1 AND user_id=$2`,id,userID);if err!=nil{http.Error(w,`{"error":"Не удалось отметить уведомление"}`,500);return};if tag.RowsAffected()==0{http.Error(w,`{"error":"Уведомление не найдено"}`,404);return};_=json.NewEncoder(w).Encode(map[string]bool{"success":true})
}
