package model

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const (
	logoImagesOptionKey = "LogoImages"
	maxLogoImages       = 10
)

var (
	ErrLogoImageExists   = errors.New("已存在同名图片，请重命名后上传，或先删除已有图片")
	ErrLogoImageLimit    = errors.New("最多保存 10 张徽标图片，请先删除已有图片")
	ErrLogoImageNotFound = errors.New("图片不存在，可能已被删除")
	errLogoImageLibrary  = errors.New("徽标图片配置必须为 JSON 对象")
	logoImageWriteMutex  sync.Mutex
	logoImageSnapshot    atomic.Pointer[logoImageLibrary]
)

type LogoImage struct {
	Content    []byte    `json:"content"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type logoImageLibrary map[string]LogoImage

func ListLogoImageNames() []string {
	names := make([]string, 0)
	if images := logoImageSnapshot.Load(); images != nil {
		for name := range *images {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// GetLogoImage returns a snapshot whose content must be treated as read-only.
func GetLogoImage(name string) (LogoImage, bool) {
	if images := logoImageSnapshot.Load(); images != nil {
		image, ok := (*images)[name]
		return image, ok
	}
	return LogoImage{}, false
}

func AddLogoImage(name string, content []byte) error {
	return changeLogoImages(func(images logoImageLibrary) error {
		if _, exists := images[name]; exists {
			return ErrLogoImageExists
		}
		if len(images) >= maxLogoImages {
			return ErrLogoImageLimit
		}
		images[name] = LogoImage{Content: content, UploadedAt: time.Now().UTC()}
		return nil
	})
}

func DeleteLogoImage(name string) error {
	return changeLogoImages(func(images logoImageLibrary) error {
		if _, exists := images[name]; !exists {
			return ErrLogoImageNotFound
		}
		delete(images, name)
		return nil
	})
}

func changeLogoImages(change func(logoImageLibrary) error) error {
	logoImageWriteMutex.Lock()
	defer logoImageWriteMutex.Unlock()

	var images logoImageLibrary
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Option{
			Key: logoImagesOptionKey, Value: "{}",
		}).Error; err != nil {
			return err
		}
		var option Option
		if err := lockForUpdate(tx).Where(&Option{Key: logoImagesOptionKey}).First(&option).Error; err != nil {
			return err
		}
		var err error
		images, err = decodeLogoImageLibrary(option.Value)
		if err != nil {
			return err
		}
		if err := change(images); err != nil {
			return err
		}
		data, err := common.Marshal(images)
		if err != nil {
			return err
		}
		// SQL logging can interpolate the entire Base64 image library.
		return tx.Session(&gorm.Session{Logger: logger.Discard}).Model(&option).Update("value", string(data)).Error
	})
	if err == nil {
		logoImageSnapshot.Store(&images)
	}
	return err
}

func decodeLogoImageLibrary(value string) (logoImageLibrary, error) {
	var images logoImageLibrary
	if err := common.UnmarshalJsonStr(value, &images); err != nil {
		return nil, err
	}
	if images == nil {
		return nil, errLogoImageLibrary
	}
	return images, nil
}

// Load the image snapshot in the same order as local writes so a periodic
// reload cannot replace a just-committed image library with an older one.
func loadOptionsWithLogoImages() ([]*Option, error) {
	logoImageWriteMutex.Lock()
	defer logoImageWriteMutex.Unlock()

	options, err := AllOption()
	if err != nil {
		return nil, err
	}
	for _, option := range options {
		if option.Key == logoImagesOptionKey {
			images, err := decodeLogoImageLibrary(option.Value)
			if err != nil {
				common.SysError("读取徽标图片配置失败: " + err.Error())
				return options, nil
			}
			logoImageSnapshot.Store(&images)
			return options, nil
		}
	}
	logoImageSnapshot.Store(nil)
	return options, nil
}
