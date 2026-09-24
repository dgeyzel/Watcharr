package addedtocontent

import (
	"log/slog"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/util"
	"gorm.io/gorm"
)

type Addable interface {
	GetId() int
	GetMediaType() util.SupportedMedia
}

type IdToTypePair struct {
	Id   int
	Type util.SupportedMedia
}

type WatchedProvider interface {
	GetWatchedItemBySupportedMediaId(userId uint, id uint, t util.SupportedMedia) (entity.Watched, error)
	GetWatchedItemsBySupportedMediaIds(userId uint, c []IdToTypePair) ([]entity.Watched, error)
}

type AddListCall[S Addable] struct {
	s     []S
	addCb func(i int, w *entity.Watched)
}

func NewAddListCall[S Addable](s []S, addCb func(i int, w *entity.Watched)) *AddListCall[S] {
	return &AddListCall[S]{
		s,
		addCb,
	}
}

// A helper for adding `Watched` data to structs of generic data.
// The actual adding of watched data will be handled by the caller
// through the `addCb` callback function.
func AddList[S Addable](
	wp WatchedProvider,
	userId uint,
	s []S,
	addCb func(i int, w *entity.Watched),
) error {
	if len(s) <= 0 {
		slog.Debug("AddList: 's' is empty.")
		return nil
	}
	contentIdAndTypePairs := []IdToTypePair{}
	for _, v := range s {
		contentIdAndTypePairs = append(contentIdAndTypePairs, IdToTypePair{
			v.GetId(),
			v.GetMediaType(),
		})
	}
	if ws, err := wp.GetWatchedItemsBySupportedMediaIds(userId, contentIdAndTypePairs); err == nil {
		for _, v := range ws {
			for i, vv := range s {
				if v.Content != nil &&
					vv.GetMediaType() == v.Content.GetTypeSupportedMedia() &&
					vv.GetId() == v.Content.TmdbID {
					addCb(i, &v)
				}
			}
		}
	} else {
		slog.Error("AddList: Getting watched items by tmdbIds failed!", "error", err)
		return err
	}
	return nil
}

func Add[S Addable](
	wp WatchedProvider,
	userId uint,
	s S,
	addCb func(w *entity.Watched),
) error {
	watchedEntry, err := wp.GetWatchedItemBySupportedMediaId(
		userId,
		uint(s.GetId()),
		s.GetMediaType(),
	)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	addCb(&watchedEntry)
	return nil
}

func AddSingularAndList[S Addable, S2 Addable](
	wp WatchedProvider,
	userId uint,
	s S,
	addCb func(w *entity.Watched),
	list []*AddListCall[S2],
) error {
	err := Add(wp, userId, s, addCb)
	if err != nil {
		slog.Error("AddSingularAndList: Failed to add singular.")
		return err
	}
	for i := range list {
		if err := AddList(wp, userId, list[i].s, list[i].addCb); err != nil {
			slog.Error("AddSingularAndList: An error occurred adding to list.")
			return err
		}
	}
	return nil
}
