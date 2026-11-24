package main

import (
 "crypto/rand"
 "encoding/hex"
 "encoding/json"
 "io"
 "log"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
)

type familyGoalPayload struct {
 Title string `json:"title"`
 Description string `json:"description"`
 TargetAmount string `json:"targetAmount"`
 StartingAmount string `json:"startingAmount"`
 DueDate string `json:"dueDate"`
 Priority string `json:"priority"`
 Status string `json:"status"`
}
type familyGoalResponse struct {
 ID int64 `json:"id"`
 Title string `json:"title"`
 Description string `json:"description"`
 TargetAmount string `json:"targetAmount"`
 StartingAmount string `json:"startingAmount"`
 CurrentAmount string `json:"currentAmount"`
 DueDate *string `json:"dueDate,omitempty"`
 Priority string `json:"priority"`
 Status string `json:"status"`
 ImageURL *string `json:"imageUrl,omitempty"`
 CreatedBy *int64 `json:"createdBy,omitempty"`
 CreatedAt time.Time `json:"createdAt"`
}
type familyGoalEntryPayload struct { Amount string `json:"amount"`;Comment string `json:"comment"`;EntryType string `json:"entryType"`;IdempotencyKey string `json:"idempotencyKey"`;AccountID int64 `json:"accountId"`;TransactionID int64 `json:"transactionId"` }
type familyGoalEntryResponse struct { ID int64 `json:"id"`;Amount string `json:"amount"`;EntryType string `json:"entryType"`;Comment string `json:"comment"`;Author string `json:"author"`;CreatedAt time.Time `json:"createdAt"` }

func familyGoalsHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID int64){
 family,err:=loadFamily(database,userID);if err!=nil{http.Error(w,`{"error":"Семейный бюджет не найден"}`,404);return}
 path:=strings.TrimSuffix(r.URL.Path,"/")
 if path=="/api/family/goals"{
  switch r.Method{
  case http.MethodGet:
   rows,err:=database.Query(r.Context(),`SELECT g.id,g.title,g.description,g.target_amount::text,g.starting_amount::text,(g.starting_amount+COALESCE(SUM(CASE WHEN e.entry_type='withdrawal' THEN -e.amount ELSE e.amount END),0))::text,g.due_date::text,g.priority,g.status,g.image_key,g.created_by,g.created_at FROM family_goals g LEFT JOIN family_goal_entries e ON e.goal_id=g.id WHERE g.family_id=$1 GROUP BY g.id ORDER BY g.created_at DESC,g.id DESC`,family.ID)
   if err!=nil{http.Error(w,`{"error":"Не удалось загрузить семейные цели"}`,500);return};defer rows.Close();items:=make([]familyGoalResponse,0)
   var store *s3Store
   for rows.Next(){var x familyGoalResponse;var due,image *string;var creator pgtype.Int8;if err:=rows.Scan(&x.ID,&x.Title,&x.Description,&x.TargetAmount,&x.StartingAmount,&x.CurrentAmount,&due,&x.Priority,&x.Status,&image,&creator,&x.CreatedAt);err!=nil{http.Error(w,`{"error":"Ошибка чтения семейных целей"}`,500);return};x.DueDate=due;if creator.Valid{v:=creator.Int64;x.CreatedBy=&v};if image!=nil&&*image!=""{if store==nil{store,_=newS3Store()};if store!=nil{if url,e:=store.presignGet(*image,7*24*time.Hour);e==nil{x.ImageURL=&url}}};items=append(items,x)}
   _=json.NewEncoder(w).Encode(items)
  case http.MethodPost:
   r.Body=http.MaxBytesReader(w,r.Body,24<<10);var in familyGoalPayload;if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
   in.Title=strings.TrimSpace(in.Title);in.Description=strings.TrimSpace(in.Description);in.TargetAmount=strings.TrimSpace(in.TargetAmount);in.StartingAmount=strings.TrimSpace(in.StartingAmount);if in.StartingAmount==""{in.StartingAmount="0"};if in.Priority==""{in.Priority="medium"}
   if in.Title==""||len([]rune(in.Title))>120||len([]rune(in.Description))>1000||!validGoalAmount(in.TargetAmount,false)||!validGoalAmount(in.StartingAmount,true)||(in.Priority!="low"&&in.Priority!="medium"&&in.Priority!="high"){http.Error(w,`{"error":"Проверьте название, суммы и приоритет"}`,400);return}
   due,err:=parseGoalDate(in.DueDate);if err!=nil{http.Error(w,`{"error":"Некорректная дата цели"}`,400);return}
   var x familyGoalResponse;var dueText *string;var creator pgtype.Int8
   err=database.QueryRow(r.Context(),`INSERT INTO family_goals(family_id,created_by,title,description,target_amount,starting_amount,due_date,priority) VALUES($1,$2,$3,$4,$5::numeric,$6::numeric,$7,$8) RETURNING id,title,description,target_amount::text,starting_amount::text,starting_amount::text,due_date::text,priority,status,created_by,created_at`,family.ID,userID,in.Title,in.Description,in.TargetAmount,in.StartingAmount,due,in.Priority).Scan(&x.ID,&x.Title,&x.Description,&x.TargetAmount,&x.StartingAmount,&x.CurrentAmount,&dueText,&x.Priority,&x.Status,&creator,&x.CreatedAt)
   if err!=nil{http.Error(w,`{"error":"Не удалось создать семейную цель"}`,500);return};x.DueDate=dueText;if creator.Valid{v:=creator.Int64;x.CreatedBy=&v};w.WriteHeader(201);_=json.NewEncoder(w).Encode(x)
  default:w.Header().Set("Allow","GET, POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
  }
  return
 }
 parts:=strings.Split(strings.Trim(strings.TrimPrefix(path,"/api/family/goals/"),"/"),"/")
 id,err:=strconv.ParseInt(parts[0],10,64);if err!=nil||id<=0{http.Error(w,`{"error":"Некорректная цель"}`,400);return}
 if len(parts)==2&&parts[1]=="entries"{familyGoalEntriesHandler(w,r,database,userID,family.ID,id);return}
 if len(parts)==2&&parts[1]=="image"{familyGoalImageHandler(w,r,database,userID,family.ID,id);return}
 if len(parts)!=1{http.Error(w,`{"error":"Некорректный адрес цели"}`,400);return}
 var creator pgtype.Int8
 if err:=database.QueryRow(r.Context(),`SELECT created_by FROM family_goals WHERE id=$1 AND family_id=$2`,id,family.ID).Scan(&creator);err!=nil{http.Error(w,`{"error":"Цель не найдена"}`,404);return}
 canManage:=family.Role=="owner"||family.Role=="admin"||(creator.Valid&&creator.Int64==userID)
 switch r.Method{
 case http.MethodDelete:
  if !canManage{http.Error(w,`{"error":"Недостаточно прав"}`,403);return}
  var imageKey *string;_ = database.QueryRow(r.Context(),`SELECT image_key FROM family_goals WHERE id=$1 AND family_id=$2`,id,family.ID).Scan(&imageKey)
  _,err:=database.Exec(r.Context(),`DELETE FROM family_goals WHERE id=$1 AND family_id=$2`,id,family.ID);if err!=nil{http.Error(w,`{"error":"Не удалось удалить семейную цель"}`,500);return};if imageKey!=nil&&*imageKey!=""{if store,e:=newS3Store();e==nil{if e:=store.delete(r.Context(),*imageKey);e!=nil{log.Printf("failed to delete family goal image %d: %v",id,e)}}};_=json.NewEncoder(w).Encode(map[string]bool{"success":true})
 case http.MethodPatch:
  if !canManage{http.Error(w,`{"error":"Недостаточно прав"}`,403);return}
  r.Body=http.MaxBytesReader(w,r.Body,24<<10);var in familyGoalPayload;if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
  in.Title=strings.TrimSpace(in.Title);in.Description=strings.TrimSpace(in.Description);in.TargetAmount=strings.TrimSpace(in.TargetAmount);in.StartingAmount=strings.TrimSpace(in.StartingAmount)
  if in.Title==""||len([]rune(in.Title))>120||len([]rune(in.Description))>1000||!validGoalAmount(in.TargetAmount,false)||!validGoalAmount(in.StartingAmount,true)||(in.Priority!="low"&&in.Priority!="medium"&&in.Priority!="high"){http.Error(w,`{"error":"Проверьте название, суммы и приоритет"}`,400);return}
  if in.Status!="active"&&in.Status!="paused"&&in.Status!="completed"{http.Error(w,`{"error":"Некорректный статус"}`,400);return}
  due,err:=parseGoalDate(in.DueDate);if err!=nil{http.Error(w,`{"error":"Некорректная дата"}`,400);return}
  _,err=database.Exec(r.Context(),`UPDATE family_goals SET title=$3,description=$4,target_amount=$5::numeric,starting_amount=$6::numeric,due_date=$7,priority=$8,status=CASE WHEN starting_amount+COALESCE((SELECT SUM(CASE WHEN entry_type='withdrawal' THEN -amount ELSE amount END) FROM family_goal_entries WHERE goal_id=family_goals.id),0)>=$5::numeric THEN 'completed' WHEN $9='paused' THEN 'paused' ELSE 'active' END,updated_at=NOW() WHERE id=$1 AND family_id=$2`,id,family.ID,in.Title,in.Description,in.TargetAmount,in.StartingAmount,due,in.Priority,in.Status)
  if err!=nil{http.Error(w,`{"error":"Не удалось обновить семейную цель"}`,500);return};_=json.NewEncoder(w).Encode(map[string]bool{"success":true})
 default:w.Header().Set("Allow","PATCH, DELETE");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
 }
}

func familyGoalEntriesHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID,familyID,goalID int64){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 if r.Method==http.MethodGet{
  rows,err:=database.Query(r.Context(),`SELECT e.id,e.amount::text,e.entry_type,e.comment,COALESCE(u.first_name||' '||u.last_name,'Удалённый участник'),e.created_at FROM family_goal_entries e LEFT JOIN users u ON u.id=e.author_user_id WHERE e.goal_id=$1 AND e.family_id=$2 ORDER BY e.created_at DESC,e.id DESC`,goalID,familyID)
  if err!=nil{http.Error(w,`{"error":"Не удалось загрузить историю цели"}`,500);return};defer rows.Close();items:=make([]familyGoalEntryResponse,0)
  for rows.Next(){var x familyGoalEntryResponse;if err:=rows.Scan(&x.ID,&x.Amount,&x.EntryType,&x.Comment,&x.Author,&x.CreatedAt);err!=nil{http.Error(w,`{"error":"Ошибка истории цели"}`,500);return};items=append(items,x)}
  _=json.NewEncoder(w).Encode(items);return
 }
 if r.Method!=http.MethodPost{w.Header().Set("Allow","GET, POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
 r.Body=http.MaxBytesReader(w,r.Body,12<<10);var in familyGoalEntryPayload;if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
 in.Amount=strings.TrimSpace(in.Amount);in.Comment=strings.TrimSpace(in.Comment);in.EntryType=strings.TrimSpace(in.EntryType);in.IdempotencyKey=strings.TrimSpace(in.IdempotencyKey);if in.EntryType==""{in.EntryType="contribution"}
 if !validGoalAmount(in.Amount,false)||(in.EntryType!="contribution"&&in.EntryType!="withdrawal")||len([]rune(in.Comment))>500||in.IdempotencyKey==""||len(in.IdempotencyKey)>40||in.AccountID<0||in.TransactionID<0{http.Error(w,`{"error":"Проверьте сумму, комментарий, счёт и ключ операции"}`,400);return}
 tx,err:=database.Begin(r.Context());if err!=nil{http.Error(w,`{"error":"Не удалось начать пополнение"}`,500);return};defer tx.Rollback(r.Context())
 var status string;if err:=tx.QueryRow(r.Context(),`SELECT status FROM family_goals WHERE id=$1 AND family_id=$2 FOR UPDATE`,goalID,familyID).Scan(&status);err!=nil{http.Error(w,`{"error":"Семейная цель не найдена"}`,404);return}
 var item familyGoalEntryResponse
 err=tx.QueryRow(r.Context(),`SELECT e.id,e.amount::text,e.entry_type,e.comment,COALESCE(u.first_name||' '||u.last_name,'Удалённый участник'),e.created_at FROM family_goal_entries e LEFT JOIN users u ON u.id=e.author_user_id WHERE e.family_id=$1 AND e.author_user_id=$2 AND e.idempotency_key=$3`,familyID,userID,in.IdempotencyKey).Scan(&item.ID,&item.Amount,&item.EntryType,&item.Comment,&item.Author,&item.CreatedAt)
 if err==nil{
  var samePayload bool
  err=tx.QueryRow(r.Context(),`SELECT EXISTS(
   SELECT 1 FROM family_goal_entries e
   WHERE e.id=$1 AND e.goal_id=$2 AND e.family_id=$3 AND e.author_user_id=$4 AND e.amount=$5::numeric AND e.entry_type=$6 AND e.comment=$7
   AND (
    ($8::bigint>0 AND EXISTS(SELECT 1 FROM transactions t WHERE t.id=e.transaction_id AND t.user_id=$4 AND t.account_id=$8 AND t.amount=$5::numeric AND t.operation_type=CASE WHEN $6='withdrawal' THEN 'income' ELSE 'expense' END))
    OR ($9::bigint>0 AND e.transaction_id=$9)
    OR ($8::bigint=0 AND $9::bigint=0 AND e.transaction_id IS NULL)
   )
  )`,item.ID,goalID,familyID,userID,in.Amount,in.EntryType,in.Comment,in.AccountID,in.TransactionID).Scan(&samePayload)
  if err!=nil{http.Error(w,`{"error":"Не удалось проверить повтор пополнения"}`,500);return}
  if !samePayload{http.Error(w,`{"error":"Ключ идемпотентности уже использован для другого пополнения"}`,409);return}
  if e:=tx.Commit(r.Context());e!=nil{http.Error(w,`{"error":"Не удалось завершить пополнение"}`,500);return}
  _=json.NewEncoder(w).Encode(item);return
 }
 if err!=pgx.ErrNoRows{http.Error(w,`{"error":"Не удалось проверить повтор пополнения"}`,500);return}
 if status=="paused"||(status=="completed"&&in.EntryType=="contribution"){http.Error(w,`{"error":"Операция недоступна для приостановленной или завершённой цели"}`,409);return}
 if in.EntryType=="withdrawal"{var enough bool;if err:=tx.QueryRow(r.Context(),`SELECT starting_amount+COALESCE((SELECT SUM(CASE WHEN entry_type='withdrawal' THEN -amount ELSE amount END) FROM family_goal_entries WHERE goal_id=$1),0)>=$2::numeric FROM family_goals WHERE id=$1`,goalID,in.Amount).Scan(&enough);err!=nil{http.Error(w,`{"error":"Не удалось проверить остаток цели"}`,500);return};if !enough{http.Error(w,`{"error":"Сумма снятия превышает остаток семейной цели"}`,409);return}}
 if in.AccountID>0&&in.TransactionID>0{http.Error(w,`{"error":"Укажите счёт или готовую операцию, но не оба поля"}`,400);return}
 if in.AccountID==0&&in.TransactionID==0{http.Error(w,`{"error":"Для операции выберите личный счёт или связанную финансовую запись"}`,400);return}
 if in.AccountID>0{
  var exists bool;if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`,in.AccountID,userID).Scan(&exists);err!=nil||!exists{http.Error(w,`{"error":"Личный счёт не найден"}`,400);return}
  var goalTitle string;if err:=tx.QueryRow(r.Context(),`SELECT title FROM family_goals WHERE id=$1 AND family_id=$2`,goalID,familyID).Scan(&goalTitle);err!=nil{http.Error(w,`{"error":"Семейная цель не найдена"}`,404);return}
  transactionKey:="family-goal:"+strconv.FormatInt(familyID,10)+":"+strconv.FormatInt(userID,10)+":"+in.IdempotencyKey
  operationType:="expense";transactionTitle:="Вклад в семейную цель: "+goalTitle
  if in.EntryType=="withdrawal"{operationType="income";transactionTitle="Вывод из семейной цели: "+goalTitle}
  var transactionID int64
  err=tx.QueryRow(r.Context(),`INSERT INTO transactions(user_id,account_id,title,operation_type,amount,category,operation_date,comment,idempotency_key) VALUES($1,$2,$3,$4,$5::numeric,'Семейные цели',CURRENT_DATE,$6,$7) ON CONFLICT(user_id,idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key RETURNING id`,userID,in.AccountID,transactionTitle,operationType,in.Amount,in.Comment,transactionKey).Scan(&transactionID)
  if err!=nil{http.Error(w,`{"error":"Не удалось создать финансовую операцию"}`,500);return}
  var transactionMatches bool
  if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM transactions WHERE id=$1 AND user_id=$2 AND account_id=$3 AND amount=$4::numeric AND operation_type=$6 AND category='Семейные цели' AND comment=$5 AND title=$7)`,transactionID,userID,in.AccountID,in.Amount,in.Comment,operationType,transactionTitle).Scan(&transactionMatches);err!=nil{http.Error(w,`{"error":"Не удалось проверить связанную операцию"}`,500);return}
  if !transactionMatches{http.Error(w,`{"error":"Ключ операции уже использован для другой финансовой записи"}`,409);return}
  in.TransactionID=transactionID
 }
 if in.TransactionID>0{
  var linkedID int64
  if err:=tx.QueryRow(r.Context(),`SELECT id FROM transactions WHERE id=$1 AND user_id=$2 AND amount=$3::numeric AND operation_type=CASE WHEN $4='withdrawal' THEN 'income' ELSE 'expense' END FOR UPDATE`,in.TransactionID,userID,in.Amount,in.EntryType).Scan(&linkedID);err!=nil{
   http.Error(w,`{"error":"Связанная операция не найдена или её сумма и тип не совпадают с пополнением"}`,400);return
  }
  in.TransactionID=linkedID
 }
 var author string;_=tx.QueryRow(r.Context(),`SELECT first_name||' '||last_name FROM users WHERE id=$1`,userID).Scan(&author)
 _,err=tx.Exec(r.Context(),`INSERT INTO family_goal_entries(goal_id,family_id,author_user_id,amount,entry_type,comment,transaction_id,idempotency_key) VALUES($1,$2,$3,$4::numeric,$5,$6,NULLIF($7,0),$8)`,goalID,familyID,userID,in.Amount,in.EntryType,in.Comment,in.TransactionID,in.IdempotencyKey)
 if err!=nil{http.Error(w,`{"error":"Не удалось сохранить пополнение"}`,500);return}
 err=tx.QueryRow(r.Context(),`SELECT id,amount::text,entry_type,comment,COALESCE((SELECT first_name||' '||last_name FROM users WHERE id=$2),'Удалённый участник'),created_at FROM family_goal_entries WHERE goal_id=$1 AND author_user_id=$2 AND idempotency_key=$3`,goalID,userID,in.IdempotencyKey).Scan(&item.ID,&item.Amount,&item.EntryType,&item.Comment,&item.Author,&item.CreatedAt)
 if err!=nil{http.Error(w,`{"error":"Не удалось прочитать пополнение"}`,500);return}
 _,err=tx.Exec(r.Context(),`UPDATE family_goals SET status=CASE WHEN starting_amount+COALESCE((SELECT SUM(CASE WHEN entry_type='withdrawal' THEN -amount ELSE amount END) FROM family_goal_entries WHERE goal_id=$1),0)>=target_amount THEN 'completed' WHEN status='paused' THEN 'paused' ELSE 'active' END,updated_at=NOW() WHERE id=$1`,goalID)
 if err!=nil{http.Error(w,`{"error":"Не удалось обновить прогресс цели"}`,500);return}
 _,_=tx.Exec(r.Context(),`INSERT INTO notifications(user_id,notification_type,title,message,related_type,related_id) SELECT user_id,'family_goal',CASE WHEN $5='withdrawal' THEN 'Вывод из семейной цели' ELSE 'Пополнение семейной цели' END,$2,'family_goal',$3 FROM family_members WHERE family_id=$1 AND user_id<>$4`,familyID,in.Amount+" ₽",goalID,userID,in.EntryType)
 if err:=tx.Commit(r.Context());err!=nil{http.Error(w,`{"error":"Не удалось завершить пополнение"}`,500);return}
 w.WriteHeader(201);_=json.NewEncoder(w).Encode(item)
}

func familyGoalImageHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID,familyID,goalID int64){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 var creator pgtype.Int8;if err:=database.QueryRow(r.Context(),`SELECT created_by FROM family_goals WHERE id=$1 AND family_id=$2`,goalID,familyID).Scan(&creator);err!=nil{http.Error(w,`{"error":"Цель не найдена"}`,404);return}
 var role string;_=database.QueryRow(r.Context(),`SELECT role FROM family_members WHERE user_id=$1 AND family_id=$2`,userID,familyID).Scan(&role)
 if role!="owner"&&role!="admin"&&(!creator.Valid||creator.Int64!=userID){http.Error(w,`{"error":"Недостаточно прав"}`,403);return}
 store,err:=newS3Store();if err!=nil{http.Error(w,`{"error":"S3-хранилище не настроено"}`,503);return}
 var old *string;_=database.QueryRow(r.Context(),`SELECT image_key FROM family_goals WHERE id=$1`,goalID).Scan(&old)
 if r.Method==http.MethodDelete{_,err=database.Exec(r.Context(),`UPDATE family_goals SET image_key=NULL WHERE id=$1`,goalID);if err!=nil{http.Error(w,`{"error":"Не удалось удалить изображение"}`,500);return};if old!=nil&&*old!=""{_=store.delete(r.Context(),*old)};_=json.NewEncoder(w).Encode(map[string]bool{"success":true});return}
 if r.Method!=http.MethodPut{w.Header().Set("Allow","PUT, DELETE");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
 r.Body=http.MaxBytesReader(w,r.Body,6<<20);if err:=r.ParseMultipartForm(6<<20);err!=nil{http.Error(w,`{"error":"Файл слишком большой"}`,400);return}
 file,_,err:=r.FormFile("image");if err!=nil{http.Error(w,`{"error":"Выберите изображение"}`,400);return};defer file.Close()
 data,err:=io.ReadAll(io.LimitReader(file,(5<<20)+1));if err!=nil||len(data)==0||len(data)>5<<20{http.Error(w,`{"error":"Изображение должно быть не больше 5 МБ"}`,400);return}
 contentType:=http.DetectContentType(data);ext:="";switch contentType{case "image/jpeg":ext="jpg";case "image/png":ext="png";case "image/webp":ext="webp";case "image/gif":ext="gif";default:http.Error(w,`{"error":"Поддерживаются JPEG, PNG, WebP и GIF"}`,400);return}
 random:=make([]byte,16);if _,err:=rand.Read(random);err!=nil{http.Error(w,`{"error":"Не удалось создать ключ"}`,500);return}
 key:="family-goals/"+strconv.FormatInt(familyID,10)+"/"+hex.EncodeToString(random)+"."+ext
 if err:=store.put(r.Context(),key,contentType,data);err!=nil{http.Error(w,`{"error":"Не удалось загрузить изображение"}`,502);return}
 if _,err:=database.Exec(r.Context(),`UPDATE family_goals SET image_key=$2,updated_at=NOW() WHERE id=$1`,goalID,key);err!=nil{_=store.delete(r.Context(),key);http.Error(w,`{"error":"Не удалось сохранить изображение"}`,500);return}
 if old!=nil&&*old!=""{if err:=store.delete(r.Context(),*old);err!=nil{log.Printf("failed to delete old family goal image %d: %v",goalID,err)}}
 url,err:=store.presignGet(key,7*24*time.Hour);if err!=nil{http.Error(w,`{"error":"Изображение сохранено, но ссылка не сформирована"}`,502);return}
 _=json.NewEncoder(w).Encode(map[string]string{"imageUrl":url})
}
