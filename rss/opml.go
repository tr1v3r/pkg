package rss

import (
	"encoding/xml"
	"fmt"
)

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
	DateCreated  string `xml:"dateCreated,omitempty"`
	DateModified string `xml:"dateModified,omitempty"`
}

// Body defines the body of an OPML document, containing multiple outlines.
type Body struct {
	Outlines OutlineArray `xml:"outline"`
}

// OutlineArray is a slice of Outline pointers.
type OutlineArray []*Outline

// AddOutline returns the array with the outline added under the specified group.
// If the group does not exist, a new one is created.
// Returns the receiver unchanged if the outline or groupText is invalid.
func (a OutlineArray) AddOutline(groupText string, o *Outline) OutlineArray {
	if o.Text == "" || o.XMLUrl == "" {
		return a
	}
	if groupText == "" {
		return a
	}

	for _, group := range a {
		if group.Text == groupText {
			group.Outlines = append(group.Outlines, o)
			return a
		}
	}
	return append(a, &Outline{Text: groupText, Outlines: []*Outline{o}})
}

// Outline defines the structure of an outline element in the OPML document.
type Outline struct {
	Type     string     `xml:"type,attr,omitempty"`
	Text     string     `xml:"text,attr,omitempty"`
	Title    string     `xml:"title,attr,omitempty"`
	XMLUrl   string     `xml:"xmlUrl,attr,omitempty"`
	HTMLUrl  string     `xml:"htmlUrl,attr,omitempty"`
	Outlines []*Outline `xml:"outline,omitempty"`
}

// ParseOPML parses XML data into an OPML document.
func ParseOPML(data []byte) (*OPML, error) {
	var opml OPML
	if err := xml.Unmarshal(data, &opml); err != nil {
		return nil, fmt.Errorf("parse OPML: %w", err)
	}
	return &opml, nil
}
