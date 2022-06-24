package service

import (
	"github.com/AliyunContainerService/image-syncer/pkg/db/models"
	"github.com/AliyunContainerService/image-syncer/pkg/db/tools"

	"gorm.io/gorm"
)

type ImageService struct {
	_db *gorm.DB
}

// create a new image service instance
func NewImageService() *ImageService {
	return &ImageService{_db: tools.GetDBConnection()}
}

func (is *ImageService) InsertImage(image *models.Image) error {
	return is._db.Create(image).Error
}

// acquire all images without sync
func (is *ImageService) GetUnsyncImages() ([]models.Image, error) {
	var unsyncedImages []models.Image
	err := is._db.Model(&models.Image{}).Where("is_sync = ?", false).Find(&unsyncedImages).Error
	return unsyncedImages, err
}

// update image sync status
func (is *ImageService) UpdateImageSyncStatus(image *models.Image, syncStatus bool) error {
	// build basic query (contains source and destination)
	query := is._db.Debug().Model(&models.Image{}).
		Where("source = ? And destination = ?", image.Source, image.Destination)
	// add tag if it is not empty
	if image.Tag != "" {
		query = query.Where("tag = ?", image.Tag)
	}
	image.IsSync = syncStatus
	err := query.Update("is_sync", syncStatus).Error
	return err
}
