package public

import (
	"github.com/sbondCo/Watcharr/database/entity"
	"gorm.io/gorm"
)

// PublicStatuses are the watched statuses visitors can see. HOLD and DROPPED
// are admin only.
var PublicStatuses = []entity.WatchedStatus{entity.FINISHED, entity.WATCHING, entity.PLANNED}

// PublicContentTypes are the content types visitors can see (no games).
var PublicContentTypes = []entity.ContentType{entity.MOVIE, entity.SHOW}

// VisibleWatched is THE definition of a "visible" item, used by every public
// endpoint. An item is visible when it:
//   - belongs to the owner (the admin),
//   - is a movie or tv show,
//   - has a public status (FINISHED, WATCHING or PLANNED),
//   - is not hidden,
//   - is not (soft) deleted (gorm adds this for the Watched model).
//
// Use on a query whose model is entity.Watched.
func VisibleWatched(ownerID uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Where("watcheds.user_id = ?", ownerID).
			Where("watcheds.hidden = ?", false).
			Where("watcheds.status IN ?", PublicStatuses).
			Where("watcheds.content_id IN (?)",
				db.Session(&gorm.Session{NewDB: true}).
					Model(&entity.Content{}).
					Select("id").
					Where("type IN ?", PublicContentTypes))
	}
}
