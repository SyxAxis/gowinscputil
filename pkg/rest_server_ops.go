package pkg

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func InitRESTServer(inipath string, restSvrPort int, restSSLEnabled bool) {

	// gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()
	baseAPIPath := "/api/v1"

	router := gin.Default()
	router.SetTrustedProxies(nil)

	// spin the router handlers out into a thread, then it can respond concurrently
	go func() {
		router.GET(baseAPIPath+"/connection", execRESTConnectionTest(inipath))
		router.GET(baseAPIPath+"/connectionv2", execRESTConnectionTestV2(inipath))

		router.GET(baseAPIPath+"/profile", execRESTProfileList(inipath, false))
		router.GET(baseAPIPath+"/profile/full", execRESTProfileList(inipath, true))

		router.GET("/ping", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"ping": "pong",
			})
		})
	}()

	if restSSLEnabled {
		router.RunTLS(":"+strconv.Itoa(restSvrPort), ".//restserver.crt", ".//restserver.key")
	} else {
		router.Run(":" + strconv.Itoa(restSvrPort))
	}

}

func execRESTConnectionTestV2(inipath string) gin.HandlerFunc {

	fn := func(c *gin.Context) {

		profile := c.Query("profile")
		threads, _ := strconv.Atoi(c.DefaultQuery("threads", "3"))

		// now call the actual test
		connectTestResults, err := executeConnectionTest(inipath, profile, threads, false)
		if err != nil {
			c.IndentedJSON(http.StatusNoContent, "null")
		}
		c.IndentedJSON(http.StatusOK, connectTestResults)
	}

	return gin.HandlerFunc(fn)
}

func execRESTConnectionTest(inipath string) gin.HandlerFunc {

	// this is only ever used here, so local is fine
	type inboundJSONRequest struct {
		ProfileName string `json:"profile"`
	}
	var inboundReq inboundJSONRequest

	// router calls must only ever get a context back,
	// so we create an anon func object, wrap our code inside
	// when the "fn" object is passed back
	// we get to call whatever we like with whatever vars we need
	// AND it gets called with context and gets the right object response!
	// this only works 'cos golang allows func objects to be passed as references

	fn := func(c *gin.Context) {
		// get the inbound JSON body and store in the struct
		if err := c.BindJSON(&inboundReq); err != nil {
			return
		}

		// now call the actual test
		connectTestResults, err := executeConnectionTest(inipath, inboundReq.ProfileName, 3, false)
		if err != nil {
			c.IndentedJSON(http.StatusNoContent, "null")
		}
		c.IndentedJSON(http.StatusOK, connectTestResults)
	}

	return gin.HandlerFunc(fn)
}

func execRESTProfileList(inipath string, fullmeta bool) gin.HandlerFunc {

	// router calls must only ever get a context back,
	// so we create an anon func object, wrap our code inside
	// when the "fn" object is passed back
	// we get to call whatever we like with whatever vars we need
	// AND it gets called with context and gets the right object response!
	// this only works 'cos golang allows func objects to be passed as references

	fn := func(c *gin.Context) {

		profile := c.DefaultQuery("profile", "ALL")

		// now call the actual test
		winSCPProfiles, err := WinSCPiniExtractProfileData(inipath, profile, true)
		if err != nil {
			c.IndentedJSON(http.StatusNoContent, "null")
		}

		if fullmeta {
			c.IndentedJSON(http.StatusOK, winSCPProfiles)
		} else {
			var profileList []string
			for _, profileMeta := range winSCPProfiles {
				profileList = append(profileList, profileMeta.ProfileName)
			}
			// FYI - profile names are already sorted by the extract function
			c.IndentedJSON(http.StatusOK, profileList)
		}
	}

	return gin.HandlerFunc(fn)
}
