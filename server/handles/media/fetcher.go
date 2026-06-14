package media

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
)

type HTTPFetcher struct {
	Client *http.Client
}

func (f HTTPFetcher) Fetch(ctx context.Context, source mediamodel.FeedSource) ([]byte, error) {
	client := f.Client
	if client == nil {
		client = httpClient(source.ProxyURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch feed failed with status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func httpClient(proxyURL string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}
