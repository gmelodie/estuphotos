package main

import (
	"context"
	"net/http"
	"time"

	"github.com/application-research/edge-ur/core"

	_ "github.com/gmelodie/estuphotos/docs"
	"github.com/spf13/viper"

	logging "github.com/ipfs/go-log/v2"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var log = logging.Logger("estuphotos")

type Photo struct {
	gorm.Model
	Name   string `json:"name"`
	UserID uint   `json:"user_id"`
	CID    string `json:"cid"`
}

type User struct {
	gorm.Model
	Handle string  `json:"handle"`
	Email  string  `json:"email"`
	ApiKey string  `json:"apikey"`
	Photos []Photo `json:photos`
}

type API struct {
	DB        *gorm.DB
	LightNode *core.LightNode
}

type HttpError struct {
	Code    int    `json:"code,omitempty"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

func (he HttpError) Error() string {
	if he.Details == "" {
		return he.Reason
	}
	return he.Reason + ": " + he.Details
}

func withUser(f func(echo.Context, *User) error) func(echo.Context) error {
	return func(c echo.Context) error {
		u, ok := c.Get("user").(*User)
		if !ok {
			return &HttpError{
				Code:    http.StatusUnauthorized,
				Reason:  "invalid API key",
				Details: "endpoint not called with proper authentication",
			}
		}
		return f(c, u)
	}
}

// handleIndex godoc
// @Summary  API greeting message
// @Produce  json
// @Success  200
// @Router   / [get]
func (a *API) handleIndex(c echo.Context) error {
	return c.JSON(http.StatusOK, "Welcome to EstuPhotos")
}

// handleUploadPhoto godoc
// @Summary  Upload a new Photo
// @Accept   json
// @Produce  json
// @Param    name  body      string  true  "name of the photo"
// @Success  202        {object}  Photo
// @Router   /photo [post]
func (a *API) handlePhotoUpload(c echo.Context) error { // TODO: add user

	// get file information from formdata
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	defer form.RemoveAll()

	mpf, err := c.FormFile("data")
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	filename := mpf.Filename
	if fvname := c.FormValue("filename"); fvname != "" {
		filename = fvname
	}

	// open file
	fi, err := mpf.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	defer fi.Close()

	// upload file to edge-ur

	fileNode, err := a.LightNode.Node.AddPinFile(context.Background(), fi, nil)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	size, err := fileNode.Size()
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	content := core.Content{
		Name:             filename,
		Size:             int64(size),
		Cid:              fileNode.Cid().String(),
		RequestingApiKey: viper.Get("API_KEY").(string),
		Created_at:       time.Now(),
		Updated_at:       time.Now(),
	}

	// queue file for uploading on edge-ur
	a.LightNode.DB.Create(&content)

	// add file to our own database
	newPhoto := Photo{
		Name: filename,
		CID:  content.Cid,
	}
	err = a.DB.Save(&newPhoto).Error
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusCreated, newPhoto)
}

// handlePhotoDownload godoc
// @Summary  Download existing photo
// @Produce  json
// @Param    cid  query     string  true  "first name"
// @Success  200        {object}  string
// @Router   /photo/{cid} [get]
func (a *API) handlePhotoDownload(c echo.Context) error {
	cid := c.Param("cid")

	var photo Photo
	err := a.DB.Model(&Photo{}).
		Where("cid = ?", cid).
		Scan(&photo).
		Error
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	// If photo is in the database, retrieve it
	// TODO: retrieve photo

	return c.JSON(http.StatusOK, photo)
}

func createDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("estuphotos.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&Photo{})
	db.AutoMigrate(&User{})

	return db, nil
}

// @title EstuPhotos
// @version 1.0

// @BasePath /
// @schemes http
func main() {
	var api API
	var err error

	logging.SetLogLevel("edge-ur", "debug")

	viper.SetConfigFile(".env")
	err = viper.ReadInConfig()
	if err != nil {
		log.Error(err)
		panic(err)
	}

	// start light edge-ur node
	ctx := context.Background()
	api.LightNode, err = core.NewCliNode(&ctx)
	if err != nil {
		log.Fatal("could not start lightNode: %s", err)
		panic(err)
	}

	// start database
	api.DB, err = createDatabase()
	if err != nil {
		log.Fatal("could not create or open database: %s", err)
		panic(err)
	}

	e := echo.New()

	e.GET("/", api.handleIndex)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// e.POST("/photo", withUser(api.handlePhotoUpload))
	// e.GET("/photo/:cid", withUser(api.handlePhotoDownload))
	e.POST("/photo", api.handlePhotoUpload)
	e.GET("/photo/:cid", api.handlePhotoDownload)

	e.Logger.Fatal(e.Start(":8080"))
}
