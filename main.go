package main

import (
	"encoding/json"
	"net/http"

	_ "github.com/gmelodie/estuphotos/docs"
	"golang.org/x/crypto/bcrypt"

	logging "github.com/ipfs/go-log/v2"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Photo struct {
	ID   uint   `gorm:"primarykey" json:"id"`
	Name string `json:"name"`
	User string `json:"user"`
}

type User struct {
	ID           uint   `gorm:"primarykey" json:"id"`
	Handle       string `json:"handle"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
}

type API struct {
	DB *gorm.DB
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// check password
// err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

// handleIndex godoc
// @Summary  API greeting message
// @Produce  json
// @Success  200
// @Router   / [get]
func (a *API) handleIndex(c echo.Context) error {
	return c.JSON(http.StatusOK, "Welcome to EstuPhotos")
}

// handleUploadPhoto godoc
// @Summary  Create a new Person
// @Accept   json
// @Produce  json
// @Param    firstName  body      string  true  "first name"
// @Param    lastName   body      string  true  "last name"
// @Param    age        body      int     true  "age"
// @Success  202        {object}  Person
// @Router   /person [post]
func (a *API) handlePersonCreate(c echo.Context) error {
	newPerson := new(Person)
	err := json.NewDecoder(c.Request().Body).Decode(&newPerson)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	a.DB.Save(&newPerson)

	return c.JSON(http.StatusCreated, newPerson)
}

// handlePersonGet godoc
// @Summary  Find a registered Person
// @Produce  json
// @Param    firstName  query     string  true  "first name"
// @Success  200        {object}  Person
// @Router   /person/{firstName} [get]
func (a *API) handlePersonGet(c echo.Context) error {
	firstName := c.Param("firstName")

	var person Person
	err := a.DB.Model(&Person{}).
		Where("first_name = ?", firstName).
		Scan(&person).
		Error
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	return c.JSON(http.StatusCreated, person)
}

// handlePersonUpdate godoc
// @Summary  Update a registered Person
// @Produce  json
// @Param    firstName  query     string  true  "original first name"
// @Param    firstName  body      string  true  "new first name"
// @Param    lastName   body      string  true  "new last name"
// @Param    age        body      int     true  "new age"
// @Success  200        {object}  Person
// @Router   /person/{firstName} [put]
func (a *API) handlePersonUpdate(c echo.Context) error {
	origFirstName := c.Param("firstName")

	updatedPerson := new(Person)
	err := json.NewDecoder(c.Request().Body).Decode(&updatedPerson)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	err = a.DB.Model(&Person{}).
		Where("first_name = ?", origFirstName).
		Updates(*updatedPerson).
		Error
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	return c.JSON(http.StatusOK, updatedPerson)
}

// handlePersonDelete godoc
// @Summary  Delete a registered Person
// @Produce  json
// @Param    firstName  query     string  true  "first name"
// @Success  200 "deleted successfully"
// @Success  404 "entry to delete not found"
// @Router   /person/{firstName} [delete]
func (a *API) handlePersonDelete(c echo.Context) error {
	firstName := c.Param("firstName")

	err := a.DB.Where("first_name = ?", firstName).Delete(&Person{}).Error
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	return c.NoContent(http.StatusOK)
}

func createDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("people.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&Person{})

	return db, nil
}

// @title People API
// @version 2.0

// @BasePath /
// @schemes http
func main() {
	var log = logging.Logger("api")
	var api API
	var err error

	api.DB, err = createDatabase()
	if err != nil {
		log.Fatal("could not create or open database: %s", err)
	}

	e := echo.New()

	e.GET("/", api.handleIndex)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.DELETE("/person/:firstName", api.handlePersonDelete)
	e.PUT("/person/:firstName", api.handlePersonUpdate)
	e.POST("/person", api.handlePersonCreate)
	e.GET("/person/:firstName", api.handlePersonGet)

	e.Logger.Fatal(e.Start(":8080"))
}
