package main

import (
	"cmp"
	"context"
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
	magenta := color.New(color.FgMagenta, color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Println(magenta("============================================"))
	fmt.Println(magenta("MEC LIVROS - EBOOK DOWNLOADER"))
	fmt.Println(magenta("============================================"))
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	if err := os.MkdirAll("epubs", 0755); err != nil {
		fmt.Printf("%s\n", red("Erro ao criar pasta epubs"))
		os.Exit(1)
	}

	cacheManager := cache.New()
	jwtToken := cacheManager.Get()
	usedCache := jwtToken != ""

	if usedCache {
		fmt.Println(yellow("Token em cache encontrado"))
		fmt.Printf("Token: %s...\n\n", jwtToken[:min(30, len(jwtToken))])
		fmt.Println(green("Usando token do cache\n"))
	}

	if jwtToken == "" {
		fmt.Println(cyan("Informe o JWT Token:"))
		fmt.Println("(No navegador: DevTools > Application > Local Storage)")
		fmt.Print("Token: ")
		fmt.Scanln(&jwtToken)
		jwtToken = strings.TrimSpace(jwtToken)

		if jwtToken == "" {
			fmt.Println(red("\nToken obrigatório"))
			os.Exit(1)
		}
		fmt.Println()
	}

	client := downloader.NewClient(jwtToken, "")

	rentals, err := client.FetchRentals(ctx)
	if err != nil || len(rentals) == 0 {
		fmt.Printf("%s\n", red("\nNenhum ebook encontrado ou token inválido"))
		os.Exit(1)
	}

	fmt.Printf("\n%sEncontrados %d ebook(s):%s\n\n", green(""), len(rentals), "")
	for i, r := range rentals {
		fmt.Printf("[%d] %s (ID: %d, %d dias restantes)\n",
			i+1, cyan(r.BookTitle), r.BookID, r.DaysRemaining)
	}

	var selected models.Rental
	switch len(rentals) {
	case 1:
		selected = rentals[0]
		fmt.Printf("\n%sAuto-selecionado: %s%s\n", green(""), selected.BookTitle, "")
	default:
		fmt.Printf("\nSelecione [1-%d]: ", len(rentals))
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

	fmt.Printf("\n%sSaída: %s.epub%s\n", cyan(""), outputName, "")
	fmt.Printf("%sIniciando download...%s\n\n", green(""), "")

	client = downloader.NewClient(jwtToken, bookID)

	manifest, webpubBaseURL, err := client.FetchManifest(ctx, bookID)
	if err != nil {
		fmt.Printf("%s\n", red("Erro ao buscar manifesto"))
		os.Exit(1)
	}

	fmt.Printf("%s%s - %s%s\n", cyan(""),
		manifest.Manifest.Metadata.Title,
		manifest.Manifest.Metadata.Author, "")
	fmt.Printf("Capítulos: %d | Recursos: %d\n\n",
		len(manifest.Manifest.ReadingOrder),
		len(manifest.Manifest.Resources))

	stats, chapters, cssFiles, fontFiles, imageFiles, err := client.DownloadAll(ctx, manifest, webpubBaseURL)
	if err != nil {
		fmt.Printf("%s\n", red("Erro no download"))
		os.Exit(1)
	}

	if stats.ChaptersSuccess == 0 {
		fmt.Println(red("Nenhum capítulo baixado"))
		os.Exit(1)
	}

	epubsPath := filepath.Join("epubs", outputName)
	builder := epub.NewBuilder(epubsPath)
	epubPath, err := builder.Build(manifest, chapters, cssFiles, fontFiles, imageFiles)
	if err != nil {
		fmt.Printf("%s\n", red("Erro ao criar EPUB"))
		os.Exit(1)
	}

	info, _ := os.Stat(epubPath)
	sizeMB := float64(info.Size()) / 1024 / 1024

	fmt.Printf("\n%sEPUB criado: %s (%.1f MB)%s\n",
		green(""), epubPath, sizeMB, "")
	fmt.Printf("Capítulos: %d/%d | CSS: %d | Fontes: %d | Imagens: %d\n",
		stats.ChaptersSuccess,
		len(manifest.Manifest.ReadingOrder),
		stats.CSSSuccess,
		stats.FontSuccess,
		stats.ImageSuccess)

	if !usedCache {
		if err := cacheManager.Save(jwtToken); err != nil {
			fmt.Printf("Erro ao salvar cache: %v\n", err)
		}
	}

	fmt.Printf("\n%sSucesso!%s\n", magenta(""), "")
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
