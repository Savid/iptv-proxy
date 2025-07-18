// Package epg provides parsing and filtering functionality for EPG (Electronic Program Guide) data.
package epg

import (
	"encoding/xml"
	"io"
)

// TV represents the root element of an EPG XML document.
type TV struct {
	XMLName  xml.Name    `xml:"tv"`
	Channels []Channel   `xml:"channel"`
	Programs []Programme `xml:"programme"`
}

// Channel represents a channel in the EPG data.
type Channel struct {
	ID          string `xml:"id,attr"`
	DisplayName string `xml:"display-name"`
	Icon        Icon   `xml:"icon"`
}

// Icon represents a channel icon in the EPG data.
type Icon struct {
	Src string `xml:"src,attr"`
}

// Programme represents a program/show in the EPG data.
type Programme struct {
	Channel     string `xml:"channel,attr"`
	Start       string `xml:"start,attr"`
	Stop        string `xml:"stop,attr"`
	Title       string `xml:"title"`
	Description string `xml:"desc"`
}

// ParseStream parses EPG XML data from an io.Reader.
func ParseStream(reader io.Reader) (*TV, error) {
	decoder := xml.NewDecoder(reader)

	var tv TV
	if err := decoder.Decode(&tv); err != nil {
		return nil, err
	}

	return &tv, nil
}
