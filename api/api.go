package api

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/tree/main/scalecloud.de-api"
)

var logger, _ = zap.NewProduction()

// album represents data about a record album.
type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

// albums slice to seed record album data.
var albums = []album{
	{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
	{ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
	{ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
}

func InitApi() {
	logger.Info("Init api")
	scalecloud.Init()
	startAPI()
}

func startAPI() {
	router := gin.Default()

	initRoutes(router)
	// initCertificate(router)
	// initTrustedPlatform(router)
	startListening(router)
}

func startListening(router *gin.Engine) {
	logger.Info("Starting listening for requests")
	router.Run(":15000")
}

func initRoutes(router *gin.Engine) {
	logger.Info("Setting up routes...")
	router.GET("/albums", getAlbums)
	router.GET("/albums/:id", getAlbumByID)
	router.POST("/albums", postAlbums)
}

func initCertificate(router *gin.Engine) {
	logger.Info("init certificate")
	error := autotls.Run(router, "api.scalecloud.de")
	if error != nil {
		logger.Error("Could not setup certificate", zap.Error(error))
	}
}

func initTrustedPlatform(router *gin.Engine) {
	logger.Info("init trusted platform")
	router.TrustedPlatform = gin.PlatformGoogleAppEngine
}

// getAlbums responds with the list of all albums as JSON.
func getAlbums(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, albums)
}

// postAlbums adds an album from JSON received in the request body.
func postAlbums(c *gin.Context) {
	var newAlbum album

	// Call BindJSON to bind the received JSON to
	// newAlbum.
	if err := c.BindJSON(&newAlbum); err != nil {
		return
	}

	// Add the new album to the slice.
	albums = append(albums, newAlbum)
	c.IndentedJSON(http.StatusCreated, newAlbum)
}

// getAlbumByID locates the album whose ID value matches the id
// parameter sent by the client, then returns that album as a response.
func getAlbumByID(c *gin.Context) {
	id := c.Param("id")

	// Loop over the list of albums, looking for
	// an album whose ID value matches the parameter.
	for _, a := range albums {
		if a.ID == id {
			c.IndentedJSON(http.StatusOK, a)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
}
