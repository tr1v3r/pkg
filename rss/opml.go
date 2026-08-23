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

// Head defines the header information of an OPML 2.0 document (§2.1).
type Head struct {
	Title           string `xml:"title,omitempty"`
	DateCreated     string `xml:"dateCreated,omitempty"`
	DateModified    string `xml:"dateModified,omitempty"`
	OwnerName       string `xml:"ownerName,omitempty"`
	OwnerEmail      string `xml:"ownerEmail,omitempty"`
	OwnerID         string `xml:"ownerId,omitempty"`
	Docs            string `xml:"docs,omitempty"`
	ExpansionState  []int  `xml:"expansionState>number"`
	VertScrollState int    `xml:"vertScrollState,omitempty"`
	WindowTop       int    `xml:"windowTop,omitempty"`
	WindowLeft      int    `xml:"windowLeft,omitempty"`
	WindowBottom    int    `xml:"windowBottom,omitempty"`
	WindowRight     int    `xml:"windowRight,omitempty"`
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
	Type         string     `xml:"type,attr,omitempty"`
	Text         string     `xml:"text,attr,omitempty"`
	IsComment    *bool      `xml:"isComment,attr,omitempty"`
	IsBreakpoint *bool      `xml:"isBreakpoint,attr,omitempty"`
	Title        string     `xml:"title,attr,omitempty"`
	XMLUrl       string     `xml:"xmlUrl,attr,omitempty"`
	HTMLUrl      string     `xml:"htmlUrl,attr,omitempty"`
	Language     string     `xml:"language,attr,omitempty"`
	Version      string     `xml:"version,attr,omitempty"`
	Description  string     `xml:"description,attr,omitempty"`
	Category     string     `xml:"category,attr,omitempty"`
	Created      string     `xml:"created,attr,omitempty"`
	Outlines     []*Outline `xml:"outline,omitempty"`
}

// ParseOPML parses XML data into an OPML document.
func ParseOPML(data []byte) (*OPML, error) {
	var opml OPML
	if err := xml.Unmarshal(data, &opml); err != nil {
		return nil, fmt.Errorf("parse OPML: %w", err)
	}
	return &opml, nil
}
