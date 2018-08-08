package rs

import (
	"encoding/base64"
	"net/http"

	"fmt"
	"github.com/tonycai653/iqshell/qiniu/api.v6/auth/digest"
	. "github.com/tonycai653/iqshell/qiniu/api.v6/conf"
	"github.com/tonycai653/iqshell/qiniu/rpc"
	"github.com/tonycai653/iqshell/qiniu/uri"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// ----------------------------------------------------------

type Client struct {
	Conn rpc.Client
}

func NewMac(mac *digest.Mac) Client {
	t := digest.NewTransport(mac, nil)
	client := &http.Client{Transport: t}
	return Client{rpc.Client{client, ""}}
}

func NewEx(t http.RoundTripper) Client {
	client := &http.Client{Transport: t}
	return Client{rpc.Client{client, ""}}
}

func NewMacEx(mac *digest.Mac, t http.RoundTripper, bindRemoteIp string) Client {
	mt := digest.NewTransport(mac, t)
	client := &http.Client{Transport: mt}
	return Client{rpc.Client{client, bindRemoteIp}}
}

// ----------------------------------------------------------

// @gist entry
type Entry struct {
	Hash     string `json:"hash"`
	Fsize    int64  `json:"fsize"`
	PutTime  int64  `json:"putTime"`
	MimeType string `json:"mimeType"`
	Customer string `json:"customer"`
	FileType int    `json:"type"`
}

type GetRet struct {
	URL      string `json:"url"`
	Hash     string `json:"hash"`
	MimeType string `json:"mimeType"`
	Fsize    int64  `json:"fsize"`
	Expiry   int64  `json:"expires"`
	Version  string `json:"version"`
}

// @endgist

func (rs Client) Get(l rpc.Logger, bucket, key, destFile string) (err error) {
	entryUri := strings.Join([]string{bucket, key}, ":")

	url := strings.Join([]string{RS_HOST, "get", uri.Encode(entryUri)}, "/")

	var data GetRet

	err = rs.Conn.Call(nil, &data, url)
	if err != nil {
		return
	}
	resp, err := rs.Conn.Get(nil, data.URL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, rerr := ioutil.ReadAll(resp.Body)
		if rerr != nil {
			return rerr
		}
		fmt.Fprintf(os.Stderr, "Qget: http respcode: %d, respbody: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}
	f, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	io.Copy(f, resp.Body)
	return
}

func (rs Client) Stat(l rpc.Logger, bucket, key string) (entry Entry, err error) {
	err = rs.Conn.Call(l, &entry, RS_HOST+URIStat(bucket, key))
	return
}

func (rs Client) Delete(l rpc.Logger, bucket, key string) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+URIDelete(bucket, key))
}

func (rs Client) Move(l rpc.Logger, bucketSrc, keySrc, bucketDest, keyDest string, force bool) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+URIMove(bucketSrc, keySrc, bucketDest, keyDest, force))
}

func (rs Client) Copy(l rpc.Logger, bucketSrc, keySrc, bucketDest, keyDest string, force bool) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+URICopy(bucketSrc, keySrc, bucketDest, keyDest, force))
}

func (rs Client) ChangeMime(l rpc.Logger, bucket, key, mime string) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+URIChangeMime(bucket, key, mime))
}

func (rs Client) ChangeType(l rpc.Logger, bucket, key string, fileType int) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+URIChangeType(bucket, key, fileType))
}

func (rs Client) DeleteAfterDays(l rpc.Logger, bucket, key string, days int) (err error) {
	return rs.Conn.Call(l, nil, RS_HOST+URIDeleteAfterDays(bucket, key, days))
}

func encodeURI(uri string) string {
	return base64.URLEncoding.EncodeToString([]byte(uri))
}

func URIDelete(bucket, key string) string {
	return fmt.Sprintf("/delete/%s", encodeURI(bucket+":"+key))
}

func URIStat(bucket, key string) string {
	return fmt.Sprintf("/stat/%s", encodeURI(bucket+":"+key))
}

func URICopy(bucketSrc, keySrc, bucketDest, keyDest string, force bool) string {
	return fmt.Sprintf("/copy/%s/%s/force/%v", encodeURI(bucketSrc+":"+keySrc), encodeURI(bucketDest+":"+keyDest), force)
}

func URIMove(bucketSrc, keySrc, bucketDest, keyDest string, force bool) string {
	return fmt.Sprintf("/move/%s/%s/force/%v", encodeURI(bucketSrc+":"+keySrc), encodeURI(bucketDest+":"+keyDest), force)
}

func URIChangeMime(bucket, key, mime string) string {
	return fmt.Sprintf("/chgm/%s/mime/%s", encodeURI(bucket+":"+key), encodeURI(mime))
}

func URIChangeType(bucket, key string, fileType int) string {
	return fmt.Sprintf("/chtype/%s/type/%d", encodeURI(bucket+":"+key), fileType)
}

func URIDeleteAfterDays(bucket, key string, days int) string {
	return fmt.Sprintf("/deleteAfterDays/%s/%d", encodeURI(bucket+":"+key), days)
}

func URIPrefetch(bucket, key string) string {
	return fmt.Sprintf("/prefetch/%s", encodeURI(bucket+":"+key))
}
