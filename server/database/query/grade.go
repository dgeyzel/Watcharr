// Package query has SQL building blocks shared by the admin and public
// watched lists, so both apply exactly the same rules.
package query

import (
	"strings"

	"github.com/sbondCo/Watcharr/database/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GradeNone is the grade filter value for "watched but not yet rated".
const GradeNone = "none"

// gradeBucket puts graded items first, then ungraded ones, and planned items
// last (planned titles have no grade slot, even if a grade is saved).
const gradeBucket = `CASE
	WHEN watcheds.status = 'PLANNED' THEN 2
	WHEN watcheds.grade IS NULL THEN 1
	ELSE 0 END`

// Planned items rank 0 so a saved (hidden) grade never affects their order.
const gradeRank = `CASE
	WHEN watcheds.status = 'PLANNED' THEN 0
	WHEN watcheds.grade = 'S' THEN 6 WHEN watcheds.grade = 'A' THEN 5
	WHEN watcheds.grade = 'B' THEN 4 WHEN watcheds.grade = 'C' THEN 3
	WHEN watcheds.grade = 'D' THEN 2 WHEN watcheds.grade = 'F' THEN 1
	ELSE 0 END`

// OrderByGrade sorts by grade, S to F (or F to S when asc), with ungraded
// items after all graded ones and planned items last in both directions.
func OrderByGrade(db *gorm.DB, asc bool) *gorm.DB {
	return db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: gradeBucket, Raw: true}}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: gradeRank, Raw: true}, Desc: !asc})
}

// FilterGrade keeps items matching any of the given grades. Values are S..F
// (exact, uppercase) or GradeNone ("watched but not yet rated": finished or
// watching with no grade). A letter never matches planned items. Unknown
// values are ignored; if none are valid no filter is applied.
func FilterGrade(db *gorm.DB, values []string) *gorm.DB {
	letters := []entity.Grade{}
	none := false
	for _, raw := range values {
		for _, v := range strings.Split(raw, ",") {
			v = strings.TrimSpace(v)
			if v == GradeNone {
				none = true
			} else if g := entity.Grade(v); g.IsValid() {
				letters = append(letters, g)
			}
		}
	}
	conds := []string{}
	args := []any{}
	if len(letters) > 0 {
		conds = append(conds, "(watcheds.grade IN ? AND watcheds.status != ?)")
		args = append(args, letters, entity.PLANNED)
	}
	if none {
		conds = append(conds, "(watcheds.grade IS NULL AND watcheds.status IN ?)")
		args = append(args, []entity.WatchedStatus{entity.FINISHED, entity.WATCHING})
	}
	if len(conds) == 0 {
		return db
	}
	return db.Where(strings.Join(conds, " OR "), args...)
}
