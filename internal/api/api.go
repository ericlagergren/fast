// Package api loads configuration data from api.fast.com.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
)

type loader struct {
	c         *http.Client
	token     string
	nurls     int
	userAgent string
}

// Option is a function that modifies the loading of an API configuration.
type Option func(*loader)

// WithClient sets the Client used to load the configuration.
func WithClient(hc *http.Client) Option {
	return func(c *loader) { c.c = hc }
}

// WithToken sets the API token used to load the configuration.
func WithToken(token string) Option {
	return func(c *loader) { c.token = token }
}

// NumURLs sets the number of URLs the configuration should contain.
func NumURLs(n int) Option {
	return func(c *loader) { c.nurls = n }
}

// WithUserAgent sets the user agent to use while loading the configuration.
func WithUserAgent(userAgent string) Option {
	return func(c *loader) { c.userAgent = userAgent }
}

const (
	// DefaultToken is a known api.fast.com token, used by Load if no other token
	// is specified.
	DefaultToken = "YXNkZmFzZGxmbnNkYWZoYXNkZmhrYWxm"
	// DefaultUserAgent is used by Load if no other user agent is specified.
	DefaultUserAgent = "fast/0.1 (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
)

// Load fetches api.fast.com's configuration.
func Load(opts ...Option) (*Config, error) {
	l := loader{
		c:         http.DefaultClient,
		userAgent: DefaultUserAgent,
	}
	for _, fn := range opts {
		fn(&l)
	}
	if l.nurls == 0 {
		l.nurls = 3
	}
	if l.token == "" {
		l.token = DefaultToken
	}

	req, err := http.NewRequest("GET", makeURL(l.token, l.nurls), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", l.userAgent)

	resp, err := l.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch c := resp.StatusCode; c {
	case http.StatusOK:
		// OK
	case http.StatusForbidden:
		return nil, errors.New("invalid API token")
	default:
		return nil, fmt.Errorf("non-200 status code: %d", c)
	}

	var cfg Config
	err = json.NewDecoder(resp.Body).Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func makeURL(token string, nurls int) string {
	v := make(url.Values, 3)
	v.Add("https", "true")
	v.Add("token", token)
	v.Add("urlCount", strconv.Itoa(nurls))
	return (&url.URL{
		Scheme:   "https",
		Host:     "api.fast.com",
		Path:     "netflix/speedtest/v2",
		RawQuery: v.Encode(),
	}).String()
}

// Config is an api.fast.com configuration.
type Config struct {
	// Client contains information on the client requesting the configuration.
	Client Client `json:"client"`
	// Targets contains
	Targets []Target `json:"targets"`
}

type Location struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

// Client contains information on the client requesting the configuration.
type Client struct {
	// ASN is the ISP's Autonomous System Number.
	//
	// For example, CenturyLink's ASN is 209.
	// https://whois.arin.net/rest/asn/AS209
	ASN string `json:"asn"`
	// ISP is the name of the ISP.
	ISP      string   `json:"isp"`
	Location Location `json:"location"`
	// IP is the client's public IP address.
	IP net.IP `json:"ip"`
}

// Target describes a particular file used to measure download speed.
type Target struct {
	// URL is file's URL.
	URL string `json:"url"`
	// Location currently contains (Netflix, Open Connect).
	Location Location `json:"location"`
	// Name is currently set to the same thing as URL.
	Name string `json:"name"`
}
