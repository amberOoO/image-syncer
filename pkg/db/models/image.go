package models

import (
	"gorm.io/gorm"
	// "image-syncer/pkg/client"
)

type Image struct {
	/* record info */
	gorm.Model

	/* image info */
	Source      string `gorm:"column:source"` // simple url with no tag! Cause tag will be splited to pairs in process, which if difficult to specifed it's status
	Tag         string `gorm:"column:tag"`
	Destination string `gorm:"column:destination"`

	/* whether an image is already synced */
	IsSync bool `gorm:"column:is_sync;default:false"`
}

func (im *Image) GetSourceWithTag() string {
	return im.Source + ":" + im.Tag
}

// func (im *Image) FormatToClient() client.URLPair{

// }
