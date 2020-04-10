// https://www.eclipse.org/paho/clients/golang/

/*
Todo
make configurable....
-- log level
-- animations and colors per state
*/

package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	ag "github.com/gaussmeter/mqttagent"
	log "github.com/sirupsen/logrus"
	randstr "github.com/thanhpk/randstr"

	//import the Paho Go MQTT library
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var debug bool = true

var geoFence string = "unset"
var speed int = -1
var state string = "unset"
var pluggedIn bool = false
var chargeLimitSoc int = -1
var batteryLevel int = -1
var host string = "ws://192.168.1.51:9001"
var car string = "1"
var topicPrefix string = "teslamate/cars/"
var lumen string = "http://192.168.1.127:9000/lumen"
var user string = ""
var pass string = ""
var loopSleep time.Duration = 250

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	//log.WithFields(log.Fields{"topic": msg.Topic(), "payload": string(msg.Payload()) }).Info()
	if msg.Topic() == "teslamate/cars/1/geofence" {
		geoFence = string(msg.Payload())
	}
	if msg.Topic() == "teslamate/cars/1/speed" {
		speed, _ = strconv.Atoi(string(msg.Payload()))
	}
	if msg.Topic() == "teslamate/cars/1/state" {
		state = string(msg.Payload())
	}
	if msg.Topic() == "teslamate/cars/1/plugged_in" {
		pluggedIn, _ = strconv.ParseBool(string(msg.Payload()))
	}
	if msg.Topic() == "teslamate/cars/1/charge_limit_soc" {
		chargeLimitSoc, _ = strconv.Atoi(string(msg.Payload()))
	}
	if msg.Topic() == "teslamate/cars/1/battery_level" {
		batteryLevel, _ = strconv.Atoi(string(msg.Payload()))
	}
}

func getSetting(setting string, defaultValue string) (value string) {
	if os.Getenv(setting) != "" {
		log.WithFields(log.Fields{"configFrom":"env", setting:os.Getenv(setting)}).Info()
		return os.Getenv(setting)
	}
	log.WithFields(log.Fields{"configFrom":"default", setting:defaultValue}).Info()
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
}

func main() {
	agent := ag.NewAgent(host, "teslamater-"+randstr.String(4), user, pass)
	err := agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}
	agent.Subscribe(topicPrefix + car + "/#", f)

	var body string = "{\"animation\": \"cylon\", \"rgbw\": \"0,0,0,255\", \"percent\": 10.0, \"velocity\" : 30 }"
	var lastBody string = ""
	var lastState string = ""
	var velocity string = "1"
	var percent string = "0"
	var lastSendTime int64 = 0
	for !agent.IsTerminated() {
		if speed == 0 {
			velocity = "1"
		} else {
			velocity = fmt.Sprintf("%d", speed)
		}
		if batteryLevel != 0 && chargeLimitSoc != 0 {
			percent = fmt.Sprintf("%f", float32(batteryLevel)/float32(chargeLimitSoc)*100)
		} else {
			percent = "10"
		}
		loopSleep = 250
		switch true {
		case geoFence == "unset" || state == "unset" || speed == -1 || batteryLevel == -1 || chargeLimitSoc == -1 :
			log.Info("too many unset values")
			loopSleep = 3000
			break
		case geoFence == "Home" && pluggedIn == true:
			body = "{\"animation\": \"bargraph\", \"rgbw\": \"0,255,0,0\", \"percent\": " + percent + ", \"velocity\" : " + velocity + "}"
			break
		case geoFence == "Home" && pluggedIn == false:
			body = "{\"animation\": \"bargraph\", \"rgbw\": \"0,128,128,0\", \"percent\": " + percent + ", \"velocity\" : " + velocity + "}"
			break
		case geoFence != "Home":
			body = "{\"animation\": \"rainbow\", \"rgbw\": \"0,0,0,0\", \"percent\": " + percent + ", \"velocity\" : " + velocity + "}"
			break
		}
		if body != lastBody || state != lastState || time.Now().Unix() - lastSendTime > 90 {
			//todo: ?escape json body in log?
			log.WithFields(log.Fields{"state": fmt.Sprintf("GeoFence: %s, Speed: %d, State: %s, Plugged In: %t, Charge Limit: %d, Charge Level: %d, Percent: %s", geoFence, speed, state, pluggedIn, chargeLimitSoc, batteryLevel, percent), "body": body}).Info()
			lastBody = body
			lastState = state
			httpClient := &http.Client{}
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
			lastSendTime = time.Now().Unix()
		}
		time.Sleep(loopSleep* time.Millisecond)
	}

}
