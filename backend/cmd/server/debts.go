package main

import (
 "encoding/json"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
)

type debtPayload struct {
 Name string `json:"name"`
 Direction string `json:"direction"`
 InitialAmount string `json:"initialAmount"`
 DueDate string `json:"dueDate"`
 Comment string `json:"comment"`
 Status string `json:"status"`
}
type debtResponse struct {
 ID int64 `json:"id"`
 Name string `json:"name"`
 Direction string `json:"direction"`
 InitialAmount string `json:"initialAmount"`
 RemainingAmount string `json:"remainingAmount"`
 DueDate *string `json:"dueDate,omitempty"`
 Comment string `json:"comment"`
 Status string `json:"status"`
 CreatedAt time.Time `json:"createdAt"`
}
type debtPaymentPayload struct {
 Amount string `json:"amount"`
 AccountID int64 `json:"accountId"`
 Date string `json:"date"`
 Comment string `json:"comment"`
 IdempotencyKey string `json:"idempotencyKey"`
}
type debtPaymentResponse struct {
 ID int64 `json:"id"`
 Amount string `json:"amount"`
 AccountID int64 `json:"accountId"`
 TransactionID int64 `json:"transactionId"`
 Date string `json:"date"`
 Comment string `json:"comment"`
 CreatedAt time.Time `json:"createdAt"`
}
func debtsHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 if r.URL.Path!="/api/debts"{debtItemHandler(w,r,database);return}
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 switch r.Method{
 case http.MethodGet:
  rows,err:=database.Query(r.Context(),`SELECT d.id,d.name,d.direction,d.initial_amount::text,(d.initial_amount-COALESCE(SUM(p.amount),0))::text,d.due_date::text,d.comment,d.status,d.created_at FROM debts d LEFT JOIN debt_payments p ON p.debt_id=d.id AND p.user_id=d.user_id WHERE d.user_id=$1 GROUP BY d.id ORDER BY d.created_at DESC,d.id DESC`,userID)
  if err!=nil{http.Error(w,`{"error":"Не удалось загрузить долги"}`,500);return};defer rows.Close();items:=make([]debtResponse,0)
  for rows.Next(){var x debtResponse;var due *string;if err:=rows.Scan(&x.ID,&x.Name,&x.Direction,&x.InitialAmount,&x.RemainingAmount,&due,&x.Comment,&x.Status,&x.CreatedAt);err!=nil{http.Error(w,`{"error":"Ошибка чтения долгов"}`,500);return};x.DueDate=due;items=append(items,x)}
  _=json.NewEncoder(w).Encode(items)
 case http.MethodPost:
  r.Body=http.MaxBytesReader(w,r.Body,16<<10);var in debtPayload
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
  in.Name=strings.TrimSpace(in.Name);in.Direction=strings.TrimSpace(in.Direction);in.InitialAmount=strings.TrimSpace(in.InitialAmount);in.Comment=strings.TrimSpace(in.Comment)
  if in.Name==""||len([]rune(in.Name))>160||(in.Direction!="i_owe"&&in.Direction!="owed_to_me")||!accountAmountPattern.MatchString(in.InitialAmount)||strings.HasPrefix(in.InitialAmount,"-")||strings.Trim(in.InitialAmount,"0.")==""||len([]rune(in.Comment))>1000{http.Error(w,`{"error":"Проверьте название, направление и сумму долга"}`,400);return}
  var due any;if in.DueDate!=""{d,e:=time.Parse("2006-01-02",in.DueDate);if e!=nil{http.Error(w,`{"error":"Некорректная дата возврата"}`,400);return};due=d}
  var x debtResponse;var dueText *string
  err=database.QueryRow(r.Context(),`INSERT INTO debts(user_id,name,direction,initial_amount,due_date,comment) VALUES($1,$2,$3,$4::numeric,$5,$6) RETURNING id,name,direction,initial_amount::text,initial_amount::text,due_date::text,comment,status,created_at`,userID,in.Name,in.Direction,in.InitialAmount,due,in.Comment).Scan(&x.ID,&x.Name,&x.Direction,&x.InitialAmount,&x.RemainingAmount,&dueText,&x.Comment,&x.Status,&x.CreatedAt)
  if err!=nil{http.Error(w,`{"error":"Не удалось создать долг"}`,500);return};x.DueDate=dueText;w.WriteHeader(201);_=json.NewEncoder(w).Encode(x)
 default:w.Header().Set("Allow","GET, POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
 }
}
func debtItemHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 parts:=strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path,"/api/debts/"),"/"),"/");id,err:=strconv.ParseInt(parts[0],10,64)
 if err!=nil||id<=0{http.Error(w,`{"error":"Некорректный идентификатор долга"}`,400);return}
 if len(parts)==2&&parts[1]=="payments"{debtPaymentsHandler(w,r,database,userID,id);return};if len(parts)!=1{http.Error(w,`{"error":"Некорректный адрес"}`,400);return}
 switch r.Method{
 case http.MethodDelete:
  tag,err:=database.Exec(r.Context(),`DELETE FROM debts WHERE id=$1 AND user_id=$2`,id,userID);if err!=nil{http.Error(w,`{"error":"Не удалось удалить долг"}`,500);return};if tag.RowsAffected()==0{http.Error(w,`{"error":"Долг не найден"}`,404);return};_=json.NewEncoder(w).Encode(map[string]bool{"success":true})
 case http.MethodPatch:
  r.Body=http.MaxBytesReader(w,r.Body,16<<10);var in debtPayload
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
  in.Name=strings.TrimSpace(in.Name);in.Comment=strings.TrimSpace(in.Comment)
  if in.Name==""||len([]rune(in.Name))>160||(in.Status!="open"&&in.Status!="closed")||len([]rune(in.Comment))>1000{http.Error(w,`{"error":"Проверьте поля долга"}`,400);return}
  var due any;if in.DueDate!=""{d,e:=time.Parse("2006-01-02",in.DueDate);if e!=nil{http.Error(w,`{"error":"Некорректная дата возврата"}`,400);return};due=d}
  tag,err:=database.Exec(r.Context(),`UPDATE debts SET name=$3,due_date=$4,comment=$5,status=$6,updated_at=NOW() WHERE id=$1 AND user_id=$2`,id,userID,in.Name,due,in.Comment,in.Status)
  if err!=nil{http.Error(w,`{"error":"Не удалось обновить долг"}`,500);return};if tag.RowsAffected()==0{http.Error(w,`{"error":"Долг не найден"}`,404);return};_=json.NewEncoder(w).Encode(map[string]bool{"success":true})
 default:w.Header().Set("Allow","PATCH, DELETE");http.Error(w,`{"error":"Метод не поддерживается"}`,405)
 }
}
func debtPaymentsHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool,userID,debtID int64){
 if r.Method==http.MethodGet{
  rows,err:=database.Query(r.Context(),`SELECT id,amount::text,account_id,transaction_id,payment_date::text,comment,created_at FROM debt_payments WHERE debt_id=$1 AND user_id=$2 ORDER BY payment_date DESC,id DESC`,debtID,userID)
  if err!=nil{http.Error(w,`{"error":"Не удалось загрузить платежи"}`,500);return};defer rows.Close();items:=make([]debtPaymentResponse,0)
  for rows.Next(){var x debtPaymentResponse;if err:=rows.Scan(&x.ID,&x.Amount,&x.AccountID,&x.TransactionID,&x.Date,&x.Comment,&x.CreatedAt);err!=nil{http.Error(w,`{"error":"Ошибка чтения платежей"}`,500);return};items=append(items,x)}
  _=json.NewEncoder(w).Encode(items);return
 }
 if r.Method!=http.MethodPost{w.Header().Set("Allow","GET, POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
 r.Body=http.MaxBytesReader(w,r.Body,16<<10);var in debtPaymentPayload
 if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
 in.Amount=strings.TrimSpace(in.Amount);in.Comment=strings.TrimSpace(in.Comment);in.IdempotencyKey=strings.TrimSpace(in.IdempotencyKey)
 date,err:=time.Parse("2006-01-02",in.Date)
 if !accountAmountPattern.MatchString(in.Amount)||strings.HasPrefix(in.Amount,"-")||strings.Trim(in.Amount,"0.")==""||in.AccountID<=0||len([]rune(in.Comment))>500||in.IdempotencyKey==""||len(in.IdempotencyKey)>80||err!=nil{http.Error(w,`{"error":"Проверьте сумму, счёт, дату и ключ платежа"}`,400);return}
 tx,err:=database.Begin(r.Context());if err!=nil{http.Error(w,`{"error":"Не удалось начать платёж"}`,500);return};defer tx.Rollback(r.Context())
 var name,direction,status string;var initial string
 err=tx.QueryRow(r.Context(),`SELECT name,direction,initial_amount::text,status FROM debts WHERE id=$1 AND user_id=$2 FOR UPDATE`,debtID,userID).Scan(&name,&direction,&initial,&status)
 if err!=nil{http.Error(w,`{"error":"Долг не найден"}`,404);return}
 var existing debtPaymentResponse
 err=tx.QueryRow(r.Context(),`SELECT id,amount::text,account_id,transaction_id,payment_date::text,comment,created_at FROM debt_payments WHERE user_id=$1 AND idempotency_key=$2`,userID,in.IdempotencyKey).Scan(&existing.ID,&existing.Amount,&existing.AccountID,&existing.TransactionID,&existing.Date,&existing.Comment,&existing.CreatedAt)
 if err==nil{if e:=tx.Commit(r.Context());e!=nil{http.Error(w,`{"error":"Не удалось завершить платёж"}`,500);return};_=json.NewEncoder(w).Encode(existing);return}
 if err!=pgx.ErrNoRows{http.Error(w,`{"error":"Не удалось проверить повтор платежа"}`,500);return}
 if status!="open"{http.Error(w,`{"error":"Долг уже закрыт"}`,409);return}
 var accountExists bool
 if err:=tx.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`,in.AccountID,userID).Scan(&accountExists);err!=nil||!accountExists{http.Error(w,`{"error":"Счёт не найден"}`,400);return}
 var enough bool
 err=tx.QueryRow(r.Context(),`SELECT d.initial_amount-COALESCE(SUM(p.amount),0)>=$2::numeric FROM debts d LEFT JOIN debt_payments p ON p.debt_id=d.id WHERE d.id=$1 GROUP BY d.id`,debtID,in.Amount).Scan(&enough)
 if err!=nil||!enough{http.Error(w,`{"error":"Платёж превышает остаток долга"}`,409);return}
 txType:="expense";if direction=="owed_to_me"{txType="income"}
 transactionKey:="debt-payment:"+strconv.FormatInt(userID,10)+":"+in.IdempotencyKey
 var transactionID int64
 err=tx.QueryRow(r.Context(),`INSERT INTO transactions(user_id,account_id,title,operation_type,amount,category,operation_date,comment,idempotency_key) VALUES($1,$2,$3,$4,$5::numeric,'Долги',$6,$7,$8) RETURNING id`,userID,in.AccountID,"Платёж по долгу: "+name,txType,in.Amount,date,in.Comment,transactionKey).Scan(&transactionID)
 if err!=nil{http.Error(w,`{"error":"Не удалось записать финансовую операцию"}`,500);return}
 var payment debtPaymentResponse
 err=tx.QueryRow(r.Context(),`INSERT INTO debt_payments(debt_id,user_id,amount,account_id,transaction_id,payment_date,comment,idempotency_key) VALUES($1,$2,$3::numeric,$4,$5,$6,$7,$8) RETURNING id,amount::text,account_id,transaction_id,payment_date::text,comment,created_at`,debtID,userID,in.Amount,in.AccountID,transactionID,date,in.Comment,in.IdempotencyKey).Scan(&payment.ID,&payment.Amount,&payment.AccountID,&payment.TransactionID,&payment.Date,&payment.Comment,&payment.CreatedAt)
 if err!=nil{http.Error(w,`{"error":"Не удалось сохранить историю платежа"}`,500);return}
 _,err=tx.Exec(r.Context(),`UPDATE debts SET status=CASE WHEN initial_amount-COALESCE((SELECT SUM(amount) FROM debt_payments WHERE debt_id=$1),0)<=0 THEN 'closed' ELSE status END,updated_at=NOW() WHERE id=$1`,debtID)
 if err!=nil{http.Error(w,`{"error":"Не удалось обновить остаток долга"}`,500);return}
 if err:=tx.Commit(r.Context());err!=nil{http.Error(w,`{"error":"Не удалось завершить платёж"}`,500);return}
 w.WriteHeader(201);_=json.NewEncoder(w).Encode(payment)
}
