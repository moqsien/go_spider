package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"search/spiders"
	"strings"
)

type MyDownloader interface {
	Run(anHao string) ([]int64, string)
	GetOnePage(anHao string, page int, getTotalItem bool, putChan chan int64)
	DoCookie(req *http.Request)
	Get(url string, toDoCookie bool) (result string)
}

func runSpider(downloader MyDownloader, anHao string) ([]int64, string) {
	pidArray, errMsg := downloader.Run(anHao)
	return pidArray, errMsg
}

func run(key string, anHao string) ([]int64, string) {
	var pidArray []int64
	var errMsg string
	if key == "jd" {
		downloader := new(spiders.JdDownloader)
		downloader.UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"
		downloader.Referer = "https://auction.jd.com/sifa_list.html"
		pidArray, errMsg = runSpider(downloader, anHao)
	} else if key == "tb" {
		downloader := new(spiders.TbDownloader)
		downloader.UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36"
		downloader.Referer = "https://sf.taobao.com/item_list.htm"
		pidArray, errMsg = runSpider(downloader, anHao)
	} else {
		fmt.Println("请传入正确的key")
	}
	return pidArray, errMsg
}

func prepareResponse(c *gin.Context) {
	key := c.Param("key")
	anHao := c.DefaultQuery("anhao", "")
	if anHao != "" && (key == "jd" || key == "tb") {
		pidArray, errMsg := run(key, anHao)
		var status int = 0
		if strings.Contains(errMsg, "未返回数据") {
			status = 20010
		}
		c.JSON(http.StatusOK, gin.H{
			"status": status,
			"result": pidArray,
			"msg":    errMsg,
		})
	} else if anHao == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": 10011,
			"result": []int64{},
			"msg":    "请传入案号",
		})
	} else if key == "jd" || key == "tb" {
		c.JSON(http.StatusOK, gin.H{
			"status": 10011,
			"result": []int64{},
			"msg":    "请选择平台，支持jd和tb",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"status": 40010,
			"result": []int64{},
			"msg":    "未知错误",
		})
	}

}

func main() {
	// anHao := "兴执字第"
	// anHao := "连执行字"
	// pidArray := getJdPids(anHao)
	// pidArray := run("tb", anHao)
	// fmt.Println(pidArray)
	r := gin.Default()
	r.GET("/getAnhao/:key/", prepareResponse)
	r.Run("0.0.0.0:5090")
}
