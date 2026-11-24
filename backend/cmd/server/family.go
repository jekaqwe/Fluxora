package main

import (
 "crypto/rand"
 "crypto/sha256"
 "encoding/hex"
 "context"
 "encoding/json"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
)

type familyInfo struct {
 ID int64 `json:"id"`
 Name string `json:"name"`
 InitialBalance string `json:"initialBalance"`
 CurrentBalance string `json:"currentBalance"`
 Role string `json:"role"`
}
type familyMemberInfo struct {
 ID int64 `json:"id"`
 UserID int64 `json:"userId"`
 FirstName string `json:"firstName"`
 LastName string `json:"lastName"`
 Username string `json:"username"`
 Role string `json:"role"`
 JoinedAt time.Time `json:"joinedAt"`
}
type familyCreatePayload struct { Name string `json:"name"`; InitialBalance string `json:"initialBalance"` }
type familyInvitePayload struct { Email string `json:"email"` }
type familyAcceptPayload struct { Token string `json:"token"` }
type familyTxPayload struct {
 Title string `json:"title"`
 Type string `json:"type"`
 Amount string `json:"amount"`
 Category string `json:"category"`
 Date string `json:"date"`
 Comment string `json:"comment"`
 IdempotencyKey string `json:"idempotencyKey"`
}
type familyTxResponse struct {
 ID int64 `json:"id"`
 Title string `json:"title"`
 Type string `json:"type"`
 Amount string `json:"amount"`
 Category string `json:"category"`
 Date string `json:"date"`
 Comment string `json:"comment"`
 Author string `json:"author"`
 CreatedAt time.Time `json:"createdAt"`
}
func familyToken() (string,string,error) {
 raw:=make([]byte,32); if _,err:=rand.Read(raw);err!=nil{return "","",err}
 token:=hex.EncodeToString(raw); sum:=sha256.Sum256([]byte(token)); return token,hex.EncodeToString(sum[:]),nil
}
func loadFamily(database *pgxpool.Pool,userID int64)(familyInfo,error) {
 var f familyInfo
 err:=database.QueryRow(context.Background(),`
  SELECT f.id,f.name,f.initial_balance::text,
   (f.initial_balance+COALESCE((SELECT SUM(CASE WHEN operation_type='income' THEN amount ELSE -amount END) FROM family_transactions WHERE family_id=f.id),0))::text,
   m.role
  FROM family_members m JOIN families f ON f.id=m.family_id WHERE m.user_id=$1
 `,userID).Scan(&f.ID,&f.Name,&f.InitialBalance,&f.CurrentBalance,&f.Role)
 return f,err
}
func familyHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool) {
 w.Header().Set("Content-Type","application/json"); w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database); if err!=nil {http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 path:=strings.TrimSuffix(r.URL.Path,"/")
 switch path {
 case "/api/family":
  switch r.Method {
  case http.MethodGet:
   family,err:=loadFamily(database,userID); if err==pgx.ErrNoRows { _=json.NewEncoder(w).Encode(map[string]any{"family":nil,"members":[]any{}});return }; if err!=nil {http.Error(w,`{"error":"Не удалось загрузить семейный бюджет"}`,500);return}
   rows,err:=database.Query(r.Context(),`SELECT m.id,u.id,u.first_name,u.last_name,u.username,m.role,m.joined_at FROM family_members m JOIN users u ON u.id=m.user_id WHERE m.family_id=$1 ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END,m.joined_at,m.id`,family.ID)
   if err!=nil {http.Error(w,`{"error":"Не удалось загрузить участников"}`,500);return}; defer rows.Close()
   members:=make([]familyMemberInfo,0); for rows.Next(){var m familyMemberInfo;if err:=rows.Scan(&m.ID,&m.UserID,&m.FirstName,&m.LastName,&m.Username,&m.Role,&m.JoinedAt);err!=nil{http.Error(w,`{"error":"Ошибка чтения участников"}`,500);return};members=append(members,m)}
   _=json.NewEncoder(w).Encode(map[string]any{"family":family,"members":members})
  case http.MethodPost:
   r.Body=http.MaxBytesReader(w,r.Body,16<<10);var in familyCreatePayload
   if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
   in.Name=strings.TrimSpace(in.Name);in.InitialBalance=strings.TrimSpace(in.InitialBalance);if in.InitialBalance==""{in.InitialBalance="0"}
   if in.Name==""||len([]rune(in.Name))>100||!accountAmountPattern.MatchString(in.InitialBalance){http.Error(w,`{"error":"Проверьте название и начальный баланс"}`,400);return}
   tx,err:=database.Begin(r.Context());if err!=nil{http.Error(w,`{"error":"Не удалось создать семейный бюджет"}`,500);return};defer tx.Rollback(r.Context())
   var familyID int64
   err=tx.QueryRow(r.Context(),`INSERT INTO families(name,owner_id,initial_balance) VALUES($1,$2,$3::numeric) RETURNING id`,in.Name,userID,in.InitialBalance).Scan(&familyID)
   if err!=nil{http.Error(w,`{"error":"Не удалось создать семейный бюджет"}`,409);return}
   _,err=tx.Exec(r.Context(),`INSERT INTO family_members(family_id,user_id,role) VALUES($1,$2,'owner')`,familyID,userID)
   if err!=nil{http.Error(w,`{"error":"Не удалось назначить владельца"}`,500);return}
   if err:=tx.Commit(r.Context());err!=nil{http.Error(w,`{"error":"Не удалось завершить создание"}`,500);return}
   family,_:=loadFamily(database,userID);w.WriteHeader(201);_=json.NewEncoder(w).Encode(family)
  default:
   w.Header().Set("Allow","GET, POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
  }
 case "/api/family/invitations":
  if r.Method!=http.MethodPost{w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
  family,err:=loadFamily(database,userID);if err!=nil{http.Error(w,`{"error":"Сначала создайте или примите семейное приглашение"}`,409);return}
  if family.Role!="owner"&&family.Role!="admin"{http.Error(w,`{"error":"Недостаточно прав"}`,403);return}
  r.Body=http.MaxBytesReader(w,r.Body,8<<10);var in familyInvitePayload
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
  in.Email=strings.ToLower(strings.TrimSpace(in.Email));if len(in.Email)>255||!strings.Contains(in.Email,"@"){http.Error(w,`{"error":"Укажите корректный email"}`,400);return}
  token,hash,err:=familyToken();if err!=nil{http.Error(w,`{"error":"Не удалось создать приглашение"}`,500);return}
  _,err=database.Exec(r.Context(),`INSERT INTO family_invitations(family_id,email,token_hash,invited_by,expires_at) VALUES($1,$2,$3,$4,NOW()+INTERVAL '7 days')`,family.ID,in.Email,hash,userID)
  if err!=nil{http.Error(w,`{"error":"Не удалось сохранить приглашение"}`,500);return}
  _,_=database.Exec(r.Context(),`INSERT INTO notifications(user_id,notification_type,title,message,related_type,related_id) SELECT id,'family_invite','Приглашение в семейный бюджет','Вас пригласили в семейный бюджет. Используйте полученный токен для принятия.','family',$1 FROM users WHERE LOWER(email)=LOWER($2)`,family.ID,in.Email)
  _=json.NewEncoder(w).Encode(map[string]string{"token":token,"message":"Приглашение создано. Передайте токен участнику вручную."})
 case "/api/family/accept":
  if r.Method!=http.MethodPost{w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
  r.Body=http.MaxBytesReader(w,r.Body,8<<10);var in familyAcceptPayload
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil||strings.TrimSpace(in.Token)==""{http.Error(w,`{"error":"Укажите токен приглашения"}`,400);return}
  sum:=sha256.Sum256([]byte(strings.TrimSpace(in.Token)));hash:=hex.EncodeToString(sum[:])
  tx,err:=database.Begin(r.Context());if err!=nil{http.Error(w,`{"error":"Не удалось принять приглашение"}`,500);return};defer tx.Rollback(r.Context())
  var familyID int64;var inviteEmail string
  err=tx.QueryRow(r.Context(),`SELECT family_id,email FROM family_invitations WHERE token_hash=$1 AND accepted_at IS NULL AND expires_at>NOW() FOR UPDATE`,hash).Scan(&familyID,&inviteEmail)
  if err!=nil{http.Error(w,`{"error":"Приглашение недействительно или истекло"}`,404);return}
  var userEmail string
  if err:=tx.QueryRow(r.Context(),`SELECT email FROM users WHERE id=$1`,userID).Scan(&userEmail);err!=nil{http.Error(w,`{"error":"Пользователь не найден"}`,404);return}
  if !strings.EqualFold(inviteEmail,userEmail){http.Error(w,`{"error":"Приглашение предназначено для другого email"}`,403);return}
  _,err=tx.Exec(r.Context(),`INSERT INTO family_members(family_id,user_id,role) VALUES($1,$2,'member')`,familyID,userID)
  if err!=nil{http.Error(w,`{"error":"Вы уже состоите в семейном бюджете или приглашение недоступно"}`,409);return}
  _,err=tx.Exec(r.Context(),`UPDATE family_invitations SET accepted_at=NOW() WHERE token_hash=$1`,hash)
  if err!=nil{http.Error(w,`{"error":"Не удалось подтвердить приглашение"}`,500);return}
  if err:=tx.Commit(r.Context());err!=nil{http.Error(w,`{"error":"Не удалось завершить принятие приглашения"}`,500);return}
  family,_:=loadFamily(database,userID);_=json.NewEncoder(w).Encode(family)
 case "/api/family/leave":
  if r.Method!=http.MethodPost{w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
  family,err:=loadFamily(database,userID);if err!=nil{http.Error(w,`{"error":"Семейный бюджет не найден"}`,404);return}
  if family.Role=="owner"{http.Error(w,`{"error":"Владелец должен передать права управления перед выходом"}`,409);return}
  _,err=database.Exec(r.Context(),`DELETE FROM family_members WHERE user_id=$1`,userID);if err!=nil{http.Error(w,`{"error":"Не удалось покинуть семейный бюджет"}`,500);return}
  _=json.NewEncoder(w).Encode(map[string]bool{"success":true})
 case "/api/family/transactions":
  familyTransactionsHandler(w,r,database,userID)
 default:
  if strings.HasPrefix(path,"/api/family/members/"){familyMemberHandler(w,r,database,userID,strings.TrimPrefix(path,"/api/family/members/"));return}
  if strings.HasPrefix(path,"/api/family/transactions/"){familyTransactionItemHandler(w,r,database,userID,strings.TrimPrefix(path,"/api/family/transactions/"));return}
  http.NotFound(w,r)
 }
}

func familyMemberHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID int64,rawID string){
 family,err:=loadFamily(database,userID);if err!=nil{http.Error(w,`{"error":"Семейный бюджет не найден"}`,404);return}
 if family.Role!="owner"&&family.Role!="admin"{http.Error(w,`{"error":"Недостаточно прав"}`,403);return}
 memberID,err:=strconv.ParseInt(rawID,10,64);if err!=nil||memberID<=0{http.Error(w,`{"error":"Некорректный участник"}`,400);return}
 var role string
 err=database.QueryRow(r.Context(),`SELECT role FROM family_members WHERE id=$1 AND family_id=$2`,memberID,family.ID).Scan(&role)
 if err!=nil{http.Error(w,`{"error":"Участник не найден"}`,404);return}
 if role=="owner"{http.Error(w,`{"error":"Нельзя изменить или удалить владельца"}`,409);return}
 if family.Role=="admin"&&role=="admin"{http.Error(w,`{"error":"Администратор не может управлять другим администратором"}`,403);return}
 if r.Method==http.MethodDelete{
  _,err=database.Exec(r.Context(),`DELETE FROM family_members WHERE id=$1 AND family_id=$2`,memberID,family.ID)
  if err!=nil{http.Error(w,`{"error":"Не удалось удалить участника"}`,500);return};_=json.NewEncoder(w).Encode(map[string]bool{"success":true});return
 }
 if r.Method==http.MethodPatch{
  if family.Role!="owner"{http.Error(w,`{"error":"Только владелец может менять роли"}`,403);return}
  var input struct{Role string `json:"role"`}
  if err:=json.NewDecoder(r.Body).Decode(&input);err!=nil||(input.Role!="admin"&&input.Role!="member"){http.Error(w,`{"error":"Допустимые роли: admin или member"}`,400);return}
  _,err=database.Exec(r.Context(),`UPDATE family_members SET role=$3 WHERE id=$1 AND family_id=$2`,memberID,family.ID,input.Role)
  if err!=nil{http.Error(w,`{"error":"Не удалось изменить роль"}`,500);return};_=json.NewEncoder(w).Encode(map[string]string{"role":input.Role});return
 }
 w.Header().Set("Allow","PATCH, DELETE");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
}
func familyTransactionsHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID int64){
 family,err:=loadFamily(database,userID);if err!=nil{http.Error(w,`{"error":"Семейный бюджет не найден"}`,404);return}
 if r.Method==http.MethodGet{
  rows,err:=database.Query(r.Context(),`SELECT t.id,t.title,t.operation_type,t.amount::text,t.category,t.operation_date::text,t.comment,CASE WHEN u.id IS NULL THEN 'Удалённый участник' ELSE u.first_name||' '||u.last_name END,t.created_at FROM family_transactions t LEFT JOIN users u ON u.id=t.author_user_id WHERE t.family_id=$1 ORDER BY t.operation_date DESC,t.id DESC LIMIT 500`,family.ID)
  if err!=nil{http.Error(w,`{"error":"Не удалось загрузить семейные операции"}`,500);return};defer rows.Close();items:=make([]familyTxResponse,0)
  for rows.Next(){var x familyTxResponse;if err:=rows.Scan(&x.ID,&x.Title,&x.Type,&x.Amount,&x.Category,&x.Date,&x.Comment,&x.Author,&x.CreatedAt);err!=nil{http.Error(w,`{"error":"Ошибка чтения семейных операций"}`,500);return};items=append(items,x)}
  _=json.NewEncoder(w).Encode(items);return
 }
 if r.Method!=http.MethodPost{w.Header().Set("Allow","GET, POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
 r.Body=http.MaxBytesReader(w,r.Body,24<<10);var in familyTxPayload
 if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
 in.Title=strings.TrimSpace(in.Title);in.Type=strings.TrimSpace(in.Type);in.Amount=strings.TrimSpace(in.Amount);in.Category=strings.TrimSpace(in.Category);in.Comment=strings.TrimSpace(in.Comment);in.IdempotencyKey=strings.TrimSpace(in.IdempotencyKey)
 date,e:=time.Parse("2006-01-02",in.Date)
 if in.Title==""||len([]rune(in.Title))>160||(in.Type!="income"&&in.Type!="expense")||!accountAmountPattern.MatchString(in.Amount)||strings.HasPrefix(in.Amount,"-")||strings.Trim(in.Amount,"0.")==""||in.Category==""||len([]rune(in.Category))>80||len([]rune(in.Comment))>1000||in.IdempotencyKey==""||len(in.IdempotencyKey)>100||e!=nil{http.Error(w,`{"error":"Проверьте поля семейной операции"}`,400);return}
 var item familyTxResponse
 err=database.QueryRow(r.Context(),`INSERT INTO family_transactions(family_id,author_user_id,operation_type,title,amount,category,operation_date,comment,idempotency_key) VALUES($1,$2,$3,$4,$5::numeric,$6,$7,$8,$9) ON CONFLICT(author_user_id,idempotency_key) DO NOTHING RETURNING id,title,operation_type,amount::text,category,operation_date::text,comment,(SELECT first_name||' '||last_name FROM users WHERE id=$2),created_at`,family.ID,userID,in.Type,in.Title,in.Amount,in.Category,date,in.Comment,in.IdempotencyKey).Scan(&item.ID,&item.Title,&item.Type,&item.Amount,&item.Category,&item.Date,&item.Comment,&item.Author,&item.CreatedAt)
created:=true
 if err==pgx.ErrNoRows{created=false;err=database.QueryRow(r.Context(),`SELECT t.id,t.title,t.operation_type,t.amount::text,t.category,t.operation_date::text,t.comment,CASE WHEN u.id IS NULL THEN 'Удалённый участник' ELSE u.first_name||' '||u.last_name END,t.created_at FROM family_transactions t LEFT JOIN users u ON u.id=t.author_user_id WHERE t.author_user_id=$1 AND t.idempotency_key=$2`,userID,in.IdempotencyKey).Scan(&item.ID,&item.Title,&item.Type,&item.Amount,&item.Category,&item.Date,&item.Comment,&item.Author,&item.CreatedAt)}
 if err!=nil{http.Error(w,`{"error":"Не удалось сохранить семейную операцию"}`,500);return}
 if created{_,_=database.Exec(r.Context(),`INSERT INTO notifications(user_id,notification_type,title,message,related_type,related_id) SELECT user_id,'family_activity','Новая семейная операция',$2,'family_transaction',$3 FROM family_members WHERE family_id=$1 AND user_id<>$4`,family.ID,in.Title,item.ID,userID)}
 w.WriteHeader(201);_=json.NewEncoder(w).Encode(item)
}

func familyTransactionItemHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID int64,rawID string){
 family,err:=loadFamily(database,userID);if err!=nil{http.Error(w,`{"error":"Семейный бюджет не найден"}`,404);return}
 id,err:=strconv.ParseInt(rawID,10,64);if err!=nil||id<=0{http.Error(w,`{"error":"Некорректная операция"}`,400);return}
 var author pgtype.Int8
 if err:=database.QueryRow(r.Context(),`SELECT author_user_id FROM family_transactions WHERE id=$1 AND family_id=$2`,id,family.ID).Scan(&author);err!=nil{http.Error(w,`{"error":"Операция не найдена"}`,404);return}
 allowed:=author.Valid&&author.Int64==userID||family.Role=="owner"||family.Role=="admin"
 if !allowed{http.Error(w,`{"error":"Недостаточно прав для изменения операции"}`,403);return}
 if r.Method==http.MethodDelete {
  _,err:=database.Exec(r.Context(),`DELETE FROM family_transactions WHERE id=$1 AND family_id=$2`,id,family.ID)
  if err!=nil{http.Error(w,`{"error":"Не удалось удалить операцию"}`,500);return}
  _=json.NewEncoder(w).Encode(map[string]bool{"success":true});return
 }
 if r.Method==http.MethodPatch {
  r.Body=http.MaxBytesReader(w,r.Body,24<<10);var in familyTxPayload
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
  in.Title=strings.TrimSpace(in.Title);in.Type=strings.TrimSpace(in.Type);in.Amount=strings.TrimSpace(in.Amount);in.Category=strings.TrimSpace(in.Category);in.Comment=strings.TrimSpace(in.Comment)
  date,e:=time.Parse("2006-01-02",in.Date)
  if in.Title==""||len([]rune(in.Title))>160||(in.Type!="income"&&in.Type!="expense")||!accountAmountPattern.MatchString(in.Amount)||strings.HasPrefix(in.Amount,"-")||strings.Trim(in.Amount,"0.")==""||in.Category==""||len([]rune(in.Category))>80||len([]rune(in.Comment))>1000||e!=nil{http.Error(w,`{"error":"Проверьте поля семейной операции"}`,400);return}
  var item familyTxResponse
  err:=database.QueryRow(r.Context(),`UPDATE family_transactions SET title=$3,operation_type=$4,amount=$5::numeric,category=$6,operation_date=$7,comment=$8,updated_at=NOW()
   WHERE id=$1 AND family_id=$2
   RETURNING id,title,operation_type,amount::text,category,operation_date::text,comment,
    COALESCE((SELECT first_name||' '||last_name FROM users WHERE id=family_transactions.author_user_id),'Удалённый участник'),created_at`,
   id,family.ID,in.Title,in.Type,in.Amount,in.Category,date,in.Comment).Scan(&item.ID,&item.Title,&item.Type,&item.Amount,&item.Category,&item.Date,&item.Comment,&item.Author,&item.CreatedAt)
  if err!=nil{http.Error(w,`{"error":"Не удалось обновить семейную операцию"}`,500);return}
  _=json.NewEncoder(w).Encode(item);return
 }
 w.Header().Set("Allow","PATCH, DELETE");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
}
