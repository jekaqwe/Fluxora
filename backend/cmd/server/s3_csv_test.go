package main

import (
 "net/url"
 "testing"
)

func TestEscapeS3KeyPreservesObjectPathSegments(t *testing.T){
 got:=escapeS3Key("avatars/42/photo image.png")
 want:="avatars/42/photo%20image.png"
 if got!=want{t.Fatalf("escapeS3Key() = %q, want %q",got,want)}
}

func TestObjectURLUsesPathStyleAndEscapedKey(t *testing.T){
 endpoint,err:=url.Parse("https://s3.regru.cloud")
 if err!=nil{t.Fatal(err)}
 store:=&s3Store{endpoint:endpoint,bucket:"fluxora.su"}
 got:=store.objectURL("goals/17/cover image.png")
 if got.EscapedPath()!="/fluxora.su/goals/17/cover%20image.png"{t.Fatalf("unexpected escaped path: %s",got.EscapedPath())}
 if got.Host!="s3.regru.cloud"{t.Fatalf("unexpected host: %s",got.Host)}
}

func TestSafeCSVCellPreventsFormulaInjection(t *testing.T){
 for _,value:=range []string{"=1+1","+SUM(A1:A2)","-2+3","@HYPERLINK(1)"}{
  got:=safeCSVCell(value)
  if got==value||got[0]!='\''{t.Fatalf("safeCSVCell(%q) = %q, expected apostrophe prefix",value,got)}
 }
 if got:=safeCSVCell("ordinary text");got!="ordinary text"{t.Fatalf("safeCSVCell changed ordinary text: %q",got)}
}

func TestNormalizeCSVAmount(t *testing.T){
 tests:=map[string]string{"001.2000":"1.2","12.00":"12","0.50":"0.5","100":"100"}
 for input,want:=range tests{if got:=normalizeCSVAmount(input);got!=want{t.Errorf("normalizeCSVAmount(%q) = %q, want %q",input,got,want)}}
}
