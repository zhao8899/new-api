package model

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type OpenAIFile struct {
	Id          string         `json:"id" gorm:"type:varchar(64);primaryKey"`
	UserId      int            `json:"user_id" gorm:"index"`
	TokenId     int            `json:"token_id" gorm:"index"`
	Object      string         `json:"object" gorm:"type:varchar(16)"`
	Bytes       int64          `json:"bytes"`
	CreatedAt   int64          `json:"created_at" gorm:"bigint;index"`
	Filename    string         `json:"filename" gorm:"type:varchar(255)"`
	Purpose     string         `json:"purpose" gorm:"type:varchar(64);index"`
	ContentData string         `json:"-" gorm:"type:text"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type OpenAIFileBrief struct {
	Id        string `json:"id"`
	Object    string `json:"object"`
	Bytes     int64  `json:"bytes"`
	CreatedAt int64  `json:"created_at"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
}

func (f *OpenAIFile) ToBrief() OpenAIFileBrief {
	return OpenAIFileBrief{
		Id:        f.Id,
		Object:    f.Object,
		Bytes:     f.Bytes,
		CreatedAt: f.CreatedAt,
		Filename:  f.Filename,
		Purpose:   f.Purpose,
	}
}

func (f *OpenAIFile) ContentBytes() ([]byte, error) {
	if strings.TrimSpace(f.ContentData) == "" {
		return nil, nil
	}
	data, err := base64.StdEncoding.DecodeString(f.ContentData)
	if err != nil {
		return nil, fmt.Errorf("decode file content failed: %w", err)
	}
	return data, nil
}

func CreateOpenAIFile(userId int, tokenId int, filename string, purpose string, content []byte) (*OpenAIFile, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	filename = strings.TrimSpace(filename)
	purpose = strings.TrimSpace(purpose)
	if filename == "" {
		return nil, errors.New("filename is required")
	}
	if purpose == "" {
		return nil, errors.New("purpose is required")
	}

	file := &OpenAIFile{
		Id:          "file-" + common.GetTimeString() + common.GetRandomString(6),
		UserId:      userId,
		TokenId:     tokenId,
		Object:      "file",
		Bytes:       int64(len(content)),
		CreatedAt:   common.GetTimestamp(),
		Filename:    filename,
		Purpose:     purpose,
		ContentData: base64.StdEncoding.EncodeToString(content),
	}
	if err := DB.Create(file).Error; err != nil {
		return nil, err
	}
	return file, nil
}

func ListOpenAIFilesByUserId(userId int) ([]OpenAIFile, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	var files []OpenAIFile
	err := DB.Where("user_id = ?", userId).Order("created_at desc").Find(&files).Error
	return files, err
}

func GetOpenAIFileByIDAndUserID(fileId string, userId int) (*OpenAIFile, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	var file OpenAIFile
	err := DB.Where("id = ? AND user_id = ?", strings.TrimSpace(fileId), userId).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func DeleteOpenAIFileByIDAndUserID(fileId string, userId int) error {
	file, err := GetOpenAIFileByIDAndUserID(fileId, userId)
	if err != nil {
		return err
	}
	return DB.Delete(file).Error
}
