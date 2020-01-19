package bm1365Model

import (
	"../../../module"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// 用于生成响应解析器
func genResponseParsers() []module.ParseResponse {
	parseLink := func(httpResp *http.Response, respDepth uint32) ([]module.Data, []error) {
		dataList := make([]module.Data, 0)
		// 检查响应
		if httpResp == nil {
			return nil, []error{fmt.Errorf("空的HTTP响应")}
		}
		httpReq := httpResp.Request
		if httpReq == nil {
			return nil, []error{fmt.Errorf("空的HTTP请求")}
		}
		reqURL := httpReq.URL
		if httpResp.StatusCode != 200 {
			err := fmt.Errorf("请求返回状态不是200 %d (requestURL: %s)",
				httpResp.StatusCode, reqURL)
			return nil, []error{err}
		}
		body := httpResp.Body
		if body == nil {
			err := fmt.Errorf("空的HTTP响应体 (requestURL: %s)",
				reqURL)
			return nil, []error{err}
		}
		// 检查HTTP响应头中的内容类型
		var matchedContentType bool
		if httpResp.Header != nil {
			contentTypes := httpResp.Header["Content-Type"]
			for _, ct := range contentTypes {
				if strings.HasPrefix(ct, "text/html") {
					matchedContentType = true
					break
				}
			}
		}
		if !matchedContentType {
			return dataList, nil
		}
		// 解析HTTP响应体
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return dataList, []error{err}
		}
		errs := make([]error, 0)

		// 生成条目
		item := make(map[string]interface{})
		j := &JcUx{dom: doc}
		j.getType()
		j.getInfo()
		j.getText()
		j.getImg()
		item["bmInfo"] = *j
		dataList = append(dataList, module.Item(item))

		// 查找img标签并提取地址
		exp := func(selection *goquery.Selection) {
			imgSrc, exists := selection.Attr("src")
			if !exists || imgSrc == "" || imgSrc == "#" || imgSrc == "/" {
				return
			}
			imgSrc = strings.TrimSpace(imgSrc)
			imgURL, err := url.Parse(imgSrc)
			if err != nil {
				errs = append(errs, err)
				return
			}
			if !imgURL.IsAbs() {
				imgURL = reqURL.ResolveReference(imgURL)
			}
			httpReq, err := http.NewRequest("GET", imgURL.String(), nil)
			if err != nil {
				errs = append(errs, err)
			} else {
				req := module.NewRequest(httpReq, respDepth)
				dataList = append(dataList, req)
			}
		}

		doc.Find(".prod_detail img").Each(func(i int, selection *goquery.Selection) {
			exp(selection)
		})
		doc.Find(".yyal img").Each(func(i int, selection *goquery.Selection) {
			exp(selection)
		})
		doc.Find(".jdgz img").Each(func(i int, selection *goquery.Selection) {
			exp(selection)
		})

		return dataList, errs
	}
	parseImg := func(httpResp *http.Response, respDepth uint32) ([]module.Data, []error) {
		// 检查响应
		if httpResp == nil {
			return nil, []error{fmt.Errorf("空的HTTP响应")}
		}
		httpReq := httpResp.Request
		if httpReq == nil {
			return nil, []error{fmt.Errorf("空的HTTP请求")}
		}
		reqURL := httpReq.URL
		if httpResp.StatusCode != 200 {
			err := fmt.Errorf("请求返回状态不是200 %d (requestURL: %s)",
				httpResp.StatusCode, reqURL)
			return nil, []error{err}
		}
		httpRespBody := httpResp.Body
		if httpRespBody == nil {
			err := fmt.Errorf("空的HTTP响应体 (requestURL: %s)",
				reqURL)
			return nil, []error{err}
		}
		// 检查HTTP响应头中的内容类型
		dataList := make([]module.Data, 0)
		var pictureFormat string
		if httpResp.Header != nil {
			contentTypes := httpResp.Header["Content-Type"]
			var contentType string
			for _, ct := range contentTypes {
				if strings.HasPrefix(ct, "image") {
					contentType = ct
					break
				}
			}
			index1 := strings.Index(contentType, "/")
			index2 := strings.Index(contentType, ";")
			if index1 > 0 {
				if index2 < 0 {
					pictureFormat = contentType[index1+1:]
				} else if index1 < index2 {
					pictureFormat = contentType[index1+1 : index2]
				}
			}
		}
		if pictureFormat == "" {
			return dataList, nil
		}
		// 生成条目
		item := make(map[string]interface{})
		item["reader"] = httpRespBody
		item["name"] = path.Base(reqURL.Path)
		item["ext"] = pictureFormat
		dataList = append(dataList, module.Item(item))
		return dataList, nil
	}
	return []module.ParseResponse{parseLink, parseImg}
}
