package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
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

func buildS3CanonicalRequest(method, escapedPath, canonicalQuery, canonicalHeaders, signedHeaders, payloadHash string) string {
	return method + "\n" + escapedPath + "\n" + canonicalQuery + "\n" +
		canonicalHeaders + signedHeaders + "\n" + payloadHash
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

func (s *s3Store) putReader(ctx context.Context, key, contentType string, body io.ReadSeeker, size int64) error {
	return s.doSignedURLReader(ctx, http.MethodPut, s.objectURL(key), contentType, body, size)
}

func (s *s3Store) delete(ctx context.Context, key string) error {
	return s.doSigned(ctx, http.MethodDelete, key, "", nil)
}

func (s *s3Store) doSigned(ctx context.Context, method, key, contentType string, data []byte) error {
	return s.doSignedURLReader(ctx, method, s.objectURL(key), contentType, bytes.NewReader(data), int64(len(data)))
}

func (s *s3Store) doSignedURLReader(ctx context.Context, method string, u *url.URL, contentType string, body io.ReadSeeker, size int64) error {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, body); err != nil {
		return err
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return err
	}
	payloadHash := hex.EncodeToString(hasher.Sum(nil))
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
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
	canonicalRequest := buildS3CanonicalRequest(method, u.EscapedPath(), "", canonicalHeaders, signedHeaders, payloadHash)
	scope := dateStamp + "/" + s.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(s.signingKey(dateStamp), stringToSign))

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}
	req.ContentLength = size
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
	canonicalRequest := buildS3CanonicalRequest(http.MethodGet, u.EscapedPath(), canonicalQuery, "host:"+u.Host+"\n", "host", "UNSIGNED-PAYLOAD")"
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

type s3ListResult struct {
 XMLName xml.Name `xml:"ListBucketResult"`
 IsTruncated bool `xml:"IsTruncated"`
 NextContinuationToken string `xml:"NextContinuationToken"`
 Contents []struct {
  Key string `xml:"Key"`
  LastModified string `xml:"LastModified"`
 } `xml:"Contents"`
}

func (s *s3Store) listKeys(ctx context.Context,prefix string)([]string,error){
 keys:=make([]string,0);token:=""
 for {
  u:=*s.endpoint
  basePath:=strings.TrimRight(u.EscapedPath(),"/")
  u.RawPath=basePath+"/"+url.PathEscape(s.bucket)
  decoded,_:=url.PathUnescape(u.RawPath);u.Path=decoded
  query:=url.Values{};query.Set("list-type","2");query.Set("prefix",prefix);if token!=""{query.Set("continuation-token",token)}
  u.RawQuery=query.Encode()
  now:=time.Now().UTC();payloadHash:=sha256Hex(nil);amzDate:=now.Format("20060102T150405Z");dateStamp:=now.Format("20060102")
  canonicalHeaders:="host:"+u.Host+"\n"+"x-amz-content-sha256:"+payloadHash+"\n"+"x-amz-date:"+amzDate+"\n"
  signedHeaders:="host;x-amz-content-sha256;x-amz-date"
  canonicalRequest:=buildS3CanonicalRequest(http.MethodGet,u.EscapedPath(),u.Query().Encode(),canonicalHeaders,signedHeaders,payloadHash)
  scope:=dateStamp+"/"+s.region+"/s3/aws4_request"
  stringToSign:="AWS4-HMAC-SHA256\n"+amzDate+"\n"+scope+"\n"+sha256Hex([]byte(canonicalRequest))
  signature:=hex.EncodeToString(hmacSHA256(s.signingKey(dateStamp),stringToSign))
  req,err:=http.NewRequestWithContext(ctx,http.MethodGet,u.String(),nil);if err!=nil{return nil,err}
  req.Header.Set("X-Amz-Date",amzDate);req.Header.Set("X-Amz-Content-Sha256",payloadHash)
  req.Header.Set("Authorization","AWS4-HMAC-SHA256 Credential="+s.access+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
  resp,err:=s.client.Do(req);if err!=nil{return nil,err}
  body,readErr:=io.ReadAll(io.LimitReader(resp.Body,8<<20));resp.Body.Close()
  if readErr!=nil{return nil,readErr};if resp.StatusCode<200||resp.StatusCode>=300{return nil,fmt.Errorf("S3 list failed with status %d: %s",resp.StatusCode,strings.TrimSpace(string(body)))}
  var result s3ListResult;if err:=xml.Unmarshal(body,&result);err!=nil{return nil,err}
  for _,item:=range result.Contents{keys=append(keys,item.Key)}
  if !result.IsTruncated||result.NextContinuationToken==""{break};token=result.NextContinuationToken
 }
 return keys,nil
}

func (s *s3Store) get(ctx context.Context,key string)([]byte,error){
 u:=s.objectURL(key);now:=time.Now().UTC();dateStamp:=now.Format("20060102");amzDate:=now.Format("20060102T150405Z");payloadHash:=sha256Hex(nil)
 canonicalHeaders:="host:"+u.Host+"\n"+"x-amz-content-sha256:"+payloadHash+"\n"+"x-amz-date:"+amzDate+"\n"
 signedHeaders:="host;x-amz-content-sha256;x-amz-date"
 canonicalRequest:=buildS3CanonicalRequest(http.MethodGet,u.EscapedPath(),"",canonicalHeaders,signedHeaders,payloadHash)
 scope:=dateStamp+"/"+s.region+"/s3/aws4_request"
 stringToSign:="AWS4-HMAC-SHA256\n"+amzDate+"\n"+scope+"\n"+sha256Hex([]byte(canonicalRequest))
 signature:=hex.EncodeToString(hmacSHA256(s.signingKey(dateStamp),stringToSign))
 req,err:=http.NewRequestWithContext(ctx,http.MethodGet,u.String(),nil);if err!=nil{return nil,err}
 req.Header.Set("X-Amz-Date",amzDate);req.Header.Set("X-Amz-Content-Sha256",payloadHash)
 req.Header.Set("Authorization","AWS4-HMAC-SHA256 Credential="+s.access+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
 resp,err:=s.client.Do(req);if err!=nil{return nil,err};defer resp.Body.Close()
 body,err:=io.ReadAll(io.LimitReader(resp.Body,1<<20));if err!=nil{return nil,err}
 if resp.StatusCode<200||resp.StatusCode>=300{return nil,fmt.Errorf("S3 get failed with status %d: %s",resp.StatusCode,strings.TrimSpace(string(body)))}
 return body,nil
}
