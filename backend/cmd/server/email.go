package main

import (
 "crypto/rand"
 "crypto/tls"
 "errors"
 "crypto/sha256"
 "encoding/base64"
 "encoding/hex"
 "encoding/json"
 "io"
 "net"
 "net/http"
 "net/mail"
 "net/smtp"
 "os"
 "strings"
 "time"
 "github.com/jackc/pgx/v5/pgxpool"
)

func sendEmailChangeMessage(to,token string)error{
 host:=strings.TrimSpace(os.Getenv("SMTP_HOST"));port:=strings.TrimSpace(os.Getenv("SMTP_PORT"));from:=strings.TrimSpace(os.Getenv("SMTP_FROM"))
 username:=os.Getenv("SMTP_USER");password:=os.Getenv("SMTP_PASSWORD")
 if host==""||from==""{return errors.New("SMTP is not configured")};if port==""{port="587"}
 subject:=base64.StdEncoding.EncodeToString([]byte("Подтверждение нового email Fluxora"))
 body:="From: "+from+"\r\nTo: "+to+"\r\nSubject: =?UTF-8?B?"+subject+"?=\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n"+
  "Для подтверждения нового адреса в Fluxora введите этот код в настройках аккаунта:\r\n\r\n"+token+"\r\n\r\nКод действует 30 минут. Если вы не запрашивали смену email, проигнорируйте письмо.\r\n"
 client,err:=smtp.Dial(net.JoinHostPort(host,port));if err!=nil{return err};defer client.Close()
 if ok,_:=client.Extension("STARTTLS");!ok{return errors.New("SMTP server does not support STARTTLS")}
 if err:=client.StartTLS(&tls.Config{ServerName:host,MinVersion:tls.VersionTLS12});err!=nil{return err}
 if username!=""{if err:=client.Auth(smtp.PlainAuth("",username,password,host));err!=nil{return err}}
 if err:=client.Mail(from);err!=nil{return err};if err:=client.Rcpt(to);err!=nil{return err}
 writer,err:=client.Data();if err!=nil{return err};if _,err:=io.WriteString(writer,body);err!=nil{_ = writer.Close();return err};if err:=writer.Close();err!=nil{return err}
 return client.Quit()
}
func emailChangeHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool) {
 w.Header().Set("Content-Type","application/json")
 w.Header().Set("Cache-Control","no-store")
 userID,err:=authenticatedUserID(r,database)
 if err!=nil { http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return }
 if r.URL.Path=="/api/account/email/change" {
  if r.Method!=http.MethodPost { w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return }
  r.Body=http.MaxBytesReader(w,r.Body,8<<10)
  var in struct { Email string `json:"email"` }
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil { http.Error(w,`{"error":"Некорректные данные"}`,400);return }
  in.Email=strings.TrimSpace(in.Email)
  parsed,err:=mail.ParseAddress(in.Email)
  if err!=nil||parsed.Address!=in.Email||len(in.Email)>255 { http.Error(w,`{"error":"Укажите корректный email"}`,400);return }
  var current string
  if err:=database.QueryRow(r.Context(),`SELECT email FROM users WHERE id=$1`,userID).Scan(&current);err!=nil { http.Error(w,`{"error":"Пользователь не найден"}`,404);return }
  if strings.EqualFold(current,in.Email) { http.Error(w,`{"error":"Это уже текущий email"}`,400);return }
  var recent bool
  if err:=database.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM email_change_requests WHERE user_id=$1 AND created_at>NOW()-INTERVAL '1 minute')`,userID).Scan(&recent);err!=nil{http.Error(w,`{"error":"Не удалось проверить запросы email"}`,500);return};if recent{http.Error(w,`{"error":"Подождите минуту перед повторной отправкой"}`,429);return}
  var taken bool
  if err:=database.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email)=LOWER($1) AND id<>$2)`,in.Email,userID).Scan(&taken);err!=nil||taken { http.Error(w,`{"error":"Этот email уже используется"}`,409);return }
  tokenBytes:=make([]byte,32)
  if _,err:=rand.Read(tokenBytes);err!=nil { http.Error(w,`{"error":"Не удалось создать код"}`,500);return }
  token:=hex.EncodeToString(tokenBytes)
  sum:=sha256.Sum256([]byte(token));tokenHash:=hex.EncodeToString(sum[:])
  if _,err:=database.Exec(r.Context(),`DELETE FROM email_change_requests WHERE user_id=$1`,userID);err!=nil { http.Error(w,`{"error":"Не удалось подготовить подтверждение"}`,500);return }
  _,err=database.Exec(r.Context(),`INSERT INTO email_change_requests(user_id,new_email,token_hash,expires_at) VALUES($1,$2,$3,NOW()+INTERVAL '30 minutes')`,userID,in.Email,tokenHash)
  if err!=nil { http.Error(w,`{"error":"Не удалось создать запрос смены email"}`,500);return }
  if err:=sendEmailChangeMessage(in.Email,token);err!=nil {
   _,_=database.Exec(r.Context(),`DELETE FROM email_change_requests WHERE user_id=$1`,userID)
   http.Error(w,`{"error":"Не удалось отправить письмо. Проверьте SMTP-настройки сервера."}`,503);return
  }
  _=json.NewEncoder(w).Encode(map[string]string{"message":"Письмо с кодом подтверждения отправлено на новый адрес."})
  return
 }
 if r.URL.Path=="/api/account/email/confirm" {
  if r.Method!=http.MethodPost { w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return }
  r.Body=http.MaxBytesReader(w,r.Body,8<<10)
  var in struct { Token string `json:"token"` }
  if err:=json.NewDecoder(r.Body).Decode(&in);err!=nil||strings.TrimSpace(in.Token)=="" { http.Error(w,`{"error":"Введите код из письма"}`,400);return }
  sum:=sha256.Sum256([]byte(strings.TrimSpace(in.Token)));tokenHash:=hex.EncodeToString(sum[:])
  tx,err:=database.Begin(r.Context())
  if err!=nil { http.Error(w,`{"error":"Не удалось подтвердить email"}`,500);return }
  defer tx.Rollback(r.Context())
  var newEmail string
  err=tx.QueryRow(r.Context(),`SELECT new_email FROM email_change_requests WHERE user_id=$1 AND token_hash=$2 AND expires_at>NOW() FOR UPDATE`,userID,tokenHash).Scan(&newEmail)
  if err!=nil { http.Error(w,`{"error":"Код неверный или срок его действия истёк"}`,400);return }
  if _,err:=tx.Exec(r.Context(),`UPDATE users SET email=$2 WHERE id=$1`,userID,newEmail);err!=nil { http.Error(w,`{"error":"Этот email уже используется"}`,409);return }
  _,_=tx.Exec(r.Context(),`DELETE FROM email_change_requests WHERE user_id=$1`,userID)
  _,_=tx.Exec(r.Context(),`INSERT INTO notifications(user_id,notification_type,title,message,related_type) VALUES($1,'account_email','Email изменён','Адрес электронной почты успешно изменён.','account')`,userID)
  if err:=tx.Commit(r.Context());err!=nil { http.Error(w,`{"error":"Не удалось завершить смену email"}`,500);return }
  _=json.NewEncoder(w).Encode(map[string]string{"message":"Email успешно изменён","email":newEmail})
  return
 }
 http.NotFound(w,r)
}
