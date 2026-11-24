package main

import (
 "crypto/sha256"
 "encoding/csv"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/jackc/pgx/v5/pgxpool"
)

type importCSVRow struct {
 Date string `json:"date"`
 Title string `json:"title"`
 Type string `json:"type"`
 Amount string `json:"amount"`
 Category string `json:"category"`
 AccountID int64 `json:"accountId"`
 TransferAccountID int64 `json:"transferAccountId"`
 Comment string `json:"comment"`
 Errors []string `json:"errors"`
}
func parseImportCSV(data []byte)([]importCSVRow,error){
 reader:=csv.NewReader(strings.NewReader(string(data)));reader.FieldsPerRecord=-1
 header,err:=reader.Read();if err!=nil{return nil,err}
 required:=[]string{"date","title","type","amount","category","accountId","transferAccountId","comment"}
 indexes:=map[string]int{}
 for i,h:=range header{indexes[strings.TrimSpace(h)]=i}
 for _,h:=range required{if _,ok:=indexes[h];!ok{return nil,fmt.Errorf("Отсутствует колонка %s",h)}}
 rows:=make([]importCSVRow,0)
 for line:=2;;line++{
  record,err:=reader.Read();if err==io.EOF{break};if err!=nil{return nil,fmt.Errorf("Строка %d: %v",line,err)}
  value:=func(key string)string{idx:=indexes[key];if idx>=len(record){return ""};return strings.TrimSpace(record[idx])}
  accountID,_:=strconv.ParseInt(value("accountId"),10,64);transferID,_:=strconv.ParseInt(value("transferAccountId"),10,64)
  rows=append(rows,importCSVRow{Date:value("date"),Title:value("title"),Type:value("type"),Amount:value("amount"),Category:value("category"),AccountID:accountID,TransferAccountID:transferID,Comment:value("comment"),Errors:[]string{}})
  if len(rows)>=5000{return nil,fmt.Errorf("Файл содержит больше 5000 строк")}
 }
 return rows,nil
}
func validateImportRow(row *importCSVRow) {
 if _,err:=time.Parse("2006-01-02",row.Date);err!=nil{row.Errors=append(row.Errors,"Некорректная дата")}
 if row.Title==""||len([]rune(row.Title))>160{row.Errors=append(row.Errors,"Некорректное название")}
 if row.Type!="income"&&row.Type!="expense"&&row.Type!="transfer"{row.Errors=append(row.Errors,"Неизвестный тип")}
 if !accountAmountPattern.MatchString(row.Amount)||strings.HasPrefix(row.Amount,"-")||strings.Trim(row.Amount,"0.")==""{row.Errors=append(row.Errors,"Некорректная сумма")}
 if row.Category==""||len([]rune(row.Category))>80{row.Errors=append(row.Errors,"Некорректная категория")}
 if row.AccountID<=0{row.Errors=append(row.Errors,"Не указан счёт")}
 if row.Type=="transfer"&&(row.TransferAccountID<=0||row.TransferAccountID==row.AccountID){row.Errors=append(row.Errors,"Некорректный счёт перевода")}
 if row.Type!="transfer"&&row.TransferAccountID!=0{row.Errors=append(row.Errors,"Второй счёт допустим только для перевода")}
 if len([]rune(row.Comment))>1000{row.Errors=append(row.Errors,"Слишком длинный комментарий")}
}
func csvHandler(w http.ResponseWriter,r *http.Request,database *pgxpool.Pool){
 userID,err:=authenticatedUserID(r,database);if err!=nil{http.Error(w,`{"error":"Пользователь не авторизован"}`,401);return}
 switch r.URL.Path{
 case "/api/export.csv":
  if r.Method!=http.MethodGet{w.Header().Set("Allow","GET");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
  fromValue,errFrom:=parseOptionalDate(r.URL.Query().Get("from"));toValue,errTo:=parseOptionalDate(r.URL.Query().Get("to"));if errFrom!=nil||errTo!=nil{http.Error(w,`{"error":"Некорректный период экспорта"}`,400);return}
  rows,err:=database.Query(r.Context(),`SELECT operation_date::text,title,operation_type,amount::text,category,account_id,COALESCE(transfer_account_id,0),comment FROM transactions WHERE user_id=$1 AND ($2::date IS NULL OR operation_date >= $2::date) AND ($3::date IS NULL OR operation_date <= $3::date) AND ($4::bigint=0 OR account_id=$4 OR transfer_account_id=$4) AND ($5='' OR category=$5) ORDER BY operation_date,id`,userID,fromValue,toValue,parseIntDefault(r.URL.Query().Get("accountId")),r.URL.Query().Get("category"))
  if err!=nil{http.Error(w,`Не удалось подготовить экспорт`,500);return};defer rows.Close()
  w.Header().Set("Content-Type","text/csv; charset=utf-8");w.Header().Set("Content-Disposition",`attachment; filename="fluxora-transactions.csv"`)
  _,_=w.Write([]byte{0xEF,0xBB,0xBF});writer:=csv.NewWriter(w);_=writer.Write([]string{"date","title","type","amount","category","accountId","transferAccountId","comment"})
  for rows.Next(){var record []string;var date,title,kind,amount,category,comment string;var account,transfer int64;if err:=rows.Scan(&date,&title,&kind,&amount,&category,&account,&transfer,&comment);err!=nil{return};record=[]string{date,title,kind,amount,category,strconv.FormatInt(account,10),strconv.FormatInt(transfer,10),comment};_=writer.Write(record)}
  writer.Flush()
 case "/api/import/preview","/api/import/confirm":
  if r.Method!=http.MethodPost{w.Header().Set("Allow","POST");http.Error(w,`{"error":"Метод не поддерживается"}`,405);return}
  r.Body=http.MaxBytesReader(w,r.Body,6<<20);if err:=r.ParseMultipartForm(6<<20);err!=nil{http.Error(w,`{"error":"Файл слишком большой или некорректный"}`,400);return}
  file,header,err:=r.FormFile("file");if err!=nil{http.Error(w,`{"error":"Выберите CSV-файл"}`,400);return};defer file.Close()
  if header.Size>5<<20{http.Error(w,`{"error":"Максимальный размер файла 5 МБ"}`,400);return}
  data,err:=io.ReadAll(io.LimitReader(file,(5<<20)+1));if err!=nil||len(data)>5<<20{http.Error(w,`{"error":"Не удалось прочитать CSV-файл"}`,400);return}
  items,err:=parseImportCSV(data);if err!=nil{http.Error(w,"{\\"error\\":\\"Ошибка CSV: "+err.Error()+"\\"}",400);return}
  valid:=0
  for i:=range items{validateImportRow(&items[i]);if len(items[i].Errors)==0{
   var source,destination bool
   _=database.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`,items[i].AccountID,userID).Scan(&source)
   if !source{items[i].Errors=append(items[i].Errors,"Счёт не найден")}
   if items[i].Type=="transfer"{_=database.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND user_id=$2)`,items[i].TransferAccountID,userID).Scan(&destination);if !destination{items[i].Errors=append(items[i].Errors,"Счёт получения не найден")}}
  };if len(items[i].Errors)==0{valid++}}
  if r.URL.Path=="/api/import/preview"{_=json.NewEncoder(w).Encode(map[string]any{"rows":items,"valid":valid,"invalid":len(items)-valid});return}
  if valid!=len(items)||len(items)==0{http.Error(w,`{"error":"Импорт отменён: исправьте ошибки в предварительном просмотре"}`,400);return}
  sum:=sha256.Sum256(data);fileKey:=hex.EncodeToString(sum[:])
  tx,err:=database.Begin(r.Context());if err!=nil{http.Error(w,`{"error":"Не удалось начать импорт"}`,500);return};defer tx.Rollback(r.Context())
  imported:=0
  for i,item:=range items{
   date,_:=time.Parse("2006-01-02",item.Date);key:="csv:"+fileKey[:40]+":"+strconv.Itoa(i)
   tag,err:=tx.Exec(r.Context(),`INSERT INTO transactions(user_id,account_id,transfer_account_id,title,operation_type,amount,category,operation_date,comment,idempotency_key) VALUES($1,$2,NULLIF($3,0),$4,$5,$6::numeric,$7,$8,$9,$10) ON CONFLICT(user_id,idempotency_key) DO NOTHING`,userID,item.AccountID,item.TransferAccountID,item.Title,item.Type,item.Amount,item.Category,date,item.Comment,key)
   if err!=nil{http.Error(w,`{"error":"Не удалось импортировать строки"}`,500);return};imported+=int(tag.RowsAffected())
  }
  if err:=tx.Commit(r.Context());err!=nil{http.Error(w,`{"error":"Не удалось завершить импорт"}`,500);return}
  _=json.NewEncoder(w).Encode(map[string]int{"imported":imported,"duplicates":len(items)-imported})
 default:http.NotFound(w,r)
 }
}
func parseIntDefault(v string)int64{n,_:=strconv.ParseInt(v,10,64);return n}

func parseOptionalDate(v string)(any,error){if strings.TrimSpace(v)==""{return nil,nil};d,err:=time.Parse("2006-01-02",v);if err!=nil{return nil,err};return d,nil}
