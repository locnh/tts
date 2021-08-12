package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	strip "github.com/grokify/html-strip-tags-go"
	log "github.com/sirupsen/logrus"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const (
	ZALO_AI_ENDPOINT = "https://api.zalo.ai/v1/tts/synthesize"
)

var (
	zalo_ai_api_key string
	zalo_speaker_id string
	speak_speed     string

	storage_path  string
	public_prefix string
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
	log.WithFields(log.Fields{
		"zalo_speaker_id": zalo_speaker_id,
		"speak_speed":     speak_speed,
		"storage_path":    storage_path,
		"public_prefix":   public_prefix,
	}).Info("Settings")

	r := gin.Default()
	r.StaticFile("/try-it.html", "./try-it.html")
	r.POST("/raw", returnRaw(true))
	r.POST("/embeded", returnRaw(false))
	r.POST("/json", returnJSON)
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

	zalo_speaker_id, provided = os.LookupEnv("ZALO_SPEAKER_ID")
	if !provided {
		zalo_speaker_id = "1"
	}

	speak_speed, provided = os.LookupEnv("ZALO_SPEAKER_SPEED")
	if !provided {
		speak_speed = "0.8"
	}

	storage_path, provided = os.LookupEnv("STORAGE_PATH")
	if !provided {
		storage_path = "."
	}

	public_prefix, provided = os.LookupEnv("PUBLIC_PREFIX")
	if !provided {
		public_prefix = "http://localhost:8080"
	}
}

func stringPurify(content string) string {
	content = strip.StripTags(content)
	content = strings.ReplaceAll(content, ".", ". ")

	return content
}

func stringFilet(content string, maxLength int) []string {
	slices := strings.Fields(stringPurify(content))
	arrChunk := []string{}
	chunk := ""

	for _, v := range slices {
		if len(chunk+strings.TrimSpace(v)) < maxLength {
			chunk = chunk + " " + v
		} else {
			arrChunk = append(arrChunk, strings.TrimSpace(chunk))
			chunk = v
		}
	}
	arrChunk = append(arrChunk, strings.TrimSpace(chunk))

	return arrChunk
}

func getRawAudioLink(payload string) string {
	params := url.Values{}
	params.Add("input", payload)
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
		if filename := processData(c); filename == "" {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": "Payload empty"})
		} else if _, err := os.Stat(storage_path + "/" + filename + ".mp3"); os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError,
				gin.H{"error": "Server Internal Error"})
		} else {
			url := public_prefix + "/" + filename + ".mp3"
			if raw {
				c.String(http.StatusOK, "%s", url)
			} else {
				c.String(http.StatusOK, "<audio scr=\"%s\" controls autoplay></audio>", url)
			}
		}
	}
}

func returnJSON(c *gin.Context) {
	if filename := processData(c); filename == "" {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": "Payload empty"})
	} else if _, err := os.Stat(storage_path + "/" + filename + ".mp3"); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError,
			gin.H{"error": "Server Internal Error"})
	} else {
		c.JSON(http.StatusOK,
			gin.H{"url": public_prefix + "/" + filename + ".mp3"})
	}
}

func processData(c *gin.Context) string {
	content, _ := ioutil.ReadAll(c.Request.Body)

	if len(content) == 0 {
		return ""
	}

	sha1HashInBytes := sha1.Sum(content)
	filename := hex.EncodeToString(sha1HashInBytes[:])

	if _, err := os.Stat(storage_path + "/" + filename + ".mp3"); os.IsNotExist(err) {
		chunks := stringFilet(string(content), 2000)
		if len(chunks) > 1 {
			for i, v := range chunks {
				url := getRawAudioLink(v)
				if url != "" {
					_ = fileDownload(url, filename+"_"+fmt.Sprintf("%d", i))

				}
			}

			err := mp3Concat(filename, len(chunks))
			if err != nil {
				log.Error(err)
			}
		} else {
			url := getRawAudioLink(chunks[0])
			if url != "" {
				_ = fileDownload(url, filename)

			}
		}
	}

	return filename
}

func fileDownload(url string, filename string) error {
	wavFile := storage_path + "/" + filename + ".wav"

	// Create blank file
	file, err := os.Create(wavFile)
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	// CDN 404 delay workaround
	var retries int = 20
	var resp *http.Response

	for retries > 0 {
		log.WithFields(log.Fields{
			"url":   url,
			"count": retries,
		}).Info("Downloading...")

		resp, err = client.Get(url)
		if err != nil {
			log.Error(err)
		}
		if resp.StatusCode == 200 {
			break
		}
		retries -= 1
		time.Sleep(500 * time.Millisecond)
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)

	defer file.Close()

	log.WithFields(log.Fields{
		"file": filename,
		"size": size,
	}).Info("Downloaded")

	// Convert to mp3
	err = wav2mp3(filename)
	if err != nil {
		log.WithFields(log.Fields{
			"file": filename,
		}).Error(err)
	}

	// Clean up wav part
	err = os.Remove(wavFile)
	if err != nil {
		log.Error(err)
	}

	return err
}

func mp3Concat(filename string, parts int) error {
	Streams := []*ffmpeg.Stream{}
	for i := 0; i < parts; i++ {
		Streams = append(Streams, ffmpeg.Input(storage_path+"/"+filename+"_"+fmt.Sprintf("%d", i)+".mp3"))
	}
	err := ffmpeg.Concat(Streams, ffmpeg.KwArgs{"v": 0, "a": 1}).
		Output(storage_path + "/" + filename + ".mp3").
		OverWriteOutput().
		Run()

	if err == nil {
		for i := 0; i < parts; i++ {
			err := os.Remove(storage_path + "/" + filename + "_" + fmt.Sprintf("%d", i) + ".mp3")
			if err != nil {
				log.Error(err)
			}
		}
	}
	return err
}

func wav2mp3(filename string) error {
	err := ffmpeg.Input(storage_path+"/"+filename+".wav").
		Output(storage_path+"/"+filename+".mp3", ffmpeg.KwArgs{"ar": 44100, "ac": 2, "b:a": "128k"}).
		OverWriteOutput().
		Run()
	return err
}
