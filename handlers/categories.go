package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Category struct {
	ID         int    `gorm:"primaryKey;column:id"    json:"id"`
	NameEn     string `gorm:"column:name_en"          json:"name_en"`
	NameRu     string `gorm:"column:name_ru"          json:"name_ru"`
	NameDe     string `gorm:"column:name_de"          json:"name_de"`
	TypeName   string `gorm:"column:type_name"        json:"type_name"`
	TypeNameRu string `gorm:"column:type_name_ru"     json:"type_name_ru"`
	TypeNameDe string `gorm:"column:type_name_de"     json:"type_name_de"`
	Entity     string `gorm:"column:entity"           json:"entity"`
}

func GetCategories(c *gin.Context) {
	var list []Category
	if err := DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateCategory(c *gin.Context) {
	var obj Category
	if !bindJSON(c, &obj) {
		return
	}
	if err := DB.Create(&obj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, obj)
}

func UpdateCategory(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}

	var obj Category
	if err := DB.First(&obj, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if !bindJSON(c, &obj) {
		return
	}
	obj.ID = id

	if err := DB.Save(&obj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, obj)
}

func DeleteCategory(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	if err := DB.Delete(&Category{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
