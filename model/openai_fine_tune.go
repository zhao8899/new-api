package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type OpenAIFineTune struct {
	ID             string         `json:"id" gorm:"type:varchar(64);primaryKey"`
	UserId         int            `json:"user_id" gorm:"index"`
	TokenId        int            `json:"token_id" gorm:"index"`
	Object         string         `json:"object" gorm:"type:varchar(32)"`
	CreatedAt      int64          `json:"created_at" gorm:"bigint;index"`
	UpdatedAt      int64          `json:"updated_at" gorm:"bigint"`
	Model          string         `json:"model" gorm:"type:varchar(128)"`
	TrainingFile   string         `json:"training_file" gorm:"type:varchar(64)"`
	ValidationFile string         `json:"validation_file" gorm:"type:varchar(64)"`
	FineTunedModel string         `json:"fine_tuned_model" gorm:"type:varchar(128)"`
	Status         string         `json:"status" gorm:"type:varchar(32);index"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

type OpenAIFineTuneDTO struct {
	ID             string `json:"id"`
	Object         string `json:"object"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at,omitempty"`
	Model          string `json:"model"`
	TrainingFile   string `json:"training_file"`
	ValidationFile string `json:"validation_file,omitempty"`
	FineTunedModel string `json:"fine_tuned_model,omitempty"`
	Status         string `json:"status"`
}

type OpenAIFineTuneEvent struct {
	ID         string         `json:"id" gorm:"type:varchar(64);primaryKey"`
	FineTuneID string         `json:"fine_tune_id" gorm:"type:varchar(64);index"`
	UserId     int            `json:"user_id" gorm:"index"`
	Object     string         `json:"object" gorm:"type:varchar(32)"`
	CreatedAt  int64          `json:"created_at" gorm:"bigint;index"`
	Level      string         `json:"level" gorm:"type:varchar(16)"`
	Message    string         `json:"message" gorm:"type:text"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

type OpenAIFineTuneEventDTO struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	CreatedAt  int64  `json:"created_at"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	FineTuneID string `json:"fine_tune_id"`
}

func (f *OpenAIFineTune) ToDTO() OpenAIFineTuneDTO {
	return OpenAIFineTuneDTO{
		ID:             f.ID,
		Object:         f.Object,
		CreatedAt:      f.CreatedAt,
		UpdatedAt:      f.UpdatedAt,
		Model:          f.Model,
		TrainingFile:   f.TrainingFile,
		ValidationFile: f.ValidationFile,
		FineTunedModel: f.FineTunedModel,
		Status:         f.Status,
	}
}

func (e *OpenAIFineTuneEvent) ToDTO() OpenAIFineTuneEventDTO {
	return OpenAIFineTuneEventDTO{
		ID:         e.ID,
		Object:     e.Object,
		CreatedAt:  e.CreatedAt,
		Level:      e.Level,
		Message:    e.Message,
		FineTuneID: e.FineTuneID,
	}
}

func CreateOpenAIFineTune(userId int, tokenId int, trainingFile string, validationFile string, modelName string) (*OpenAIFineTune, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	trainingFile = strings.TrimSpace(trainingFile)
	modelName = strings.TrimSpace(modelName)
	validationFile = strings.TrimSpace(validationFile)
	if trainingFile == "" {
		return nil, errors.New("training file is required")
	}
	if modelName == "" {
		return nil, errors.New("model is required")
	}

	now := common.GetTimestamp()
	fineTune := &OpenAIFineTune{
		ID:             "ft-" + common.GetTimeString() + common.GetRandomString(6),
		UserId:         userId,
		TokenId:        tokenId,
		Object:         "fine-tune",
		CreatedAt:      now,
		UpdatedAt:      now,
		Model:          modelName,
		TrainingFile:   trainingFile,
		ValidationFile: validationFile,
		Status:         "pending",
	}
	if err := DB.Create(fineTune).Error; err != nil {
		return nil, err
	}
	if err := appendOpenAIFineTuneEvent(fineTune.ID, userId, "info", "Fine-tune job created"); err != nil {
		return nil, err
	}
	return fineTune, nil
}

func ListOpenAIFineTunesByUserID(userId int) ([]OpenAIFineTune, error) {
	var items []OpenAIFineTune
	err := DB.Where("user_id = ?", userId).Order("created_at desc").Find(&items).Error
	return items, err
}

func GetOpenAIFineTuneByIDAndUserID(id string, userId int) (*OpenAIFineTune, error) {
	var item OpenAIFineTune
	err := DB.Where("id = ? AND user_id = ?", strings.TrimSpace(id), userId).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func CancelOpenAIFineTuneByIDAndUserID(id string, userId int) (*OpenAIFineTune, error) {
	item, err := GetOpenAIFineTuneByIDAndUserID(id, userId)
	if err != nil {
		return nil, err
	}
	if item.Status != "cancelled" {
		item.Status = "cancelled"
		item.UpdatedAt = common.GetTimestamp()
		if err := DB.Model(item).Updates(map[string]any{
			"status":     item.Status,
			"updated_at": item.UpdatedAt,
		}).Error; err != nil {
			return nil, err
		}
		if err := appendOpenAIFineTuneEvent(item.ID, userId, "info", "Fine-tune job cancelled"); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func ListOpenAIFineTuneEventsByFineTuneIDAndUserID(id string, userId int) ([]OpenAIFineTuneEvent, error) {
	var items []OpenAIFineTuneEvent
	err := DB.Where("fine_tune_id = ? AND user_id = ?", strings.TrimSpace(id), userId).Order("created_at desc, id desc").Find(&items).Error
	return items, err
}

func appendOpenAIFineTuneEvent(fineTuneID string, userId int, level string, message string) error {
	event := &OpenAIFineTuneEvent{
		ID:         "ftevent-" + common.GetTimeString() + common.GetRandomString(6),
		FineTuneID: fineTuneID,
		UserId:     userId,
		Object:     "fine-tune-event",
		CreatedAt:  common.GetTimestamp(),
		Level:      strings.TrimSpace(level),
		Message:    strings.TrimSpace(message),
	}
	return DB.Create(event).Error
}
