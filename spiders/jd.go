package spiders

import (
	"fmt"
	"github.com/tidwall/gjson"
	"math/rand"
	"net/url"
	"regexp"
	"search/download"
	"sync"
	"time"
)

var JdUrl string = `https://api.m.jd.com/api?appid=paimai-search-soa&functionId=paimai_unifiedSearch&body=%s&loginType=3&jsonp=jQuery%d&_=%d`
var JdBody string = `{"apiType":2,"page":"%v","pageSize":%v,"reqSource":0,"keyword":"%v"}`
var JdPageSize int64 = 40

// var wg sync.WaitGroup

type JdDownloader struct {
	download.Downloader
	wg sync.WaitGroup
}

func (this *JdDownloader) GetOnePage(anHao string, page int, getTotalItem bool, putChan chan int64) {
	defer this.wg.Done()
	defer func() {
		if info := recover(); info != nil {
			fmt.Println(info)
			if getTotalItem {
				close(putChan)
			} else {
				putChan <- int64(0)
			}
			// close(putChan)
		}
	}()
	stamp := time.Now().Unix() * 1000
	rand.Seed(time.Now().Unix())
	randInt := rand.Intn(100000)
	rawBody := fmt.Sprintf(JdBody, page, JdPageSize, anHao)
	realBody := url.QueryEscape(rawBody)
	realUrl := fmt.Sprintf(JdUrl, realBody, randInt, stamp)
	// fmt.Println(realUrl)
	r := this.Get(realUrl, false)
	reg := regexp.MustCompile(`jQuery\d+\((.+?)\);`)
	matched := reg.FindStringSubmatch(r)
	if len(matched) >= 2 {
		json := matched[1]
		if getTotalItem {
			totalItem := gjson.Get(json, "totalItem").Int()
			putChan <- int64(totalItem)
		} else {
			pidArray := gjson.Get(json, "datas.#.id").Array()
			for _, value := range pidArray {
				putChan <- int64(value.Int())
			}
		}
	}
}

func (this *JdDownloader) Run(anHao string) ([]int64, string) {
	ch := make(chan int64, 50)
	sigChan := make(chan struct{})
	var msg string = "请求成功"
	pidArray := []int64{}
	// defer close(ch)
	this.wg.Add(1)
	go this.GetOnePage(anHao, 1, true, ch)
	totalItem, isNotClosed := <-ch
	if !isNotClosed {
		return []int64{}, "解析出错"
	}
	totalPage := (totalItem / JdPageSize) + 1
	if totalPage <= 1 {
		this.wg.Add(1)
		go this.GetOnePage(anHao, 1, false, ch)
	} else {
		for i := 1; i <= int(totalPage); i++ {
			this.wg.Add(1)
			go this.GetOnePage(anHao, i, false, ch)
		}
	}
	// 不断从管道中获取pid放入结果集
	go func() {
		for {
			pid, ok := <-ch
			if ok || pid != int64(0) {
				if pid != int64(0) && pid != int64(1) {
					pidArray = append(pidArray, pid)
				} else if pid == int64(0) {
					msg = "有请求错误，请稍后重试"
				} else {
					msg = "未返回数据，请稍后重试"
				}
			} else {
				break
			}
		}
	}()
	// 发送结束信号
	go func(putChan chan int64, signalChan chan struct{}) {
		<-signalChan
		close(putChan)
	}(ch, sigChan)
	this.wg.Wait()
	for {
		if len(ch) == 0 {
			sigChan <- struct{}{}
			break
		}
	}
	return pidArray, msg
}
