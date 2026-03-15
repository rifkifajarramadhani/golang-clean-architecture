package models

type User struct {
	ID       int `gorm:"primaryKey"`
	Username string
	Email    string `gorm:"unique"`
	Password string
}
