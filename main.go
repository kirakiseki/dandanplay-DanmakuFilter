package main

import (
	"crypto/tls"
	"dandanplay-DanmakuFilter/utils"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strings"
)

type Danmakus struct {
	Code int     `json:"code"`
	Data [][]any `json:"data"`
}

func FilterDanmaku(danmaku string, rules []utils.Rule) bool {
	for _, rule := range rules {
		switch rule.Type {
		case "regex":
			r, err := regexp.Compile(rule.Rule)
			if err != nil {
				utils.Inst.Logger.Error().Err(err).Msgf("Failed to compile regex: %s", rule.Rule)
				return true
			}

			if r.MatchString(danmaku) {
				return false
			}

		case "keyword":
			if strings.Contains(danmaku, rule.Rule) {
				return false
			}
		}

	}
	return true
}

func GetCookie(url, name string) string {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		utils.Inst.Logger.Fatal().Err(err).Msg("Failed to create cookie jar")
		return ""
	}

	client := &http.Client{Transport: tr, Jar: cookieJar}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		utils.Inst.Logger.Fatal().Err(err).Msg("Failed to create cookie request")
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		utils.Inst.Logger.Fatal().Err(err).Msg("Failed to fetch cookie")
		return ""
	}
	defer resp.Body.Close()
	for _, cook := range cookieJar.Cookies(resp.Request.URL) {
		if cook.Name == name {
			return cook.Value
		}
	}
	utils.Inst.Logger.Fatal().Err(err).Msg("Failed to found cookie")
	return ""
}

func main() {
	utils.Inst.Logger = utils.Zerolog()
	utils.Inst.GinEngine = utils.Gin()

	baseURL := os.Getenv("BASEURL")
	if baseURL == "" {
		utils.Inst.Logger.Fatal().Msg("BASEURL is not set")
	}

	ruleFiles := utils.ReadRules()
	rules := utils.ParseRules(ruleFiles)

	utils.Inst.Logger.Info().Msgf("Initialized %d rules", len(rules))

	utils.Inst.GinEngine.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	utils.Inst.GinEngine.POST("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	utils.Inst.GinEngine.GET("/filter", func(c *gin.Context) {
		url := c.Query("id")

		cookie := GetCookie(baseURL, "_ncfa")

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		req, err := http.NewRequest("GET", url, nil)
		client := &http.Client{
			Transport: tr,
		}
		req.Header.Set("Cookie", "_ncfa="+cookie)

		resp, err := client.Do(req)
		if err != nil {
			utils.Inst.Logger.Error().Err(err).Msg("Failed to fetch danmaku")
			c.String(200, "failed to fetch")
			return
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			utils.Inst.Logger.Error().Err(err).Msg("Failed to read danmaku")
			c.String(200, "failed to read")
			return
		}

		var danmakus Danmakus
		var respDanmakus Danmakus

		err = json.Unmarshal(body, &danmakus)
		if err != nil {
			utils.Inst.Logger.Error().Err(err).Msg("Failed to parse danmaku")
			c.String(200, "failed to parse")
			return
		}

		respDanmakus.Code = danmakus.Code

		for _, d := range danmakus.Data {
			if FilterDanmaku(d[4].(string), rules) {
				respDanmakus.Data = append(respDanmakus.Data, d)
			} else {
				utils.Inst.Logger.Info().Msgf("Filtered danmaku: %s", d[4].(string))
			}
		}

		utils.Inst.Logger.Info().Msgf("Source danmakus: %d, filtered danmakus: %d", len(danmakus.Data), len(respDanmakus.Data))
		utils.Inst.Logger.Info().Msgf("Deleted %d danmakus", len(danmakus.Data)-len(respDanmakus.Data))

		c.JSON(200, respDanmakus)
	})

	utils.Inst.GinEngine.Run(":1412")

}
