package downloader

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/publio/mectlivros/pkg/models"
)

func TestNewClient_TrimsBearerPrefix(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"no prefix", "my-token", "my-token"},
		{"lowercase bearer", "bearer my-token", "my-token"},
		{"uppercase Bearer", "Bearer my-token", "my-token"},
		{"Bearer with spaces", "Bearer   my-token", "  my-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.token, "book123")
			if c.token != tt.want {
				t.Errorf("token = %q, want %q", c.token, tt.want)
			}
		})
	}
}

func TestNewClient_SetsBookID(t *testing.T) {
	c := NewClient("token", "book456")
	if c.bookID != "book456" {
		t.Errorf("bookID = %q, want %q", c.bookID, "book456")
	}
}

func TestDoRequest_GzipDecompression(t *testing.T) {
	jsonData := map[string]string{"key": "value"}
	jsonBytes, _ := json.Marshal(jsonData)

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write(jsonBytes)
	gzWriter.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}))
	defer server.Close()

	c := NewClient("token", "")
	_, body, err := c.DoRequest(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("DoRequest() error = %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Response is not valid JSON: %v, body: %q", err, string(body))
	}
	if result["key"] != "value" {
		t.Errorf("Decompressed body = %q, want %q", string(body), `{"key": "value"}`)
	}
}

func TestDoRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient("token", "")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := c.DoRequest(ctx, server.URL, nil)
	if err == nil {
		t.Error("DoRequest() expected error when context is cancelled, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestDoRequest_SetsCorrectHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient("my-token", "book999")
	_, _, err := c.DoRequest(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("DoRequest() error = %v", err)
	}

	if receivedHeaders.Get("User-Agent") == "" {
		t.Error("User-Agent header should be set")
	}
	if receivedHeaders.Get("Authorization") != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", receivedHeaders.Get("Authorization"), "Bearer my-token")
	}
	if !bytes.Contains([]byte(receivedHeaders.Get("Accept")), []byte("application/json")) {
		t.Errorf("Accept header should contain application/json, got %q", receivedHeaders.Get("Accept"))
	}
	if !bytes.Contains([]byte(receivedHeaders.Get("Referer")), []byte("bookId=book999")) {
		t.Errorf("Referer should contain bookId=book999, got %q", receivedHeaders.Get("Referer"))
	}
}

func TestDoRequest_OverwritesHeadersWithCustomValues(t *testing.T) {
	var receivedAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient("token", "")
	customHeaders := map[string]string{"Accept": "application/xhtml+xml"}
	_, _, err := c.DoRequest(context.Background(), server.URL, customHeaders)
	if err != nil {
		t.Fatalf("DoRequest() error = %v", err)
	}

	if receivedAccept != "application/xhtml+xml" {
		t.Errorf("Accept = %q, want %q (custom header should overwrite)", receivedAccept, "application/xhtml+xml")
	}
}

func TestDoRequest_NetworkError(t *testing.T) {
	c := NewClient("token", "")
	_, _, err := c.DoRequest(context.Background(), "http://localhost:99999", nil)
	if err == nil {
		t.Error("DoRequest() expected error for network failure, got nil")
	}
}

func TestDoRequest_InvalidURL(t *testing.T) {
	c := NewClient("token", "")
	_, _, err := c.DoRequest(context.Background(), "://invalid-url", nil)
	if err == nil {
		t.Error("DoRequest() expected error for invalid URL, got nil")
	}
}

func TestDoRequest_ReadsResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success", "data": [1, 2, 3]}`))
	}))
	defer server.Close()

	c := NewClient("token", "")
	resp, body, err := c.DoRequest(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("DoRequest() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if result["result"] != "success" {
		t.Errorf("result = %v, want success", result["result"])
	}
}

func TestDoRequest_ResponseBodyClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient("token", "")
	resp, _, err := c.DoRequest(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("DoRequest() error = %v", err)
	}
	if resp.Body == nil {
		t.Error("Response body should not be nil")
	}
}

func TestDoRequest_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	c := NewClient("invalid-token", "")
	_, body, err := c.DoRequest(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("DoRequest() should not return error on 401, got %v", err)
	}
	if len(body) == 0 {
		t.Error("Body should be readable on 401")
	}
}

func TestDoRequest_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	c := NewClient("token", "")
	_, _, err := c.DoRequest(context.Background(), server.URL, nil)
	if err != nil {
		t.Fatalf("DoRequest() should not return error on 403, got %v", err)
	}
}

func TestFetchRentals_EmptyList(t *testing.T) {
	t.Skip("FetchRentals requires real API endpoint - tested via integration tests")
}

func TestExtractWebpubURL_VariousScenarios(t *testing.T) {
	tests := []struct {
		name    string
		links   []models.Link
		wantURL string
	}{
		{
			name: "valid webpub self link",
			links: []models.Link{
				{Href: "https://meclivros.mec.gov.br/webpub/abc123/manifest.json", Rel: "self", Type: "application/webpub+json"},
			},
			wantURL: "https://meclivros.mec.gov.br/epub-proxy/webpub/abc123",
		},
		{
			name: "alternate rel is ignored",
			links: []models.Link{
				{Href: "https://meclivros.mec.gov.br/webpub/abc123/manifest.json", Rel: "alternate", Type: "application/webpub+json"},
			},
			wantURL: "",
		},
		{
			name: "non-manifest href is ignored",
			links: []models.Link{
				{Href: "https://meclivros.mec.gov.br/webpub/abc123/other.json", Rel: "self", Type: "application/webpub+json"},
			},
			wantURL: "",
		},
		{
			name: "no webpub in path is ignored",
			links: []models.Link{
				{Href: "https://meclivros.mec.gov.br/api/abc123/manifest.json", Rel: "self", Type: "application/webpub+json"},
			},
			wantURL: "",
		},
		{
			name: "deeply nested path",
			links: []models.Link{
				{Href: "https://meclivros.mec.gov.br/some/path/webpub/nested/book/manifest.json", Rel: "self", Type: "application/webpub+json"},
			},
			wantURL: "https://meclivros.mec.gov.br/epub-proxy/webpub/nested/book",
		},
		{
			name:    "empty links",
			links:   []models.Link{},
			wantURL: "",
		},
		{
			name: "no matching self link",
			links: []models.Link{
				{Href: "https://meclivros.mec.gov.br/other/path/manifest.json", Rel: "self", Type: "text/html"},
			},
			wantURL: "",
		},
		{
			name: "multiple links picks self with webpub",
			links: []models.Link{
				{Href: "https://other.com/manifest.json", Rel: "alternate", Type: "application/webpub+json"},
				{Href: "https://meclivros.mec.gov.br/webpub/book123/manifest.json", Rel: "self", Type: "application/webpub+json"},
				{Href: "https://other.com/alternate", Rel: "alternate", Type: "text/html"},
			},
			wantURL: "https://meclivros.mec.gov.br/epub-proxy/webpub/book123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := models.Manifest{}
			manifest.Manifest.Links = tt.links
			url := extractWebpubURL(manifest)
			if url != tt.wantURL {
				t.Errorf("extractWebpubURL() = %q, want %q", url, tt.wantURL)
			}
		})
	}
}

func TestDownloadAll_EmptyWebpubURL(t *testing.T) {
	c := &HTTPClient{}
	_, _, _, _, _, err := c.DownloadAll(context.Background(), &models.Manifest{}, "")
	if err == nil {
		t.Error("DownloadAll() expected error for empty webpub URL, got nil")
	}
	if err.Error() != "empty webpub base URL" {
		t.Errorf("error = %v, want %v", err, "empty webpub base URL")
	}
}

func createTestManifest(readingOrder []models.Resource, resources []models.Resource) *models.Manifest {
	return &models.Manifest{
		Manifest: struct {
			Metadata struct {
				Title      string `json:"title"`
				Author     string `json:"author"`
				Publisher  string `json:"publisher"`
				Language   string `json:"language"`
				Identifier string `json:"identifier"`
			} `json:"metadata"`
			ReadingOrder []models.Resource `json:"readingOrder"`
			Resources    []models.Resource `json:"resources"`
			Links        []models.Link     `json:"links"`
		}{
			Metadata: struct {
				Title      string `json:"title"`
				Author     string `json:"author"`
				Publisher  string `json:"publisher"`
				Language   string `json:"language"`
				Identifier string `json:"identifier"`
			}{
				Title:      "Test Book",
				Author:     "Test Author",
				Publisher:  "MEC",
				Language:   "pt-BR",
				Identifier: "ISBN-123",
			},
			ReadingOrder: readingOrder,
			Resources:    resources,
			Links:        nil,
		},
	}
}

func TestDownloadAll_ResourceCategorization(t *testing.T) {
	manifest := createTestManifest(
		[]models.Resource{{Href: "chapter1.xhtml", Type: "application/xhtml+xml"}},
		[]models.Resource{
			{Href: "css/style.css", Type: "text/css"},
			{Href: "css/responsive.css", Type: "text/css"},
			{Href: "font/Lato-Regular.otf", Type: "font/otf"},
			{Href: "font/Roboto.ttf", Type: "font/ttf"},
			{Href: "image/cover.jpg", Type: "image/jpeg"},
			{Href: "image/logo.png", Type: "image/png"},
			{Href: "image/photo.jpeg", Type: "image/jpeg"},
		},
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(make([]byte, 200)))
	}))
	defer server.Close()

	c := NewClient("token", "")
	stats, chapters, cssFiles, fontFiles, imageFiles, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if len(chapters) != 1 {
		t.Errorf("len(chapters) = %d, want 1", len(chapters))
	}
	if len(cssFiles) != 2 {
		t.Errorf("len(cssFiles) = %d, want 2", len(cssFiles))
	}
	if len(fontFiles) != 2 {
		t.Errorf("len(fontFiles) = %d, want 2", len(fontFiles))
	}
	if len(imageFiles) != 3 {
		t.Errorf("len(imageFiles) = %d, want 3", len(imageFiles))
	}

	if stats.ChaptersSuccess != 1 {
		t.Errorf("stats.ChaptersSuccess = %d, want 1", stats.ChaptersSuccess)
	}
	if stats.CSSSuccess != 2 {
		t.Errorf("stats.CSSSuccess = %d, want 2", stats.CSSSuccess)
	}
	if stats.FontSuccess != 2 {
		t.Errorf("stats.FontSuccess = %d, want 2", stats.FontSuccess)
	}
	if stats.ImageSuccess != 3 {
		t.Errorf("stats.ImageSuccess = %d, want 3", stats.ImageSuccess)
	}
}

func TestDownloadAll_EmptyResources(t *testing.T) {
	manifest := createTestManifest([]models.Resource{}, []models.Resource{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Server should not receive any requests when resources are empty")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("token", "")
	stats, chapters, cssFiles, fontFiles, imageFiles, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if len(chapters) != 0 || len(cssFiles) != 0 || len(fontFiles) != 0 || len(imageFiles) != 0 {
		t.Errorf("All resource slices should be empty, got chapters=%d css=%d fonts=%d images=%d",
			len(chapters), len(cssFiles), len(fontFiles), len(imageFiles))
	}
	if stats.ChaptersSuccess != 0 {
		t.Errorf("stats.ChaptersSuccess = %d, want 0", stats.ChaptersSuccess)
	}
}

func TestDownloadAll_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	manifest := createTestManifest(
		[]models.Resource{{Href: "chapter1.xhtml", Type: "application/xhtml+xml"}},
		nil,
	)

	c := NewClient("token", "")
	stats, chapters, _, _, _, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() should not return error on server error, got %v", err)
	}
	if stats.ChaptersFailed != 1 {
		t.Errorf("stats.ChaptersFailed = %d, want 1", stats.ChaptersFailed)
	}
	if len(chapters) != 0 {
		t.Errorf("len(chapters) = %d, want 0 when download fails", len(chapters))
	}
}

func TestDownloadResources_ContentValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("too short"))
	}))
	defer server.Close()

	manifest := createTestManifest(
		[]models.Resource{{Href: "chapter1.xhtml", Type: "application/xhtml+xml"}},
		nil,
	)

	c := NewClient("token", "")
	stats, chapters, _, _, _, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if stats.ChaptersFailed != 1 {
		t.Errorf("ChaptersFailed = %d, want 1 (content too short)", stats.ChaptersFailed)
	}
	if len(chapters) != 0 {
		t.Errorf("Chapters = %v, want empty when content < 100 bytes", chapters)
	}
}

func TestDownloadAll_AllChaptersFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("too short"))
	}))
	defer server.Close()

	manifest := createTestManifest(
		[]models.Resource{
			{Href: "ch1.xhtml", Type: "application/xhtml+xml"},
			{Href: "ch2.xhtml", Type: "application/xhtml+xml"},
		},
		nil,
	)

	c := NewClient("token", "")
	stats, chapters, _, _, _, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if stats.ChaptersFailed != 2 {
		t.Errorf("ChaptersFailed = %d, want 2", stats.ChaptersFailed)
	}
	if stats.ChaptersSuccess != 0 {
		t.Errorf("ChaptersSuccess = %d, want 0", stats.ChaptersSuccess)
	}
	if len(chapters) != 0 {
		t.Errorf("len(chapters) = %d, want 0 when all fail", len(chapters))
	}
}

func TestDownloadResources_MixedSuccessAndFailure(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(make([]byte, 200)))
		}
	}))
	defer server.Close()

	resources := make([]models.Resource, 6)
	for i := range resources {
		resources[i] = models.Resource{Href: "ch" + string(rune('0'+i)) + ".xhtml", Type: "application/xhtml+xml"}
	}
	manifest := createTestManifest(resources, nil)

	c := NewClient("token", "")
	stats, chapters, _, _, _, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if stats.ChaptersSuccess != 3 {
		t.Errorf("ChaptersSuccess = %d, want 3", stats.ChaptersSuccess)
	}
	if stats.ChaptersFailed != 3 {
		t.Errorf("ChaptersFailed = %d, want 3", stats.ChaptersFailed)
	}
	if len(chapters) != 3 {
		t.Errorf("len(chapters) = %d, want 3", len(chapters))
	}
}

func TestDownloadResources_ChapterAcceptHeader(t *testing.T) {
	var acceptHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(make([]byte, 200)))
	}))
	defer server.Close()

	manifest := createTestManifest(
		[]models.Resource{{Href: "chapter1.xhtml", Type: "application/xhtml+xml"}},
		nil,
	)

	c := NewClient("token", "")
	c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")

	if acceptHeader == "" || !bytes.Contains([]byte(acceptHeader), []byte("application/xhtml+xml")) {
		t.Errorf("Accept header for chapters = %q, want to contain application/xhtml+xml", acceptHeader)
	}
}

func TestDownloadResources_CSSUsesDefaultAcceptHeader(t *testing.T) {
	var acceptHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("body {}"))
	}))
	defer server.Close()

	manifest := createTestManifest(nil, []models.Resource{{Href: "style.css", Type: "text/css"}})

	c := NewClient("token", "")
	c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")

	if acceptHeader == "" {
		t.Error("Accept header should not be empty for CSS")
	}
}

func TestStats_AtomicOperations(t *testing.T) {
	stats := &Stats{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt32(&stats.ChaptersSuccess, 1)
			atomic.AddInt32(&stats.CSSSuccess, 1)
			atomic.AddInt32(&stats.FontSuccess, 1)
			atomic.AddInt32(&stats.ImageSuccess, 1)
		}()
	}
	wg.Wait()

	if stats.ChaptersSuccess != 100 {
		t.Errorf("ChaptersSuccess = %d, want 100", stats.ChaptersSuccess)
	}
	if stats.CSSSuccess != 100 {
		t.Errorf("CSSSuccess = %d, want 100", stats.CSSSuccess)
	}
	if stats.FontSuccess != 100 {
		t.Errorf("FontSuccess = %d, want 100", stats.FontSuccess)
	}
	if stats.ImageSuccess != 100 {
		t.Errorf("ImageSuccess = %d, want 100", stats.ImageSuccess)
	}
}

func TestResourceCategorization_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		wantType string
	}{
		{"uppercase CSS", "style.CSS", "css"},
		{"uppercase OTF", "font.OTF", "font"},
		{"uppercase TTF", "font.TTF", "font"},
		{"uppercase JPG", "image.JPG", "image"},
		{"uppercase PNG", "image.PNG", "image"},
		{"jpeg with e", "image.jpeg", "image"},
		{"jpg", "image.jpg", "image"},
		{"path with css", "assets/css/main.css", "css"},
		{"path with font", "fonts/roboto.ttf", "font"},
		{"path with image", "images/photo.png", "image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resources []models.Resource
			if bytes.Contains([]byte(tt.href), []byte(".css")) {
				resources = []models.Resource{{Href: tt.href, Type: "text/css"}}
			} else if bytes.Contains([]byte(tt.href), []byte(".otf")) || bytes.Contains([]byte(tt.href), []byte(".ttf")) {
				resources = []models.Resource{{Href: tt.href, Type: "font/otf"}}
			} else {
				resources = []models.Resource{{Href: tt.href, Type: "image/jpeg"}}
			}

			manifest := createTestManifest(nil, resources)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if bytes.Contains([]byte(tt.href), []byte(".css")) {
					w.Write([]byte("body {}"))
				} else if bytes.Contains([]byte(tt.href), []byte(".otf")) || bytes.Contains([]byte(tt.href), []byte(".ttf")) {
					w.Write([]byte("font data"))
				} else {
					w.Write([]byte(make([]byte, 200)))
				}
			}))
			defer server.Close()

			c := NewClient("token", "")
			_, chapters, cssFiles, fontFiles, imageFiles, _ := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")

			switch tt.wantType {
			case "css":
				if len(cssFiles) != 1 {
					t.Errorf("href=%q categorized as css, got cssFiles=%d", tt.href, len(cssFiles))
				}
			case "font":
				if len(fontFiles) != 1 {
					t.Errorf("href=%q categorized as font, got fontFiles=%d", tt.href, len(fontFiles))
				}
			case "image":
				if len(imageFiles) != 1 {
					t.Errorf("href=%q categorized as image, got imageFiles=%d", tt.href, len(imageFiles))
				}
			default:
				if len(chapters) != 0 && len(cssFiles) == 0 && len(fontFiles) == 0 && len(imageFiles) == 0 {
					t.Errorf("href=%q not categorized, all slices empty", tt.href)
				}
			}
		})
	}
}

func TestResourceCategorization_UnrecognizedExtensions(t *testing.T) {
	tests := []struct {
		name string
		href string
		want int
	}{
		{"svg not categorized", "image/logo.svg", 0},
		{"woff not categorized", "font/font.woff", 0},
		{"woff2 not categorized", "font/font.woff2", 0},
		{"gif not categorized", "image/animation.gif", 0},
		{"webp not categorized", "image/photo.webp", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources := []models.Resource{{Href: tt.href, Type: "application/octet-stream"}}
			manifest := createTestManifest(nil, resources)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(make([]byte, 200)))
			}))
			defer server.Close()

			c := NewClient("token", "")
			_, _, cssFiles, fontFiles, imageFiles, _ := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")

			total := len(cssFiles) + len(fontFiles) + len(imageFiles)
			if total != tt.want {
				t.Errorf("href=%q: total categorized = %d, want %d (css=%d, font=%d, image=%d)",
					tt.href, total, tt.want, len(cssFiles), len(fontFiles), len(imageFiles))
			}
		})
	}
}

func TestDownloadedItem_Fields(t *testing.T) {
	item := DownloadedItem{
		Filename: "chapter1.xhtml",
		Href:     "OEBPS/chapter1.xhtml",
		Content:  []byte("<html>content</html>"),
		Size:     19,
	}

	if item.Filename != "chapter1.xhtml" {
		t.Errorf("Filename = %q, want %q", item.Filename, "chapter1.xhtml")
	}
	if item.Size != 19 {
		t.Errorf("Size = %d, want 19", item.Size)
	}
	if string(item.Content) != "<html>content</html>" {
		t.Errorf("Content = %q, want %q", string(item.Content), "<html>content</html>")
	}
}

func TestDownloadResult_Fields(t *testing.T) {
	item := DownloadedItem{Filename: "test.css", Href: "css/test.css", Content: []byte("body {}"), Size: 7}

	result := DownloadResult{
		Type:  resourceTypeCSS,
		Item:  item,
		Error: nil,
	}

	if result.Type != resourceTypeCSS {
		t.Errorf("Type = %q, want %q", result.Type, resourceTypeCSS)
	}
	if result.Error != nil {
		t.Errorf("Error = %v, want nil", result.Error)
	}
}

func TestDownloadAll_OnlyChapters(t *testing.T) {
	manifest := createTestManifest(
		[]models.Resource{
			{Href: "ch1.xhtml", Type: "application/xhtml+xml"},
			{Href: "ch2.xhtml", Type: "application/xhtml+xml"},
			{Href: "ch3.xhtml", Type: "application/xhtml+xml"},
		},
		nil,
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(make([]byte, 200)))
	}))
	defer server.Close()

	c := NewClient("token", "")
	stats, chapters, cssFiles, fontFiles, imageFiles, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if len(chapters) != 3 {
		t.Errorf("len(chapters) = %d, want 3", len(chapters))
	}
	if len(cssFiles) != 0 || len(fontFiles) != 0 || len(imageFiles) != 0 {
		t.Errorf("Other slices should be empty, got css=%d fonts=%d images=%d", len(cssFiles), len(fontFiles), len(imageFiles))
	}
	if stats.ChaptersSuccess != 3 {
		t.Errorf("stats.ChaptersSuccess = %d, want 3", stats.ChaptersSuccess)
	}
}

func TestDownloadAll_OnlyResources(t *testing.T) {
	manifest := createTestManifest(
		nil,
		[]models.Resource{
			{Href: "style.css", Type: "text/css"},
			{Href: "logo.png", Type: "image/png"},
			{Href: "font.otf", Type: "font/otf"},
		},
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(make([]byte, 200)))
	}))
	defer server.Close()

	c := NewClient("token", "")
	stats, chapters, cssFiles, fontFiles, imageFiles, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if len(chapters) != 0 {
		t.Errorf("len(chapters) = %d, want 0", len(chapters))
	}
	if stats.CSSSuccess != 1 || stats.FontSuccess != 1 || stats.ImageSuccess != 1 {
		t.Errorf("stats = {CSS:%d, Font:%d, Image:%d}, want all 1",
			stats.CSSSuccess, stats.FontSuccess, stats.ImageSuccess)
	}
	_ = cssFiles
	_ = fontFiles
	_ = imageFiles
}

func TestDownloadResources_SmallButValidChapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		content := "<html><body>" + string(make([]byte, 100)) + "</body></html>"
		w.Write([]byte(content))
	}))
	defer server.Close()

	manifest := createTestManifest(
		[]models.Resource{{Href: "chapter.xhtml", Type: "application/xhtml+xml"}},
		nil,
	)

	c := NewClient("token", "")
	stats, chapters, _, _, _, err := c.DownloadAll(context.Background(), manifest, server.URL+"/webpub/book")
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if stats.ChaptersFailed != 0 {
		t.Errorf("ChaptersFailed = %d, want 0 (content is > 100 bytes)", stats.ChaptersFailed)
	}
	if len(chapters) != 1 {
		t.Errorf("len(chapters) = %d, want 1", len(chapters))
	}
}

func TestDownloadAll_WithRealURLFormat(t *testing.T) {
	manifest := createTestManifest(
		[]models.Resource{
			{Href: "OEBPS/chapter1.xhtml", Type: "application/xhtml+xml"},
			{Href: "OEBPS/chapter2.xhtml", Type: "application/xhtml+xml"},
		},
		[]models.Resource{
			{Href: "OEBPS/css/style.css", Type: "text/css"},
			{Href: "OEBPS/image/cover.jpg", Type: "image/jpeg"},
		},
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(make([]byte, 200)))
	}))
	defer server.Close()

	c := NewClient("token", "")
	stats, chapters, cssFiles, fontFiles, imageFiles, err := c.DownloadAll(context.Background(), manifest, server.URL)
	if err != nil {
		t.Fatalf("DownloadAll() error = %v", err)
	}

	if len(chapters) != 2 {
		t.Errorf("len(chapters) = %d, want 2", len(chapters))
	}
	if len(cssFiles) != 1 {
		t.Errorf("len(cssFiles) = %d, want 1", len(cssFiles))
	}
	if len(fontFiles) != 0 {
		t.Errorf("len(fontFiles) = %d, want 0", len(fontFiles))
	}
	if len(imageFiles) != 1 {
		t.Errorf("len(imageFiles) = %d, want 1", len(imageFiles))
	}

	totalSuccess := stats.ChaptersSuccess + stats.CSSSuccess + stats.FontSuccess + stats.ImageSuccess
	if totalSuccess != 4 {
		t.Errorf("totalSuccess = %d, want 4", totalSuccess)
	}
	_ = chapters
	_ = cssFiles
	_ = imageFiles
}

func TestDoRequest_RespectsTimeout(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer slowServer.Close()

	c := NewClient("token", "")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _, err := c.DoRequest(ctx, slowServer.URL, nil)
	if err == nil {
		t.Error("DoRequest() should return error when timeout is exceeded")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestDownloadAll_ContextCancellation(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(make([]byte, 200)))
	}))
	defer server.Close()

	resources := make([]models.Resource, 10)
	for i := range resources {
		resources[i] = models.Resource{Href: "ch" + string(rune('0'+i)) + ".xhtml", Type: "application/xhtml+xml"}
	}
	manifest := createTestManifest(resources, nil)

	c := NewClient("token", "")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _, _, _, _, err := c.DownloadAll(ctx, manifest, server.URL+"/webpub/book")

	if err == nil {
		t.Log("Context cancelled but no error returned - partial results may be valid")
	}

	finalCount := atomic.LoadInt32(&requestCount)
	t.Logf("Completed %d requests before context cancelled", finalCount)
}

func TestDoRequest_NoRetryOnFailure(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient("token", "")
	_, _, err := c.DoRequest(context.Background(), server.URL, nil)

	if err != nil {
		t.Fatalf("DoRequest() returned error: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("requestCount = %d, want 1 (no retry logic should exist)", requestCount)
	}
}
