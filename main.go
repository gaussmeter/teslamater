// https://www.eclipse.org/paho/clients/golang/

/*
Todo

manage MQTT connection/reconnection:wq

make configurable....
-- MQTT user, pass, connection.
-- car number
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

var geoFence string = ""
var speed int = 0
var state string = ""
var pluggedIn bool = false
var chargeLimitSoc int = 0
var batteryLevel int = 0

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

func main() {
	agent := ag.NewAgent("ws://192.168.1.51:9001", "teslamater-"+randstr.String(4))
	err := agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}
	agent.Subscribe("teslamate/cars/1/#", f)

	var body string = "{\"animation\": \"cylon\", \"rgbw\": \"0,0,0,255\", \"percent\": 10.0, \"velocity\" : 30 }"
	var lastBody string = ""
	var lastState string = ""
	var velocity string = "1"
	var percent string = "0"
	for !agent.IsTerminated() {
		if speed == 0 {
			velocity = "1"
		} else {
			velocity = fmt.Sprintf("%d", speed)
		}
		percent = fmt.Sprintf("%f", float32(batteryLevel)/float32(chargeLimitSoc)*100)
		switch true {
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
		if body != lastBody || state != lastState { // Todo: or longer than 90 seconds from last change
			//todo: ?escape json body in log?
			log.WithFields(log.Fields{"state": fmt.Sprintf("GeoFence: %s, Speed: %d, State: %s, Plugged In: %t, Charge Limit: %d, Charge Level: %d, Percent: %s", geoFence, speed, state, pluggedIn, chargeLimitSoc, batteryLevel, percent), "body": body}).Info()
			lastBody = body
			lastState = state
			httpClient := &http.Client{}
			req, err := http.NewRequest(http.MethodPut, "http://192.168.1.127:9000/lumen", strings.NewReader(body))
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
		}
		time.Sleep(250 * time.Millisecond)
	}

}
