package downloader

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/publio/mectlivros/pkg/models"
)

const (
	baseURL    = "https://meclivros.mec.gov.br"
	maxWorkers = 8
	timeout    = 30 * time.Second

	resourceTypeChapter = "chapter"
	resourceTypeCSS     = "css"
	resourceTypeFont    = "font"
	resourceTypeImage   = "image"
)

// HTTPClient wraps an http.Client with proper configuration
type HTTPClient struct {
	client *http.Client
	token  string
	bookID string
}

// NewClient creates a new HTTP client for MeC Livros API
func NewClient(token, bookID string) *HTTPClient {
	cleanToken := strings.TrimPrefix(strings.TrimPrefix(token, "Bearer "), "bearer ")

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &HTTPClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		token:  cleanToken,
		bookID: bookID,
	}
}

// DoRequest performs an HTTP request with retries
func (c *HTTPClient) DoRequest(ctx context.Context, url string, headers map[string]string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json,application/xhtml+xml,text/html,*/*")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Referer", "https://meclivros.mec.gov.br/leitura?bookId="+c.bookID)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("read body: %w", err)
	}

	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(strings.NewReader(string(body)))
		if err == nil {
			defer reader.Close()
			decompressed, err := io.ReadAll(reader)
			if err == nil {
				body = decompressed
			}
		}
	}

	return resp, body, nil
}

// FetchRentals retrieves rented books from the API
func (c *HTTPClient) FetchRentals(ctx context.Context) ([]models.Rental, error) {
	url := baseURL + "/api/backend/rentals?page=1&limit=50&status=active"

	_, body, err := c.DoRequest(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch rentals: %w", err)
	}

	var resp models.RentalsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal rentals: %w", err)
	}

	return resp.Rentals, nil
}

// FetchManifest retrieves the book manifest and extracts webpub URL
func (c *HTTPClient) FetchManifest(ctx context.Context, bookID string) (*models.Manifest, string, error) {
	url := baseURL + "/api/backend/books/" + bookID + "/manifest"

	_, body, err := c.DoRequest(ctx, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("fetch manifest: %w", err)
	}

	var manifest models.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, "", fmt.Errorf("unmarshal manifest: %w", err)
	}

	webpubBaseURL := extractWebpubURL(manifest)

	return &manifest, webpubBaseURL, nil
}

func extractWebpubURL(manifest models.Manifest) string {
	for _, link := range manifest.Manifest.Links {
		if link.Rel == "self" && strings.Contains(link.Href, "manifest.json") {
			if idx := strings.Index(link.Href, "/webpub/"); idx != -1 {
				pathPart := strings.TrimSuffix(link.Href[idx+8:], "/manifest.json")
				return baseURL + "/epub-proxy/webpub/" + pathPart
			}
		}
	}
	return ""
}

// DownloadedItem represents a successfully downloaded resource
type DownloadedItem struct {
	Filename string
	Href     string
	Content  []byte
	Size     int
}

// Stats tracks download statistics
type Stats struct {
	ChaptersSuccess int32
	ChaptersFailed  int32
	CSSSuccess      int32
	FontSuccess     int32
	ImageSuccess    int32
}

// DownloadResult is the result of a download operation
type DownloadResult struct {
	Type  string
	Item  DownloadedItem
	Error error
}

// DownloadAll downloads all resources in parallel
func (c *HTTPClient) DownloadAll(ctx context.Context, manifest *models.Manifest, webpubBaseURL string) (*Stats, []DownloadedItem, []DownloadedItem, []DownloadedItem, []DownloadedItem, error) {
	if webpubBaseURL == "" {
		return nil, nil, nil, nil, nil, errors.New("empty webpub base URL")
	}

	stats := &Stats{}

	chapters, err := c.downloadResources(ctx, manifest.Manifest.ReadingOrder, webpubBaseURL, resourceTypeChapter, stats)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("download chapters: %w", err)
	}

	// Categorize resources
	var cssResources, fontResources, imageResources []models.Resource
	for _, r := range manifest.Manifest.Resources {
		href := strings.ToLower(r.Href)
		switch {
		case strings.Contains(href, ".css"):
			cssResources = append(cssResources, r)
		case strings.Contains(href, ".otf"), strings.Contains(href, ".ttf"):
			fontResources = append(fontResources, r)
		case strings.Contains(href, ".jpg"), strings.Contains(href, ".jpeg"), strings.Contains(href, ".png"):
			imageResources = append(imageResources, r)
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	var cssFiles, fontFiles, imageFiles []DownloadedItem

	wg.Go(func() {
		var err error
		cssFiles, err = c.downloadResources(ctx, cssResources, webpubBaseURL, resourceTypeCSS, stats)
		if err != nil {
			errChan <- fmt.Errorf("download CSS: %w", err)
		}
	})

	wg.Go(func() {
		var err error
		fontFiles, err = c.downloadResources(ctx, fontResources, webpubBaseURL, resourceTypeFont, stats)
		if err != nil {
			errChan <- fmt.Errorf("download fonts: %w", err)
		}
	})

	wg.Go(func() {
		var err error
		imageFiles, err = c.downloadResources(ctx, imageResources, webpubBaseURL, resourceTypeImage, stats)
		if err != nil {
			errChan <- fmt.Errorf("download images: %w", err)
		}
	})

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	return stats, chapters, cssFiles, fontFiles, imageFiles, nil
}

func (c *HTTPClient) downloadResources(ctx context.Context, resources []models.Resource, webpubBaseURL, resourceType string, stats *Stats) ([]DownloadedItem, error) {
	if len(resources) == 0 {
		return nil, nil
	}

	results := make([]DownloadedItem, 0, len(resources))
	resultChan := make(chan DownloadedItem, len(resources))

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	for _, resource := range resources {
		wg.Add(1)
		go func(href string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			url := webpubBaseURL + "/" + href
			filename := path.Base(href)

			headers := map[string]string{}
			if resourceType == resourceTypeChapter {
				headers["Accept"] = "application/xhtml+xml,text/html"
			}

			_, body, err := c.DoRequest(ctx, url, headers)
			if err != nil {
				atomic.AddInt32(&stats.ChaptersFailed, 1)
				return
			}

			// Validate content
			if resourceType == resourceTypeChapter && len(body) < 100 {
				atomic.AddInt32(&stats.ChaptersFailed, 1)
				return
			}

			resultChan <- DownloadedItem{
				Filename: filename,
				Href:     href,
				Content:  body,
				Size:     len(body),
			}
		}(resource.Href)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for item := range resultChan {
		results = append(results, item)
		switch resourceType {
		case resourceTypeChapter:
			atomic.AddInt32(&stats.ChaptersSuccess, 1)
		case resourceTypeCSS:
			atomic.AddInt32(&stats.CSSSuccess, 1)
		case resourceTypeFont:
			atomic.AddInt32(&stats.FontSuccess, 1)
		case resourceTypeImage:
			atomic.AddInt32(&stats.ImageSuccess, 1)
		}
	}

	return results, nil
}
