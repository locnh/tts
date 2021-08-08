package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	strip "github.com/grokify/html-strip-tags-go"
	log "github.com/sirupsen/logrus"
)

const (
	ZALO_CREDITS     = "Đọc bởi ZALO TTS"
	ZALO_AI_ENDPOINT = "https://api.zalo.ai/v1/tts/synthesize"
)

var (
	zalo_ai_api_key string
	zalo_speaker_id string
	speak_speed     string
)

type ZaloTTS struct {
	Error_code    int8        `json:"error_code"`
	Error_message string      `json:"error_message"`
	Data          ZaloTTSData `json:"data"`
}

type ZaloTTSData struct {
	Url string `json:"url"`
}

func main() {
	r := gin.Default()
	r.POST("/raw", returnRaw(true))
	r.POST("/embeded", returnRaw(false))
	r.Run()
}

func init() {
	apiKey, provided := os.LookupEnv("ZALO_AI_API_KEY")
	if !provided {
		log.Fatal("ZALO_AI_API_KEY is not set")
		os.Exit(128)
	} else {
		zalo_ai_api_key = apiKey
	}

	_, provided = os.LookupEnv("ZALO_SPEAKER_ID")
	if !provided {
		zalo_speaker_id = "1"
	}

	_, provided = os.LookupEnv("ZALO_SPEAKER_ID")
	if !provided {
		speak_speed = "0.8"
	}
}

func purifier(content string) string {
	return strip.StripTags(content)
}

func getRawAudioLink(payload string) string {
	params := url.Values{}
	params.Add("input", payload+ZALO_CREDITS)
	params.Add("speaker_id", zalo_speaker_id)
	params.Add("speed", speak_speed)
	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest("POST", ZALO_AI_ENDPOINT, body)
	if err != nil {
		log.Error(err)
	}
	req.Header.Set("Apikey", zalo_ai_api_key)
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)

		var tts ZaloTTS
		_ = json.Unmarshal([]byte(bodyString), &tts)

		return tts.Data.Url
	} else {
		return ""
	}
}

func returnRaw(raw bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		content, _ := ioutil.ReadAll(c.Request.Body)
		url := getRawAudioLink(purifier(string(content)))

		if url != "" {
			if raw {
				c.String(http.StatusOK, url)
			} else {
				c.String(http.StatusOK,
					"<audio controls autoplay><source src=\"%s\" type=\"audio/mpeg\"></audio>",
					url)
			}
		} else {
			c.JSON(http.StatusInternalServerError, "InternalServerError")
		}
	}
}
