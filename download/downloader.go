package download

import (
	// _ "context"
	"fmt"
	// "github.com/go-redis/redis/v8"
	"github.com/garyburd/redigo/redis"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

type Downloader struct {
	UserAgent string
	Referer   string
}

// var CookieRedisClient *redis.Client
var pool *redis.Pool

// var ctx = context.Background()

func init() {
	pool = &redis.Pool{
		MaxIdle:     8,
		MaxActive:   0,
		IdleTimeout: 100,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "172.18.255.7:6379", redis.DialDatabase(1), redis.DialPassword("youcheng7347"))
		},
	}
	// CookieRedisClient = redis.NewClient(&redis.Options{
	// 	Addr:     "172.18.255.7:6379",
	// 	Password: "youcheng7347", // no password set
	// 	DB:       1,              // use default DB
	// })
}

func (this *Downloader) DoCookie(req *http.Request) {
	// cookieStr, err := CookieRedisClient.Get(ctx, "zc_cookies").Result()
	conn := pool.Get()
	defer conn.Close()
	cookieStr, err := redis.String(conn.Do("GET", "zc_cookies"))
	if err != nil {
		panic(fmt.Sprintf("获取cookie失败：%v", err))
	}
	x5sec := gjson.Get(cookieStr, "x5sec").String()
	cookie1 := &http.Cookie{
		Name:  "x5sec",
		Value: x5sec,
	}
	req.AddCookie(cookie1)
	rand.Seed(time.Now().Unix())
	randIntStr := fmt.Sprintf("%v", rand.Intn(1000000000))
	// fmt.Println(randIntStr)
	cookie2 := &http.Cookie{
		Name:  "unb",
		Value: randIntStr,
	}
	req.AddCookie(cookie2)
}

func (this *Downloader) Get(url string, toDoCookie bool) (result string) {
	client := &http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
	req, _ := http.NewRequest("GET", url, nil)
	// fmt.Println(this.UserAgent)
	// fmt.Println(url)
	req.Header.Set("User-Agent", this.UserAgent)
	req.Header.Set("Referer", this.Referer)
	if toDoCookie {
		this.DoCookie(req)
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("请求失败：%v", err))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("读取请求结果失败：%v", err))
	}
	result = string(body)
	// fmt.Println(result)
	return
}
