package main

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/application-research/edge-ur/core"
	"golang.org/x/xerrors"

	_ "github.com/gmelodie/estuphotos/docs"
	"github.com/gmelodie/estuphotos/util"
	"github.com/spf13/viper"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	"gorm.io/gorm"
)

var log = logging.Logger("estuphotos")

type API struct {
	DB        *gorm.DB
	LightNode *core.LightNode
}

func (a *API) AuthRequired() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			//	Check first if the Token is available. We should not continue if the
			//	token isn't even available.
			auth, err := util.ExtractAuth(c)
			if err != nil {
				return err
			}

			u, err := a.checkTokenAuth(auth)
			if err != nil {
				return err
			}

			c.Set("user", u)
			return next(c)
		}
	}
}

func (a *API) checkTokenAuth(token string) (*util.User, error) {
	var user util.User
	if err := a.DB.First(&user, "apikey = ?", token).Error; err != nil {
		if xerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &util.HttpError{
				Code:    http.StatusUnauthorized,
				Reason:  "Invalid Token",
				Details: "no user exists for the spicified api key",
			}
		}
		return nil, err
	}

	return &user, nil
}

// handleIndex godoc
// @Summary  API greeting message
// @Produce  json
// @Success  200
// @Router   / [get]
func (a *API) handleIndex(c echo.Context) error {
	return c.JSON(http.StatusOK, "Welcome to EstuPhotos")
}

// handleUserRegister godoc
// @Summary  Register a new user
// @Accept   json
// @Produce  json
// @Success  202        {object}  User
// @Router   /user/{username} [post]
func (a *API) handleUserRegister(c echo.Context) error {
	username := c.Param("username")

	apiKey := util.GenerateToken(64)
	// add user to our own database
	newUser := util.User{
		Handle: username,
		ApiKey: apiKey,
	}

	err := a.DB.Create(&newUser).Error
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusCreated, newUser)

}

// handleUploadPhoto godoc
// @Summary  Upload a new Photo
// @Accept   json
// @Produce  json
// @Param    name  body      string  true  "name of the photo"
// @Success  202        {object}  Photo
// @Router   /photo [post]
func (a *API) handlePhotoUpload(c echo.Context, u *util.User) error {

	// authenticate user
	auth, err := util.ExtractAuth(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	err = a.DB.First(&util.User{}).
		Where("apikey = ?", auth).
		Error
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

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
	newPhoto := util.Photo{
		Name:   filename,
		CID:    content.Cid,
		UserID: u.ID,
	}
	err = a.DB.Create(&newPhoto).Error
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
	cidStr := c.Param("cid")

	var photo util.Photo
	err := a.DB.Model(&util.Photo{}).
		Where("cid = ?", cidStr).
		Scan(&photo).
		Error
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	// If photo is in the database, retrieve it
	// TODO: retrieve photo
	parsedCID, err := cid.Parse(photo.CID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	photoReader, err := a.LightNode.Node.GetFile(context.Background(), parsedCID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.Stream(http.StatusOK, "application/octet-stream", photoReader)
	// return c.JSON(http.StatusOK, photoReader)
}

// @title EstuPhotos
// @version 1.0

// @BasePath /
// @schemes http
func main() {
	var api API
	var err error

	rand.Seed(time.Now().UnixNano())

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
	api.DB, err = util.CreateDatabase()
	if err != nil {
		log.Fatal("could not create or open database: %s", err)
		panic(err)
	}

	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())

	e.GET("/", api.handleIndex)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.POST("/user/:username", api.handleUserRegister)

	e.POST("/photo", util.WithUser(api.handlePhotoUpload), api.AuthRequired())
	// e.POST("/photo", api.handlePhotoUpload)
	e.GET("/photo/:cid", api.handlePhotoDownload)

	e.Logger.Fatal(e.Start(":8080"))
}
