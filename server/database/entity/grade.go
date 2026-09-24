package entity

import (
	"encoding/json"
	"errors"
)

// Grade is the owner's S-F letter grade for a title. It is a separate score
// from the numeric Rating (which stays admin only), and is never derived from
// it. Stored as nullable text, null means "not graded".
type Grade string

const (
	GradeS Grade = "S"
	GradeA Grade = "A"
	GradeB Grade = "B"
	GradeC Grade = "C"
	GradeD Grade = "D"
	GradeF Grade = "F"
)

// Grades from highest to lowest.
var Grades = []Grade{GradeS, GradeA, GradeB, GradeC, GradeD, GradeF}

var ErrInvalidGrade = errors.New("grade must be one of S, A, B, C, D, F (uppercase) or null")

// IsValid reports if g is exactly one of S, A, B, C, D, F. Lowercase and any
// other value (A+, E, ...) is invalid.
func (g Grade) IsValid() bool {
	switch g {
	case GradeS, GradeA, GradeB, GradeC, GradeD, GradeF:
		return true
	}
	return false
}

// Rank orders grades for sorting: S=6 ... F=1, anything else 0.
func (g Grade) Rank() int {
	switch g {
	case GradeS:
		return 6
	case GradeA:
		return 5
	case GradeB:
		return 4
	case GradeC:
		return 3
	case GradeD:
		return 2
	case GradeF:
		return 1
	}
	return 0
}

// RankOf is Rank for a possibly nil grade (nil = 0, not graded).
func RankOf(g *Grade) int {
	if g == nil {
		return 0
	}
	return g.Rank()
}

// ParseGrade parses user input. "" clears the grade (returns nil), invalid
// values return ErrInvalidGrade.
func ParseGrade(s string) (*Grade, error) {
	if s == "" {
		return nil, nil
	}
	g := Grade(s)
	if !g.IsValid() {
		return nil, ErrInvalidGrade
	}
	return &g, nil
}

// OptionalGrade is a grade in a json request that tells apart "not
// provided" (Set=false), "clear it" (null or "", Set=true, Value=nil) and a
// new grade. Invalid values fail json decoding with ErrInvalidGrade.
type OptionalGrade struct {
	Set   bool
	Value *Grade
}

func (o *OptionalGrade) UnmarshalJSON(b []byte) error {
	o.Set = true
	o.Value = nil
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return ErrInvalidGrade
	}
	g, err := ParseGrade(s)
	if err != nil {
		return err
	}
	o.Value = g
	return nil
}
