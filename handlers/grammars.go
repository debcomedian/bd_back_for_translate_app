package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Grammars struct {
	ID            int    `gorm:"primaryKey;column:id"   json:"id"`
	TitleRu       string `gorm:"column:title_ru"        json:"title_ru"`
	TitleEn       string `gorm:"column:title_en"        json:"title_en"`
	TitleDe       string `gorm:"column:title_de"        json:"title_de"`
	DescriptionRu string `gorm:"column:description_ru"  json:"description_ru"`
	DescriptionEn string `gorm:"column:description_en"  json:"description_en"`
	DescriptionDe string `gorm:"column:description_de"  json:"description_de"`
	Language      string `gorm:"column:language"        json:"language"`
}

type GrammarRules struct {
	ID                int    `gorm:"primaryKey;column:id"              json:"id"`
	GrammarID         int    `gorm:"column:grammar_id;index"           json:"grammar_id"`
	RuleNameRu        string `gorm:"column:rule_name_ru"               json:"rule_name_ru"`
	RuleNameEn        string `gorm:"column:rule_name_en"               json:"rule_name_en"`
	RuleNameDe        string `gorm:"column:rule_name_de"               json:"rule_name_de"`
	RuleDescriptionRu string `gorm:"column:rule_description_ru"        json:"rule_description_ru"`
	RuleDescriptionEn string `gorm:"column:rule_description_en"        json:"rule_description_en"`
	RuleDescriptionDe string `gorm:"column:rule_description_de"        json:"rule_description_de"`
}

type GrammarExamples struct {
	ID        int    `gorm:"primaryKey;column:id"    json:"id"`
	RuleID    int    `gorm:"column:rule_id;index"    json:"rule_id"`
	ExampleRu string `gorm:"column:example_ru"       json:"example_ru"`
	ExampleEn string `gorm:"column:example_en"       json:"example_en"`
	ExampleDe string `gorm:"column:example_de"       json:"example_de"`
}

type GrammarExceptions struct {
	ID            int    `gorm:"primaryKey;column:id"       json:"id"`
	RuleID        int    `gorm:"column:rule_id;index"       json:"rule_id"`
	DescriptionRu string `gorm:"column:description_ru"      json:"description_ru"`
	DescriptionEn string `gorm:"column:description_en"      json:"description_en"`
	DescriptionDe string `gorm:"column:description_de"      json:"description_de"`
	ExplanationRu string `gorm:"column:explanation_ru"      json:"explanation_ru"`
	ExplanationEn string `gorm:"column:explanation_en"      json:"explanation_en"`
	ExplanationDe string `gorm:"column:explanation_de"      json:"explanation_de"`
}

func GetGrammars(c *gin.Context) {
	var g []Grammars
	if err := DB.Find(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, g)

}

func CreateGrammars(c *gin.Context) {
	var g Grammars
	if err := c.BindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := DB.Create(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, g)
}

func UpdateGrammars(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var g Grammars
	if err = DB.First(&g, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "grammar not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var input Grammars
	if err = c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err = DB.Session(&gorm.Session{FullSaveAssociations: true}).
		Model(&Grammars{ID: id}).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, input)
}

func DeleteGrammars(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err = DB.Delete(&Grammars{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ************** GrammarRules **************

func GetGrammarRules(c *gin.Context) {
	var items []GrammarRules
	if err := DB.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func CreateGrammarRules(c *gin.Context) {
	var item GrammarRules
	if !bindJSON(c, &item) {
		return
	}
	if err := DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func UpdateGrammarRules(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}

	var item GrammarRules
	if err := DB.First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var input GrammarRules
	if !bindJSON(c, &input) {
		return
	}
	input.ID = id

	if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).
		Model(&item).
		Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, input)
}

func DeleteGrammarRules(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	if err := DB.Delete(&GrammarRules{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ************** GrammarExamples **************

func GetGrammarExamples(c *gin.Context) {
	var items []GrammarExamples
	if err := DB.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func CreateGrammarExamples(c *gin.Context) {
	var item GrammarExamples
	if !bindJSON(c, &item) {
		return
	}
	if err := DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func UpdateGrammarExamples(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}

	var item GrammarExamples
	if err := DB.First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var input GrammarExamples
	if !bindJSON(c, &input) {
		return
	}
	input.ID = id

	if err := DB.Model(&item).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, input)
}

func DeleteGrammarExamples(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	if err := DB.Delete(&GrammarExamples{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ************** GrammarExceptions **************

func GetGrammarExceptions(c *gin.Context) {
	var items []GrammarExceptions
	if err := DB.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func CreateGrammarExceptions(c *gin.Context) {
	var item GrammarExceptions
	if !bindJSON(c, &item) {
		return
	}
	if err := DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func UpdateGrammarExceptions(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}

	var item GrammarExceptions
	if err := DB.First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var input GrammarExceptions
	if !bindJSON(c, &input) {
		return
	}
	input.ID = id

	if err := DB.Model(&item).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, input)
}

func DeleteGrammarExceptions(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	if err := DB.Delete(&GrammarExceptions{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
