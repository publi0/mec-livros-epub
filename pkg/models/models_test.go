package models

import (
	"encoding/json"
	"testing"
)

func TestRental_JSON_Unmarshal(t *testing.T) {
	data := `{
		"book_id": 123,
		"book_title": "Matemática",
		"book_author": "João Silva",
		"days_remaining": 15
	}`

	var r Rental
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if r.BookID != 123 {
		t.Errorf("BookID = %d, want 123", r.BookID)
	}
	if r.BookTitle != "Matemática" {
		t.Errorf("BookTitle = %q, want %q", r.BookTitle, "Matemática")
	}
	if r.BookAuthor != "João Silva" {
		t.Errorf("BookAuthor = %q, want %q", r.BookAuthor, "João Silva")
	}
	if r.DaysRemaining != 15 {
		t.Errorf("DaysRemaining = %d, want 15", r.DaysRemaining)
	}
}

func TestRentalsResponse_Unmarshal(t *testing.T) {
	data := `{
		"rentals": [
			{"book_id": 1, "book_title": "Livro 1", "book_author": "Autor 1", "days_remaining": 10},
			{"book_id": 2, "book_title": "Livro 2", "book_author": "Autor 2", "days_remaining": 5}
		]
	}`

	var resp RentalsResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(resp.Rentals) != 2 {
		t.Errorf("len(Rentals) = %d, want 2", len(resp.Rentals))
	}
	if resp.Rentals[0].BookTitle != "Livro 1" {
		t.Errorf("Rentals[0].BookTitle = %q, want %q", resp.Rentals[0].BookTitle, "Livro 1")
	}
	if resp.Rentals[1].BookTitle != "Livro 2" {
		t.Errorf("Rentals[1].BookTitle = %q, want %q", resp.Rentals[1].BookTitle, "Livro 2")
	}
}

func TestResource_JSON(t *testing.T) {
	data := `{"href": "chapter1.xhtml", "type": "application/xhtml+xml"}`

	var r Resource
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if r.Href != "chapter1.xhtml" {
		t.Errorf("Href = %q, want %q", r.Href, "chapter1.xhtml")
	}
	if r.Type != "application/xhtml+xml" {
		t.Errorf("Type = %q, want %q", r.Type, "application/xhtml+xml")
	}
}

func TestLink_JSON(t *testing.T) {
	data := `{"href": "https://example.com/manifest.json", "rel": "self", "type": "application/webpub+json"}`

	var l Link
	if err := json.Unmarshal([]byte(data), &l); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if l.Href != "https://example.com/manifest.json" {
		t.Errorf("Href = %q, want %q", l.Href, "https://example.com/manifest.json")
	}
	if l.Rel != "self" {
		t.Errorf("Rel = %q, want %q", l.Rel, "self")
	}
	if l.Type != "application/webpub+json" {
		t.Errorf("Type = %q, want %q", l.Type, "application/webpub+json")
	}
}

func TestManifest_Unmarshal(t *testing.T) {
	data := `{
		"manifest": {
			"metadata": {
				"title": "Test Book",
				"author": "Test Author",
				"publisher": "MEC",
				"language": "pt-BR",
				"identifier": "ISBN-123"
			},
			"readingOrder": [
				{"href": "ch1.xhtml", "type": "application/xhtml+xml"}
			],
			"resources": [
				{"href": "style.css", "type": "text/css"}
			],
			"links": [
				{"href": "https://example.com/manifest.json", "rel": "self", "type": "application/webpub+json"}
			]
		}
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if m.Manifest.Metadata.Title != "Test Book" {
		t.Errorf("Metadata.Title = %q, want %q", m.Manifest.Metadata.Title, "Test Book")
	}
	if m.Manifest.Metadata.Author != "Test Author" {
		t.Errorf("Metadata.Author = %q, want %q", m.Manifest.Metadata.Author, "Test Author")
	}
	if len(m.Manifest.ReadingOrder) != 1 {
		t.Errorf("len(ReadingOrder) = %d, want 1", len(m.Manifest.ReadingOrder))
	}
	if len(m.Manifest.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want 1", len(m.Manifest.Resources))
	}
	if len(m.Manifest.Links) != 1 {
		t.Errorf("len(Links) = %d, want 1", len(m.Manifest.Links))
	}
}

func TestManifest_EmptyReadingOrder(t *testing.T) {
	data := `{
		"manifest": {
			"metadata": {"title": "Empty", "author": "", "publisher": "", "language": "", "identifier": ""},
			"readingOrder": [],
			"resources": [],
			"links": []
		}
	}`

	var m Manifest
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(m.Manifest.ReadingOrder) != 0 {
		t.Errorf("len(ReadingOrder) = %d, want 0", len(m.Manifest.ReadingOrder))
	}
}

func TestResource_HrefCanContainSpecialCharacters(t *testing.T) {
	data := `{"href": "assets/img/photo%20space.jpg", "type": "image/jpeg"}`

	var r Resource
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if r.Href != "assets/img/photo%20space.jpg" {
		t.Errorf("Href = %q, want %q", r.Href, "assets/img/photo%20space.jpg")
	}
}
