package models

type Trailer struct {
    MovieBase
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

