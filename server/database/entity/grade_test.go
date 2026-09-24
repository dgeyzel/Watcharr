package entity

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestGradeIsValid(t *testing.T) {
	for _, g := range []string{"S", "A", "B", "C", "D", "F"} {
		if !Grade(g).IsValid() {
			t.Errorf("%q should be valid", g)
		}
	}
	for _, g := range []string{"", "a", "s", "f", "E", "A+", "B-", "SS", " A", "A ", "AA", "1", "none"} {
		if Grade(g).IsValid() {
			t.Errorf("%q should be invalid", g)
		}
	}
}

func TestGradeRankOrder(t *testing.T) {
	prev := 7
	for _, g := range Grades {
		if r := g.Rank(); r >= prev || r <= 0 {
			t.Fatalf("rank of %s = %d, expected strictly below %d and above 0", g, r, prev)
		} else {
			prev = r
		}
	}
	if GradeS.Rank() != 6 || GradeF.Rank() != 1 || Grade("x").Rank() != 0 || RankOf(nil) != 0 {
		t.Fatal("unexpected rank values")
	}
}

func TestParseGrade(t *testing.T) {
	if g, err := ParseGrade(""); g != nil || err != nil {
		t.Fatalf(`ParseGrade("") = %v, %v; want nil, nil`, g, err)
	}
	if g, err := ParseGrade("B"); err != nil || g == nil || *g != GradeB {
		t.Fatalf(`ParseGrade("B") = %v, %v`, g, err)
	}
	for _, s := range []string{"b", "A+", "E"} {
		if _, err := ParseGrade(s); !errors.Is(err, ErrInvalidGrade) {
			t.Errorf("ParseGrade(%q) err = %v, want ErrInvalidGrade", s, err)
		}
	}
}

func TestOptionalGradeJSON(t *testing.T) {
	type req struct {
		Grade OptionalGrade `json:"grade"`
	}
	cases := []struct {
		body    string
		set     bool
		value   string // "" means nil
		wantErr bool
	}{
		{`{}`, false, "", false},
		{`{"grade":null}`, true, "", false},
		{`{"grade":""}`, true, "", false},
		{`{"grade":"A"}`, true, "A", false},
		{`{"grade":"S"}`, true, "S", false},
		{`{"grade":"a"}`, false, "", true},
		{`{"grade":"A+"}`, false, "", true},
		{`{"grade":"E"}`, false, "", true},
		{`{"grade":5}`, false, "", true},
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
		if r.Grade.Value != nil {
			got = string(*r.Grade.Value)
		}
		if r.Grade.Set != c.set || got != c.value {
			t.Errorf("%s: got set=%v value=%q, want set=%v value=%q", c.body, r.Grade.Set, got, c.set, c.value)
		}
	}
}
