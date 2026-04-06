package models

type Rental struct {
	BookID        int    `json:"book_id"`
	BookTitle     string `json:"book_title"`
	BookAuthor    string `json:"book_author"`
	DaysRemaining int    `json:"days_remaining"`
}

type RentalsResponse struct {
	Rentals []Rental `json:"rentals"`
}

type Manifest struct {
	Manifest struct {
		Metadata struct {
			Title      string `json:"title"`
			Author     string `json:"author"`
			Publisher  string `json:"publisher"`
			Language   string `json:"language"`
			Identifier string `json:"identifier"`
		} `json:"metadata"`
		ReadingOrder []Resource `json:"readingOrder"`
		Resources    []Resource `json:"resources"`
		Links        []Link     `json:"links"`
	} `json:"manifest"`
}

type Resource struct {
	Href string `json:"href"`
	Type string `json:"type"`
}

type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
	Type string `json:"type"`
}
