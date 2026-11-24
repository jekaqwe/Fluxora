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
  {"monthly anchor restored after February",time.Date(2026,2,28,0,0,0,0,time.UTC),"monthly","2026-03-31"},
  {"yearly leap anchor restored",time.Date(2025,2,28,0,0,0,0,time.UTC),"yearly","2028-02-29"},
  {"yearly leap",time.Date(2024,2,29,0,0,0,0,time.UTC),"yearly","2025-02-28"},
 }
 for _,tt:=range tests{t.Run(tt.name,func(t *testing.T){got:=nextRecurringDate(tt.date,tt.frequency,func()int{if tt.name=="monthly anchor restored after February"{return 31};if tt.name=="yearly leap anchor restored"{return 29};return tt.date.Day()}(),func()int{if tt.name=="yearly leap anchor restored"{return 2};return int(tt.date.Month())}()).Format("2006-01-02");if got!=tt.want{t.Fatalf("got %s, want %s",got,tt.want)}})}
}
