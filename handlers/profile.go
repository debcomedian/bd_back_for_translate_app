package handlers

import (
	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var user models.User
	if err := DB.First(&user, uid).Error; err != nil {
		handleDBErr(c, err)
		return
	}

	respondOK(c, publicUserView{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Level:    user.Level,
	})
}

func GetUserWordProgress(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var list []models.UserWordProgress
	if err := DB.Where("user_id = ?", uid).Order("word_id ASC").Find(&list).Error; err != nil {
		handleDBErr(c, err)
		return
	}
	respondOK(c, list)
}
