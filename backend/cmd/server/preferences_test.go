package main

import "testing"

func TestDefaultUserPreferences(t *testing.T) {
 prefs:=defaultUserPreferences()
 if prefs.Theme!="light"||!prefs.Notifications["operations"]||!prefs.Notifications["goals"]||!prefs.Notifications["family"] {
  t.Fatalf("unexpected notification defaults: %#v",prefs)
 }
 if !prefs.Privacy["profile"]||!prefs.Privacy["achievements"]||!prefs.Privacy["activity"] {
  t.Fatalf("unexpected privacy defaults: %#v",prefs)
 }
}

func TestMergePreferenceFlags(t *testing.T) {
 current:=map[string]bool{"operations":true,"goals":true,"family":true}
 merged,ok:=mergePreferenceFlags(current,map[string]bool{"goals":false},map[string]bool{"operations":true,"goals":true,"family":true})
 if !ok||merged["goals"]||!merged["operations"]||!merged["family"] {
  t.Fatalf("unexpected merged flags: %#v, ok=%v",merged,ok)
 }
 if _,ok:=mergePreferenceFlags(current,map[string]bool{"unknown":false},map[string]bool{"operations":true});ok {
  t.Fatal("expected unknown preference key to be rejected")
 }
}
