package main

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestEscapeS3KeyPreservesObjectPathSegments(t *testing.T) {
	got := escapeS3Key("avatars/42/photo image.png")
	want := "avatars/42/photo%20image.png"
	if got != want {
		t.Fatalf("escapeS3Key() = %q, want %q", got, want)
	}
}

func TestObjectURLUsesPathStyleAndEscapedKey(t *testing.T) {
	endpoint, err := url.Parse("https://s3.regru.cloud")
	if err != nil {
		t.Fatal(err)
	}
	store := &s3Store{endpoint: endpoint, bucket: "fluxora.su"}
	got := store.objectURL("goals/17/cover image.png")
	if got.EscapedPath() != "/fluxora.su/goals/17/cover%20image.png" {
		t.Fatalf("unexpected escaped path: %s", got.EscapedPath())
	}
	if got.Host != "s3.regru.cloud" {
		t.Fatalf("unexpected host: %s", got.Host)
	}
}

func TestSafeCSVCellPreventsFormulaInjection(t *testing.T) {
	for _, value := range []string{"=1+1", "+SUM(A1:A2)", "-2+3", "@HYPERLINK(1)"} {
		got := safeCSVCell(value)
		if got == value || got[0] != '\'' {
			t.Fatalf("safeCSVCell(%q) = %q, expected apostrophe prefix", value, got)
		}
	}
	if got := safeCSVCell("ordinary text"); got != "ordinary text" {
		t.Fatalf("safeCSVCell changed ordinary text: %q", got)
	}
}

func TestS3PutReaderUploadsSeekableStream(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.ContentLength != int64(len("streamed archive payload")) {
			t.Errorf("content length = %d", r.ContentLength)
		}
		if r.Header.Get("X-Amz-Content-Sha256") == "" {
			t.Error("missing payload hash")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}
		received = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	store := &s3Store{
		endpoint: endpoint,
		bucket:   "test-bucket",
		region:   "test-region-1",
		access:   "test-access",
		secret:   "test-secret",
		client:   server.Client(),
	}
	file, err := os.CreateTemp(t.TempDir(), "backup-*.dump")
	if err != nil {
		t.Fatal(err)
	}
	payload := "streamed archive payload"
	if _, err := io.Copy(file, strings.NewReader(payload)); err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	if err := store.putReader(context.Background(), "backups/test.dump", "application/octet-stream", file, info.Size()); err != nil {
		t.Fatalf("putReader() error = %v", err)
	}
	if received != payload {
		t.Fatalf("uploaded body = %q, want %q", received, payload)
	}
}

func TestBuildS3CanonicalRequestUsesSingleHeaderSeparator(t *testing.T) {
	headers := "host:storage.example\n" +
		"x-amz-content-sha256:abc123\n" +
		"x-amz-date:20261124T120000Z\n"
	got := buildS3CanonicalRequest(http.MethodGet, "/bucket/object", "", headers,
		"host;x-amz-content-sha256;x-amz-date", "abc123")
	want := "GET\n/bucket/object\n\n" +
		"host:storage.example\n" +
		"x-amz-content-sha256:abc123\n" +
		"x-amz-date:20261124T120000Z\n" +
		"host;x-amz-content-sha256;x-amz-date\nabc123"
	if got != want {
		t.Fatalf("canonical request mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestS3MultipartUploadUsesPartsAndCompletes(t *testing.T) {
	const partSize = int64(5 * 1024 * 1024)
	payload := strings.Repeat("a", int(partSize)) + strings.Repeat("b", int(partSize)) + "final-part"
	uploaded := make(map[int]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		switch {
		case r.Method == http.MethodPost && query.Has("uploads"):
			_, _ = io.WriteString(w, "<InitiateMultipartUploadResult><UploadId>test-upload-id</UploadId></InitiateMultipartUploadResult>")
		case r.Method == http.MethodPut && query.Get("uploadId") == "test-upload-id":
			part, err := strconv.Atoi(query.Get("partNumber"))
			if err != nil {
				t.Errorf("invalid part number: %v", err)
				w.WriteHeader(400)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read part: %v", err)
				w.WriteHeader(500)
				return
			}
			if int64(len(body)) != r.ContentLength {
				t.Errorf("part %d length mismatch", part)
			}
			uploaded[part] = string(body)
			w.Header().Set("ETag", "etag-"+strconv.Itoa(part))
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && query.Get("uploadId") == "test-upload-id":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read completion: %v", err)
			}
			var complete s3MultipartCompletion
			if err := xml.Unmarshal(body, &complete); err != nil {
				t.Errorf("decode completion: %v", err)
			}
			if len(complete.Parts) != 3 {
				t.Errorf("completion has %d parts, want 3", len(complete.Parts))
			}
			_, _ = io.WriteString(w, "<CompleteMultipartUploadResult><ETag>final-etag</ETag></CompleteMultipartUploadResult>")
		case r.Method == http.MethodDelete && query.Get("uploadId") == "test-upload-id":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected multipart request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	store := &s3Store{endpoint: endpoint, bucket: "test-bucket", region: "test-region-1", access: "test-access", secret: "test-secret", client: server.Client()}
	reader := strings.NewReader(payload)
	if err := store.putReaderMultipartWithPartSize(context.Background(), "backups/large.dump", "application/octet-stream", reader, int64(len(payload)), partSize); err != nil {
		t.Fatalf("multipart upload failed: %v", err)
	}
	if uploaded[1] != strings.Repeat("a", int(partSize)) || uploaded[2] != strings.Repeat("b", int(partSize)) || uploaded[3] != "final-part" {
		t.Fatalf("multipart parts do not match source: sizes=%d,%d,%d", len(uploaded[1]), len(uploaded[2]), len(uploaded[3]))
	}
}
