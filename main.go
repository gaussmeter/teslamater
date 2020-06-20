// https://www.eclipse.org/paho/clients/golang/

/*
Todo
make configurable....
-- log level
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	ag "github.com/gaussmeter/mqttagent"
	log "github.com/sirupsen/logrus"
	randstr "github.com/thanhpk/randstr"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	pkger "github.com/markbates/pkger"
)

type Config struct {
	HomePluggedInAsleep HomePluggedInAsleep `json:"homepluggedinasleep"`
	HomePluggedInAwake  HomePluggedInAwake  `json:"homepluggedinawake"`
	HomeUnpluggedAsleep HomeUnpluggedAsleep `json:"homeunpluggedasleep"`
	HomeUnpluggedAwake  HomeUnpluggedAwake  `json:"homeunpluggedawake"`
	NotHomeAlseep       NotHomeAlseep       `json:"nothomeasleep"`
	NotHomeAwake        NotHomeAwake        `json:"nothomeawake"`
	Default             Default             `json:"default"`
	UnHealthy           UnHealthy           `json:"unhealthy"`
}
type Lumen struct {
	Bright    int    `json:"bright"`
	Animation string `json:"animation"`
	Percent   int    `json:"percent"`
	Velocity  int    `json:"velocity"`
	R         int    `json:"r"`
	G         int    `json:"g"`
	B         int    `json:"b"`
	W         int    `json:"w"`
	R2        int    `json:"r2"`
	G2        int    `json:"g2"`
	B2        int    `json:"b2"`
	W2        int    `json:"w2"`
}
type Default struct {
	Lumen Lumen `json:"lumen"`
}
type NotHomeAlseep struct {
	Lumen Lumen `json:"lumen"`
}
type HomePluggedInAsleep struct {
	Lumen Lumen `json:"lumen"`
}
type HomeUnpluggedAsleep struct {
	Lumen Lumen `json:"lumen"`
}
type NotHomeAwake struct {
	Lumen Lumen `json:"lumen"`
}
type HomePluggedInAwake struct {
	Lumen Lumen `json:"lumen"`
}
type HomeUnpluggedAwake struct {
	Lumen Lumen `json:"lumen"`
}
type UnHealthy struct {
	Lumen Lumen `json:"lumen"`
}

var config Config

var debug bool = true

var geoFence string = "unset"
var speed int = -1
var state string = "unset"
var pluggedIn bool = false
var chargeLimitSoc int = -1
var batteryLevel int = -1
var healthy bool = true
var host string = "ws://192.168.1.51:9001"
var car string = "1"
var home string = "Home"
var topicPrefix string = "teslamate/cars/"
var lumen string = "http://192.168.1.127:9000/lumen"
var user string = ""
var pass string = ""
var loopSleep time.Duration = 250

var httpClient = &http.Client{ Timeout: time.Second * 5 }

var geoFenceMq MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		geoFence = string(msg.Payload())
}
var speedMq MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		speed, _ = strconv.Atoi(string(msg.Payload()))
}
var stateMq MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		state = string(msg.Payload())
}
var pluggedInMq MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		pluggedIn, _ = strconv.ParseBool(string(msg.Payload()))
}
var chargeLimitSocMq MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		chargeLimitSoc, _ = strconv.Atoi(string(msg.Payload()))
}
var batteryLevelMq MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		batteryLevel, _ = strconv.Atoi(string(msg.Payload()))
}
var healthyMq MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		healthy, _ = strconv.ParseBool(string(msg.Payload()))
}

func getSetting(setting string, defaultValue string) (value string) {
	if os.Getenv(setting) != "" {
		log.WithFields(log.Fields{"configFrom": "env", setting: os.Getenv(setting)}).Info()
		return os.Getenv(setting)
	}
	log.WithFields(log.Fields{"configFrom": "default", setting: defaultValue}).Info()
	return defaultValue
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	// get config from environment
	log.SetReportCaller(false)
	host = getSetting("MQTT_HOST", host)
	user = getSetting("MQTT_USER", user)
	pass = getSetting("MQTT_PASS", pass)
	lumen = getSetting("LUMEN_HOST", lumen)
	car = getSetting("CAR_NUMBER", car)
	home = getSetting("GEOFENCE_HOME", home)
}

func main() {
	configJSON, err := os.Open("config.json")
	if err != nil {
		log.WithFields(log.Fields{"Error": err.Error()}).Warn("config.json not found, default will be used.")
	}
	defaultJSON, err := pkger.Open("/default.json")
	if err != nil {
		log.WithFields(log.Fields{"Error": err.Error()}).Error()
		os.Exit(1)
	}
	byteValue, _ := ioutil.ReadAll(defaultJSON)
	json.Unmarshal(byteValue, &config)
	byteValue, _ = ioutil.ReadAll(configJSON)
	json.Unmarshal(byteValue, &config)
	configJSON.Close()
	defaultJSON.Close()

	agent := ag.NewAgent(host, "teslamater-"+randstr.String(4), user, pass)
	err = agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}
	agent.Subscribe(topicPrefix+car+"/geofence", geoFenceMq)
	agent.Subscribe(topicPrefix+car+"/speed", speedMq)
	agent.Subscribe(topicPrefix+car+"/state", stateMq)
	agent.Subscribe(topicPrefix+car+"/plugged_in", pluggedInMq)
	agent.Subscribe(topicPrefix+car+"/charge_limit_soc", chargeLimitSocMq)
	agent.Subscribe(topicPrefix+car+"/battery_level", batteryLevelMq)
	agent.Subscribe(topicPrefix+car+"/healthy", healthyMq)

	var body string = ""
	var lastBody string = ""
	var lastState string = ""
	var velocity int = 1
	var percent int = 0
	var lastSendTime int64 = 0
	for !agent.IsTerminated() {
		if speed == 0 {
			velocity = 1
		} else {
			velocity = speed
		}
		if batteryLevel != 0 && chargeLimitSoc != 0 {
			percent = int((float32(batteryLevel) / float32(chargeLimitSoc)) * 100)
		} else {
			percent = 10
		}
		loopSleep = 250
		switch true {
		case healthy == false:
			out, _ := json.Marshal(config.UnHealthy.Lumen)
			body = string(out)
			break
		case state == "unset" || speed == -1 || batteryLevel == -1 || chargeLimitSoc == -1:
			out, _ := json.Marshal(config.Default.Lumen)
			body = string(out)
			log.Info("too many unset values")
			loopSleep = 3000
			break
		case geoFence == home && pluggedIn == true && state != "asleep":
			config.HomePluggedInAwake.Lumen.Percent = percent
			config.HomePluggedInAwake.Lumen.Velocity = velocity
			out, _ := json.Marshal(config.HomePluggedInAwake.Lumen)
			body = string(out)
			break
		case geoFence == home && pluggedIn == true && state == "asleep":
			config.HomePluggedInAsleep.Lumen.Percent = percent
			config.HomePluggedInAsleep.Lumen.Velocity = velocity
			out, _ := json.Marshal(config.HomePluggedInAsleep.Lumen)
			body = string(out)
			break
		case geoFence == home && pluggedIn == false && state != "asleep":
			config.HomeUnpluggedAwake.Lumen.Percent = percent
			config.HomeUnpluggedAwake.Lumen.Velocity = velocity
			out, _ := json.Marshal(config.HomeUnpluggedAwake.Lumen)
			body = string(out)
			break
		case geoFence == home && pluggedIn == false && state == "asleep":
			config.HomeUnpluggedAsleep.Lumen.Percent = percent
			config.HomeUnpluggedAsleep.Lumen.Velocity = velocity
			out, _ := json.Marshal(config.HomeUnpluggedAsleep.Lumen)
			body = string(out)
			break
		case geoFence != home && state != "asleep":
			config.NotHomeAwake.Lumen.Percent = percent
			config.NotHomeAwake.Lumen.Velocity = velocity
			out, _ := json.Marshal(config.NotHomeAwake.Lumen)
			body = string(out)
			break
		case geoFence != home && state == "asleep":
			config.NotHomeAlseep.Lumen.Percent = percent
			config.NotHomeAlseep.Lumen.Velocity = velocity
			out, _ := json.Marshal(config.NotHomeAlseep.Lumen)
			body = string(out)
			break
		}
		if body != lastBody || state != lastState || time.Now().Unix() - lastSendTime > 90 {
			//todo: ?escape json body in log?
			log.WithFields(log.Fields{"state": fmt.Sprintf("GeoFence: %s, Speed: %d, State: %s, Plugged In: %t, Healthy: %t, Charge Limit: %d, Charge Level: %d, Percent: %d", geoFence, speed, state, pluggedIn, healthy, chargeLimitSoc, batteryLevel, percent), "body": body}).Info()
			lastBody = body
			lastState = state
			req, err := http.NewRequest(http.MethodPut, lumen, strings.NewReader(body))
			if debug == true && err != nil {
				log.WithFields(log.Fields{"error": err.Error()}).Info()
			}
			resp, err := httpClient.Do(req)
			if debug == true && err == nil {
				log.WithFields(log.Fields{"statusCode": resp.StatusCode}).Info("response")
			}
			if debug == true && err != nil {
				log.WithFields(log.Fields{"error": err.Error()}).Info()
			}
			resp.Body.Close()
			lastSendTime = time.Now().Unix()
		}
		time.Sleep(loopSleep * time.Millisecond)
		httpClient.CloseIdleConnections()
	}

}
