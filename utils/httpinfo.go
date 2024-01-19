package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/mssola/user_agent"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type Whois struct {
	Pro  string `json:"pro"`
	Ip   string `json:"ip"`
	City string `json:"city"`
}

func GBKToUTF8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := io.ReadAll(reader)
	if e != nil {
		return nil, e

	}
	return d, nil

}

func HttpGetClientOs(userAgent string) string {
	return user_agent.New(userAgent).OS()
}

func HttpGetClientBrowser(userAgent string) string {
	ua := user_agent.New(userAgent)
	n, v := ua.Browser()
	return n + "-" + v
}

func HttpGetClientLocation(ip string) string {
	if ip == "" {
		return ""
	}

	if ip == "::1" || ip == "127.0.0.1" {
		return "内网ip"
	}

	whois := Whois{}
	c, _ := client.NewClient()
	url := "http://whois.pconline.com.cn/ipJson.jsp?json=true&ip=" + ip

	status, body, err := c.Get(context.Background(), nil, url)
	if err != nil {
		return ""
	}

	if status != 200 {
		return ""
	}

	body, err = GBKToUTF8(body)
	if err != nil {
		return ""
	}

	err = json.Unmarshal(body, &whois)
	if err != nil {
		return ""
	}
	return whois.Pro + " " + whois.City
}
