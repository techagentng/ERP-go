package models

type Trailer struct {
    MovieBase
    TrailerID    uint   `gorm:"primaryKey" json:"trailer_id"`
    LogLine      string `gorm:"type:text"`
    ProductYear  string `gorm:"size:4"`
    Star1        string `gorm:"type:text;not null"`
    Star2        string `gorm:"type:text;not null"`
    Star3        string `gorm:"type:text;not null"`
    VideoURLs    string `gorm:"type:text"` 
    PictureURLs  string `gorm:"type:text"` 
	UserID       uint     `json:"user_id"`
	User         User     `gorm:"foreignKey:UserID" json:"user"`
}

type UploadProgress struct {
	TotalFiles      int     `json:"total_files"`
	UploadedFiles   int     `json:"uploaded_files"`
	Percentage      float64 `json:"percentage"`
}
