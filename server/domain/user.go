package domain

// User
type (
	UserBioUpdateRequest struct {
		NewBio string `json:"newBio" binding:"max=128"`
	}
)
