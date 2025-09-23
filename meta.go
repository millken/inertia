package inertia

import (
	"fmt"
	"maps"
	"strings"
)

// Meta represents HTML meta information
type Meta struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Keywords    string            `json:"keywords,omitempty"`
	Author      string            `json:"author,omitempty"`
	Viewport    string            `json:"viewport,omitempty"`
	Charset     string            `json:"charset,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
	OpenGraph   OpenGraph         `json:"og,omitempty"`
	Twitter     TwitterCard       `json:"twitter,omitempty"`
}

// OpenGraph represents Open Graph meta tags
type OpenGraph struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
	SiteName    string `json:"site_name,omitempty"`
}

// TwitterCard represents Twitter Card meta tags
type TwitterCard struct {
	Card        string `json:"card,omitempty"`
	Site        string `json:"site,omitempty"`
	Creator     string `json:"creator,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
}

// NewMeta creates a new Meta instance with default values
func NewMeta() *Meta {
	return &Meta{
		Charset:  "UTF-8",
		Viewport: "width=device-width, initial-scale=1.0",
		Custom:   make(map[string]string),
	}
}

// SetTitle sets the page title
func (m *Meta) SetTitle(title string) *Meta {
	m.Title = title
	return m
}

// SetDescription sets the page description
func (m *Meta) SetDescription(description string) *Meta {
	m.Description = description
	return m
}

// SetKeywords sets the page keywords
func (m *Meta) SetKeywords(keywords string) *Meta {
	m.Keywords = keywords
	return m
}

// SetAuthor sets the page author
func (m *Meta) SetAuthor(author string) *Meta {
	m.Author = author
	return m
}

// AddCustom adds a custom meta tag
func (m *Meta) AddCustom(name, content string) *Meta {
	if m.Custom == nil {
		m.Custom = make(map[string]string)
	}
	m.Custom[name] = content
	return m
}

// SetOpenGraph sets Open Graph meta tags
func (m *Meta) SetOpenGraph(og OpenGraph) *Meta {
	m.OpenGraph = og
	return m
}

// SetTwitterCard sets Twitter Card meta tags
func (m *Meta) SetTwitterCard(twitter TwitterCard) *Meta {
	m.Twitter = twitter
	return m
}

// ToHTML converts Meta to HTML meta tags
func (m *Meta) ToHTML() string {
	var sb strings.Builder

	if m.Charset != "" {
		sb.WriteString(fmt.Sprintf(`<meta charset="%s">`, m.Charset))
		sb.WriteString("\n")
	}

	if m.Viewport != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="viewport" content="%s">`, m.Viewport))
		sb.WriteString("\n")
	}

	if m.Title != "" {
		sb.WriteString(fmt.Sprintf(`<title>%s</title>`, m.Title))
		sb.WriteString("\n")
	}

	if m.Description != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="description" content="%s">`, m.Description))
		sb.WriteString("\n")
	}

	if m.Keywords != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="keywords" content="%s">`, m.Keywords))
		sb.WriteString("\n")
	}

	if m.Author != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="author" content="%s">`, m.Author))
		sb.WriteString("\n")
	}

	// Custom meta tags
	for name, content := range m.Custom {
		sb.WriteString(fmt.Sprintf(`<meta name="%s" content="%s">`, name, content))
		sb.WriteString("\n")
	}

	// Open Graph tags
	if m.OpenGraph.Title != "" {
		sb.WriteString(fmt.Sprintf(`<meta property="og:title" content="%s">`, m.OpenGraph.Title))
		sb.WriteString("\n")
	}
	if m.OpenGraph.Description != "" {
		sb.WriteString(fmt.Sprintf(`<meta property="og:description" content="%s">`, m.OpenGraph.Description))
		sb.WriteString("\n")
	}
	if m.OpenGraph.Image != "" {
		sb.WriteString(fmt.Sprintf(`<meta property="og:image" content="%s">`, m.OpenGraph.Image))
		sb.WriteString("\n")
	}
	if m.OpenGraph.URL != "" {
		sb.WriteString(fmt.Sprintf(`<meta property="og:url" content="%s">`, m.OpenGraph.URL))
		sb.WriteString("\n")
	}
	if m.OpenGraph.Type != "" {
		sb.WriteString(fmt.Sprintf(`<meta property="og:type" content="%s">`, m.OpenGraph.Type))
		sb.WriteString("\n")
	}
	if m.OpenGraph.SiteName != "" {
		sb.WriteString(fmt.Sprintf(`<meta property="og:site_name" content="%s">`, m.OpenGraph.SiteName))
		sb.WriteString("\n")
	}

	// Twitter Card tags
	if m.Twitter.Card != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="twitter:card" content="%s">`, m.Twitter.Card))
		sb.WriteString("\n")
	}
	if m.Twitter.Site != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="twitter:site" content="%s">`, m.Twitter.Site))
		sb.WriteString("\n")
	}
	if m.Twitter.Creator != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="twitter:creator" content="%s">`, m.Twitter.Creator))
		sb.WriteString("\n")
	}
	if m.Twitter.Title != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="twitter:title" content="%s">`, m.Twitter.Title))
		sb.WriteString("\n")
	}
	if m.Twitter.Description != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="twitter:description" content="%s">`, m.Twitter.Description))
		sb.WriteString("\n")
	}
	if m.Twitter.Image != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="twitter:image" content="%s">`, m.Twitter.Image))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m Meta) Clone() Meta {
	clone := Meta{
		Title:       m.Title,
		Description: m.Description,
		Keywords:    m.Keywords,
		Author:      m.Author,
		Viewport:    m.Viewport,
		Charset:     m.Charset,
		Custom:      make(map[string]string),
		OpenGraph:   m.OpenGraph,
		Twitter:     m.Twitter,
	}
	maps.Copy(clone.Custom, m.Custom)
	return clone
}

// UseMeta creates a middleware that initializes Meta in context
func UseMeta(defaultMeta *Meta) HandlerFunc {
	return func(c *Context) {
		c.SetMeta(defaultMeta.Clone())
		c.Next()
	}
}
