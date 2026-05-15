// Package crawler provides a concurrent web crawler and scraper built on the
// Colly framework (github.com/gocolly/colly).  Each target URL is visited via
// Colly's async collector; results are gathered through a channel and returned
// to the caller for streaming to disk.
package crawler

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/shelaghhaney/webcrawler/internal/models"
	"github.com/shelaghhaney/webcrawler/internal/parser"
)

// Config groups the tunable parameters for a crawl run.
type Config struct {
	// UserAgent is sent in every HTTP request so Wikipedia can identify the bot.
	UserAgent string

	// RequestDelay is the minimum pause between consecutive requests to the
	// same domain, keeping the crawler polite.
	RequestDelay time.Duration

	// Parallelism caps the maximum number of goroutines Colly uses at once.
	Parallelism int
}

// DefaultConfig returns a Config with sensible defaults for crawling Wikipedia.
func DefaultConfig() Config {
	return Config{
		UserAgent:    "WikiRoboticsBot/1.0 (educational web-crawl assignment)",
		RequestDelay: 500 * time.Millisecond,
		Parallelism:  5,
	}
}

// CrawlURLs visits every URL concurrently using a shared Colly collector,
// scrapes plain text from each Wikipedia article body, and returns the
// collected pages.  Order of results is not guaranteed.
func CrawlURLs(urls []string, cfg Config) []models.Page {
	if len(urls) == 0 {
		return []models.Page{}
	}

	// paragraphsByURL and titleByURL accumulate content as Colly fires
	// callbacks; a mutex guards both maps since callbacks run concurrently.
	paragraphsByURL := make(map[string][]string, len(urls))
	titleByURL      := make(map[string]string,   len(urls))
	var mapMu sync.Mutex

	c := buildCollector(cfg)

	// --- Colly callbacks -----------------------------------------------------

	// Extract the article title from the <h1 id="firstHeading"> element.
	c.OnHTML("h1#firstHeading", func(e *colly.HTMLElement) {
		url   := e.Request.URL.String()
		title := parser.CleanText(e.Text)
		mapMu.Lock()
		titleByURL[url] = title
		mapMu.Unlock()
	})

	// Extract each <p> paragraph inside the main article content area.
	c.OnHTML("#mw-content-text p", func(e *colly.HTMLElement) {
		text := parser.CleanText(e.Text)
		if !parser.IsUsefulParagraph(text) {
			return
		}
		url := e.Request.URL.String()
		mapMu.Lock()
		paragraphsByURL[url] = append(paragraphsByURL[url], text)
		mapMu.Unlock()
	})

	c.OnRequest(func(r *colly.Request) {
		log.Printf("[crawler] visiting %s", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("[crawler] error fetching %s – status %d – %v",
			r.Request.URL, r.StatusCode, err)
	})

	c.OnScraped(func(r *colly.Response) {
		url := r.Request.URL.String()
		mapMu.Lock()
		title      := titleByURL[url]
		paragraphs := paragraphsByURL[url]
		mapMu.Unlock()

		if len(paragraphs) == 0 {
			log.Printf("[crawler] no usable content at %s", url)
			return
		}
		log.Printf("[crawler] scraped %q – %d paragraphs", title, len(paragraphs))
	})

	// --- Queue every target URL then wait ------------------------------------
	for _, u := range urls {
		if err := c.Visit(u); err != nil {
			log.Printf("[crawler] could not queue %s – %v", u, err)
		}
	}
	c.Wait() // blocks until all async visits complete

	// --- Assemble results ----------------------------------------------------
	pages := make([]models.Page, 0, len(urls))
	for _, u := range urls {
		paragraphs, ok := paragraphsByURL[u]
		if !ok || len(paragraphs) == 0 {
			continue
		}
		pages = append(pages, models.Page{
			URL:   u,
			Title: titleByURL[u],
			Body:  strings.Join(paragraphs, "\n\n"),
		})
	}
	return pages
}

// buildCollector creates and configures a Colly async collector with the
// rate-limit and user-agent settings from cfg.
func buildCollector(cfg Config) *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent(cfg.UserAgent),
		colly.Async(true),
	)

	// Rate-limit to the Wikipedia domain so we stay polite.
	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*wikipedia.org*",
		Parallelism: cfg.Parallelism,
		Delay:       cfg.RequestDelay,
	}); err != nil {
		log.Printf("[crawler] warning: could not apply rate-limit rule – %v", err)
	}

	return c
}
