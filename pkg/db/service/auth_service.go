package service

import (
	"github.com/AliyunContainerService/image-syncer/pkg/db/models"
	"github.com/AliyunContainerService/image-syncer/pkg/db/tools"

	"gorm.io/gorm"
)

type AuthService struct {
	_db *gorm.DB
}

func NewAuthService() *AuthService {
	return &AuthService{_db: tools.GetDBConnection()}
}

func (as *AuthService) InsertAuth(auth *models.Auth) error {
	// 0 represents unspecified, so specified auto
	return as._db.Create(auth).Error
}

func (as *AuthService) GetAllAuth() ([]models.Auth, error) {
	var (
		auths []models.Auth
		err   error
	)
	err = as._db.Model(&models.Auth{}).Find(&auths).Error
	return auths, err
}

func (as *AuthService) UpdatePassword(auth *models.Auth, password string) (err error) {
	// figure out whether auth exists based on id or registry
	query := as._db.Model(&models.Auth{})
	if auth.ID != 0 {
		query.Where("id = ?", auth.ID)
	} else if auth.Registry != "" {
		query.Where("registry = ?", auth.Registry)
	} else {
		return gorm.ErrInvalidField
	}
	// find the first repo
	err = query.First(auth).Error
	if err != nil {
		return err
	}
	// update password
	err = as._db.Model(&models.Auth{}).Where(auth).Update("password", password).Error
	return err
}
