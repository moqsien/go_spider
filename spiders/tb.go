package spiders

import (
	"fmt"
	"github.com/antchfx/htmlquery"
	"github.com/axgle/mahonia"
	"github.com/tidwall/gjson"
	"net/url"
	"search/download"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TbDownloader struct {
	download.Downloader
	wg sync.WaitGroup
}

var TbUrl string = "https://sf.taobao.com/item_list.htm?q=%v&page=%v"
var TbPageSize int64 = 40

func (this *TbDownloader) GetOnePage(anHao string, page int, getTotalItem bool, putChan chan int64) {
	defer this.wg.Done()
	defer func() {
		if info := recover(); info != nil {
			fmt.Println(info)
			if getTotalItem {
				close(putChan)
			} else {
				putChan <- int64(0)
			}
		}
	}()
	enc := mahonia.NewEncoder("gbk")
	q := enc.ConvertString(anHao)
	q_ := url.QueryEscape(q)
	realUrl := fmt.Sprintf(TbUrl, q_, page)
	r := ""
	// 验证无法通过时最多重试3次
	for i := 0; i <= 3; i++ {
		r = this.Get(realUrl, true)
		if !strings.Contains(r, "霸下通用 web 页面-验证码") {
			break
		}
		// 失败一次等2秒重试
		time.Sleep(2 * time.Second)
	}
	html, err := htmlquery.Parse(strings.NewReader(r))
	if err != nil {
		panic(fmt.Sprintf("解析错误：%v", err))
	} else {
		if getTotalItem {
			total := htmlquery.FindOne(html, "//em[@class='page-total']")
			totalPage := int64(0)
			if total != nil {
				totalPage, _ = strconv.ParseInt(htmlquery.InnerText(total), 0, 64)
			}
			putChan <- totalPage
		} else {
			data := htmlquery.FindOne(html, "//script[@id='sf-item-list-data']")
			if data != nil {
				jsonStr := htmlquery.InnerText(data)
				if jsonStr != "" {
					pidArray := gjson.Get(jsonStr, "data.#.id").Array()
					for _, value := range pidArray {
						putChan <- int64(value.Int())
					}
				}
			}
		}
	}
}

func (this *TbDownloader) Run(anHao string) ([]int64, string) {
	ch := make(chan int64, 50)
	sigChan := make(chan struct{})
	var msg string = "请求成功"
	pidArray := []int64{}
	// defer close(ch)
	this.wg.Add(1)
	go this.GetOnePage(anHao, 1, true, ch)
	totalPage, isNotClosed := <-ch
	if !isNotClosed {
		return []int64{}, "cookie获取失败"
	}
	if totalPage == 1 {
		this.wg.Add(1)
		go this.GetOnePage(anHao, 1, false, ch)
	} else if totalPage > 1 {
		for i := 1; i <= int(totalPage); i++ {
			this.wg.Add(1)
			go this.GetOnePage(anHao, i, false, ch)
		}
	} else {
		return []int64{}, "没有查询到相关数据"
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
	// close(ch)
	// time.Sleep(1 * time.Second)
	for {
		if len(ch) == 0 {
			sigChan <- struct{}{}
			break
		}
	}
	// for pid := range ch {
	// 	if pid != int64(0) && pid != int64(1) {
	// 		pidArray = append(pidArray, pid)
	// 	} else if pid == int64(0) {
	// 		msg = "请求错误，请稍后重试"
	// 	} else {
	// 		msg = "未返回数据，请稍后重试"
	// 	}
	// }
	return pidArray, msg
}
