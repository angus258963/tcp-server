package dcard

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	url    = "https://www.dcard.tw/_api/search/posts" // https://www.dcard.tw/_api/search/posts?query=%E5%8F%B0&limit=1
	method = "GET"
)

type Request struct {
	Timeout time.Duration
	Limit   string
	Offset  string
}

type Post struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Excerpt    string `json:"excerpt"`
	LikeCount  int64  `json:"likeCount"`
	School     string `json:"school"`
	Department string `json:"department"`
}
type Posts []*Post

func New(timeout, limit, offset int) *Request {
	return &Request{
		Timeout: time.Duration(timeout) * time.Second,
		Limit:   strconv.Itoa(limit),
		Offset:  strconv.Itoa(offset),
	}
}

func (d *Request) Search(text string) (Posts, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Println("http.NewRequest failed")
		return nil, err
	}
	query := req.URL.Query()
	query.Set("query", text)
	query.Set("limit", d.Limit)
	query.Set("offset", d.Offset)

	req.URL.RawQuery = query.Encode()

	res, err := d.sendReq(req)
	if err != nil {
		log.Println("sendReq failed")
		return nil, err
	}

	return res, nil
}

func (d *Request) sendReq(req *http.Request) (Posts, error) {
	client := &http.Client{Timeout: d.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("client.Do failed")
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status failed")
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("ioutil.ReadAll failed")
		return nil, err
	}

	posts := Posts{}
	if err := json.Unmarshal(bytes, &posts); err != nil {
		log.Println("json.Unmarshal failed")
		return nil, err
	}
	// time.Sleep(time.Second * 3)

	return posts, nil
}
