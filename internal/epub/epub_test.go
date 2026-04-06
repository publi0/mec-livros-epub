package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/publio/mectlivros/internal/downloader"
	"github.com/publio/mectlivros/pkg/models"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("my-book")
	if builder.outputName != "my-book" {
		t.Errorf("outputName = %q, want %q", builder.outputName, "my-book")
	}
}

func TestBuilder_Build_CreatesEPUBFile(t *testing.T) {
	builder := NewBuilder(filepath.Join(t.TempDir(), "test-book"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "Test Book"
	manifest.Manifest.Metadata.Author = "Test Author"
	manifest.Manifest.Metadata.Language = "pt-BR"
	manifest.Manifest.Metadata.Identifier = "ISBN-123"
	manifest.Manifest.Metadata.Publisher = "MEC"

	chapters := []downloader.DownloadedItem{
		{Filename: "chapter1.xhtml", Href: "OEBPS/chapter1.xhtml", Content: []byte("<html><body><p>Chapter 1</p></body></html>"), Size: 45},
		{Filename: "chapter2.xhtml", Href: "OEBPS/chapter2.xhtml", Content: []byte("<html><body><p>Chapter 2</p></body></html>"), Size: 45},
	}

	cssFiles := []downloader.DownloadedItem{
		{Filename: "style.css", Href: "css/style.css", Content: []byte("body { margin: 0; }"), Size: 17},
	}

	fontFiles := []downloader.DownloadedItem{}
	imageFiles := []downloader.DownloadedItem{}

	outputPath, err := builder.Build(manifest, chapters, cssFiles, fontFiles, imageFiles)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Build() did not create EPUB file")
	}

	if !strings.HasSuffix(outputPath, ".epub") {
		t.Errorf("outputPath = %q, want suffix .epub", outputPath)
	}
}

func TestBuilder_Build_CreatesValidZIP(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewBuilder(filepath.Join(tmpDir, "valid-zip"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "Valid ZIP Test"
	manifest.Manifest.Metadata.Author = "Author"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID-123"
	manifest.Manifest.Metadata.Publisher = "Publisher"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body>Content</body></html>"), Size: 29},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("Failed to open EPUB as ZIP: %v", err)
	}
	defer r.Close()

	foundMimetype := false
	for _, f := range r.File {
		if f.Name == "mimetype" {
			foundMimetype = true
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open mimetype: %v", err)
			}
			content := make([]byte, 30)
			n, _ := rc.Read(content)
			if string(content[:n]) != "application/epub+zip" {
				t.Errorf("mimetype content = %q, want %q", string(content[:n]), "application/epub+zip")
			}
			rc.Close()
		}
		if f.Name == "META-INF/container.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open container.xml: %v", err)
			}
			content := make([]byte, 200)
			n, _ := rc.Read(content)
			if !strings.Contains(string(content[:n]), "container") {
				t.Errorf("container.xml should contain 'container'")
			}
			rc.Close()
		}
	}

	if !foundMimetype {
		t.Error("EPUB does not contain mimetype file")
	}
}

func TestBuilder_Build_IncludesChaptersInSpine(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewBuilder(filepath.Join(tmpDir, "spine-test"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "Spine Test"
	manifest.Manifest.Metadata.Author = "Author"
	manifest.Manifest.Metadata.Language = "pt"
	manifest.Manifest.Metadata.Identifier = "ID-456"
	manifest.Manifest.Metadata.Publisher = "Pub"

	chapters := []downloader.DownloadedItem{
		{Filename: "c1.xhtml", Href: "OEBPS/c1.xhtml", Content: []byte("<html><body>1</body></html>"), Size: 23},
		{Filename: "c2.xhtml", Href: "OEBPS/c2.xhtml", Content: []byte("<html><body>2</body></html>"), Size: 23},
		{Filename: "c3.xhtml", Href: "OEBPS/c3.xhtml", Content: []byte("<html><body>3</body></html>"), Size: 23},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	var opfContent string
	for _, f := range r.File {
		if f.Name == "OEBPS/content.opf" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open content.opf: %v", err)
			}
			content := make([]byte, 2000)
			n, _ := rc.Read(content)
			opfContent = string(content[:n])
			rc.Close()
			break
		}
	}

	if !strings.Contains(opfContent, "<spine>") {
		t.Error("OPF should contain <spine> element")
	}
	if strings.Count(opfContent, "<itemref") != 3 {
		t.Errorf("OPF should contain 3 <itemref> elements, got %d", strings.Count(opfContent, "<itemref"))
	}
}

func TestBuilder_Build_HandlesEmptyCSS(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewBuilder(filepath.Join(tmpDir, "no-css"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "No CSS"
	manifest.Manifest.Metadata.Author = "Author"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID-789"
	manifest.Manifest.Metadata.Publisher = "Pub"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body>Content</body></html>"), Size: 29},
	}

	_, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() with nil CSS should not error: %v", err)
	}
}

func TestBuilder_Build_HandlesEmptyFonts(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewBuilder(filepath.Join(tmpDir, "no-fonts"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "No Fonts"
	manifest.Manifest.Metadata.Author = "Author"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID-999"
	manifest.Manifest.Metadata.Publisher = "Pub"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body>Content</body></html>"), Size: 29},
	}

	_, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() with nil fonts should not error: %v", err)
	}
}

func TestBuilder_Build_HandlesEmptyImages(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewBuilder(filepath.Join(tmpDir, "no-images"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "No Images"
	manifest.Manifest.Metadata.Author = "Author"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID-111"
	manifest.Manifest.Metadata.Publisher = "Pub"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body>Content</body></html>"), Size: 29},
	}

	_, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() with nil images should not error: %v", err)
	}
}

func TestBuilder_Build_AddsImageResources(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewBuilder(filepath.Join(tmpDir, "with-images"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "With Images"
	manifest.Manifest.Metadata.Author = "Author"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID-222"
	manifest.Manifest.Metadata.Publisher = "Pub"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body>Content</body></html>"), Size: 29},
	}

	images := []downloader.DownloadedItem{
		{Filename: "photo.jpg", Href: "image/photo.jpg", Content: []byte("fake jpeg data"), Size: 16},
		{Filename: "logo.png", Href: "image/logo.png", Content: []byte("fake png data"), Size: 15},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, nil, images)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	imageFiles := []string{}
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "OEBPS/image/") {
			imageFiles = append(imageFiles, f.Name)
		}
	}

	if len(imageFiles) != 2 {
		t.Errorf("Expected 2 image files, got %d", len(imageFiles))
	}
}

func TestBuilder_Build_AddsFontResources(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewBuilder(filepath.Join(tmpDir, "with-fonts"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "With Fonts"
	manifest.Manifest.Metadata.Author = "Author"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID-333"
	manifest.Manifest.Metadata.Publisher = "Pub"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body>Content</body></html>"), Size: 29},
	}

	fonts := []downloader.DownloadedItem{
		{Filename: "font.otf", Href: "font/font.otf", Content: []byte("fake font data"), Size: 14},
		{Filename: "font.ttf", Href: "font/font.ttf", Content: []byte("fake font data"), Size: 14},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, fonts, nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	fontFiles := []string{}
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "OEBPS/font/") {
			fontFiles = append(fontFiles, f.Name)
		}
	}

	if len(fontFiles) != 2 {
		t.Errorf("Expected 2 font files, got %d", len(fontFiles))
	}
}

func TestBuilder_Build_AddsCSSResources(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewBuilder(filepath.Join(tmpDir, "with-css"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "With CSS"
	manifest.Manifest.Metadata.Author = "Author"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID-444"
	manifest.Manifest.Metadata.Publisher = "Pub"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body>Content</body></html>"), Size: 29},
	}

	cssFiles := []downloader.DownloadedItem{
		{Filename: "style.css", Href: "css/style.css", Content: []byte("body { color: black; }"), Size: 23},
	}

	outputPath, err := builder.Build(manifest, chapters, cssFiles, nil, nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	cssFilesFound := []string{}
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "OEBPS/css/") {
			cssFilesFound = append(cssFilesFound, f.Name)
		}
	}

	if len(cssFilesFound) != 1 {
		t.Errorf("Expected 1 CSS file, got %d", len(cssFilesFound))
	}
}

func TestGenerateOPF_WritesMetadata(t *testing.T) {
	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "Title Test"
	manifest.Manifest.Metadata.Author = "Author Test"
	manifest.Manifest.Metadata.Language = "pt-BR"
	manifest.Manifest.Metadata.Identifier = "ISBN-ABC"
	manifest.Manifest.Metadata.Publisher = "Publisher Test"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("content"), Size: 7},
	}

	opf := generateOPF(manifest, chapters, nil, nil, nil, []string{"item1"})

	if !strings.Contains(opf, "<dc:title>Title Test</dc:title>") {
		t.Error("OPF should contain correct dc:title")
	}
	if !strings.Contains(opf, "<dc:creator>Author Test</dc:creator>") {
		t.Error("OPF should contain correct dc:creator")
	}
	if !strings.Contains(opf, "<dc:language>pt-BR</dc:language>") {
		t.Error("OPF should contain correct dc:language")
	}
	if !strings.Contains(opf, "<dc:identifier>ISBN-ABC</dc:identifier>") {
		t.Error("OPF should contain correct dc:identifier")
	}
	if !strings.Contains(opf, "<dc:publisher>Publisher Test</dc:publisher>") {
		t.Error("OPF should contain correct dc:publisher")
	}
}

func TestGenerateOPF_CreatesManifestItems(t *testing.T) {
	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "T"
	manifest.Manifest.Metadata.Author = "A"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID"
	manifest.Manifest.Metadata.Publisher = "P"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("content"), Size: 7},
	}
	cssFiles := []downloader.DownloadedItem{
		{Filename: "style.css", Href: "css/style.css", Content: []byte("css"), Size: 3},
	}
	fontFiles := []downloader.DownloadedItem{
		{Filename: "font.otf", Href: "font/font.otf", Content: []byte("font"), Size: 4},
	}
	imageFiles := []downloader.DownloadedItem{
		{Filename: "img.jpg", Href: "image/img.jpg", Content: []byte("img"), Size: 3},
	}

	opf := generateOPF(manifest, chapters, cssFiles, fontFiles, imageFiles, []string{"item1"})

	if !strings.Contains(opf, `media-type="application/xhtml+xml"`) {
		t.Error("OPF should contain xhtml media type")
	}
	if !strings.Contains(opf, `media-type="text/css"`) {
		t.Error("OPF should contain css media type")
	}
	if !strings.Contains(opf, `media-type="font/otf"`) {
		t.Error("OPF should contain otf font media type")
	}
	if !strings.Contains(opf, `media-type="image/jpeg"`) {
		t.Error("OPF should contain jpeg media type")
	}
}

func TestGenerateOPF_HandlesTTFFont(t *testing.T) {
	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "T"
	manifest.Manifest.Metadata.Author = "A"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID"
	manifest.Manifest.Metadata.Publisher = "P"

	fontFiles := []downloader.DownloadedItem{
		{Filename: "font.ttf", Href: "font/font.ttf", Content: []byte("font"), Size: 4},
	}

	opf := generateOPF(manifest, nil, nil, fontFiles, nil, nil)

	if !strings.Contains(opf, `media-type="application/font-sfnt"`) {
		t.Error("OPF should contain sfnt media type for TTF")
	}
}

func TestGenerateOPF_HandlesPNGImage(t *testing.T) {
	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "T"
	manifest.Manifest.Metadata.Author = "A"
	manifest.Manifest.Metadata.Language = "en"
	manifest.Manifest.Metadata.Identifier = "ID"
	manifest.Manifest.Metadata.Publisher = "P"

	imageFiles := []downloader.DownloadedItem{
		{Filename: "logo.png", Href: "image/logo.png", Content: []byte("png"), Size: 3},
	}

	opf := generateOPF(manifest, nil, nil, nil, imageFiles, nil)

	if !strings.Contains(opf, `media-type="image/png"`) {
		t.Error("OPF should contain png media type")
	}
}

func TestBuilder_Build_WithNonASCIITitle(t *testing.T) {
	builder := NewBuilder(filepath.Join(t.TempDir(), "test-na-title"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "São Paulo: História e Cultura"
	manifest.Manifest.Metadata.Author = "José Silva"
	manifest.Manifest.Metadata.Language = "pt-BR"
	manifest.Manifest.Metadata.Identifier = "ISBN-ABC"
	manifest.Manifest.Metadata.Publisher = "MEC"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body><p>Capítulo 1</p></body></html>"), Size: 45},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() error with non-ASCII title = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Build() did not create EPUB file with non-ASCII title")
	}
}

func TestBuilder_Build_WithNonASCIIAuthor(t *testing.T) {
	builder := NewBuilder(filepath.Join(t.TempDir(), "test-na-author"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "Book Title"
	manifest.Manifest.Metadata.Author = "Rui Mário"
	manifest.Manifest.Metadata.Language = "pt"
	manifest.Manifest.Metadata.Identifier = "ISBN-123"
	manifest.Manifest.Metadata.Publisher = "Editora Nacional"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body><p>Chapter</p></body></html>"), Size: 40},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() error with non-ASCII author = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Build() did not create EPUB file with non-ASCII author")
	}
}

func TestBuilder_Build_WithCJKCharacters(t *testing.T) {
	builder := NewBuilder(filepath.Join(t.TempDir(), "test-cjk"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "日本語の本"
	manifest.Manifest.Metadata.Author = "山田太郎"
	manifest.Manifest.Metadata.Language = "ja"
	manifest.Manifest.Metadata.Identifier = "ISBN-JP"
	manifest.Manifest.Metadata.Publisher = "日本出版社"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body><p>章1</p></body></html>"), Size: 30},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() error with CJK characters = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Build() did not create EPUB file with CJK characters")
	}
}

func TestBuilder_Build_WithEmoji(t *testing.T) {
	builder := NewBuilder(filepath.Join(t.TempDir(), "test-emoji"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "📚 Livros Infantis"
	manifest.Manifest.Metadata.Author = "Maria 👩‍👧‍👦"
	manifest.Manifest.Metadata.Language = "pt-BR"
	manifest.Manifest.Metadata.Identifier = "ISBN-EMOJI"
	manifest.Manifest.Metadata.Publisher = "Editora"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body><p>📖 Chapter</p></body></html>"), Size: 35},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() error with emoji = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Build() did not create EPUB file with emoji")
	}
}

func TestGenerateOPF_WithNonASCIIMetadata(t *testing.T) {
	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "São Paulo"
	manifest.Manifest.Metadata.Author = "José"
	manifest.Manifest.Metadata.Language = "pt-BR"
	manifest.Manifest.Metadata.Identifier = "ISBN-123"
	manifest.Manifest.Metadata.Publisher = "MEC - Ministério"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("content"), Size: 7},
	}

	opf := generateOPF(manifest, chapters, nil, nil, nil, []string{"item1"})

	if !strings.Contains(opf, "São Paulo") {
		t.Error("OPF should contain non-ASCII title")
	}
	if !strings.Contains(opf, "José") {
		t.Error("OPF should contain non-ASCII author")
	}
	if !strings.Contains(opf, "pt-BR") {
		t.Error("OPF should contain pt-BR language")
	}
}

func TestBuilder_Build_PreservesUTF8InOPF(t *testing.T) {
	builder := NewBuilder(filepath.Join(t.TempDir(), "test-utf8-opf"))

	manifest := &models.Manifest{}
	manifest.Manifest.Metadata.Title = "Çãõ áéíóú"
	manifest.Manifest.Metadata.Author = "Niño"
	manifest.Manifest.Metadata.Language = "pt"
	manifest.Manifest.Metadata.Identifier = "ISBN-UTF"
	manifest.Manifest.Metadata.Publisher = "Publisher"

	chapters := []downloader.DownloadedItem{
		{Filename: "ch1.xhtml", Href: "OEBPS/ch1.xhtml", Content: []byte("<html><body>Çãõ</body></html>"), Size: 25},
	}

	outputPath, err := builder.Build(manifest, chapters, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	var opfContent string
	for _, f := range r.File {
		if f.Name == "OEBPS/content.opf" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open content.opf: %v", err)
			}
			content := make([]byte, 2000)
			n, _ := rc.Read(content)
			opfContent = string(content[:n])
			rc.Close()
			break
		}
	}

	if !strings.Contains(opfContent, "Çãõ") {
		t.Error("OPF should contain UTF-8 characters Çãõ")
	}
}
