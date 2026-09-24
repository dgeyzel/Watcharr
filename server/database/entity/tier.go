package entity

import (
	"encoding/json"
	"errors"
)

// Tier is the owner's S-F tier for a title. It is a separate score
// from the numeric Rating (which stays admin only), and is never derived from
// it. Stored as nullable text, null means "not tiered".
type Tier string

const (
	TierS Tier = "S"
	TierA Tier = "A"
	TierB Tier = "B"
	TierC Tier = "C"
	TierD Tier = "D"
	TierF Tier = "F"
)

// Tiers from highest to lowest.
var Tiers = []Tier{TierS, TierA, TierB, TierC, TierD, TierF}

var ErrInvalidTier = errors.New("tier must be one of S, A, B, C, D, F (uppercase) or null")

// IsValid reports if g is exactly one of S, A, B, C, D, F. Lowercase and any
// other value (A+, E, ...) is invalid.
func (g Tier) IsValid() bool {
	switch g {
	case TierS, TierA, TierB, TierC, TierD, TierF:
		return true
	}
	return false
}

// Rank orders tiers for sorting: S=6 ... F=1, anything else 0.
func (g Tier) Rank() int {
	switch g {
	case TierS:
		return 6
	case TierA:
		return 5
	case TierB:
		return 4
	case TierC:
		return 3
	case TierD:
		return 2
	case TierF:
		return 1
	}
	return 0
}

// RankOf is Rank for a possibly nil tier (nil = 0, not tiered).
func RankOf(g *Tier) int {
	if g == nil {
		return 0
	}
	return g.Rank()
}

// ParseTier parses user input. "" clears the tier (returns nil), invalid
// values return ErrInvalidTier.
func ParseTier(s string) (*Tier, error) {
	if s == "" {
		return nil, nil
	}
	g := Tier(s)
	if !g.IsValid() {
		return nil, ErrInvalidTier
	}
	return &g, nil
}

// OptionalTier is a tier in a json request that tells apart "not
// provided" (Set=false), "clear it" (null or "", Set=true, Value=nil) and a
// new tier. Invalid values fail json decoding with ErrInvalidTier.
type OptionalTier struct {
	Set   bool
	Value *Tier
}

func (o *OptionalTier) UnmarshalJSON(b []byte) error {
	o.Set = true
	o.Value = nil
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return ErrInvalidTier
	}
	g, err := ParseTier(s)
	if err != nil {
		return err
	}
	o.Value = g
	return nil
}
