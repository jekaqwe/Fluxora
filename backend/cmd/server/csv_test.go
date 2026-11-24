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

func TestNormalizeCSVAmount(t *testing.T) {
 cases:=map[string]string{"100.00":"100","12.30":"12.3","0.50":"0.5","100":"100","0.00":"0"}
 for input,want:=range cases { if got:=normalizeCSVAmount(input);got!=want { t.Errorf("normalizeCSVAmount(%q)=%q, want %q",input,got,want) } }
}
