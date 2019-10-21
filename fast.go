// Fast measures the network's download speed using fast.com.
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"text/tabwriter"

	"github.com/ericlagergren/fast/internal/api"
	"github.com/gonum/stat"
)

func main() {
	var (
		token     string
		nurls     int
		userAgent string
		chatty    bool
	)
	flag.StringVar(&token, "token", api.DefaultToken, "api.fast.com access token")
	flag.IntVar(&nurls, "urls", 3, "number of URLs to try")
	flag.StringVar(&userAgent, "user-agent", api.DefaultUserAgent, "user agent to use")
	flag.BoolVar(&chatty, "v", false, "be verbose")
	flag.Parse()

	if chatty {
		fmt.Fprintln(os.Stderr, "Retrieving fast.com configuration...")
	}
	c, err := api.Load(
		api.WithToken(token),
		api.NumURLs(nurls),
		api.WithUserAgent(userAgent),
	)
	if err != nil {
		log.Fatal(err)
	}

	if chatty {
		isp := c.Client.ISP
		if isp == "" {
			isp = "???"
		}
		fmt.Fprintf(os.Stderr, "Testing from %s (%s)...\n\n", isp, c.Client.IP)
	}

	w := new(tabwriter.Writer)
	initWriter(w)

	tprintln(w, "server\t# iters\tspeed (Mbit/s)")

	x := make([]float64, 0, len(c.Targets))
	weights := make([]float64, 0, len(c.Targets))
	for i, t := range c.Targets {
		url := t.URL
		tprintf(w, "%s", parseHost(url))

		r := testing.Benchmark(func(b *testing.B) {
			var once sync.Once
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					resp, err := http.DefaultClient.Get(url)
					if err != nil {
						b.Fatal(err)
					}
					nw, err := io.Copy(ioutil.Discard, resp.Body)
					resp.Body.Close()
					if err != nil {
						b.Fatal(err)
					}
					once.Do(func() { b.SetBytes(nw) })
				}
			})
		})
		mbps := float64(r.Bytes*int64(r.N)*8) / 1e6 / r.T.Seconds()
		x = append(x, mbps)
		weights = append(weights, float64(r.N))

		tprintf(w, "\t%d\t%.3f\n", r.N, mbps)
		// Align the "RESULT: ..." section. This only works because all the URLs
		// are the same size.
		if i != len(c.Targets)-1 {
			w.Flush()
			initWriter(w)
		}
	}

	mean, std := stat.MeanStdDev(x, weights)
	tprintf(w, "\t\t%.3f ±%.3f\n", mean, std)
	w.Flush()
}

func initWriter(w *tabwriter.Writer) {
	w.Init(os.Stdout, 20, 1, 3, ' ', tabwriter.StripEscape)
}

func tprintf(w *tabwriter.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...)
}

func tprintln(w *tabwriter.Writer, args ...interface{}) {
	fmt.Fprintln(w, args...)
}

func parseHost(url_ string) string {
	u, err := url.Parse(url_)
	if err != nil {
		return err.Error()
	}
	return u.Host
}
