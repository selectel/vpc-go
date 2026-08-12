package api

// PageLink is one link in a paginated collection response.
type PageLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// NextPageLink returns the URL of the next page, or an empty string on the last page.
func NextPageLink(links []PageLink) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}

	return ""
}
