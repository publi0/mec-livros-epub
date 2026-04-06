package epub

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/publio/mectlivros/internal/downloader"
	"github.com/publio/mectlivros/pkg/models"
)

// Builder creates EPUB files
type Builder struct {
	outputName string
}

// NewBuilder creates a new EPUB builder
func NewBuilder(outputName string) *Builder {
	return &Builder{outputName: outputName}
}

// Build creates the EPUB file from downloaded resources
func (b *Builder) Build(
	manifest *models.Manifest,
	chapters []downloader.DownloadedItem,
	cssFiles []downloader.DownloadedItem,
	fontFiles []downloader.DownloadedItem,
	imageFiles []downloader.DownloadedItem,
) (string, error) {

	tempDir := fmt.Sprintf("epub_temp_%d", os.Getpid())
	defer os.RemoveAll(tempDir)

	os.MkdirAll(filepath.Join(tempDir, "OEBPS", "css"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "OEBPS", "font"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "OEBPS", "image"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "META-INF"), 0755)

	// Write chapters and collect IDs
	chapterIDs := make([]string, 0, len(chapters))
	for i, ch := range chapters {
		id := fmt.Sprintf("item%d", i+1)
		chapterIDs = append(chapterIDs, id)
		path := filepath.Join(tempDir, "OEBPS", ch.Filename)
		if err := os.WriteFile(path, ch.Content, 0644); err != nil {
			return "", err
		}
	}

	for _, item := range cssFiles {
		path := filepath.Join(tempDir, "OEBPS", "css", item.Filename)
		if err := os.WriteFile(path, item.Content, 0644); err != nil {
			return "", err
		}
	}

	for _, item := range fontFiles {
		path := filepath.Join(tempDir, "OEBPS", "font", item.Filename)
		if err := os.WriteFile(path, item.Content, 0644); err != nil {
			return "", err
		}
	}

	for _, item := range imageFiles {
		path := filepath.Join(tempDir, "OEBPS", "image", item.Filename)
		if err := os.WriteFile(path, item.Content, 0644); err != nil {
			return "", err
		}
	}

	opf := generateOPF(manifest, chapters, cssFiles, fontFiles, imageFiles, chapterIDs)
	opfPath := filepath.Join(tempDir, "OEBPS", "content.opf")
	if err := os.WriteFile(opfPath, []byte(opf), 0644); err != nil {
		return "", err
	}

	container := `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
    <rootfiles>
        <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
    </rootfiles>
</container>`
	containerPath := filepath.Join(tempDir, "META-INF", "container.xml")
	if err := os.WriteFile(containerPath, []byte(container), 0644); err != nil {
		return "", err
	}

	outputFile := b.outputName + ".epub"
	if err := createZip(tempDir, outputFile); err != nil {
		return "", err
	}

	return outputFile, nil
}

func generateOPF(
	manifest *models.Manifest,
	chapters []downloader.DownloadedItem,
	cssFiles []downloader.DownloadedItem,
	fontFiles []downloader.DownloadedItem,
	imageFiles []downloader.DownloadedItem,
	chapterIDs []string,
) string {
	var opf strings.Builder

	opf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
    <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
        <dc:title>`)
	opf.WriteString(manifest.Manifest.Metadata.Title)
	opf.WriteString(`</dc:title>
        <dc:creator>`)
	opf.WriteString(manifest.Manifest.Metadata.Author)
	opf.WriteString(`</dc:creator>
        <dc:language>`)
	opf.WriteString(manifest.Manifest.Metadata.Language)
	opf.WriteString(`</dc:language>
        <dc:identifier>`)
	opf.WriteString(manifest.Manifest.Metadata.Identifier)
	opf.WriteString(`</dc:identifier>
        <dc:publisher>`)
	opf.WriteString(manifest.Manifest.Metadata.Publisher)
	opf.WriteString(`</dc:publisher>
    </metadata>
    <manifest>
`)

	itemID := 0

	for _, ch := range chapters {
		itemID++
		fmt.Fprintf(&opf, "        <item id=\"item%d\" href=\"%s\" media-type=\"application/xhtml+xml\"/>\n", itemID, ch.Filename)
	}

	for _, item := range cssFiles {
		itemID++
		fmt.Fprintf(&opf, "        <item id=\"item%d\" href=\"css/%s\" media-type=\"text/css\"/>\n", itemID, item.Filename)
	}

	for _, item := range fontFiles {
		itemID++
		media := "font/otf"
		if !strings.Contains(item.Filename, ".otf") {
			media = "application/font-sfnt"
		}
		fmt.Fprintf(&opf, "        <item id=\"item%d\" href=\"font/%s\" media-type=\"%s\"/>\n", itemID, item.Filename, media)
	}

	for _, item := range imageFiles {
		itemID++
		media := "image/jpeg"
		if strings.Contains(item.Filename, ".png") {
			media = "image/png"
		}
		fmt.Fprintf(&opf, "        <item id=\"item%d\" href=\"image/%s\" media-type=\"%s\"/>\n", itemID, item.Filename, media)
	}

	opf.WriteString(`    </manifest>
    <spine>
`)

	// Spine with correct IDs
	for _, id := range chapterIDs {
		fmt.Fprintf(&opf, "        <itemref idref=\"%s\"/>\n", id)
	}

	opf.WriteString(`    </spine>
</package>`)

	return opf.String()
}

func createZip(sourceDir, outputFile string) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// Add mimetype first (uncompressed)
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		return err
	}
	w.Write([]byte("application/epub+zip"))

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		if relPath == "mimetype" {
			return nil
		}

		w, err := zw.Create(relPath)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = w.Write(data)
		return err
	})
}
