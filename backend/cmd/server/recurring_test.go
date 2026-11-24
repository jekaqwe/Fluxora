package main

import (
 "testing"
 "time"
)

func TestNextRecurringDate(t *testing.T) {
 tests:=[]struct{name string;date time.Time;frequency string;want string}{
  {"daily",time.Date(2026,11,24,0,0,0,0,time.UTC),"daily","2026-11-25"},
  {"weekly",time.Date(2026,11,24,0,0,0,0,time.UTC),"weekly","2026-12-01"},
  {"monthly month end",time.Date(2026,1,31,0,0,0,0,time.UTC),"monthly","2026-02-28"},
  {"monthly leap",time.Date(2028,1,31,0,0,0,0,time.UTC),"monthly","2028-02-29"},
  {"yearly leap",time.Date(2024,2,29,0,0,0,0,time.UTC),"yearly","2025-02-28"},
 }
 for _,tt:=range tests{t.Run(tt.name,func(t *testing.T){got:=nextRecurringDate(tt.date,tt.frequency).Format("2006-01-02");if got!=tt.want{t.Fatalf("got %s, want %s",got,tt.want)}})}
}
