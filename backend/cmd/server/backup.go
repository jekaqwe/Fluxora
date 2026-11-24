package main

import (
 "context"
 "log"
 "os"
 "os/exec"
 "path"
 "strings"
 "time"
)

func runDailyBackup(){
 databaseURL:=strings.TrimSpace(os.Getenv("DATABASE_URL"));if databaseURL==""{log.Println("backup: DATABASE_URL is not configured");return}
 store,err:=newS3Store();if err!=nil{log.Printf("backup: S3 is not configured: %v",err);return}
 ctx,cancel:=context.WithTimeout(context.Background(),2*time.Hour);defer cancel()
 store.client.Timeout=2*time.Hour
 command:=exec.CommandContext(ctx,"pg_dump","--format=custom","--no-owner","--no-acl","--dbname",databaseURL)
 dump,err:=command.Output();if err!=nil{log.Printf("backup: pg_dump failed: %v",err);return}
 if len(dump)==0{log.Println("backup: pg_dump returned an empty archive");return}
 key:=path.Join("backups","postgres",time.Now().UTC().Format("2006-01-02")+".dump")
 if err:=store.put(ctx,key,"application/octet-stream",dump);err!=nil{log.Printf("backup: upload failed: %v",err);return}
 log.Printf("backup: PostgreSQL archive uploaded: %s (%d bytes)",key,len(dump))
 keys,err:=store.listKeys(ctx,"backups/postgres/")
 if err!=nil{log.Printf("backup: retention listing failed: %v",err);return}
 cutoff:=time.Now().UTC().Truncate(24*time.Hour).AddDate(0,0,-6)
 for _,oldKey:=range keys{
  base:=path.Base(oldKey);date,err:=time.Parse("2006-01-02.dump",base);if err!=nil{continue}
  if date.Before(cutoff){if err:=store.delete(ctx,oldKey);err!=nil{log.Printf("backup: could not delete expired archive %s: %v",oldKey,err)}else{log.Printf("backup: expired archive removed: %s",oldKey)}}
 }
}

func startBackupScheduler(){
 go func(){
  for{
   now:=time.Now().UTC()
   next:=time.Date(now.Year(),now.Month(),now.Day(),3,0,0,0,time.UTC)
   if !next.After(now){next=next.AddDate(0,0,1)}
   timer:=time.NewTimer(time.Until(next))
   <-timer.C
   runDailyBackup()
  }
 }()
}
