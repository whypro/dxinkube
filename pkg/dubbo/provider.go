package dubbo

import (
	"fmt"
	neturl "net/url"
	"sort"
	"strings"
	"time"
)

type Provider struct {
	scheme  string
	Addr    string
	Service string
	params  map[string]string
}

func NewProvider() *Provider {
	return &Provider{
		params: make(map[string]string),
	}
}

func (p *Provider) Url() string {
	return fmt.Sprintf("%s://%s/%s", p.scheme, p.Addr, p.Service)
}

func (p *Provider) query() string {
	sortedKeys := make([]string, 0, len(p.params))
	for k := range p.params {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	paramsSlice := make([]string, 0, len(p.params))
	for _, k := range sortedKeys {
		paramsSlice = append(paramsSlice, fmt.Sprintf("%s=%s", k, p.params[k]))
	}
	return strings.Join(paramsSlice, "&")
}

func (p *Provider) String() string {
	return p.Url() + "?" + p.query()
}

func (p *Provider) SetTimestamp() {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	p.params["timestamp"] = ts
}

func (p *Provider) Key() string {
	return p.Url()
}

func (p *Provider) Parse(url string) error {
	unescapedURL, err := neturl.QueryUnescape(url)
	if err != nil {
		return err
	}
	// TODO: use RE
	schemeAndOther := strings.Split(unescapedURL, "://")
	p.scheme = schemeAndOther[0]
	urlAndPath := strings.Split(schemeAndOther[1], "/")
	p.Addr = urlAndPath[0]
	pathAndParams := strings.Split(urlAndPath[1], "?")
	p.Service = pathAndParams[0]
	params := strings.Split(pathAndParams[1], "&")
	for _, param := range params {
		pSlice := strings.Split(param, "=")
		k, v := pSlice[0], pSlice[1]
		p.params[k] = v
	}
	return nil
}
