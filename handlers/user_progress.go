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

type TextTest struct {
	ID      int    `gorm:"primaryKey;column:id" json:"id"`
	TextID  int    `gorm:"column:text_id"       json:"text_id"`
	TitleRu string `gorm:"column:title_ru"      json:"title_ru"`
	TitleEn string `gorm:"column:title_en"      json:"title_en"`
	TitleDe string `gorm:"column:title_de"      json:"title_de"`
	Lang    string `gorm:"column:lang"          json:"lang"`
}

type TextQuestion struct {
	ID     int    `gorm:"primaryKey;column:id"    json:"id"`
	TestID int    `gorm:"column:test_id"          json:"test_id"`
	BodyRu string `gorm:"column:body_ru"          json:"body_ru"`
	BodyEn string `gorm:"column:body_en"          json:"body_en"`
	BodyDe string `gorm:"column:body_de"          json:"body_de"`
	Answer string `gorm:"column:answer;size:2"    json:"answer"`
}

type UserTextTestProgress struct {
	UserID    int       `gorm:"primaryKey;column:user_id"    json:"user_id"`
	TestID    int       `gorm:"primaryKey;column:test_id"    json:"test_id"`
	Score     int       `gorm:"column:score"                 json:"score"`
	Total     int       `gorm:"column:total"                 json:"total"`
	Timestamp time.Time `gorm:"column:timestamp"             json:"timestamp"`
}

func GetTextTests(c *gin.Context) {
	var list []TextTest
	if err := DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateTextTest(c *gin.Context) {
	var t TextTest
	if !bindJSON(c, &t) {
		return
	}
	if err := DB.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func UpdateTextTest(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	var t TextTest
	if err := DB.First(&t, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var input TextTest
	if !bindJSON(c, &input) {
		return
	}
	t.TitleRu = input.TitleRu
	t.TitleEn = input.TitleEn
	t.TitleDe = input.TitleDe
	t.Lang = input.Lang
	if err := DB.Save(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

func DeleteTextTest(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	if err := DB.Delete(&TextTest{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func GetAllTextQuestions(c *gin.Context) {
	var list []TextQuestion
	if err := DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetTextQuestions(c *gin.Context) {
	tid, _ := strconv.Atoi(c.Param("id"))
	var list []TextQuestion
	if err := DB.Where("test_id = ?", tid).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateTextQuestion(c *gin.Context) {
	tid, _ := strconv.Atoi(c.Param("id"))
	var q TextQuestion
	if !bindJSON(c, &q) {
		return
	}
	q.TestID = tid
	if err := DB.Create(&q).Error; err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, q)
}

func UpdateTextQuestion(c *gin.Context) {
	qid, ok := getID(c)
	if !ok {
		return
	}
	var q TextQuestion
	if err := DB.First(&q, qid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound,
				gin.H{"error": "question not found"})
		} else {
			c.JSON(http.StatusInternalServerError,
				gin.H{"error": err.Error()})
		}
		return
	}
	var in TextQuestion
	if !bindJSON(c, &in) {
		return
	}
	q.BodyRu = in.BodyRu
	q.BodyEn = in.BodyEn
	q.BodyDe = in.BodyDe
	q.Answer = in.Answer
	DB.Save(&q)
	c.JSON(http.StatusOK, q)
}

func DeleteTextQuestion(c *gin.Context) {
	qid, ok := getID(c)
	if !ok {
		return
	}
	DB.Delete(&TextQuestion{}, qid)
	c.Status(http.StatusNoContent)
}

func GetUserTextTestProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var lst []UserTextTestProgress
	if err := DB.Where("user_id = ?", uid).Find(&lst).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, lst)
}

func UpsertUserTextTestProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	tid, _ := strconv.Atoi(c.Param("testId"))
	var p UserTextTestProgress
	if !bindJSON(c, &p) {
		return
	}
	p.UserID = uid
	p.TestID = tid
	p.Timestamp = time.Now()
	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "test_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"score", "total", "timestamp"}),
	}).Create(&p)
	c.JSON(http.StatusOK, p)
}

func DeleteUserTextTestProgress(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	tid, _ := strconv.Atoi(c.Param("testId"))
	DB.Delete(&UserTextTestProgress{}, "user_id = ? AND test_id = ?", uid, tid)
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

type GrammarTest struct {
	ID        int    `gorm:"primaryKey;column:id"      json:"id"`
	GrammarID int    `gorm:"column:grammar_id"         json:"grammar_id"`
	TitleRu   string `gorm:"column:title_ru"          json:"title_ru"`
	TitleEn   string `gorm:"column:title_en"           json:"title_en"`
	TitleDe   string `gorm:"column:title_de"           json:"title_de"`
}

type GrammarQuestion struct {
	ID       int    `gorm:"primaryKey;column:id" json:"id"`
	TestID   int    `gorm:"column:test_id"       json:"test_id"`
	Type     string `gorm:"column:type"          json:"type"`
	BodyRu   string `gorm:"column:body_ru"       json:"body_ru"`
	BodyEn   string `gorm:"column:body_en"       json:"body_en"`
	BodyDe   string `gorm:"column:body_de"       json:"body_de"`
	AnswerRu string `gorm:"column:answer_ru"     json:"answer_ru"`
	AnswerEn string `gorm:"column:answer_en"     json:"answer_en"`
	AnswerDe string `gorm:"column:answer_de"     json:"answer_de"`
}

type GrammarAnswer struct {
	ID         int    `gorm:"primaryKey;column:id" json:"id"`
	QuestionID int    `gorm:"column:question_id"   json:"question_id"`
	Idx        int    `gorm:"column:idx"           json:"idx"`
	TextRu     string `gorm:"column:text_ru"       json:"text_ru"`
	TextEn     string `gorm:"column:text_en"       json:"text_en"`
	TextDe     string `gorm:"column:text_de"       json:"text_de"`
	IsCorrect  bool   `gorm:"column:is_correct"    json:"is_correct"`
}

type UserGrammarTestProgress struct {
	UserID    int       `gorm:"primaryKey;column:user_id" json:"user_id"`
	TestID    int       `gorm:"primaryKey;column:test_id" json:"test_id"`
	Score     int       `gorm:"column:score"              json:"score"`
	Total     int       `gorm:"column:total"              json:"total"`
	Timestamp time.Time `gorm:"column:timestamp"          json:"timestamp"`
}

// GET /api/grammar_questions
func GetAllGrammarQuestions(c *gin.Context) {
	var list []GrammarQuestion
	if handleDBErr(c, DB.Find(&list).Error) { return }
	c.JSON(http.StatusOK, list)
}

// GET /api/grammar_tests/:id/questions
func GetGrammarQuestions(c *gin.Context) {
	tid, ok := getID(c)
	if !ok { return }

	var list []GrammarQuestion
	if handleDBErr(c, DB.Where("test_id = ?", tid).Find(&list).Error) { return }
	c.JSON(http.StatusOK, list)
}

// POST /api/grammar_tests/:id/questions
func CreateGrammarQuestion(c *gin.Context) {
	tid, ok := getID(c)
	if !ok { return }

	var q GrammarQuestion
	if !bindJSON(c, &q) { return }

	switch q.Type {
	case "TF", "INPUT", "MC":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be TF/INPUT/MC"})
		return
	}

	q.TestID = tid
	if handleDBErr(c, DB.Create(&q).Error) { return }
	c.JSON(http.StatusCreated, q)
}

// PUT /api/grammar_questions/:qid
func UpdateGrammarQuestion(c *gin.Context) {
	qid, ok := getID(c)
	if !ok { return }

	var orig GrammarQuestion
	if err := DB.First(&orig, qid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		} else {
			handleDBErr(c, err)
		}
		return
	}

	var in GrammarQuestion
	if !bindJSON(c, &in) { return }

	if in.Type != "" && in.Type != "TF" && in.Type != "INPUT" && in.Type != "MC" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be TF/INPUT/MC"})
		return
	}

	in.ID = qid
	if handleDBErr(c, DB.Model(&orig).Updates(in).Error) { return }
	c.JSON(http.StatusOK, in)
}

// DELETE /api/grammar_questions/:qid
func DeleteGrammarQuestion(c *gin.Context) {
	qid, ok := getID(c)
	if !ok { return }

	if handleDBErr(c, DB.Delete(&GrammarQuestion{}, qid).Error) { return }
	c.Status(http.StatusNoContent)
}

// ---------- ХЭНДЛЕРЫ ОТВЕТОВ ----------

// GET /api/grammar_answers
func GetAllGrammarAnswers(c *gin.Context) {
	var list []GrammarAnswer
	if handleDBErr(c, DB.Find(&list).Error) { return }
	c.JSON(http.StatusOK, list)
}

// GET /api/grammar_questions/:id/answers
func GetGrammarAnswers(c *gin.Context) {
	qid, ok := getID(c)
	if !ok { return }

	var list []GrammarAnswer
	if handleDBErr(c, DB.Where("question_id = ?", qid).Order("idx").Find(&list).Error) { return }
	c.JSON(http.StatusOK, list)
}

// POST /api/grammar_questions/:id/answers
func CreateGrammarAnswer(c *gin.Context) {
	qid, ok := getID(c)
	if !ok { return }

	var a GrammarAnswer
	if !bindJSON(c, &a) { return }
	a.QuestionID = qid

	if handleDBErr(c, DB.Create(&a).Error) { return }
	c.JSON(http.StatusCreated, a)
}

// PUT /api/grammar_answers/:aid
func UpdateGrammarAnswer(c *gin.Context) {
	aid, ok := getID(c)
	if !ok { return }

	var orig GrammarAnswer
	if err := DB.First(&orig, aid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "answer not found"})
		} else {
			handleDBErr(c, err)
		}
		return
	}

	var in GrammarAnswer
	if !bindJSON(c, &in) { return }
	in.ID = aid

	if handleDBErr(c, DB.Model(&orig).Updates(in).Error) { return }
	c.JSON(http.StatusOK, in)
}

// DELETE /api/grammar_answers/:aid
func DeleteGrammarAnswer(c *gin.Context) {
	aid, ok := getID(c)
	if !ok { return }

	if handleDBErr(c, DB.Delete(&GrammarAnswer{}, aid).Error) { return }
	c.Status(http.StatusNoContent)
}


// GET  /api/grammar_tests
func GetGrammarTests(c *gin.Context) {
    var list []GrammarTest
    if handleDBErr(c, DB.Find(&list).Error) {
        return
    }
    c.JSON(http.StatusOK, list)
}

// POST /api/grammar_tests
func CreateGrammarTest(c *gin.Context) {
    var t GrammarTest
    if !bindJSON(c, &t) {
        return
    }
    if handleDBErr(c, DB.Create(&t).Error) {
        return
    }
    c.JSON(http.StatusCreated, t)
}

// PUT  /api/grammar_tests/:id
func UpdateGrammarTest(c *gin.Context) {
    id, ok := getID(c)
    if !ok {
        return
    }

    var orig GrammarTest
    if err := DB.First(&orig, id).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "grammar_test not found"})
        } else if handleDBErr(c, err) {
        }
        return
    }

    var in GrammarTest
    if !bindJSON(c, &in) {
        return
    }
    in.ID = id

    if handleDBErr(c, DB.Model(&orig).Updates(in).Error) {
        return
    }
    c.JSON(http.StatusOK, in)
}

// DELETE /api/grammar_tests/:id
func DeleteGrammarTest(c *gin.Context) {
    id, ok := getID(c)
    if !ok {
        return
    }
    if handleDBErr(c, DB.Delete(&GrammarTest{}, id).Error) {
        return
    }
    c.Status(http.StatusNoContent)
}



// POST/PUT /api/users/:id/grammar_tests/:testId/progress
func UpsertUserGrammarTestProgress(c *gin.Context) {
    uid, _ := strconv.Atoi(c.Param("id"))
    tid, _ := strconv.Atoi(c.Param("testId"))

    var p UserGrammarTestProgress
    if !bindJSON(c, &p) {
        return
    }
    p.UserID = uid
    p.TestID = tid
    p.Timestamp = time.Now()

    DB.Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "user_id"}, {Name: "test_id"}},
        DoUpdates: clause.AssignmentColumns([]string{"score", "total", "timestamp"}),
    }).Create(&p)

    c.JSON(http.StatusOK, p)
}