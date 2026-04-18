package httptape

import (
	"mime"
	"sort"
	"strconv"
	"strings"
)

// MediaType represents a parsed media type (e.g., "application/json; charset=utf-8").
// It is a value type with no I/O.
type MediaType struct {
	// Type is the top-level type (e.g., "application", "text", "image").
	Type string
	// Subtype is the subtype (e.g., "json", "plain", "png").
	// For structured syntax suffixes, this includes the full subtype
	// (e.g., "vnd.api+json").
	Subtype string
	// Suffix is the structured syntax suffix without the '+' prefix
	// (e.g., "json" for "application/vnd.api+json"). Empty if no suffix.
	Suffix string
	// Params holds media type parameters (e.g., charset=utf-8).
	// The "q" parameter is extracted into QValue and not included here.
	Params map[string]string
	// QValue is the quality factor from the Accept header (0.0-1.0).
	// Defaults to 1.0 if not specified.
	QValue float64
}

// ParseMediaType parses a single media type string (e.g., "application/json;
// charset=utf-8") into a MediaType. Parameters including q-value are parsed.
// Returns an error if the string is fundamentally malformed (no "/" separator).
// Uses mime.ParseMediaType from stdlib internally.
func ParseMediaType(s string) (MediaType, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return MediaType{}, &mediaTypeError{input: s, reason: "empty media type"}
	}

	mediaType, params, err := mime.ParseMediaType(s)
	if err != nil {
		return MediaType{}, &mediaTypeError{input: s, reason: err.Error()}
	}

	slashIdx := strings.IndexByte(mediaType, '/')
	if slashIdx < 0 {
		return MediaType{}, &mediaTypeError{input: s, reason: "missing '/' separator"}
	}

	typ := mediaType[:slashIdx]
	subtype := mediaType[slashIdx+1:]

	// Extract structured syntax suffix (e.g., "+json" from "vnd.api+json").
	var suffix string
	if plusIdx := strings.LastIndexByte(subtype, '+'); plusIdx >= 0 {
		suffix = subtype[plusIdx+1:]
	}

	// Extract q-value from params.
	qvalue := 1.0
	filteredParams := make(map[string]string, len(params))
	for k, v := range params {
		if strings.EqualFold(k, "q") {
			if q, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
				if q < 0 {
					q = 0
				}
				if q > 1 {
					q = 1
				}
				qvalue = q
			}
			continue
		}
		filteredParams[k] = v
	}

	return MediaType{
		Type:    typ,
		Subtype: subtype,
		Suffix:  suffix,
		Params:  filteredParams,
		QValue:  qvalue,
	}, nil
}

// ParseAccept parses an Accept header value into a slice of MediaType entries,
// sorted by precedence (highest q-value first, then specificity descending).
// Malformed individual media ranges are silently skipped.
// An empty or missing Accept header returns a single entry for "*/*" with q=1.0.
func ParseAccept(accept string) []MediaType {
	accept = strings.TrimSpace(accept)
	if accept == "" {
		return []MediaType{{Type: "*", Subtype: "*", QValue: 1.0}}
	}

	parts := strings.Split(accept, ",")
	var result []MediaType
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mt, err := ParseMediaType(part)
		if err != nil {
			continue // silently skip malformed entries
		}
		result = append(result, mt)
	}

	if len(result) == 0 {
		return []MediaType{{Type: "*", Subtype: "*", QValue: 1.0}}
	}

	// Sort by q-value descending, then specificity descending.
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].QValue != result[j].QValue {
			return result[i].QValue > result[j].QValue
		}
		return Specificity(result[i]) > Specificity(result[j])
	})

	return result
}

// IsJSON reports whether the media type represents JSON content.
// True for: application/json, any type with +json suffix,
// application/ld+json, application/problem+json, etc.
func IsJSON(mt MediaType) bool {
	if mt.Type == "application" && mt.Subtype == "json" {
		return true
	}
	return mt.Suffix == "json"
}

// IsText reports whether the media type represents human-readable text content.
// True for: text/*, application/xml, application/javascript,
// application/x-www-form-urlencoded, any type with +xml suffix.
// False for types that IsJSON returns true for (JSON is handled separately).
func IsText(mt MediaType) bool {
	if IsJSON(mt) {
		return false
	}
	if mt.Type == "text" {
		return true
	}
	if mt.Type == "application" {
		switch mt.Subtype {
		case "xml", "javascript", "x-www-form-urlencoded":
			return true
		}
	}
	if mt.Suffix == "xml" {
		return true
	}
	return false
}

// IsBinary reports whether the media type represents binary content.
// Returns true when neither IsJSON nor IsText returns true, or when
// the media type is empty/unknown. This is the fallback classification.
func IsBinary(mt MediaType) bool {
	return !IsJSON(mt) && !IsText(mt)
}

// MatchesMediaRange reports whether a response Content-Type satisfies an
// Accept media range. Type and subtype are compared; parameters (except q)
// are ignored per ADR-41 Q2 resolution. Supports wildcards: */* matches
// anything, type/* matches any subtype of type.
func MatchesMediaRange(accept, contentType MediaType) bool {
	// Full wildcard matches anything.
	if accept.Type == "*" && accept.Subtype == "*" {
		return true
	}
	// Type must match (case-insensitive, but already lowered by mime.ParseMediaType).
	if !strings.EqualFold(accept.Type, contentType.Type) {
		return false
	}
	// Subtype wildcard matches any subtype of the same type.
	if accept.Subtype == "*" {
		return true
	}
	// Exact subtype match.
	return strings.EqualFold(accept.Subtype, contentType.Subtype)
}

// Specificity returns a specificity score for a media range:
//
//	3 for exact type/subtype (e.g., application/json)
//	2 for subtype wildcard (e.g., application/*)
//	1 for full wildcard (*/* )
//
// Used to rank among multiple matching media ranges in an Accept header.
func Specificity(mt MediaType) int {
	if mt.Type == "*" && mt.Subtype == "*" {
		return 1
	}
	if mt.Subtype == "*" {
		return 2
	}
	return 3
}

// mediaTypeError describes a media type parsing failure.
type mediaTypeError struct {
	input  string
	reason string
}

func (e *mediaTypeError) Error() string {
	return "httptape: invalid media type " + strconv.Quote(e.input) + ": " + e.reason
}
