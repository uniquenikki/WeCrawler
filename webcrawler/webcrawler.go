package webcrawler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
)

// Crawler holds the configuration and state of the crawler.
type Crawler struct {
	Domains     []string
	RateLimit   time.Duration
	Concurrency int
	ProductURLs map[string][]string
	mu          sync.Mutex
	semaphore   chan struct{}
}

// NewCrawler creates a new instance of the Crawler.
func CreateNewCrawler(domains []string, rateLimit time.Duration, concurrency int) *Crawler {
	return &Crawler{
		Domains:     domains,
		RateLimit:   rateLimit,
		Concurrency: concurrency,
		ProductURLs: make(map[string][]string),
		semaphore:   make(chan struct{}, concurrency),
	}
}

// IsProductURL determines if a URL is likely a product page.
func (c *Crawler) IsProductURL(link string) bool {
	productRegex := regexp.MustCompile(`/product/|/item/|/p/|/dp/`)
	return productRegex.MatchString(link)
}

// Fetch fetches the HTML content of a given URL.
func (c *Crawler) Fetch(link string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(link)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch %s: HTTP %d", link, resp.StatusCode)
	}

	body, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	html, _ := body.Html()
	return html, nil
}

// ParseLinks extracts all valid links from the HTML content.
func (c *Crawler) ParseLinks(html, baseURL string) ([]string, error) {
	links := []string{}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			absoluteURL := resolveURL(baseURL, href)
			if isSameDomain(baseURL, absoluteURL) {
				links = append(links, absoluteURL)
			}
		}
	})

	return links, nil
}

// CrawlDomain discovers product URLs for a single domain.
func (c *Crawler) CrawlDomain(domain string) {
	baseURL := "https://" + domain
	visited := make(map[string]bool)
	toVisit := []string{baseURL}

	for len(toVisit) > 0 {
		c.semaphore <- struct{}{}
		currentURL := toVisit[0]
		toVisit = toVisit[1:]

		if visited[currentURL] {
			<-c.semaphore
			continue
		}
		visited[currentURL] = true

		log.Printf("Crawling: %s\n", currentURL)
		html, err := c.Fetch(currentURL)
		if err != nil {
			log.Printf("Error fetching %s: %v\n", currentURL, err)
			<-c.semaphore
			continue
		}

		links, err := c.ParseLinks(html, baseURL)
		if err != nil {
			log.Printf("Error parsing links on %s: %v\n", currentURL, err)
			<-c.semaphore
			continue
		}

		for _, link := range links {
			if c.IsProductURL(link) {
				c.mu.Lock()
				c.ProductURLs[domain] = append(c.ProductURLs[domain], link)
				c.mu.Unlock()
			} else if !visited[link] {
				toVisit = append(toVisit, link)
			}
		}
		time.Sleep(c.RateLimit)
		<-c.semaphore
	}
}

// SaveResults saves the crawled product URLs to a JSON file.
func (c *Crawler) SaveResults(filename string) {
	file, err := json.MarshalIndent(c.ProductURLs, "", "  ")
	if err != nil {
		log.Fatalf("Error saving results: %v", err)
	}

	err = writeFile(filename, file)
	if err != nil {
		log.Fatalf("Error writing file: %v", err)
	}
	log.Printf("Results saved to %s\n", filename)
}

// Run executes the crawler across all domains.
func (c *Crawler) RunCrawler() {
	wg := sync.WaitGroup{}
	for _, domain := range c.Domains {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			c.CrawlDomain(domain)
		}(domain)
	}
	wg.Wait()
}

func WebCrawler(c *gin.Context) {
	domains := []string{"www.aliexpress.com"}
	crawler := CreateNewCrawler(domains, 10*time.Millisecond, 50)
	crawler.RunCrawler()
	crawler.SaveResults("product_urls.json")
	filename := "product_urls.json"
	c.File(filename)
}

// resolveURL resolves a relative URL to an absolute URL.
func resolveURL(baseURL, href string) string {
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	parsedHref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return parsedBase.ResolveReference(parsedHref).String()
}

// isSameDomain checks if two URLs belong to the same domain.
func isSameDomain(baseURL, href string) bool {
	baseDomain, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	hrefDomain, err := url.Parse(href)
	if err != nil {
		return false
	}
	return baseDomain.Host == hrefDomain.Host
}

// writeFile writes data to a file.
func writeFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}
