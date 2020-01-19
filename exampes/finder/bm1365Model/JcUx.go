package bm1365Model

import (
	"../../../module"
	"../../../scheduler"
	"encoding/json"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var excelFile *ExcelFile = nil

type ExcelFile struct {
	ef     *excelize.File
	row    int
	sheet  string
	efLook sync.Mutex
}

type JcUx struct {
	Scope        string
	Price        string
	Origin       string
	Manufacturer string
	Category1    string
	Category2    string
	Category3    string
	Agency       string
	Phone        string
	Address      string
	Email        string
	Info         string
	Images       string
	Title        string
	dom          *goquery.Document
}

// startPage 从第几页开始
// pageNum 一共爬多少页
func InitReqList(startPage int, pageNum int, sched scheduler.Scheduler) {
	logger.Info("开始拉取初始地址列表")
	ExcelInit()
	for i := startPage; i < startPage+pageNum; i++ {
		resp, err := http.PostForm("http://www.bml365.com/show/prod/getpmore/", url.Values{"type": {"0"}, "page": {strconv.Itoa(i)}, "order": {"favorite_desc"}, "city": {"0"}})
		if err != nil {
			logger.Errorf("拉取初始地址发生异常：%s", err.Error())
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logger.Errorf("初始地址获取body发生异常：%s", err.Error())
			}

			var f interface{}
			err = json.Unmarshal(body, &f)
			if err != nil {
				logger.Errorf("初始地址body转换json：%s", err.Error())
			}
			m := f.(map[string]interface{})
			p := m["page"].(map[string]interface{})
			l := p["list"].([]interface{})
			for _, val := range l {
				v := val.(map[string]interface{})
				createId := int64(v["create_id"].(float64))
				id := int64(v["id"].(float64))
				url := "http://www.bml365.com/qy/prod/v/" +
					strconv.FormatInt(createId, 10) + "-" + strconv.FormatInt(id, 10)
				HTTPReq, err := http.NewRequest("GET", url, nil)
				if err != nil {
					logger.Fatal(err)
					return
				}
				req := module.NewRequest(HTTPReq, 0)
				sched.SendReq(req)
			}
			resp.Body.Close()
		}
	}
}

func ExcelInit() {
	sheet := "Sheet1"
	f := excelize.NewFile()
	// 创建一个工作表
	index := f.NewSheet(sheet)
	f.SetActiveSheet(index)
	axis := getRid(1)
	f.SetCellValue(sheet, axis[0], "标题")
	f.SetCellValue(sheet, axis[1], "大类别")
	f.SetCellValue(sheet, axis[2], "子类别")
	f.SetCellValue(sheet, axis[3], "子子类别")
	f.SetCellValue(sheet, axis[4], "适用范围")
	f.SetCellValue(sheet, axis[5], "价格")
	f.SetCellValue(sheet, axis[6], "产地")
	f.SetCellValue(sheet, axis[7], "代理公司")
	f.SetCellValue(sheet, axis[8], "生产厂家")
	f.SetCellValue(sheet, axis[9], "电话")
	f.SetCellValue(sheet, axis[10], "地址")
	f.SetCellValue(sheet, axis[11], "邮箱")
	f.SetCellValue(sheet, axis[12], "内容")
	f.SetCellValue(sheet, axis[13], "图片")

	logger.Info("初始化表格成功！")
	excelFile = &ExcelFile{
		ef:    f,
		row:   2,
		sheet: sheet,
	}
}

func SaveExcel(savePath string) {
	if excelFile != nil {
		if err := excelFile.ef.SaveAs(savePath); err != nil {
			logger.Errorf("保存表格文件失败：%s", err)
		}
		logger.Info("保存表格文件成功！")
	}
}

func (j *JcUx) exportJcUx() {
	f := excelFile
	f.efLook.Lock()
	defer f.efLook.Unlock()
	axis := getRid(f.row)
	sheet := f.sheet
	f.ef.SetCellValue(sheet, axis[0], j.Title)
	f.ef.SetCellValue(sheet, axis[1], j.Category1)
	f.ef.SetCellValue(sheet, axis[2], j.Category2)
	f.ef.SetCellValue(sheet, axis[3], j.Category3)
	f.ef.SetCellValue(sheet, axis[4], j.Scope)
	f.ef.SetCellValue(sheet, axis[5], j.Price)
	f.ef.SetCellValue(sheet, axis[6], j.Origin)
	f.ef.SetCellValue(sheet, axis[7], j.Agency)
	f.ef.SetCellValue(sheet, axis[8], j.Manufacturer)
	f.ef.SetCellValue(sheet, axis[9], j.Phone)
	f.ef.SetCellValue(sheet, axis[10], j.Address)
	f.ef.SetCellValue(sheet, axis[11], j.Email)
	f.ef.SetCellValue(sheet, axis[12], j.Info)
	f.ef.SetCellValue(sheet, axis[13], j.Images)
	excelFile.row++
}

func (j *JcUx) getType() {
	j.dom.Find(".visible-xs-block .bread div p a").Each(func(i int, selection *goquery.Selection) {
		switch i {
		case 2:
			j.Category1 = selection.Text()
		case 3:
			j.Category2 = selection.Text()
		case 4:
			j.Category3 = selection.Text()
		}
	})
}

func (j *JcUx) getInfo() {
	j.dom.Find(".visible-xs-block div[style] .col-sm-7 div[style] h3").Each(func(i int, selection *goquery.Selection) {
		j.Title = selection.Text()
	})
	j.dom.Find(".visible-xs-block div[style] .col-sm-7 div[style] p").Each(func(i int, selection *goquery.Selection) {
		te := strings.Split(selection.Text(), ":")

		if len(te) < 2 {
			te[1] = ""
		}
		switch strings.TrimSpace(te[0]) {
		case "适用范围":
			j.Scope = strings.TrimSpace(te[1])
		case "价格":
			j.Price = strings.TrimSpace(te[1])
		case "产地":
			j.Origin = strings.TrimSpace(te[1])
		case "生产厂家":
			j.Manufacturer = strings.TrimSpace(te[1])
		case "代理公司":
			j.Agency = strings.TrimSpace(te[1])
		case "电话":
			j.Phone = strings.TrimSpace(te[1])
		case "地址":
			j.Address = strings.TrimSpace(te[1])
		case "邮箱":
			j.Email = strings.TrimSpace(te[1])
		}

	})
}

func (j *JcUx) getText() {
	j.dom.Find(".visible-xs-block .prod_detail").Each(func(i int, selection *goquery.Selection) {
		te := strings.Join(strings.Fields(selection.Text()), "")
		j.Info = te
	})
}

func (j *JcUx) getImg() {
	strs := []string{}
	exp := func(selection *goquery.Selection) {
		imgSrc, exists := selection.Attr("src")
		if !exists || imgSrc == "" || imgSrc == "#" || imgSrc == "/" {
			return
		}
		imgName := filepath.Base(imgSrc)
		strs = append(strs, imgName)
	}

	j.dom.Find(".prod_detail img").Each(func(i int, selection *goquery.Selection) {
		exp(selection)
	})
	j.dom.Find(".yyal img").Each(func(i int, selection *goquery.Selection) {
		exp(selection)
	})
	j.dom.Find(".jdgz img").Each(func(i int, selection *goquery.Selection) {
		exp(selection)
	})
	str := strings.Join(strs, ",")
	j.Images = str
}

func getRid(row int) []string {
	var first = 65
	axis := make([]string, 14)
	for i := 0; i < len(axis); i++ {
		axis[i] = string(rune(first+i)) + strconv.Itoa(row)
	}
	return axis
}
