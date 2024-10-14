package rss

import "encoding/xml"

// OPML defines the root element structure of an OPML document.
type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

// Head defines the header information of an OPML document.
type Head struct {
	Title        string `xml:"title"`
	DateCreated  string `xml:"dateCreated"`
	DateModified string `xml:"dateModified"`
}

// Body defines the body of an OPML document, containing multiple outlines.
type Body struct {
	Outlines OutlineArray `xml:"outline"`
}

// OutlineArray is a slice of Outline pointers.
type OutlineArray []*Outline

// Append adds a new outline to the OutlineArray, grouped by the specified groupText and groupTitle.
func (a OutlineArray) Append(groupTitle string, o *Outline) OutlineArray {
	if o.Text == "" || o.XMLUrl == "" { // invalid outline
		return a
	}
	if groupTitle == "" { // group info cannot be empty
		return a
	}

	for _, group := range a {
		if group.Title == groupTitle { // group found
			group.Outlines = append(group.Outlines, o)
			return a
		}
	}
	return append(a, &Outline{Title: groupTitle, Outlines: []*Outline{o}})
}

// Outline defines the structure of an outline element in the OPML document.
type Outline struct {
	Type     string     `xml:"type,attr,omitempty"`
	Text     string     `xml:"text,attr,omitempty"`
	Title    string     `xml:"title,attr,omitempty"`
	XMLUrl   string     `xml:"xmlUrl,attr,omitempty"`
	HTMLUrl  string     `xml:"htmlUrl,attr,omitempty"`
	Outlines []*Outline `xml:"outline,omitempty"` // 用于支持嵌套结构
}
