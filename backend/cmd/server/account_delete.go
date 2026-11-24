package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "log"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
 "golang.org/x/crypto/bcrypt"
)

var errFamilyOwnerAlone=errors.New("владелец является единственным участником семейного бюджета")

type deletionLedgerEntry struct {
 UserID int64 `json:"user_id"`
 UserCreatedAt time.Time `json:"user_created_at"`
 Status string `json:"status"`
 DeletedAt time.Time `json:"deleted_at"`
}

func deleteUserData(ctx context.Context,database *pgxpool.Pool,userID int64,allowOwnerAlone bool)([]string,error){
 tx,err:=database.Begin(ctx);if err!=nil{return nil,err};defer tx.Rollback(ctx)
 var email string
 if err:=tx.QueryRow(ctx,`SELECT email FROM users WHERE id=$1 FOR UPDATE`,userID).Scan(&email);err!=nil{return nil,err}
 _,err=tx.Exec(ctx,`DELETE FROM family_invitations WHERE LOWER(email)=LOWER($1)`,email);if err!=nil{return nil,err}
 keys:=make([]string,0)
 var avatar pgtype.Text
 if err:=tx.QueryRow(ctx,`SELECT avatar_key FROM users WHERE id=$1`,userID).Scan(&avatar);err!=nil{return nil,err}
 if avatar.Valid&&avatar.String!=""{keys=append(keys,avatar.String)}
 goalRows,err:=tx.Query(ctx,`SELECT image_key FROM goals WHERE user_id=$1 AND image_key IS NOT NULL`,userID);if err!=nil{return nil,err};for goalRows.Next(){var key string;if err:=goalRows.Scan(&key);err!=nil{goalRows.Close();return nil,err};if key!=""{keys=append(keys,key)}};if err:=goalRows.Err();err!=nil{goalRows.Close();return nil,err};goalRows.Close()
 var familyID pgtype.Int8
 err=tx.QueryRow(ctx,`SELECT family_id FROM family_members WHERE user_id=$1 AND role='owner' LIMIT 1 FOR UPDATE`,userID).Scan(&familyID)
 if err!=nil&&err!=pgx.ErrNoRows{return nil,err}
 if familyID.Valid{
  var successor int64
  err=tx.QueryRow(ctx,`SELECT user_id FROM family_members WHERE family_id=$1 AND user_id<>$2 ORDER BY CASE role WHEN 'admin' THEN 0 ELSE 1 END,joined_at,id LIMIT 1 FOR UPDATE`,familyID.Int64,userID).Scan(&successor)
  if err==pgx.ErrNoRows{
   if !allowOwnerAlone{return nil,errFamilyOwnerAlone}
   familyGoalRows,queryErr:=tx.Query(ctx,`SELECT image_key FROM family_goals WHERE family_id=$1 AND image_key IS NOT NULL`,familyID.Int64)
   if queryErr!=nil{return nil,queryErr}
   for familyGoalRows.Next(){var key string;if scanErr:=familyGoalRows.Scan(&key);scanErr!=nil{familyGoalRows.Close();return nil,scanErr};if key!=""{keys=append(keys,key)}}
   if rowsErr:=familyGoalRows.Err();rowsErr!=nil{familyGoalRows.Close();return nil,rowsErr}
   familyGoalRows.Close()
   if _,err:=tx.Exec(ctx,`DELETE FROM families WHERE id=$1`,familyID.Int64);err!=nil{return nil,err}
  }else if err!=nil{return nil,err
  }else{
   if _,err:=tx.Exec(ctx,`UPDATE family_members SET role='owner' WHERE family_id=$1 AND user_id=$2`,familyID.Int64,successor);err!=nil{return nil,err}
   if _,err:=tx.Exec(ctx,`UPDATE families SET owner_id=$2,updated_at=NOW() WHERE id=$1`,familyID.Int64,successor);err!=nil{return nil,err}
  }
 }
 tag,err:=tx.Exec(ctx,`DELETE FROM users WHERE id=$1`,userID);if err!=nil{return nil,err};if tag.RowsAffected()==0{return nil,pgx.ErrNoRows}
 if err:=tx.Commit(ctx);err!=nil{return nil,err}
 return keys,nil
}

func deleteAccountHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store")
 if r.Method!=http.MethodPost{w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 r.Body=http.MaxBytesReader(w,r.Body,12<<10);var input struct{Password string `json:"password"`;Confirmation string `json:"confirmation"`}
 if err:=json.NewDecoder(r.Body).Decode(&input);err!=nil{http.Error(w,`{"error":"Некорректные данные"}`,400);return}
 if input.Confirmation!="DELETE"||strings.TrimSpace(input.Password)==""{http.Error(w,`{"error":"Введите пароль и подтверждение DELETE"}`,400);return}
 var passwordHash string;var userCreatedAt time.Time
 if err:=database.QueryRow(r.Context(),`SELECT password_hash,created_at FROM users WHERE id=$1`,userID).Scan(&passwordHash,&userCreatedAt);err!=nil{http.Error(w,`{"error":"Пользователь не найден"}`,404);return}
 if bcrypt.CompareHashAndPassword([]byte(passwordHash),[]byte(input.Password))!=nil{http.Error(w,`{"error":"Неверный пароль"}`,401);return}
 store,err:=newS3Store();if err!=nil{http.Error(w,`{"error":"Удаление аккаунта временно недоступно: не настроено защищённое хранилище журнала удаления"}`,503);return}
 ledger:=deletionLedgerEntry{UserID:userID,UserCreatedAt:userCreatedAt,DeletedAt:time.Now().UTC(),Status:"pending"}
 body,err:=json.Marshal(ledger);if err!=nil{http.Error(w,`{"error":"Не удалось подготовить журнал удаления"}`,500);return}
 ledgerName:=strconv.FormatInt(userID,10)+"-"+ledger.UserCreatedAt.UTC().Format("20060102T150405.000000000Z")+".json"
 pendingKey:="deletion-ledger/pending/"+ledgerName
 committedKey:="deletion-ledger/committed/"+ledgerName
 if err:=store.put(r.Context(),pendingKey,"application/json",body);err!=nil{http.Error(w,`{"error":"Не удалось сохранить защищённый журнал удаления"}`,503);return}
 keys,err:=deleteUserData(r.Context(),database,userID,false)
 if err!=nil{_ = store.delete(r.Context(),pendingKey);if errors.Is(err,errFamilyOwnerAlone){http.Error(w,`{"error":"Вы единственный участник семейного бюджета. Перед удалением аккаунта добавьте другого участника и передайте ему владение."}`,409);return};http.Error(w,`{"error":"Не удалось удалить аккаунт"}`,500);return}
 ledger.Status="committed"
 committedBody,marshalErr:=json.Marshal(ledger)
 ledgerCommitted:=false
 if marshalErr==nil {
  for attempt:=0;attempt<3;attempt++ {
   if putErr:=store.put(r.Context(),committedKey,"application/json",committedBody);putErr==nil{ledgerCommitted=true;break}else{log.Printf("account deletion: committed ledger write attempt %d failed: %v",attempt+1,putErr)}
  }
 }
 if ledgerCommitted{_ = store.delete(r.Context(),pendingKey)}
 cleanupFailed:=false
 for _,key:=range keys{if err:=store.delete(r.Context(),key);err!=nil{cleanupFailed=true;log.Printf("account deletion: could not remove S3 object %s: %v",key,err)}}
 message:="Аккаунт удалён. Резервные копии могут содержать данные до 7 дней и автоматически истекут."
 if cleanupFailed{message+=" Некоторые файлы не удалось удалить из S3."}
 if !ledgerCommitted{message+=" ВНИМАНИЕ: не удалось подтвердить запись журнала удаления. Не восстанавливайте резервную копию до устранения этой проблемы."}
 _=json.NewEncoder(w).Encode(map[string]any{"success":true,"message":message,"cleanupWarning":cleanupFailed,"ledgerWarning":!ledgerCommitted})
}

func applyDeletionLedger(ctx context.Context,database *pgxpool.Pool)error{
 store,err:=newS3Store();if err!=nil{return err}
 keys,err:=store.listKeys(ctx,"deletion-ledger/committed/");if err!=nil{return err}
 for _,key:=range keys{
  raw,err:=store.get(ctx,key);if err!=nil{return fmt.Errorf("read committed deletion ledger %s: %w",key,err)}
  var entry deletionLedgerEntry;if err:=json.Unmarshal(raw,&entry);err!=nil{return fmt.Errorf("decode committed deletion ledger %s: %w",key,err)};if entry.Status!="committed"{return fmt.Errorf("deletion ledger %s is not committed",key)}
  var createdAt time.Time
  err=database.QueryRow(ctx,`SELECT created_at FROM users WHERE id=$1`,entry.UserID).Scan(&createdAt)
  if err==pgx.ErrNoRows{continue};if err!=nil{return err}
  if !createdAt.Equal(entry.UserCreatedAt){log.Printf("deletion ledger account incarnation mismatch for user id %d",entry.UserID);continue}
  if _,err:=deleteUserData(ctx,database,entry.UserID,true);err!=nil{return err}
  log.Printf("deletion ledger reapplied for user id %d",entry.UserID)
 }
 return nil
}
