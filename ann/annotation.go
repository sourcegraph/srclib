package ann

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx/types"
)

// An Ann is a source code annotation.
//
// Annotations are unique on (Repo, CommitID, UnitType, Unit, File,
// Start, End, Type).
type Ann struct {
	// Repo is the URI of the repository that contains this annotation.
	Repo string `json:",omitempty"`

	// CommitID refers to the commit that contains this annotation.
	CommitID string `db:"commit_id" json:",omitempty"`

	// UnitType is the source unit type that the annotation exists
	// on. It is either the source unit type during whose processing
	// the annotation was detected/created. Multiple annotations may
	// exist on the same file from different source unit types if a
	// file is contained in multiple source units.
	UnitType string `db:"unit_type" json:",omitempty"`

	// Unit is the source unit name that the annotation exists on. See
	// UnitType for more information.
	Unit string `json:",omitempty"`

	// Type is the type of the annotation. See this package's type
	// constants for a list of possible types.
	Type string

	// File is the filename that contains this annotation.
	File string

	// Start is the byte offset of the first byte in the file.
	Start int

	// End is the byte offset of the last byte in the annotation.
	End int `json:",omitempty"`

	// Data contains arbitrary JSON data that is specific to this
	// annotation type (e.g., the link URL for Link annotations).
	Data types.JsonText `json:",omitempty"`
}

const (
	// Link is a type of annotation that refers to an arbitrary URL
	// (typically pointing to an external web page).
	Link = "link"
)

// LinkURL parses and returns a's link URL, if a's type is Link and if
// its Data contains a valid URL (encoded as a JSON string).
func (a *Ann) LinkURL() (*url.URL, error) {
	if a.Type != Link {
		return nil, &ErrType{Expected: Link, Actual: a.Type, Op: "LinkURL"}
	}
	var urlStr string
	if err := json.Unmarshal(a.Data, &urlStr); err != nil {
		return nil, err
	}
	return url.Parse(urlStr)
}

// SetLinkURL sets a's Type to Link and Data to the JSON
// representation of the URL string. If the URL is invalid, an error
// is returned.
func (a *Ann) SetLinkURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	b, err := json.Marshal(u.String())
	if err != nil {
		return err
	}
	a.Type = Link
	a.Data = b
	return nil
}

// ErrType indicates that an operation performed on an annotation
// expected the annotation to be a different type (e.g., calling
// LinkURL on a non-link annotation).
type ErrType struct {
	Expected, Actual string // Expected and actual types
	Op               string // The name of the operation or method that was called
}

func (e *ErrType) Error() string {
	return fmt.Sprintf("%s called on annotation type %q, expected type %q", e.Op, e.Actual, e.Expected)
}

func (a *Ann) sortKey() string {
	return strings.Join([]string{a.Repo, a.CommitID, a.UnitType, a.Unit, a.Type, a.File, strconv.Itoa(a.Start), strconv.Itoa(a.End)}, ":")
}

// Sorting

type Anns []*Ann

func (vs Anns) Len() int           { return len(vs) }
func (vs Anns) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs Anns) Less(i, j int) bool { return vs[i].sortKey() < vs[j].sortKey() }
