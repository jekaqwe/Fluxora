package main

import (
 "strings"
 "testing"
)

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
 data:=[]byte("Дата;Описание;Тип;Сумма;Категория;Счёт\n2026-11-24;Кофе;expense;250,00;Еда;7\n")
 rows,err:=parseImportCSV(data)
 if err!=nil { t.Fatalf("semicolon-delimited CSV failed: %v",err) }
 if len(rows)!=1||rows[0].Title!="Кофе"||rows[0].Amount!="250.00"||rows[0].AccountID!=7 { t.Fatalf("unexpected semicolon-delimited row: %#v",rows) }
}

func TestParseImportCSVAllowsExactly5000Rows(t *testing.T) {
 var b strings.Builder
 b.WriteString("date,title,type,amount,category,accountId\n")
 row := "2026-11-24,Coffee,expense,2.50,Food,7\n"
 b.WriteString(strings.Repeat(row, 5000))
 rows, err := parseImportCSV([]byte(b.String()))
 if err != nil {
  t.Fatalf("expected 5000 rows to be accepted, got error: %v", err)
 }
 if len(rows) != 5000 {
  t.Fatalf("expected 5000 rows, got %d", len(rows))
 }
}

func TestParseImportCSVRejectsMoreThan5000Rows(t *testing.T) {
 var b strings.Builder
 b.WriteString("date,title,type,amount,category,accountId\n")
 row := "2026-11-24,Coffee,expense,2.50,Food,7\n"
 b.WriteString(strings.Repeat(row, 5001))
 if _, err := parseImportCSV([]byte(b.String())); err == nil {
  t.Fatal("expected files with more than 5000 rows to be rejected")
 }
}

func TestCSVSuggestedMappingDoesNotTreatAccountNameAsID(t *testing.T) {
 mapping := csvSuggestedMapping([]string{"date", "title", "type", "amount", "category", "accountName"})
 if _, ok := mapping["accountId"]; ok {
  t.Fatalf("accountName must not be auto-mapped to numeric accountId: %#v", mapping)
 }
}


func TestTinkoffCSVProfileAndAccountCurrencyAmounts(t *testing.T) {
	data := []byte("\uFEFFИмя счёта;Дата операции;Сумма операции;Валюта операции;Сумма в валюте счёта;Валюта счёта;Статус;Категория по-умолчанию;Описание;Сообщение\n" +
		"Black;17.03.2026 14:38:23;-1,00;USD;-86,15;RUB;Ок;Переводы;Между своими счетами;\n" +
		"Black;18.03.2026 10:00:00;12,50;RUB;12,50;RUB;Ок;Пополнения;Пополнение;\n" +
		"Black;19.03.2026 10:00:00;-5,00;RUB;-5,00;RUB;Ошибка;Покупки;Неуспешная операция;\n")
	headers, err := parseCSVHeaders(data)
	if err != nil {
		t.Fatalf("parseCSVHeaders failed: %v", err)
	}
	if !isTinkoffCSV(headers) || csvFormat(headers) != "tinkoff" {
		t.Fatalf("Tinkoff format was not detected: %#v", headers)
	}
	mapping := csvSuggestedMapping(headers)
	mapping["accountId"] = csvTinkoffAccountPrefix + "42"
	rows, skipped, err := parseImportCSVWithMappingStats(data, mapping)
	if err != nil {
		t.Fatalf("Tinkoff CSV parse failed: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("expected one failed-status row to be skipped, got %d", skipped)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two successful rows, got %d", len(rows))
	}
	if rows[0].Date != "17.03.2026 14:38:23" || rows[0].Amount != "86.15" || rows[0].Type != "expense" || rows[0].AccountID != 42 {
		t.Fatalf("unexpected converted USD operation: %#v", rows[0])
	}
	if rows[1].Date != "18.03.2026 10:00:00" || rows[1].Amount != "12.50" || rows[1].Type != "income" {
		t.Fatalf("unexpected income operation: %#v", rows[1])
	}
}

func TestParseImportDateSupportsTinkoffDateTime(t *testing.T) {
	parsed, err := parseImportDate("24.11.2026 09:49:42")
	if err != nil {
		t.Fatalf("Tinkoff date parsing failed: %v", err)
	}
	if got := parsed.Format("2006-01-02"); got != "2026-11-24" {
		t.Fatalf("unexpected normalized date: %s", got)
	}
}

