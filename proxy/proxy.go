package proxy

import (
	"io"
	"log"
	"net/http"
)

type Proxy struct {
	Protocol     string
	Host         string
	Port         string
	LatencyInSec int32
	UpdatedTs    int64
}

var (
	SeedProxy  = "http://localhost:1081"
	ProxySites = []string{
		"https://free-proxy-list.net/",
	}
)

func (p *Proxy) ToString() string {
	return fmt.Sprintf("%s://%s:%s", p.Protocol, p.Host, p.Port)
}

func GetProxiedClient(p string) *http.Client {
	if len(p) == 0 {
		return &http.Client{}
	}
	proxyUrl, err := url.Parse(p)
	if err != nil {
		log.Println(err)
		return &http.Client{}
	}
	proxiedClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	return proxiedClient
}

func GetProxyList() []*Proxy {
	bytes, err := crawlProxySite("https://free-proxy-list.net/", SeedProxy)
	if err != nil {
		log.Println(err)
		return nil
	}

}

func crawlProxySite(url, proxy string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")

	client := GetProxiedClient(proxy)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Crawler Error: Failed to crawl \"" + url + "\"")
		return nil, err
	}

	b := resp.Body
	defer b.Close()

	data, err := ioutil.ReadAll(b)
	if err != nil {
		log.Println("Crawler Error: ioutil.ReadAll -> %v", err)
		return nil, err
	}

	return data, err
}
