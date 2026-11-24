package main

import (
 "context"
 "encoding/json"
 "log"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
)

type recurringPayload struct {
 Title string `json:"title"`
 Type string `json:"type"`
 Amount string `json:"amount"`
 Category string `json:"category"`
 AccountID int64 `json:"accountId"`
 Frequency string `json:"frequency"`
 NextDate string `json:"nextDate"`
 EndDate string `json:"endDate"`
 Status string `json:"status"`
}
type recurringResponse struct {
 ID int64 `json:"id"`
 Title string `json:"title"`
 Type string `json:"type"`
 Amount string `json:"amount"`
 Category string `json:"category"`
 AccountID *int64 `json:"accountId,omitempty"`
 Frequency string `json:"frequency"`
 NextDate string `json:"nextDate"`
 EndDate *string `json:"endDate,omitempty"`
 Status string `json:"status"`
}
type recurringOccurrence struct {
 ID int64 `json:"id"`
 RecurringID int64 `json:"recurringId"`
 Title string `json:"title"`
 Type string `json:"type"`
 Amount string `json:"amount"`
 Category string `json:"category"`
 AccountID *int64 `json:"accountId,omitempty"`
 Date string `json:"date"`
}
func nextRecurringDate(d time.Time,frequency string) time.Time {
 switch frequency {
 case "daily": return d.AddDate(0,0,1)
 case "weekly": return d.AddDate(0,0,7)
 case "monthly":
  first:=time.Date(d.Year(),d.Month(),1,0,0,0,0,time.UTC).AddDate(0,1,0)
  day:=d.Day();last:=first.AddDate(0,1,-1).Day();if day>last{day=last}
  return time.Date(first.Year(),first.Month(),day,0,0,0,0,time.UTC)
 case "yearly":
  year:=d.Year()+1;month:=d.Month();day:=d.Day()
  first:=time.Date(year,month,1,0,0,0,0,time.UTC);last:=first.AddDate(0,1,-1).Day();if day>last{day=last}
  return time.Date(year,month,day,0,0,0,0,time.UTC)
 default:return d
 }
}
func recurringPayloadDate(v string)(time.Time,error){return time.Parse("2006-01-02",v)}
func recurringHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 if r.URL.Path!="/api/recurring"{recurringItemHandler(w,r,database);return}
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 switch r.Method{
 case http.MethodGet:
  rows,err:=database.Query(r.Context(),`SELECT id,title,operation_type,amount::text,category,account_id,frequency,next_date::text,end_date::text,status FROM recurring_operations WHERE user_id=$1 ORDER BY next_date,id`,userID)
  if err!=nil{http.Error(w,`{"error":"Не удалось загрузить регулярные операции"}`,500);return};defer rows.Close()
  items:=make([]recurringResponse,0)
  for rows.Next(){var x recurringResponse;var account pgtype.Int8;var end *string;if err:=rows.Scan(&x.ID,&x.Title,&x.Type,&x.Amount,&x.Category,&account,&x.Frequency,&x.NextDate,&end,&x.Status);err!=nil{http.Error(w,`{"error":"Ошибка чтения расписания"}`,500);return};if account.Valid{v:=account.Int64;x.AccountID=&v};x.EndDate=end;items=append(items,x)}
  missedRows,err:=database.Query(r.Context(),`SELECT o.id,o.recurring_id,r.title,r.operation_type,r.amount::text,r.category,r.account_id,o.occurrence_date::text FROM recurring_occurrences o JOIN recurring_operations r ON r.id=o.recurring_id WHERE o.user_id=$1 AND o.status='pending' ORDER BY o.occurrence_date,o.id`,userID)
  if err!=nil{http.Error(w,`{"error":"Не удалось загрузить пропущенные даты"}`,500);return};defer missedRows.Close()
  missed:=make([]recurringOccurrence,0)
  for missedRows.Next(){var x recurringOccurrence;var account pgtype.Int8;if err:=missedRows.Scan(&x.ID,&x.RecurringID,&x.Title,&x.Type,&x.Amount,&x.Category,&account,&x.Date);err!=nil{http.Error(w,`{"error":"Ошибка чтения пропущенных дат"}`,500);return};if account.Valid{v:=account.Int64;x.AccountID=&v};missed=append(missed,x)}
  _=json.NewEncoder(w).Encode(map[string]any{"items":items,"missed":missed})
 case http.MethodPost:
  r.Body=http.MaxBytesReader(w,r.Body,24<<10);var in recurringPayload
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
  in.Title=strings.TrimSpace(in.Title);in.Type=strings.TrimSpace(in.Type);in.Amount=strings.TrimSpace(in.Amount);in.Category=strings.TrimSpace(in.Category)
  next,err:=recurringPayloadDate(in.NextDate);end,errEnd:=recurringPayloadDate(in.EndDate)
  if in.Title==""||len([]rune(in.Title))>160||(in.Type!="income"&&in.Type!="expense")||!accountAmountPattern.MatchString(in.Amount)||strings.HasPrefix(in.Amount,"-")||strings.Trim(in.Amount,"0.")==""||in.Category==""||len([]rune(in.Category))>80||in.AccountID<=0||err!=nil||(in.EndDate!=""&&errEnd!=nil)|| (in.EndDate!=""&&end.Before(next)) {
   http.Error(w,`{"error":"Проверьте параметры регулярной операции"}`,400);return
  }
  if in.Frequency!="daily"&&in.Frequency!="weekly"&&in.Frequency!="monthly"&&in.Frequency!="yearly"{http.Error(w,`{"error":"Неизвестная периодичность"}`,400);return}
  var exists bool;if err:=database.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`,in.AccountID,userID).Scan(&exists);err!=nil||!exists{http.Error(w,`{"error":"Счёт не найден"}`,400);return}
  var item recurringResponse
  err=database.QueryRow(r.Context(),`INSERT INTO recurring_operations(user_id,account_id,title,operation_type,amount,category,frequency,next_date,end_date) VALUES($1,$2,$3,$4,$5::numeric,$6,$7,$8,$9) RETURNING id,title,operation_type,amount::text,category,account_id,frequency,next_date::text,end_date::text,status`,userID,in.AccountID,in.Title,in.Type,in.Amount,in.Category,in.Frequency,next,in.EndDate).Scan(&item.ID,&item.Title,&item.Type,&item.Amount,&item.Category,&item.AccountID,&item.Frequency,&item.NextDate,&item.EndDate,&item.Status)
  if err!=nil{http.Error(w,`{"error":"Не удалось создать расписание"}`,500);return};w.WriteHeader(201);_=json.NewEncoder(w).Encode(item)
 default:w.Header().Set("Allow","GET, POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
 }
}
func recurringItemHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 parts:=strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path,"/api/recurring/"),"/"),"/")
 if len(parts)==3&&parts[1]=="occurrences"{occurrenceAction(w,r,database,userID,parts[0],parts[2]);return}
 if len(parts)!=1{http.Error(w,`{"error":"Некорректный адрес"}`,400);return}
 id,err:=strconv.ParseInt(parts[0],10,64);if err!=nil||id<=0{http.Error(w,`{"error":"Некорректный идентификатор"}`,400);return}
 switch r.Method{
 case http.MethodDelete:
  tag,err:=database.Exec(r.Context(),`DELETE FROM recurring_operations WHERE id=$1 AND user_id=$2`,id,userID);if err!=nil{http.Error(w,`{"error":"Не удалось удалить расписание"}`,500);return};if tag.RowsAffected()==0{http.Error(w,`{"error":"Расписание не найдено"}`,404);return};_=json.NewEncoder(w).Encode(map[string]bool{"success":true})
 case http.MethodPatch:
  r.Body=http.MaxBytesReader(w,r.Body,12<<10);var in struct{Status string `json:"status"`;AccountID int64 `json:"accountId"`}
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
  if in.Status!="active"&&in.Status!="paused"{http.Error(w,`{"error":"Статус должен быть active или paused"}`,400);return}
  if in.Status=="active"&&in.AccountID<=0{http.Error(w,`{"error":"Для возобновления выберите счёт"}`,400);return}
  if in.AccountID>0{var ok bool;if err:=database.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`,in.AccountID,userID).Scan(&ok);err!=nil||!ok{http.Error(w,`{"error":"Счёт не найден"}`,400);return}}
  tag,err:=database.Exec(r.Context(),`UPDATE recurring_operations SET status=$3,account_id=CASE WHEN $4>0 THEN $4 ELSE account_id END,updated_at=NOW() WHERE id=$1 AND user_id=$2`,id,userID,in.Status,in.AccountID)
  if err!=nil{http.Error(w,`{"error":"Не удалось обновить расписание"}`,500);return};if tag.RowsAffected()==0{http.Error(w,`{"error":"Расписание не найдено"}`,404);return};_=json.NewEncoder(w).Encode(map[string]bool{"success":true})
 default:w.Header().Set("Allow","PATCH, DELETE");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
 }
}
func occurrenceAction(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID int64,rawID,action string){
 if r.Method!=http.MethodPost{w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
 id,err:=strconv.ParseInt(rawID,10,64);if err!=nil||id<=0||(action!="confirm"&&action!="skip"){http.Error(w,`{"error":"Некорректное действие"}`,400);return}
 tx,err:=database.Begin(r.Context());if err!=nil{http.Error(w,`{"error":"Не удалось обработать дату"}`,500);return};defer tx.Rollback(r.Context())
 var recurringID,accountID int64;var title,kind,amount,category,date string;var status string
 err=tx.QueryRow(r.Context(),`SELECT o.recurring_id,r.account_id,r.title,r.operation_type,r.amount::text,r.category,o.occurrence_date::text,o.status FROM recurring_occurrences o JOIN recurring_operations r ON r.id=o.recurring_id WHERE o.id=$1 AND o.user_id=$2 FOR UPDATE OF o`,id,userID).Scan(&recurringID,&accountID,&title,&kind,&amount,&category,&date,&status)
 if err!=nil{http.Error(w,`{"error":"Пропущенная дата не найдена"}`,404);return}
 if status!="pending"{http.Error(w,`{"error":"Эта дата уже обработана"}`,409);return}
 var transactionID any
 if action=="confirm"{
  if accountID<=0{http.Error(w,`{"error":"Сначала назначьте счёт регулярной операции"}`,409);return}
  parsed,_:=time.Parse("2006-01-02",date)
  var idValue int64
  err=tx.QueryRow(r.Context(),`INSERT INTO transactions(user_id,account_id,title,operation_type,amount,category,operation_date,comment,idempotency_key) VALUES($1,$2,$3,$4,$5::numeric,$6,$7,'Создано из регулярной операции',$8) ON CONFLICT(user_id,idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key RETURNING id`,userID,accountID,title,kind,amount,category,parsed,"recurring-occurrence:"+strconv.FormatInt(id,10)).Scan(&idValue)
  if err!=nil{http.Error(w,`{"error":"Не удалось создать операцию"}`,500);return};transactionID=idValue
 }
 newStatus:="skipped";if action=="confirm"{newStatus="confirmed"}
 _,err=tx.Exec(r.Context(),`UPDATE recurring_occurrences SET status=$2,transaction_id=$3,resolved_at=NOW() WHERE id=$1`,id,newStatus,transactionID)
 if err!=nil{http.Error(w,`{"error":"Не удалось отметить дату"}`,500);return}
 if err:=tx.Commit(r.Context());err!=nil{http.Error(w,`{"error":"Не удалось завершить действие"}`,500);return}
 _=json.NewEncoder(w).Encode(map[string]bool{"success":true})
}

func processRecurringDue(database *pgxpool.Pool){
 ctx:=context.Background();today:=time.Now().UTC().Truncate(24*time.Hour)
 rows,err:=database.Query(ctx,`SELECT id FROM recurring_operations WHERE status='active' AND next_date<=CURRENT_DATE ORDER BY id LIMIT 500`)
 if err!=nil{log.Printf("recurring scheduler: %v",err);return}
 ids:=make([]int64,0);for rows.Next(){var id int64;if rows.Scan(&id)==nil{ids=append(ids,id)}};rows.Close()
 for _,id:=range ids{processOneRecurring(ctx,database,id,today)}
}
func processOneRecurring(ctx context.Context,database *pgxpool.Pool,id int64,today time.Time){
 tx,err:=database.Begin(ctx);if err!=nil{log.Printf("recurring begin: %v",err);return};defer tx.Rollback(ctx)
 var userID,accountID int64;var title,kind,amount,category,frequency string;var next time.Time;var end *time.Time
 err=tx.QueryRow(ctx,`SELECT user_id,account_id,title,operation_type,amount::text,category,frequency,next_date,end_date FROM recurring_operations WHERE id=$1 AND status='active' FOR UPDATE`,id).Scan(&userID,&accountID,&title,&kind,&amount,&category,&frequency,&next,&end)
 if err!=nil{return}
 next=next.UTC();if end!=nil{v:=end.UTC();end=&v}
 for next.Before(today){
  if end!=nil&&next.After(*end){_,_=tx.Exec(ctx,`UPDATE recurring_operations SET status='completed',updated_at=NOW() WHERE id=$1`,id);_ = tx.Commit(ctx);return}
  _,err=tx.Exec(ctx,`INSERT INTO recurring_occurrences(recurring_id,user_id,occurrence_date) VALUES($1,$2,$3) ON CONFLICT(recurring_id,occurrence_date) DO NOTHING`,id,userID,next)
  if err!=nil{log.Printf("recurring missed date: %v",err);return};next=nextRecurringDate(next,frequency)
 }
 if next.Equal(today){
  if end!=nil&&next.After(*end){_,_=tx.Exec(ctx,`UPDATE recurring_operations SET status='completed',updated_at=NOW() WHERE id=$1`,id);_=tx.Commit(ctx);return}
  key:="recurring:"+strconv.FormatInt(id,10)+":"+next.Format("2006-01-02")
  _,err=tx.Exec(ctx,`INSERT INTO transactions(user_id,account_id,title,operation_type,amount,category,operation_date,comment,idempotency_key) VALUES($1,$2,$3,$4,$5::numeric,$6,$7,'Автоматическая регулярная операция',$8) ON CONFLICT(user_id,idempotency_key) DO NOTHING`,userID,accountID,title,kind,amount,category,next,key)
  if err!=nil{log.Printf("recurring create transaction: %v",err);return};next=nextRecurringDate(next,frequency)
 }
 status:="active";if end!=nil&&next.After(*end){status="completed"}
 _,err=tx.Exec(ctx,`UPDATE recurring_operations SET next_date=$2,status=$3,updated_at=NOW() WHERE id=$1`,id,next,status)
 if err!=nil{log.Printf("recurring update schedule: %v",err);return}
 if err:=tx.Commit(ctx);err!=nil{log.Printf("recurring commit: %v",err)}
}
func startRecurringScheduler(database *pgxpool.Pool){
 go func(){processRecurringDue(database);ticker:=time.NewTicker(time.Minute);defer ticker.Stop();for range ticker.C{processRecurringDue(database)}}()
}
