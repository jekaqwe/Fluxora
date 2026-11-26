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

func runDailyBackup() {
 databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
 if databaseURL == "" {
  log.Println("backup: DATABASE_URL is not configured")
  return
 }
 store, err := newS3Store()
 if err != nil {
  log.Printf("backup: S3 is not configured: %v", err)
  return
 }
 ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
 defer cancel()
 store.client.Timeout = 2 * time.Hour

 // Keep pg_dump on disk rather than buffering the entire database in RAM.
 dumpFile, err := os.CreateTemp("", "fluxora-postgres-*.dump")
 if err != nil {
  log.Printf("backup: cannot create temporary archive: %v", err)
  return
 }
 dumpPath := dumpFile.Name()
 defer func() {
  _ = dumpFile.Close()
  _ = os.Remove(dumpPath)
 }()

 command := exec.CommandContext(ctx, "pg_dump", "--format=custom", "--no-owner", "--no-acl", "--dbname", databaseURL)
 command.Stdout = dumpFile
 if err := command.Run(); err != nil {
  log.Printf("backup: pg_dump failed: %v", err)
  return
 }
 info, err := dumpFile.Stat()
 if err != nil {
  log.Printf("backup: cannot inspect archive: %v", err)
  return
 }
 if info.Size() == 0 {
  log.Println("backup: pg_dump returned an empty archive")
  return
 }
 if _, err := dumpFile.Seek(0, 0); err != nil {
  log.Printf("backup: cannot rewind archive: %v", err)
  return
 }

 key := path.Join("backups", "postgres", time.Now().UTC().Format("2006-01-02")+".dump")
 if err := store.putReader(ctx, key, "application/octet-stream", dumpFile, info.Size()); err != nil {
  log.Printf("backup: upload failed: %v", err)
  return
 }
 log.Printf("backup: PostgreSQL archive uploaded: %s (%d bytes)", key, info.Size())

 keys, err := store.listKeys(ctx, "backups/postgres/")
 if err != nil {
  log.Printf("backup: retention listing failed: %v", err)
  return
 }
 cutoff := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -6)
 for _, oldKey := range keys {
  base := path.Base(oldKey)
  date, err := time.Parse("2006-01-02.dump", base)
  if err != nil {
   continue
  }
  if date.Before(cutoff) {
   if err := store.delete(ctx, oldKey); err != nil {
    log.Printf("backup: could not delete expired archive %s: %v", oldKey, err)
   } else {
    log.Printf("backup: expired archive removed: %s", oldKey)
   }
  }
 }
}

func dailyBackupExists() bool {
 store, err := newS3Store()
 if err != nil {
  log.Printf("backup: cannot check today's archive: %v", err)
  return false
 }
 ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
 defer cancel()
 keys, err := store.listKeys(ctx, "backups/postgres/")
 if err != nil {
  log.Printf("backup: today's archive check failed: %v", err)
  return false
 }
 expected := time.Now().UTC().Format("2006-01-02") + ".dump"
 for _, key := range keys {
  if path.Base(key) == expected { return true }
 }
 return false
}

func startBackupScheduler() {
 go func() {
  // Check immediately on startup, then retry every 15 minutes. This covers
  // transient S3/network failures instead of waiting until the next day.
  const retryInterval = 15 * time.Minute
  for {
   if !dailyBackupExists() {
    runDailyBackup()
   }

   timer := time.NewTimer(retryInterval)
   <-timer.C
  }
 }()
}
