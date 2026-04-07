package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/publio/mectlivros/internal/cache"
	"github.com/publio/mectlivros/internal/downloader"
	"github.com/publio/mectlivros/internal/epub"
	"github.com/publio/mectlivros/pkg/models"
)

const operationTimeout = 10 * time.Minute

func main() {
	cyan := color.New(color.FgCyan).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	if err := os.MkdirAll("epubs", 0755); err != nil {
		fmt.Fprintln(os.Stderr, red("error: cannot create epubs/ — check write permissions"))
		os.Exit(1)
	}

	cacheManager := cache.New()
	jwtToken := cacheManager.Get()
	usedCache := jwtToken != ""

	if jwtToken == "" {
		fmt.Println("JWT token:")
		fmt.Println("(DevTools > Application > Local Storage > token)")
		fmt.Print("> ")
		fmt.Scanln(&jwtToken)
		jwtToken = strings.TrimSpace(jwtToken)

		if jwtToken == "" {
			fmt.Fprintln(os.Stderr, "error: token is required")
			os.Exit(1)
		}
		fmt.Println()
	}

	client := downloader.NewClient(jwtToken, "")

	rentals, err := client.FetchRentals(ctx)
	if err != nil {
		if errors.Is(err, downloader.ErrUnauthorized) {
			fmt.Fprintln(os.Stderr, red("error: token expired — get a new one from DevTools > Application > Local Storage, then: rm ~/.mec_livros_token"))
		} else {
			fmt.Fprintf(os.Stderr, red("error: %v\n"), err)
		}
		os.Exit(1)
	}
	if len(rentals) == 0 {
		fmt.Fprintln(os.Stderr, "no active rentals — check meclivros.mec.gov.br for active loans")
		os.Exit(1)
	}

	fmt.Printf("%d ebook(s):\n", len(rentals))
	for i, r := range rentals {
		fmt.Printf("[%d] %s (ID: %d, %d dias restantes)\n",
			i+1, cyan(r.BookTitle), r.BookID, r.DaysRemaining)
	}

	var selected models.Rental
	switch len(rentals) {
	case 1:
		selected = rentals[0]
		fmt.Printf("Auto-selecionado: %s\n", selected.BookTitle)
	default:
		fmt.Printf("Selecione [1-%d]: ", len(rentals))
		var choice int
		fmt.Scanln(&choice)
		if choice < 1 || choice > len(rentals) {
			fmt.Println(red("Opção inválida"))
			os.Exit(1)
		}
		selected = rentals[choice-1]
	}

	bookID := strconv.Itoa(selected.BookID)

	safeTitle := sanitizeFilename(selected.BookTitle)
	safeAuthor := sanitizeFilename(selected.BookAuthor)

	outputName := cmp.Or(safeAuthor, "Autor")
	if outputName != "Autor" && outputName != "" {
		outputName = safeTitle + " - " + safeAuthor
	} else {
		outputName = safeTitle
	}

	client = downloader.NewClient(jwtToken, bookID)

	manifest, webpubBaseURL, err := client.FetchManifest(ctx, bookID)
	if err != nil {
		fmt.Fprintf(os.Stderr, red("error: could not fetch manifest: %v\n"), err)
		os.Exit(1)
	}

	fmt.Printf("%s - %s\n", cyan(manifest.Manifest.Metadata.Title), manifest.Manifest.Metadata.Author)
	fmt.Printf("Capítulos: %d | Recursos: %d\n",
		len(manifest.Manifest.ReadingOrder),
		len(manifest.Manifest.Resources))

	stats, chapters, cssFiles, fontFiles, imageFiles, err := client.DownloadAll(ctx, manifest, webpubBaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, red("error: download failed: %v\n"), err)
		os.Exit(1)
	}

	if stats.ChaptersSuccess == 0 {
		fmt.Fprintln(os.Stderr, red("error: no chapters downloaded — book may be DRM-protected"))
		os.Exit(1)
	}

	epubsPath := filepath.Join("epubs", outputName)
	builder := epub.NewBuilder(epubsPath)
	epubPath, err := builder.Build(manifest, chapters, cssFiles, fontFiles, imageFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, red("error: could not build EPUB: %v\n"), err)
		os.Exit(1)
	}

	info, _ := os.Stat(epubPath)
	sizeMB := float64(info.Size()) / 1024 / 1024

	fmt.Printf("%s (%.1f MB)\n", epubPath, sizeMB)
	fmt.Printf("Capítulos: %d/%d | CSS: %d | Fontes: %d | Imagens: %d\n",
		stats.ChaptersSuccess,
		len(manifest.Manifest.ReadingOrder),
		stats.CSSSuccess,
		stats.FontSuccess,
		stats.ImageSuccess)

	if !usedCache {
		if err := cacheManager.Save(jwtToken); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not save token cache: %v\n", err)
		}
	}
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "livro"
	}

	var b strings.Builder
	b.Grow(len(s))

	invalid := map[rune]bool{
		'<': true, '>': true, ':': true, '"': true,
		'/': true, '\\': true, '|': true, '?': true, '*': true,
	}

	for _, r := range s {
		if !invalid[r] {
			b.WriteRune(r)
		}
	}

	result := strings.TrimSpace(b.String())
	if len(result) > 50 {
		result = result[:50]
	}

	if result == "" {
		return "livro"
	}
	return result
}
