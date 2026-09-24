package entity

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestTierIsValid(t *testing.T) {
	for _, g := range []string{"S", "A", "B", "C", "D", "F"} {
		if !Tier(g).IsValid() {
			t.Errorf("%q should be valid", g)
		}
	}
	for _, g := range []string{"", "a", "s", "f", "E", "A+", "B-", "SS", " A", "A ", "AA", "1", "none"} {
		if Tier(g).IsValid() {
			t.Errorf("%q should be invalid", g)
		}
	}
}

func TestTierRankOrder(t *testing.T) {
	prev := 7
	for _, g := range Tiers {
		if r := g.Rank(); r >= prev || r <= 0 {
			t.Fatalf("rank of %s = %d, expected strictly below %d and above 0", g, r, prev)
		} else {
			prev = r
		}
	}
	if TierS.Rank() != 6 || TierF.Rank() != 1 || Tier("x").Rank() != 0 || RankOf(nil) != 0 {
		t.Fatal("unexpected rank values")
	}
}

func TestParseTier(t *testing.T) {
	if g, err := ParseTier(""); g != nil || err != nil {
		t.Fatalf(`ParseTier("") = %v, %v; want nil, nil`, g, err)
	}
	if g, err := ParseTier("B"); err != nil || g == nil || *g != TierB {
		t.Fatalf(`ParseTier("B") = %v, %v`, g, err)
	}
	for _, s := range []string{"b", "A+", "E"} {
		if _, err := ParseTier(s); !errors.Is(err, ErrInvalidTier) {
			t.Errorf("ParseTier(%q) err = %v, want ErrInvalidTier", s, err)
		}
	}
}

func TestOptionalTierJSON(t *testing.T) {
	type req struct {
		Tier OptionalTier `json:"tier"`
	}
	cases := []struct {
		body    string
		set     bool
		value   string // "" means nil
		wantErr bool
	}{
		{`{}`, false, "", false},
		{`{"tier":null}`, true, "", false},
		{`{"tier":""}`, true, "", false},
		{`{"tier":"A"}`, true, "A", false},
		{`{"tier":"S"}`, true, "S", false},
		{`{"tier":"a"}`, false, "", true},
		{`{"tier":"A+"}`, false, "", true},
		{`{"tier":"E"}`, false, "", true},
		{`{"tier":5}`, false, "", true},
	}
	for _, c := range cases {
		var r req
		err := json.Unmarshal([]byte(c.body), &r)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s: expected error", c.body)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.body, err)
			continue
		}
		got := ""
		if r.Tier.Value != nil {
			got = string(*r.Tier.Value)
		}
		if r.Tier.Set != c.set || got != c.value {
			t.Errorf("%s: got set=%v value=%q, want set=%v value=%q", c.body, r.Tier.Set, got, c.set, c.value)
		}
	}
}
