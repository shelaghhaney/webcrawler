// Command crawler fetches Wikipedia articles on intelligent systems and
// robotics, strips all HTML markup, and writes the plain-text results to a
// JSON Lines (.jl) file – one JSON object per line.
//
// Usage:
//
//	./crawler [-out results.jl] [-cpuprofile cpu.prof] [-memprofile mem.prof]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/shelaghhaney/webcrawler/internal/crawler"
	"github.com/shelaghhaney/webcrawler/internal/models"
)

// targetURLs is the canonical list of Wikipedia pages for the firm's research
// interests in intelligent systems and robotics.
var targetURLs = []string{
	"https://en.wikipedia.org/wiki/Robotics",
	"https://en.wikipedia.org/wiki/Robot",
	"https://en.wikipedia.org/wiki/Reinforcement_learning",
	"https://en.wikipedia.org/wiki/Robot_Operating_System",
	"https://en.wikipedia.org/wiki/Intelligent_agent",
	"https://en.wikipedia.org/wiki/Software_agent",
	"https://en.wikipedia.org/wiki/Robotic_process_automation",
	"https://en.wikipedia.org/wiki/Chatbot",
	"https://en.wikipedia.org/wiki/Applications_of_artificial_intelligence",
	"https://en.wikipedia.org/wiki/Android_(robot)",
}

func main() {
	outputPath  := flag.String("out",        "results.jl", "path for the JSON Lines output file")
	cpuProfPath := flag.String("cpuprofile", "",           "write CPU profile to file (optional)")
	memProfPath := flag.String("memprofile", "",           "write memory profile to file (optional)")
	flag.Parse()

	// --- Optional CPU profiling ---------------------------------------------
	if *cpuProfPath != "" {
		f, err := os.Create(*cpuProfPath)
		if err != nil {
			log.Fatalf("[main] cannot create CPU profile file: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("[main] cannot start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	// --- File-based logging -------------------------------------------------
	logFile, err := os.OpenFile("crawler.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		log.Fatalf("[main] cannot open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	fmt.Println("=== Wikipedia Robotics & AI Web Crawler ===")
	fmt.Printf("Target pages : %d\n", len(targetURLs))
	fmt.Printf("Output file  : %s\n\n", *outputPath)

	// --- Open output file ---------------------------------------------------
	outFile, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("[main] cannot create output file %q: %v", *outputPath, err)
	}
	defer outFile.Close()

	// --- Run the crawl ------------------------------------------------------
	cfg := crawler.DefaultConfig()
	log.Printf("[main] starting crawl of %d URLs with parallelism=%d", len(targetURLs), cfg.Parallelism)

	start := time.Now()
	pages := crawler.CrawlURLs(targetURLs, cfg)
	elapsed := time.Since(start)

	// --- Write JSON Lines output --------------------------------------------
	enc := json.NewEncoder(outFile)
	if err := writeJSONLines(enc, pages); err != nil {
		log.Fatalf("[main] output write error: %v", err)
	}

	// --- Print summary ------------------------------------------------------
	fmt.Printf("Pages scraped : %d / %d\n", len(pages), len(targetURLs))
	fmt.Printf("Elapsed time  : %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Output file   : %s\n", *outputPath)
	fmt.Printf("Log file      : crawler.log\n")
	log.Printf("[main] complete – %d pages scraped in %s", len(pages), elapsed)

	// --- Optional memory profiling ------------------------------------------
	if *memProfPath != "" {
		f, err := os.Create(*memProfPath)
		if err != nil {
			log.Printf("[main] cannot create memory profile file: %v", err)
		} else {
			defer f.Close()
			runtime.GC()
			if profErr := pprof.WriteHeapProfile(f); profErr != nil {
				log.Printf("[main] cannot write memory profile: %v", profErr)
			}
			fmt.Printf("Memory profile: %s\n", *memProfPath)
		}
	}
}

// writeJSONLines encodes each Page as a single JSON object on its own line,
// producing a valid NDJSON file.
func writeJSONLines(enc *json.Encoder, pages []models.Page) error {
	for _, p := range pages {
		if err := enc.Encode(p); err != nil {
			return fmt.Errorf("encoding page %q: %w", p.URL, err)
		}
	}
	return nil
}
