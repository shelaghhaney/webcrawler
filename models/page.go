// Package models defines shared data structures used across the webcrawler application.
package models

// Page holds the scraped content extracted from a single Wikipedia page.
// It mirrors the structure written to each JSON line in the output .jl file.
type Page struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Body  string `json:"body"`
}
