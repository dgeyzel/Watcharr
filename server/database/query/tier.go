// Package query has SQL building blocks shared by the admin and public
// watched lists, so both apply exactly the same rules.
package query

import (
	"strings"

	"github.com/sbondCo/Watcharr/database/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TierNone is the tier filter value for "watched but not yet rated".
const TierNone = "none"

// tierBucket puts tiered items first, then untiered ones, and planned items
// last (planned titles have no tier slot, even if a tier is saved).
const tierBucket = `CASE
	WHEN watcheds.status = 'PLANNED' THEN 2
	WHEN watcheds.tier IS NULL THEN 1
	ELSE 0 END`

// Planned items rank 0 so a saved (hidden) tier never affects their order.
const tierRank = `CASE
	WHEN watcheds.status = 'PLANNED' THEN 0
	WHEN watcheds.tier = 'S' THEN 6 WHEN watcheds.tier = 'A' THEN 5
	WHEN watcheds.tier = 'B' THEN 4 WHEN watcheds.tier = 'C' THEN 3
	WHEN watcheds.tier = 'D' THEN 2 WHEN watcheds.tier = 'F' THEN 1
	ELSE 0 END`

// OrderByTier sorts by tier, S to F (or F to S when asc), with untiered
// items after all tiered ones and planned items last in both directions.
func OrderByTier(db *gorm.DB, asc bool) *gorm.DB {
	return db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: tierBucket, Raw: true}}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: tierRank, Raw: true}, Desc: !asc})
}

// FilterTier keeps items matching any of the given tiers. Values are S..F
// (exact, uppercase) or TierNone ("watched but not yet rated": finished or
// watching with no tier). A letter never matches planned items. Unknown
// values are ignored; if none are valid no filter is applied.
func FilterTier(db *gorm.DB, values []string) *gorm.DB {
	letters := []entity.Tier{}
	none := false
	for _, raw := range values {
		for _, v := range strings.Split(raw, ",") {
			v = strings.TrimSpace(v)
			if v == TierNone {
				none = true
			} else if g := entity.Tier(v); g.IsValid() {
				letters = append(letters, g)
			}
		}
	}
	conds := []string{}
	args := []any{}
	if len(letters) > 0 {
		conds = append(conds, "(watcheds.tier IN ? AND watcheds.status != ?)")
		args = append(args, letters, entity.PLANNED)
	}
	if none {
		conds = append(conds, "(watcheds.tier IS NULL AND watcheds.status IN ?)")
		args = append(args, []entity.WatchedStatus{entity.FINISHED, entity.WATCHING})
	}
	if len(conds) == 0 {
		return db
	}
	return db.Where(strings.Join(conds, " OR "), args...)
}
