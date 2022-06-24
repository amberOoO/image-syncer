package models

import (
	"gorm.io/gorm"
)

type Auth struct {
	/* record info */
	gorm.Model

	/* Repo authentication info */
	Registry string `gorm:"column:registry"`
	Username string `gorm:"column:username"`
	Password string `gorm:"column:password"`
	Insecure bool   `gorm:"default:true"`
}

// cycle used
// func (a *Auth) FormatToClient() client.Auth{
// 	return client.Auth{
// 		Username: a.Username,
// 		Password: a.Password,
// 		Insecure: a.Insecure,
// 	}
// }
