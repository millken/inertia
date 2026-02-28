package inertia

import (
	"html"
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

// writeMetaTag writes a single <meta> tag to sb if content is non-empty,
// escaping attribute values for XSS safety.
func writeMetaTag(sb *strings.Builder, attr, name, content string) {
	if content == "" {
		return
	}
	sb.WriteString("<meta ")
	sb.WriteString(attr)
	sb.WriteString(`="`)
	sb.WriteString(html.EscapeString(name))
	sb.WriteString(`" content="`)
	sb.WriteString(html.EscapeString(content))
	sb.WriteString("\">\n")
}

// ToHTML converts Meta to HTML meta tags
func (m *Meta) ToHTML() string {
	var sb strings.Builder

	if m.Charset != "" {
		sb.WriteString(`<meta charset="`)
		sb.WriteString(html.EscapeString(m.Charset))
		sb.WriteString("\">\n")
	}

	writeMetaTag(&sb, "name", "viewport", m.Viewport)

	if m.Title != "" {
		sb.WriteString("<title>")
		sb.WriteString(html.EscapeString(m.Title))
		sb.WriteString("</title>\n")
	}

	writeMetaTag(&sb, "name", "description", m.Description)
	writeMetaTag(&sb, "name", "keywords", m.Keywords)
	writeMetaTag(&sb, "name", "author", m.Author)

	for name, content := range m.Custom {
		writeMetaTag(&sb, "name", name, content)
	}

	writeMetaTag(&sb, "property", "og:title", m.OpenGraph.Title)
	writeMetaTag(&sb, "property", "og:description", m.OpenGraph.Description)
	writeMetaTag(&sb, "property", "og:image", m.OpenGraph.Image)
	writeMetaTag(&sb, "property", "og:url", m.OpenGraph.URL)
	writeMetaTag(&sb, "property", "og:type", m.OpenGraph.Type)
	writeMetaTag(&sb, "property", "og:site_name", m.OpenGraph.SiteName)

	writeMetaTag(&sb, "name", "twitter:card", m.Twitter.Card)
	writeMetaTag(&sb, "name", "twitter:site", m.Twitter.Site)
	writeMetaTag(&sb, "name", "twitter:creator", m.Twitter.Creator)
	writeMetaTag(&sb, "name", "twitter:title", m.Twitter.Title)
	writeMetaTag(&sb, "name", "twitter:description", m.Twitter.Description)
	writeMetaTag(&sb, "name", "twitter:image", m.Twitter.Image)

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
