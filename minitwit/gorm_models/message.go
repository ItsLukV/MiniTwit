package gorm_models

type Message struct {
	Message_id int `gorm:"primaryKey"`
	Author_id  uint
	Text       string
	Pub_date   string
	Flagged    int
}
