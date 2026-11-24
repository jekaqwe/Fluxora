package main

import "testing"

func TestParseImportCSV(t *testing.T) {
 data:=[]byte("date,title,type,amount,category,accountId,transferAccountId,comment\n2026-11-24,Salary,income,1200.50,Work,7,0,Monthly salary\n")
 rows,err:=parseImportCSV(data)
 if err!=nil { t.Fatalf("parseImportCSV returned error: %v",err) }
 if len(rows)!=1 { t.Fatalf("expected one row, got %d",len(rows)) }
 if rows[0].Title!="Salary"||rows[0].AccountID!=7||rows[0].Amount!="1200.50" { t.Fatalf("unexpected parsed row: %#v",rows[0]) }
}

func TestParseImportCSVRequiresAllColumns(t *testing.T) {
 _,err:=parseImportCSV([]byte("date,title,type,amount\n"))
 if err==nil { t.Fatal("expected missing-column error") }
}

func TestValidateImportRow(t *testing.T) {
 valid:=importCSVRow{Date:"2026-11-24",Title:"Coffee",Type:"expense",Amount:"250.00",Category:"Food",AccountID:3,Errors:[]string{}}
 validateImportRow(&valid)
 if len(valid.Errors)!=0 { t.Fatalf("valid row produced errors: %v",valid.Errors) }

 invalid:=importCSVRow{Date:"24.11.2026",Title:"",Type:"transfer",Amount:"-1",Category:"",AccountID:0,TransferAccountID:0,Errors:[]string{}}
 validateImportRow(&invalid)
 if len(invalid.Errors)<5 { t.Fatalf("expected validation errors, got %v",invalid.Errors) }
}

func TestSafeCSVCell(t *testing.T) {
 cases:=map[string]string{
  "=1+1":"'=1+1",
  "  @SUM(A1:A2)":"'  @SUM(A1:A2)",
  "+cmd":"'+cmd",
  "-cmd":"'-cmd",
  "ordinary text":"ordinary text",
  "":"",
 }
 for input,want:=range cases { if got:=safeCSVCell(input);got!=want { t.Errorf("safeCSVCell(%q)=%q, want %q",input,got,want) } }
}

func TestParseImportCSVWithUTF8BOM(t *testing.T) {
 data := []byte("\uFEFFdate,title,type,amount,category,accountId,transferAccountId,comment\n2026-11-24,Salary,income,1200.50,Work,7,0,Monthly salary\n")
 rows, err := parseImportCSV(data)
 if err != nil { t.Fatalf("parseImportCSV rejected exported CSV with BOM: %v", err) }
 if len(rows) != 1 || rows[0].Title != "Salary" {
  t.Fatalf("unexpected parsed rows: %#v", rows)
 }
}

func TestParseImportCSVWithCustomColumnMapping(t *testing.T) {
 data:=[]byte("Дата операции,Описание,Вид,Сумма,Категория,Счёт,Заметка\n2026-11-24,Кофе,expense,250.00,Еда,7,Утренний кофе\n")
 mapping:=map[string]string{"date":"Дата операции","title":"Описание","type":"Вид","amount":"Сумма","category":"Категория","accountId":"Счёт","comment":"Заметка"}
 rows,err:=parseImportCSVWithMapping(data,mapping)
 if err!=nil { t.Fatalf("custom mapping failed: %v",err) }
 if len(rows)!=1||rows[0].Title!="Кофе"||rows[0].AccountID!=7||rows[0].Comment!="Утренний кофе" { t.Fatalf("unexpected mapped row: %#v",rows) }
}

func TestCSVSuggestedMappingRussianHeaders(t *testing.T) {
 headers:=[]string{"Дата операции","Описание","Тип","Сумма","Категория","Счёт","Комментарий"}
 mapping:=csvSuggestedMapping(headers)
 if mapping["date"]!="Дата операции"||mapping["title"]!="Описание"||mapping["accountId"]!="Счёт"||mapping["comment"]!="Комментарий" { t.Fatalf("unexpected suggested mapping: %#v",mapping) }
}

func TestParseImportCSVWithSemicolonDelimiter(t *testing.T) {
 data:=[]byte("Дата;Описание;Тип;Сумма;Категория;Счёт\n2026-11-24;Кофе;expense;250.00;Еда;7\n")
 rows,err:=parseImportCSV(data)
 if err!=nil { t.Fatalf("semicolon-delimited CSV failed: %v",err) }
 if len(rows)!=1||rows[0].Title!="Кофе"||rows[0].Amount!="250.00"||rows[0].AccountID!=7 { t.Fatalf("unexpected semicolon-delimited row: %#v",rows) }
}
