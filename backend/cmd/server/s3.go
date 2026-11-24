package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type s3Store struct {
	endpoint *url.URL
	bucket   string
	region   string
	access   string
	secret   string
	client   *http.Client
}

func newS3Store() (*s3Store, error) {
	endpointValue := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	region := strings.TrimSpace(os.Getenv("S3_REGION"))
	access := strings.TrimSpace(os.Getenv("S3_ACCESS_KEY_ID"))
	secret := strings.TrimSpace(os.Getenv("S3_SECRET_ACCESS_KEY"))
	if endpointValue == "" || bucket == "" || region == "" || access == "" || secret == "" {
		return nil, errors.New("S3 is not configured")
	}
	endpoint, err := url.Parse(endpointValue)
	if err != nil || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.Host == "" {
		return nil, errors.New("invalid S3 endpoint")
	}
	return &s3Store{
		endpoint: endpoint,
		bucket: bucket,
		region: region,
		access: access,
		secret: secret,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func (s *s3Store) objectURL(key string) *url.URL {
	u := *s.endpoint
	escapedKey := escapeS3Key(key)
	basePath := strings.TrimRight(u.EscapedPath(), "/")
	u.RawPath = basePath + "/" + url.PathEscape(s.bucket) + "/" + escapedKey
	decodedPath, _ := url.PathUnescape(u.RawPath)
	u.Path = decodedPath
	u.RawQuery = ""
	u.Fragment = ""
	return &u
}

func escapeS3Key(key string) string {
	parts := strings.Split(key, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func (s *s3Store) signingKey(date string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+s.secret), date)
	regionKey := hmacSHA256(dateKey, s.region)
	serviceKey := hmacSHA256(regionKey, "s3")
	return hmacSHA256(serviceKey, "aws4_request")
}

func (s *s3Store) put(ctx context.Context, key, contentType string, data []byte) error {
	return s.doSigned(ctx, http.MethodPut, key, contentType, data)
}

func (s *s3Store) delete(ctx context.Context, key string) error {
	return s.doSigned(ctx, http.MethodDelete, key, "", nil)
}

func (s *s3Store) doSigned(ctx context.Context, method, key, contentType string, data []byte) error {
	u := s.objectURL(key)
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	payloadHash := sha256Hex(data)
	canonicalHeaders := ""
	signedHeaders := ""
	if contentType != "" {
		canonicalHeaders += "content-type:" + contentType + "\n"
		signedHeaders = "content-type;"
	}
	canonicalHeaders += "host:" + u.Host + "\n"
	signedHeaders += "host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders += "x-amz-content-sha256:" + payloadHash + "\n"
	canonicalHeaders += "x-amz-date:" + amzDate + "\n"
	canonicalRequest := method + "\n" + u.EscapedPath() + "\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash
	scope := dateStamp + "/" + s.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(s.signingKey(dateStamp), stringToSign))

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+s.access+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("S3 request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *s3Store) presignGet(key string, lifetime time.Duration) (string, error) {
	u := s.objectURL(key)
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	scope := dateStamp + "/" + s.region + "/s3/aws4_request"

	query := url.Values{}
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", s.access+"/"+scope)
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", fmt.Sprintf("%d", int(lifetime.Seconds())))
	query.Set("X-Amz-SignedHeaders", "host")
	canonicalQuery := query.Encode()
	canonicalRequest := http.MethodGet + "\n" + u.EscapedPath() + "\n" + canonicalQuery + "\nhost:" + u.Host + "\n\nhost\nUNSIGNED-PAYLOAD"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(s.signingKey(dateStamp), stringToSign))
	u.RawQuery = canonicalQuery + "&X-Amz-Signature=" + signature
	return u.String(), nil
}

func randomObjectID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
