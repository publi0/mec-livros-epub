package main

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
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
	// Only log warnings and errors
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	slog.SetDefault(logger)

	magenta := color.New(color.FgMagenta, color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Println(magenta("============================================"))
	fmt.Println(magenta("📚 MEC LIVROS - EBOOK DOWNLOADER (Go 1.26)"))
	fmt.Println(magenta("============================================"))
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	if err := os.MkdirAll("epubs", 0755); err != nil {
		slog.Error("failed to create epubs directory", "error", err)
		fmt.Printf("%s\n", red("❌ Erro ao criar pasta epubs"))
		os.Exit(1)
	}

	cacheManager := cache.New()
	cachedToken := cacheManager.Get()

	var jwtToken string

	switch {
	case cachedToken != "":
		fmt.Println(yellow("🔐 Token em cache encontrado"))
		fmt.Printf("   Token: %s...\n\n", cachedToken[:min(30, len(cachedToken))])
	default:
		fmt.Println(cyan("🔐 Informe o JWT Token:"))
		fmt.Println("   (No navegador: DevTools > Application > Local Storage)")
		fmt.Print("   Token: ")
		fmt.Scanln(&jwtToken)
		jwtToken = strings.TrimSpace(jwtToken)

		if jwtToken == "" {
			fmt.Println(red("\n❌ Token obrigatório"))
			os.Exit(1)
		}
		fmt.Println()
	}

	client := downloader.NewClient(jwtToken, "")

	slog.Info("fetching rentals")
	rentals, err := client.FetchRentals(ctx)
	if err != nil || len(rentals) == 0 {
		slog.Error("no rentals found", "error", err)
		fmt.Printf("%s\n", red("\n❌ Nenhum ebook encontrado ou token inválido"))
		os.Exit(1)
	}

	slog.Info("rentals found", "count", len(rentals))

	fmt.Printf("\n%s📚 Encontrados %d ebook(s):%s\n\n", green(""), len(rentals), "")
	for i, r := range rentals {
		fmt.Printf("   [%d] %s (ID: %d, %d dias restantes)\n",
			i+1, cyan(r.BookTitle), r.BookID, r.DaysRemaining)
	}

	var selected models.Rental
	switch len(rentals) {
	case 1:
		selected = rentals[0]
		fmt.Printf("\n%s🎯 Auto-selecionado: %s%s\n", green(""), selected.BookTitle, "")
	default:
		fmt.Printf("\n%sSelecione [1-%d]: %s", cyan(""), len(rentals), "")
		var choice int
		fmt.Scanln(&choice)
		if choice < 1 || choice > len(rentals) {
			fmt.Println(red("❌ Opção inválida"))
			os.Exit(1)
		}
		selected = rentals[choice-1]
	}

	slog.Info("selected book",
		"id", selected.BookID,
		"title", selected.BookTitle,
		"author", selected.BookAuthor,
	)

	bookID := strconv.Itoa(selected.BookID)

	// Output filename with proper sanitization
	safeTitle := sanitizeFilename(selected.BookTitle)
	safeAuthor := sanitizeFilename(selected.BookAuthor)

	outputName := cmp.Or(safeAuthor, "Autor")
	if outputName != "Autor" && outputName != "" {
		outputName = safeTitle + " - " + safeAuthor
	} else {
		outputName = safeTitle
	}

	fmt.Printf("\n%s📁 Saída: %s.epub%s\n", cyan(""), outputName, "")
	fmt.Printf("%s🚀 Iniciando download...%s\n\n", green(""), "")

	client = downloader.NewClient(jwtToken, bookID)

	slog.Info("fetching manifest", "book_id", bookID)
	manifest, webpubBaseURL, err := client.FetchManifest(ctx, bookID)
	if err != nil {
		slog.Error("manifest fetch failed", "error", err)
		fmt.Printf("%s\n", red("❌ Erro ao buscar manifesto"))
		os.Exit(1)
	}

	fmt.Printf("%s📖 %s - %s%s\n", cyan(""),
		manifest.Manifest.Metadata.Title,
		manifest.Manifest.Metadata.Author, "")
	fmt.Printf("   Capítulos: %d | Recursos: %d\n\n",
		len(manifest.Manifest.ReadingOrder),
		len(manifest.Manifest.Resources))

	stats, chapters, cssFiles, fontFiles, imageFiles, err := client.DownloadAll(ctx, manifest, webpubBaseURL)
	if err != nil {
		slog.Error("download failed", "error", err)
		fmt.Printf("%s\n", red("❌ Erro no download"))
		os.Exit(1)
	}

	if stats.ChaptersSuccess == 0 {
		fmt.Println(red("❌ Nenhum capítulo baixado"))
		os.Exit(1)
	}

	slog.Info("download complete",
		"chapters", stats.ChaptersSuccess,
		"css", stats.CSSSuccess,
		"fonts", stats.FontSuccess,
		"images", stats.ImageSuccess,
	)

	epubsPath := filepath.Join("epubs", outputName)
	builder := epub.NewBuilder(epubsPath)
	epubPath, err := builder.Build(manifest, chapters, cssFiles, fontFiles, imageFiles)
	if err != nil {
		slog.Error("epub build failed", "error", err)
		fmt.Printf("%s\n", red("❌ Erro ao criar EPUB"))
		os.Exit(1)
	}

	info, _ := os.Stat(epubPath)
	sizeMB := float64(info.Size()) / 1024 / 1024

	fmt.Printf("\n%s✅ EPUB criado: %s (%.1f MB)%s\n",
		green(""), epubPath, sizeMB, "")
	fmt.Printf("   Capítulos: %d/%d | CSS: %d | Fontes: %d | Imagens: %d\n",
		stats.ChaptersSuccess,
		len(manifest.Manifest.ReadingOrder),
		stats.CSSSuccess,
		stats.FontSuccess,
		stats.ImageSuccess)

	err = cacheManager.Save(jwtToken)
	if err != nil {
		slog.Error("failed to save cache", "error", err)
	}

	slog.Info("success", "epub", epubPath, "size_mb", sizeMB)
	fmt.Printf("\n%s🎉 Sucesso!%s\n", magenta(""), "")
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "livro"
	}

	// Use strings.Builder for efficient string building
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
