package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/scalecloud.de-api"
	"github.com/scalecloud/scalecloud.de-api/stripe"
	"go.uber.org/zap"
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
	initHeaders(router)
	initRoutes(router)
	// initCertificate(router)
	// initTrustedPlatform(router)
	startListening(router)
}

func initHeaders(router *gin.Engine) {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200"}
	config.AllowMethods = []string{"GET"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	config.ExposeHeaders = []string{"Content-Length"}
	config.MaxAge = 12 * time.Hour
	router.Use(cors.New(config))
}

func startListening(router *gin.Engine) {
	logger.Info("Starting listening for requests")
	router.Run(":15000")
}

func initRoutes(router *gin.Engine) {
	logger.Info("Setting up routes...")
	// Subscription

	// Account
	dashboard := router.Group("/dashboard")
	dashboard.Use(AuthRequired)
	{
		dashboard.GET("/subscriptions", getSubscriptionsOverview)
		dashboard.GET("/subscription/:id", getSubscriptionByID)
	}
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

func getSubscriptionsOverview(c *gin.Context) {
	subscriptionsOverview, error := scalecloud.GetSubscriptionsOverview(c)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("getSubscriptionsOverview", zap.Any("subscriptionsOverview", subscriptionsOverview))
	if subscriptionsOverview != nil {
		c.IndentedJSON(http.StatusOK, subscriptionsOverview)
	} else {
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionsOverview not found"})
	}
}

func getSubscriptionByID(c *gin.Context) {
	subscriptionID := c.Param("id")
	logger.Info("getSubscriptionByID", zap.String("subscriptionID", subscriptionID))
	subscriptionDetail, error := scalecloud.GetSubscriptionByID(c, subscriptionID)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("Found subscriptionDetail", zap.Any("subscriptionDetail", subscriptionDetail))
	if subscriptionDetail != (stripe.SubscriptionDetail{}) {
		c.IndentedJSON(http.StatusOK, subscriptionDetail)
	} else {
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionDetail not found"})
	}
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

// Authentication
func AuthRequired(c *gin.Context) {
	token, hasAuth := getBearerToken(c)
	if hasAuth && token != "" && scalecloud.IsAuthenticated(c, token) {
		logger.Debug("Authenticated", zap.String("token:", token))
		c.Next()
	} else {
		logger.Warn("Unauthorized", zap.String("token:", token))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

func getBearerToken(c *gin.Context) (token string, ok bool) {
	jwtToken := c.Request.Header.Get("Authorization")
	if jwtToken == "" {
		return "", false
	} else {
		return jwtToken, true
	}
}
