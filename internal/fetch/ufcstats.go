package fetch

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL       = "http://ufcstats.com"
	challengePath = "/__c"
	upcomingPath  = "/statistics/events/upcoming"
	completedPath = "/statistics/events/completed"
	userAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

type Client struct {
	http *http.Client
}

func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookiejar: %w", err)
	}
	return &Client{
		http: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}, nil
}

func (c *Client) get(path string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: HTTP %d", path, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func isChallenge(body string) bool {
	return strings.Contains(body, "Checking your browser") || strings.Contains(body, `nonce="`)
}

func (c *Client) solveChallenge() error {
	body, err := c.get("/statistics/events/completed?page=1")
	if err != nil {
		return err
	}
	if !isChallenge(body) {
		return nil
	}

	nonce, err := extractNonce(body)
	if err != nil {
		return err
	}
	n, err := solvePow(nonce, "00")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("nonce", nonce)
	form.Set("n", strconv.Itoa(n))

	req, err := http.NewRequest(http.MethodPost, baseURL+challengePath, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", baseURL+"/statistics/events/completed")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s: HTTP %d", challengePath, resp.StatusCode)
	}
	return nil
}

func (c *Client) fetch(path string) (string, error) {
	body, err := c.get(path)
	if err != nil {
		return "", err
	}
	if !isChallenge(body) {
		return body, nil
	}
	if err := c.solveChallenge(); err != nil {
		return "", fmt.Errorf("solve challenge: %w", err)
	}
	return c.get(path)
}

func (c *Client) Upcoming() (string, error) {
	return c.fetch(upcomingPath)
}

func (c *Client) EventDetail(id string) (string, error) {
	return c.fetch("/event-details/" + id)
}

func (c *Client) Completed(page int) (string, error) {
	if page < 1 {
		page = 1
	}
	return c.fetch(fmt.Sprintf("%s?page=%d", completedPath, page))
}
