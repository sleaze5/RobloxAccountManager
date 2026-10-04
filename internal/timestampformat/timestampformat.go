// Package timestampformat parses user-defined timestamp formats. A format is
// plain text in which each token starts with "$" and "$$" writes a literal "$".
package timestampformat

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const MaxLength = 256

type Kind string

const (
	KindText           Kind = "text"
	KindYear           Kind = "YYYY"
	KindYearShort      Kind = "YY"
	KindMonthName      Kind = "MMMM"
	KindMonthNameShort Kind = "MMM"
	KindMonthPadded    Kind = "MM"
	KindMonth          Kind = "M"
	KindDayPadded      Kind = "DD"
	KindDay            Kind = "D"
	KindWeekday        Kind = "dddd"
	KindWeekdayShort   Kind = "ddd"
	KindHour24Padded   Kind = "HH"
	KindHour24         Kind = "H"
	KindHour12Padded   Kind = "hh"
	KindHour12         Kind = "h"
	KindMinutePadded   Kind = "mm"
	KindMinute         Kind = "m"
	KindSecondPadded   Kind = "ss"
	KindSecond         Kind = "s"
	KindMillisecond    Kind = "SSS"
	KindMeridiem       Kind = "A"
	KindMeridiemLower  Kind = "a"
	KindRelative       Kind = "relative"
)

// tokens is sorted longest first, so "$MMMM" matches before "$MMM", "$MM", and "$M".
var tokens = func() []Kind {
	list := []Kind{
		KindYear, KindYearShort, KindMonthName, KindMonthNameShort, KindMonthPadded, KindMonth,
		KindDayPadded, KindDay, KindWeekday, KindWeekdayShort, KindHour24Padded, KindHour24,
		KindHour12Padded, KindHour12, KindMinutePadded, KindMinute, KindSecondPadded, KindSecond,
		KindMillisecond, KindMeridiem, KindMeridiemLower, KindRelative,
	}
	slices.SortStableFunc(list, func(a, b Kind) int { return cmp.Compare(len(b), len(a)) })
	return list
}()

type Segment struct {
	Kind Kind   `json:"kind"`
	Text string `json:"text,omitempty"`
}

type Format struct {
	Source   string    `json:"source"`
	Segments []Segment `json:"segments"`
}

// ParseError is shown to the user, so its message is a complete sentence.
type ParseError struct {
	Message string
}

func (err *ParseError) Error() string { return err.Message }

func Parse(source string) (Format, error) {
	if strings.TrimSpace(source) == "" {
		return Format{}, &ParseError{Message: "Enter a format."}
	}
	if utf8.RuneCountInString(source) > MaxLength {
		return Format{}, &ParseError{Message: fmt.Sprintf("Use %d characters or fewer.", MaxLength)}
	}
	segments := []Segment{}
	var text strings.Builder
	for index := 0; index < len(source); {
		next := strings.IndexByte(source[index:], '$')
		if next < 0 {
			text.WriteString(source[index:])
			break
		}
		text.WriteString(source[index : index+next])
		index += next + 1
		if strings.HasPrefix(source[index:], "$") {
			text.WriteByte('$')
			index++
			continue
		}
		kind, ok := matchToken(source[index:])
		if !ok {
			return Format{}, &ParseError{Message: fmt.Sprintf(
				"Unknown token at character %d. Use $$ to show a $ sign.",
				utf8.RuneCountInString(source[:index]),
			)}
		}
		segments = flushText(segments, &text)
		segments = append(segments, Segment{Kind: kind})
		index += len(kind)
	}
	return Format{Source: source, Segments: flushText(segments, &text)}, nil
}

func matchToken(rest string) (Kind, bool) {
	for _, kind := range tokens {
		if strings.HasPrefix(rest, string(kind)) {
			return kind, true
		}
	}
	return "", false
}

func flushText(segments []Segment, text *strings.Builder) []Segment {
	if text.Len() == 0 {
		return segments
	}
	segments = append(segments, Segment{Kind: KindText, Text: text.String()})
	text.Reset()
	return segments
}
