// Package crawler_test exercises CrawlURLs using an in-process httptest.Server
// that returns Wikipedia-shaped HTML pages.  No live network connection is
// needed; Colly is exercised end-to-end through the public API.
package crawler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shelaghhaney/webcrawler/internal/crawler"
)

// wikiHTML returns a minimal Wikipedia-shaped HTML document so that Colly's
// CSS selectors (h1#firstHeading, #mw-content-text p) find the expected nodes.
func wikiHTML(title, paras string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s - Wikipedia</title></head>
<body>
<h1 id="firstHeading">%s</h1>
<div id="mw-content-text"><div class="mw-parser-output">
%s
</div></div>
</body>
</html>`, title, title, paras)
}

// startServer launches an httptest.Server with pre-defined routes and returns
// the server and a cleanup function.
func startServer(routes map[string]string) (*httptest.Server, func()) {
	mux := http.NewServeMux()
	for path, body := range routes {
		body := body
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, body)
		})
	}
	srv := httptest.NewServer(mux)
	return srv, srv.Close
}

// zeroDelayConfig returns a Config with no inter-request delay so unit tests
// run quickly, while still exercising the Colly rate-limiter path.
func zeroDelayConfig() crawler.Config {
	cfg := crawler.DefaultConfig()
	cfg.RequestDelay = 0
	cfg.Parallelism = 3
	return cfg
}

// ---------------------------------------------------------------------------

func TestCrawlURLs_ExtractsTitleAndBody(t *testing.T) {
	para := "<p>Robotics is an interdisciplinary branch of computer science and engineering.</p>"
	srv, cleanup := startServer(map[string]string{"/wiki/Robotics": wikiHTML("Robotics", para)})
	defer cleanup()

	pages := crawler.CrawlURLs([]string{srv.URL + "/wiki/Robotics"}, zeroDelayConfig())

	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if pages[0].Title != "Robotics" {
		t.Errorf("title: got %q, want %q", pages[0].Title, "Robotics")
	}
	if !strings.Contains(pages[0].Body, "Robotics") {
		t.Errorf("body missing expected keyword; body=%q", pages[0].Body)
	}
}

func TestCrawlURLs_CrawlsMultiplePagesConcurrently(t *testing.T) {
	para := "<p>This is a substantive paragraph with enough words to pass the word-count filter.</p>"
	routes := map[string]string{
		"/wiki/Robot":    wikiHTML("Robot", para),
		"/wiki/Chatbot":  wikiHTML("Chatbot", para),
		"/wiki/Robotics": wikiHTML("Robotics", para),
	}
	srv, cleanup := startServer(routes)
	defer cleanup()

	urls := []string{
		srv.URL + "/wiki/Robot",
		srv.URL + "/wiki/Chatbot",
		srv.URL + "/wiki/Robotics",
	}

	start := time.Now()
	pages := crawler.CrawlURLs(urls, zeroDelayConfig())
	elapsed := time.Since(start)

	if len(pages) != 3 {
		t.Errorf("expected 3 pages, got %d", len(pages))
	}
	// With parallelism=3 and zero delay, three local pages must finish fast.
	if elapsed > 5*time.Second {
		t.Errorf("crawl took %v – expected concurrent execution to finish faster", elapsed)
	}
}

func TestCrawlURLs_FiltersShortParagraphs(t *testing.T) {
	html := wikiHTML("TestPage",
		`<p>Too short.</p>`+
			`<p>This is a substantive paragraph with enough words to pass the filter.</p>`)
	srv, cleanup := startServer(map[string]string{"/wiki/TestPage": html})
	defer cleanup()

	pages := crawler.CrawlURLs([]string{srv.URL + "/wiki/TestPage"}, zeroDelayConfig())

	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if strings.Contains(pages[0].Body, "Too short") {
		t.Error("short paragraph should have been filtered from body")
	}
}

func TestCrawlURLs_EmptyInputReturnsEmptySlice(t *testing.T) {
	pages := crawler.CrawlURLs([]string{}, crawler.DefaultConfig())
	if len(pages) != 0 {
		t.Errorf("expected 0 pages for empty input, got %d", len(pages))
	}
}

func TestCrawlURLs_HTTP404PageIsSkipped(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/missing", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	pages := crawler.CrawlURLs([]string{srv.URL + "/missing"}, zeroDelayConfig())
	if len(pages) != 0 {
		t.Errorf("404 page should be skipped; got %d pages", len(pages))
	}
}
