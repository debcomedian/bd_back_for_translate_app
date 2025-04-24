package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID        int       `gorm:"primaryKey;column:id"    	    json:"id"`
	Username  string    `gorm:"column:username;unique;not null" json:"username"`
	Email     string    `gorm:"column:email;unique;not null"    json:"email"`
	Level     string    `gorm:"column:level;size:2;not null"    json:"level"`
	CreatedAt time.Time `gorm:"column:created_at"               json:"created_at"`
}

func GetUsers(c *gin.Context) {
	var list []User
	if err := DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateUser(c *gin.Context) {
	var u User
	if !bindJSON(c, &u) {
		return
	}
	if err := DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u)
}

func UpdateUser(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	var u User
	if err := DB.First(&u, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var input User
	if !bindJSON(c, &input) {
		return
	}
	u.Username = input.Username
	u.Email = input.Email
	u.Level = input.Level
	if err := DB.Save(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func DeleteUser(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	if err := DB.Delete(&User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type WordProgress struct {
	UserID        int       `gorm:"primaryKey;column:user_id"       json:"user_id"`
	WordID        int       `gorm:"primaryKey;column:word_id"       json:"word_id"`
	RepeatCount   int       `gorm:"column:repeat_count"             json:"repeat_count"`
	LastPracticed time.Time `gorm:"column:last_practiced"           json:"last_practiced"`
}

func GetUserWordProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var list []WordProgress
	if err := DB.Where("user_id = ?", uid).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateUserWordProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	wid, _ := strconv.Atoi(c.Param("wordId"))
	p := WordProgress{
		UserID:        uid,
		WordID:        wid,
		RepeatCount:   0,
		LastPracticed: time.Now(),
	}
	if err := DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func UpdateUserWordProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	wid, _ := strconv.Atoi(c.Param("wordId"))

	var input struct {
		RepeatCount int `json:"repeat_count"`
	}
	if !bindJSON(c, &input) {
		return
	}

	if err := DB.Model(&WordProgress{}).
		Where("user_id = ? AND word_id = ?", uid, wid).
		Updates(map[string]interface{}{
			"repeat_count":   input.RepeatCount,
			"last_practiced": time.Now(),
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var p WordProgress
	DB.Where("user_id = ? AND word_id = ?", uid, wid).First(&p)
	c.JSON(http.StatusOK, p)
}

func DeleteUserWordProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	wid, _ := strconv.Atoi(c.Param("wordId"))
	if err := DB.Delete(&WordProgress{}, "user_id = ? AND word_id = ?", uid, wid).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type TextProgress struct {
	UserID        int       `gorm:"primaryKey;column:user_id"       json:"user_id"`
	TextID        int       `gorm:"primaryKey;column:text_id"       json:"text_id"`
	RepeatCount   int       `gorm:"column:repeat_count"             json:"repeat_count"`
	LastPracticed time.Time `gorm:"column:last_practiced"           json:"last_practiced"`
}

func GetUserTextProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var list []TextProgress
	if err := DB.Where("user_id = ?", uid).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func UpsertUserTextProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	tid, _ := strconv.Atoi(c.Param("textId"))
	var p TextProgress
	if !bindJSON(c, &p) {
		return
	}
	p.UserID = uid
	p.TextID = tid
	p.LastPracticed = time.Now()
	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "text_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"repeat_count", "last_practiced"}),
	}).Create(&p)
	c.JSON(http.StatusOK, p)
}

func DeleteUserTextProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	tid, _ := strconv.Atoi(c.Param("textId"))
	if err := DB.Delete(&TextProgress{}, "user_id = ? AND text_id = ?", uid, tid).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type GrammarProgress struct {
	UserID        int       `gorm:"primaryKey;column:user_id"         json:"user_id"`
	GrammarID     int       `gorm:"primaryKey;column:grammar_id"      json:"grammar_id"`
	RepeatCount   int       `gorm:"column:repeat_count"               json:"repeat_count"`
	LastPracticed time.Time `gorm:"column:last_practiced"             json:"last_practiced"`
}

func GetUserGrammarProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var list []GrammarProgress
	if err := DB.Where("user_id = ?", uid).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func UpsertUserGrammarProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	gid, _ := strconv.Atoi(c.Param("grammarId"))
	var p GrammarProgress
	if !bindJSON(c, &p) {
		return
	}
	p.UserID = uid
	p.GrammarID = gid
	p.LastPracticed = time.Now()
	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "grammar_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"repeat_count", "last_practiced"}),
	}).Create(&p)
	c.JSON(http.StatusOK, p)
}

func DeleteUserGrammarProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	gid, _ := strconv.Atoi(c.Param("grammarId"))
	if err := DB.Delete(&GrammarProgress{}, "user_id = ? AND grammar_id = ?", uid, gid).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
