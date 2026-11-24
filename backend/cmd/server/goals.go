package main

import (
 "crypto/rand"
 "encoding/hex"
 "encoding/json"
 "io"
 "log"
 "net/http"
 "github.com/jackc/pgx/v5"
 "regexp"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
)

type goalPayload struct {
 Title string `json:"title"`
 Description string `json:"description"`
 TargetAmount string `json:"targetAmount"`
 StartingAmount string `json:"startingAmount"`
 DueDate string `json:"dueDate"`
 Priority string `json:"priority"`
 Status string `json:"status"`
}
type goalResponse struct {
 ID int64 `json:"id"`
 Title string `json:"title"`
 Description string `json:"description"`
 TargetAmount string `json:"targetAmount"`
 StartingAmount string `json:"startingAmount"`
 CurrentAmount string `json:"currentAmount"`
 ImageURL *string `json:"imageUrl,omitempty"`
 DueDate *string `json:"dueDate,omitempty"`
 Priority string `json:"priority"`
 Status string `json:"status"`
 CreatedAt time.Time `json:"createdAt"`
}
type goalEntryPayload struct {
 Type string `json:"type"`
 Amount string `json:"amount"`
 Comment string `json:"comment"`
 TransactionID int64 `json:"transactionId"`
 AccountID int64 `json:"accountId"`
 IdempotencyKey string `json:"idempotencyKey"`
}
type goalEntryResponse struct {
 ID int64 `json:"id"`
 Type string `json:"type"`
 Amount string `json:"amount"`
 Comment string `json:"comment"`
 TransactionID *int64 `json:"transactionId,omitempty"`
 CreatedAt time.Time `json:"createdAt"`
}
var goalAmountPattern = regexp.MustCompile("^(?:0|[1-9][0-9]{0,17})(?:\\.[0-9]{1,2})?$")
func validGoalAmount(v string, zero bool) bool {
 if !goalAmountPattern.MatchString(v) { return false }
 return zero || strings.Trim(v,"0.")!=""
}

func goalsHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool) {
 if r.URL.Path!="/api/goals" { goalItemHandler(w,r,database); return }
 w.Header().Set("Content-Type","application/json"); w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database)
 if err!=nil { http.Error(w, `{"error":"Пользователь не авторизован"}`,http.StatusUnauthorized); return }
 switch r.Method {
 case http.MethodGet:
  rows,err:=database.Query(r.Context(),`
   SELECT g.id,g.title,g.description,g.target_amount::text,g.starting_amount::text,
    (g.starting_amount+COALESCE(SUM(CASE WHEN e.entry_type='contribution' THEN e.amount ELSE -e.amount END),0))::text,
    g.due_date::text,g.priority,g.status,g.image_key,g.created_at
   FROM goals g LEFT JOIN goal_entries e ON e.goal_id=g.id AND e.user_id=g.user_id
   WHERE g.user_id=$1 GROUP BY g.id ORDER BY g.created_at DESC,g.id DESC
  `,userID)
  if err!=nil { http.Error(w,`{"error":"Не удалось загрузить цели"}`,500); return }
  defer rows.Close(); items:=make([]goalResponse,0)
  for rows.Next() {
   var item goalResponse; var due,imageKey *string
   if err:=rows.Scan(&item.ID,&item.Title,&item.Description,&item.TargetAmount,&item.StartingAmount,&item.CurrentAmount,&due,&item.Priority,&item.Status,&imageKey,&item.CreatedAt); err!=nil { http.Error(w,`{"error":"Ошибка чтения целей"}`,500); return }
   item.DueDate=due
   if imageKey!=nil&&*imageKey!=""{if store,e:=newS3Store();e==nil{if url,e:=store.presignGet(*imageKey,7*24*time.Hour);e==nil{item.ImageURL=&url}}}
   items=append(items,item)
  }
  _=json.NewEncoder(w).Encode(items)
 case http.MethodPost:
  r.Body=http.MaxBytesReader(w,r.Body,32<<10); var in goalPayload
  if err:=json.NewDecoder(r.Body).Decode(&in); err!=nil { http.Error(w,`{"error":"Некорректные данные"}`,400); return }
  in.Title=strings.TrimSpace(in.Title); in.Description=strings.TrimSpace(in.Description)
  in.TargetAmount=strings.TrimSpace(in.TargetAmount); in.StartingAmount=strings.TrimSpace(in.StartingAmount)
  if in.StartingAmount=="" { in.StartingAmount="0" }; if in.Priority=="" { in.Priority="medium" }
  if in.Title==""||len([]rune(in.Title))>120||len([]rune(in.Description))>1000||!validGoalAmount(in.TargetAmount,false)||!validGoalAmount(in.StartingAmount,true) {
   http.Error(w,`{"error":"Проверьте название и суммы цели"}`,400); return
  }
  if in.Priority!="low"&&in.Priority!="medium"&&in.Priority!="high" { http.Error(w,`{"error":"Некорректный приоритет"}`,400); return }
  due,err:=parseGoalDate(in.DueDate); if err!=nil { http.Error(w,`{"error":"Некорректная дата цели"}`,400); return }
  var item goalResponse; var dueText *string
  err=database.QueryRow(r.Context(),`
   INSERT INTO goals(user_id,title,description,target_amount,starting_amount,due_date,priority)
   VALUES($1,$2,$3,$4::numeric,$5::numeric,$6,$7)
   RETURNING id,title,description,target_amount::text,starting_amount::text,starting_amount::text,due_date::text,priority,status,created_at
  `,userID,in.Title,in.Description,in.TargetAmount,in.StartingAmount,due,in.Priority).Scan(&item.ID,&item.Title,&item.Description,&item.TargetAmount,&item.StartingAmount,&item.CurrentAmount,&dueText,&item.Priority,&item.Status,&item.CreatedAt)
  if err!=nil { http.Error(w,`{"error":"Не удалось создать цель"}`,500); return }
  item.DueDate=dueText; w.WriteHeader(201); _=json.NewEncoder(w).Encode(item)
 default:
  w.Header().Set("Allow","GET, POST"); http.Error(w,`{"error":"Метод не поддерживается"}`,405)
 }
}

func parseGoalDate(v string) (any,error) {
 if strings.TrimSpace(v)=="" { return nil,nil }
 d,err:=time.Parse("2006-01-02",v); if err!=nil { return nil,err }; return d,nil
}

func goalItemHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool) {
 w.Header().Set("Content-Type","application/json"); w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database); if err!=nil { http.Error(w,`{"error":"Пользователь не авторизован"}`,401); return }
 parts:=strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path,"/api/goals/"),"/"),"/")
 id,err:=strconv.ParseInt(parts[0],10,64); if err!=nil||id<=0 { http.Error(w,`{"error":"Некорректный идентификатор цели"}`,400); return }
 if len(parts)==2&&parts[1]=="entries" { goalEntriesHandler(w,r,database,userID,id); return }
 if len(parts)==2&&parts[1]=="image" { goalImageHandler(w,r,database,userID,id); return }
 if len(parts)!=1 { http.Error(w,`{"error":"Некорректный адрес цели"}`,400); return }
 switch r.Method {
 case http.MethodPatch:
  r.Body=http.MaxBytesReader(w,r.Body,32<<10); var in goalPayload
  if err:=json.NewDecoder(r.Body).Decode(&in); err!=nil { http.Error(w,`{"error":"Некорректные данные"}`,400); return }
  in.Title=strings.TrimSpace(in.Title); in.Description=strings.TrimSpace(in.Description)
  in.TargetAmount=strings.TrimSpace(in.TargetAmount); in.StartingAmount=strings.TrimSpace(in.StartingAmount)
  if in.Title==""||len([]rune(in.Title))>120||len([]rune(in.Description))>1000||!validGoalAmount(in.TargetAmount,false)||!validGoalAmount(in.StartingAmount,true) { http.Error(w,`{"error":"Проверьте название и суммы цели"}`,400); return }
  if (in.Priority!="low"&&in.Priority!="medium"&&in.Priority!="high")||(in.Status!="active"&&in.Status!="paused"&&in.Status!="completed") { http.Error(w,`{"error":"Некорректный приоритет или статус"}`,400); return }
  due,err:=parseGoalDate(in.DueDate); if err!=nil { http.Error(w,`{"error":"Некорректная дата цели"}`,400); return }
  var item goalResponse; var dueText *string
  err=database.QueryRow(r.Context(),`
   UPDATE goals SET title=$3,description=$4,target_amount=$5::numeric,starting_amount=$6::numeric,due_date=$7,priority=$8,status=CASE
			 WHEN starting_amount+COALESCE((SELECT SUM(CASE WHEN entry_type='contribution' THEN amount ELSE -amount END) FROM goal_entries WHERE goal_id=goals.id),0)>=$5::numeric THEN 'completed'
			 WHEN $9='paused' THEN 'paused' ELSE 'active' END,updated_at=NOW()
   WHERE id=$1 AND user_id=$2
   RETURNING id,title,description,target_amount::text,starting_amount::text,(starting_amount+COALESCE((SELECT SUM(CASE WHEN entry_type='contribution' THEN amount ELSE -amount END) FROM goal_entries WHERE goal_id=goals.id),0))::text,due_date::text,priority,status,created_at
  `,id,userID,in.Title,in.Description,in.TargetAmount,in.StartingAmount,due,in.Priority,in.Status).Scan(&item.ID,&item.Title,&item.Description,&item.TargetAmount,&item.StartingAmount,&item.CurrentAmount,&dueText,&item.Priority,&item.Status,&item.CreatedAt)
  if err!=nil { http.Error(w,`{"error":"Цель не найдена или не удалось сохранить"}`,404); return }; item.DueDate=dueText; _=json.NewEncoder(w).Encode(item)
 case http.MethodDelete:
  var imageKey *string
  if err:=database.QueryRow(r.Context(),"SELECT image_key FROM goals WHERE id=$1 AND user_id=$2",id,userID).Scan(&imageKey);err!=nil{http.Error(w,`{"error":"Цель не найдена"}`,404);return}
  tag,err:=database.Exec(r.Context(),"DELETE FROM goals WHERE id=$1 AND user_id=$2",id,userID)
  if err!=nil { http.Error(w,`{"error":"Не удалось удалить цель"}`,500); return }; if tag.RowsAffected()==0 { http.Error(w,`{"error":"Цель не найдена"}`,404); return }
  if imageKey!=nil&&*imageKey!=""{if store,e:=newS3Store();e==nil{if e:=store.delete(r.Context(),*imageKey);e!=nil{log.Printf("failed to delete goal image %d: %v",id,e)}}}
  _=json.NewEncoder(w).Encode(map[string]bool{"success":true})
 default:
  w.Header().Set("Allow","PATCH, DELETE"); http.Error(w,`{"error":"Метод не поддерживается"}`,405)
 }
}

func goalEntriesHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID,goalID int64) {
 if r.Method==http.MethodGet {
  rows,err:=database.Query(r.Context(),`
   SELECT e.id,e.entry_type,e.amount::text,e.comment,e.transaction_id,e.created_at
   FROM goal_entries e JOIN goals g ON g.id=e.goal_id
   WHERE e.goal_id=$1 AND g.user_id=$2 AND e.user_id=$2 ORDER BY e.created_at DESC,e.id DESC
  `,goalID,userID)
  if err!=nil { http.Error(w,`{"error":"Не удалось загрузить историю"}`,500); return }
  defer rows.Close(); items:=make([]goalEntryResponse,0)
  for rows.Next() {
   var item goalEntryResponse; var txID pgtype.Int8
   if err:=rows.Scan(&item.ID,&item.Type,&item.Amount,&item.Comment,&txID,&item.CreatedAt); err!=nil { http.Error(w,`{"error":"Ошибка чтения истории"}`,500); return }
   if txID.Valid { value:=txID.Int64; item.TransactionID=&value }; items=append(items,item)
  }
  _=json.NewEncoder(w).Encode(items); return
 }
 if r.Method!=http.MethodPost { w.Header().Set("Allow","GET, POST"); http.Error(w,`{"error":"Метод не поддерживается"}`,405); return }
 r.Body=http.MaxBytesReader(w,r.Body,16<<10); var in goalEntryPayload
 if err:=json.NewDecoder(r.Body).Decode(&in); err!=nil { http.Error(w,`{"error":"Некорректные данные"}`,400); return }
 in.Type=strings.TrimSpace(in.Type); in.Amount=strings.TrimSpace(in.Amount); in.Comment=strings.TrimSpace(in.Comment); in.IdempotencyKey=strings.TrimSpace(in.IdempotencyKey)
 if (in.Type!="contribution"&&in.Type!="withdrawal")||!validGoalAmount(in.Amount,false)||len([]rune(in.Comment))>500||in.IdempotencyKey==""||len(in.IdempotencyKey)>60||in.AccountID<0||in.TransactionID<0 {
  http.Error(w,`{"error":"Проверьте тип, сумму, комментарий и ключ операции"}`,400); return
 }
 tx,err:=database.Begin(r.Context()); if err!=nil { http.Error(w,`{"error":"Не удалось начать операцию"}`,500); return }; defer tx.Rollback(r.Context())
 var goalStatus string
 if err:=tx.QueryRow(r.Context(),"SELECT status FROM goals WHERE id=$1 AND user_id=$2 FOR UPDATE",goalID,userID).Scan(&goalStatus); err!=nil { http.Error(w,`{"error":"Цель не найдена"}`,404); return }
 var item goalEntryResponse; var txID pgtype.Int8
 err=tx.QueryRow(r.Context(),`SELECT id,entry_type,amount::text,comment,transaction_id,created_at FROM goal_entries WHERE user_id=$1 AND idempotency_key=$2`,userID,in.IdempotencyKey).Scan(&item.ID,&item.Type,&item.Amount,&item.Comment,&txID,&item.CreatedAt)
 if err==nil {
  expectedType:="expense";if in.Type=="withdrawal"{expectedType="income"}
  var samePayload bool
  err=tx.QueryRow(r.Context(),`SELECT EXISTS(
   SELECT 1 FROM goal_entries e
   WHERE e.id=$1 AND e.goal_id=$2 AND e.entry_type=$3 AND e.amount=$4::numeric AND e.comment=$5
   AND (
    ($6::bigint>0 AND EXISTS(SELECT 1 FROM transactions t WHERE t.id=e.transaction_id AND t.user_id=$7 AND t.account_id=$6 AND t.amount=$4::numeric AND t.operation_type=$8))
    OR ($9::bigint>0 AND e.transaction_id=$9)
    OR ($6::bigint=0 AND $9::bigint=0 AND e.transaction_id IS NULL)
   )
  )`,item.ID,goalID,in.Type,in.Amount,in.Comment,in.AccountID,userID,expectedType,in.TransactionID).Scan(&samePayload)
  if err!=nil { http.Error(w,`{"error":"Не удалось проверить повтор операции"}`,500); return }
  if !samePayload { http.Error(w,`{"error":"Ключ идемпотентности уже использован для другой записи цели"}`,409); return }
  if txID.Valid { value:=txID.Int64; item.TransactionID=&value }
  if commitErr:=tx.Commit(r.Context()); commitErr!=nil { http.Error(w,`{"error":"Не удалось завершить операцию"}`,500); return }
  _=json.NewEncoder(w).Encode(item); return
 }
 if err!=pgx.ErrNoRows { http.Error(w,`{"error":"Не удалось проверить повтор операции"}`,500); return }
 if goalStatus=="paused"||goalStatus=="completed" { http.Error(w,`{"error":"Пополнение или снятие недоступно для приостановленной или завершённой цели"}`,409); return }
 if in.Type=="withdrawal" {
  var sufficient bool
  err=tx.QueryRow(r.Context(),`SELECT g.starting_amount+COALESCE(SUM(CASE WHEN e.entry_type='contribution' THEN e.amount ELSE -e.amount END),0) >= $2::numeric FROM goals g LEFT JOIN goal_entries e ON e.goal_id=g.id WHERE g.id=$1 GROUP BY g.id`,goalID,in.Amount).Scan(&sufficient)
  if err!=nil||!sufficient { http.Error(w,`{"error":"Недостаточно средств в цели для снятия"}`,409); return }
 }
 var linked any
 if in.AccountID>0&&in.TransactionID>0{http.Error(w,`{"error":"Укажите счёт или готовую операцию, но не оба поля"}`,400);return}
 if in.AccountID==0&&in.TransactionID==0{http.Error(w,`{"error":"Для записи цели выберите личный счёт или связанную финансовую операцию"}`,400);return}
 if in.AccountID>0{
  var accountExists bool
  if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`,in.AccountID,userID).Scan(&accountExists);err!=nil||!accountExists{http.Error(w,`{"error":"Счёт не найден"}`,400);return}
  var goalTitle string
  if err:=tx.QueryRow(r.Context(),`SELECT title FROM goals WHERE id=$1 AND user_id=$2`,goalID,userID).Scan(&goalTitle);err!=nil{http.Error(w,`{"error":"Цель не найдена"}`,404);return}
  txType:="expense";if in.Type=="withdrawal"{txType="income"}
  txKey:="goal-entry:"+strconv.FormatInt(userID,10)+":"+in.IdempotencyKey
  var transactionID int64
  err=tx.QueryRow(r.Context(),`INSERT INTO transactions(user_id,account_id,title,operation_type,amount,category,operation_date,comment,idempotency_key) VALUES($1,$2,$3,$4,$5::numeric,'Цели',CURRENT_DATE,$6,$7) ON CONFLICT(user_id,idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key RETURNING id`,userID,in.AccountID,map[bool]string{true:"Снятие из цели: ",false:"Пополнение цели: "}[in.Type=="withdrawal"]+goalTitle,txType,in.Amount,in.Comment,txKey).Scan(&transactionID)
  if err!=nil{http.Error(w,`{"error":"Не удалось создать связанную финансовую операцию"}`,500);return}
  var transactionMatches bool
  if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM transactions WHERE id=$1 AND user_id=$2 AND account_id=$3 AND amount=$4::numeric AND operation_type=$5 AND category='Цели' AND comment=$6)`,transactionID,userID,in.AccountID,in.Amount,txType,in.Comment).Scan(&transactionMatches);err!=nil{http.Error(w,`{"error":"Не удалось проверить связанную операцию"}`,500);return}
  if !transactionMatches{http.Error(w,`{"error":"Ключ операции уже использован для другой финансовой записи"}`,409);return}
  linked=transactionID
 }
 if in.TransactionID>0 {
  expectedOperation:="expense";if in.Type=="withdrawal"{expectedOperation="income"}
  var linkedID int64
  if err:=tx.QueryRow(r.Context(),`SELECT id FROM transactions WHERE id=$1 AND user_id=$2 AND amount=$3::numeric AND operation_type=$4 FOR UPDATE`,in.TransactionID,userID,in.Amount,expectedOperation).Scan(&linkedID);err!=nil{
   http.Error(w,`{"error":"Связанная операция не найдена или её сумма и тип не совпадают с записью цели"}`,400);return
  }
  linked=linkedID
 }
 if linked!=nil{
  var alreadyLinked bool
  if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM goal_entries WHERE transaction_id=$1 UNION ALL SELECT 1 FROM family_goal_entries WHERE transaction_id=$1)`,linked).Scan(&alreadyLinked);err!=nil{http.Error(w,`{"error":"Не удалось проверить связанную операцию"}`,500);return}
  if alreadyLinked{http.Error(w,`{"error":"Эта финансовая операция уже связана с другой целью"}`,409);return}
 }
 err=tx.QueryRow(r.Context(),`
  INSERT INTO goal_entries(goal_id,user_id,entry_type,amount,comment,transaction_id,idempotency_key)
  VALUES($1,$2,$3,$4::numeric,$5,$6,$7)
  ON CONFLICT(user_id,idempotency_key) DO NOTHING
  RETURNING id,entry_type,amount::text,comment,transaction_id,created_at
 `,goalID,userID,in.Type,in.Amount,in.Comment,linked,in.IdempotencyKey).Scan(&item.ID,&item.Type,&item.Amount,&item.Comment,&txID,&item.CreatedAt)
 if err==pgx.ErrNoRows {
  err=tx.QueryRow(r.Context(),`SELECT id,entry_type,amount::text,comment,transaction_id,created_at FROM goal_entries WHERE user_id=$1 AND idempotency_key=$2`,userID,in.IdempotencyKey).Scan(&item.ID,&item.Type,&item.Amount,&item.Comment,&txID,&item.CreatedAt)
 }
 if err!=nil { http.Error(w,`{"error":"Не удалось сохранить запись"}`,500); return }
 if txID.Valid { value:=txID.Int64; item.TransactionID=&value }
 _,err=tx.Exec(r.Context(),`
  UPDATE goals SET status=CASE
   WHEN starting_amount+COALESCE((SELECT SUM(CASE WHEN entry_type='contribution' THEN amount ELSE -amount END) FROM goal_entries WHERE goal_id=$1),0)>=target_amount THEN 'completed'
   WHEN status='completed' THEN 'active' ELSE status END,updated_at=NOW() WHERE id=$1
 `,goalID)
 if err!=nil { http.Error(w,`{"error":"Не удалось обновить прогресс цели"}`,500); return }
 _,_=tx.Exec(r.Context(),`INSERT INTO notifications(user_id,notification_type,title,message,related_type,related_id) VALUES($1,'goal_entry','Обновлён прогресс цели',$2,'goal',$3)`,userID,in.Type+" · "+in.Amount+" ₽",goalID)
 if err:=tx.Commit(r.Context()); err!=nil { http.Error(w,`{"error":"Не удалось завершить операцию"}`,500); return }
 w.WriteHeader(201); _=json.NewEncoder(w).Encode(item)
}

func goalImageHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID,goalID int64){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 store,err:=newS3Store();if err!=nil{http.Error(w,`{"error":"S3-хранилище не настроено"}`,503);return}
 var oldKey *string
 if err:=database.QueryRow(r.Context(),`SELECT image_key FROM goals WHERE id=$1 AND user_id=$2`,goalID,userID).Scan(&oldKey);err!=nil{http.Error(w,`{"error":"Цель не найдена"}`,404);return}
 if r.Method==http.MethodDelete{
  if _,err:=database.Exec(r.Context(),`UPDATE goals SET image_key=NULL,updated_at=NOW() WHERE id=$1 AND user_id=$2`,goalID,userID);err!=nil{http.Error(w,`{"error":"Не удалось удалить изображение цели"}`,500);return}
  if oldKey!=nil&&*oldKey!=""{if err:=store.delete(r.Context(),*oldKey);err!=nil{log.Printf("failed to delete goal image for goal %d: %v",goalID,err)}}
  _=json.NewEncoder(w).Encode(map[string]bool{"success":true});return
 }
 if r.Method!=http.MethodPut{w.Header().Set("Allow","PUT, DELETE");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
 r.Body=http.MaxBytesReader(w,r.Body,6<<20)
 if err:=r.ParseMultipartForm(6<<20);err!=nil{http.Error(w,`{"error":"Файл слишком большой"}`,400);return}
 file,_,err:=r.FormFile("image");if err!=nil{http.Error(w,`{"error":"Выберите изображение"}`,400);return};defer file.Close()
 data,err:=io.ReadAll(io.LimitReader(file,(5<<20)+1));if err!=nil||len(data)==0||len(data)>5<<20{http.Error(w,`{"error":"Изображение должно быть не больше 5 МБ"}`,400);return}
 contentType:=http.DetectContentType(data);ext:=""
 switch contentType{case "image/jpeg":ext="jpg";case "image/png":ext="png";case "image/webp":ext="webp";case "image/gif":ext="gif";default:http.Error(w,`{"error":"Поддерживаются JPEG, PNG, WebP и GIF"}`,400);return}
 random:=make([]byte,16);if _,err:=rand.Read(random);err!=nil{http.Error(w,`{"error":"Не удалось создать ключ файла"}`,500);return}
 key:="goals/"+strconv.FormatInt(userID,10)+"/"+hex.EncodeToString(random)+"."+ext
 if err:=store.put(r.Context(),key,contentType,data);err!=nil{http.Error(w,`{"error":"Не удалось загрузить изображение в S3"}`,502);return}
 tag,err:=database.Exec(r.Context(),`UPDATE goals SET image_key=$3,updated_at=NOW() WHERE id=$1 AND user_id=$2`,goalID,userID,key)
 if err!=nil||tag.RowsAffected()==0{_ = store.delete(r.Context(),key);http.Error(w,`{"error":"Не удалось сохранить изображение цели"}`,500);return}
 if oldKey!=nil&&*oldKey!=""{if err:=store.delete(r.Context(),*oldKey);err!=nil{log.Printf("failed to delete replaced goal image for goal %d: %v",goalID,err)}}
 url,err:=store.presignGet(key,7*24*time.Hour);if err!=nil{http.Error(w,`{"error":"Изображение сохранено, но не удалось сформировать ссылку"}`,502);return}
 _=json.NewEncoder(w).Encode(map[string]string{"imageUrl":url})
}
