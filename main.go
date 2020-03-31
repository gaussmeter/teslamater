// https://www.eclipse.org/paho/clients/golang/

/*
Todo

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

	//import the Paho Go MQTT library
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var debug bool = false

var geoFence string = ""
var speed int = 0
var state string = ""
var pluggedIn bool = false
var chargeLimitSoc int = 0
var batteryLevel int = 0

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
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
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("ws://192.168.1.51:9001")
	opts.SetClientID("go-simple")
	opts.SetDefaultPublishHandler(f)

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//subscribe to the topic /teslamate/cars/1/# and request messages to be delivered
	//at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe("teslamate/cars/1/#", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	var body string = "{\"animation\": \"cylon\", \"rgbw\": \"0,0,0,255\", \"percent\": 10.0, \"velocity\" : 30 }"
	var lastBody string = ""
	var lastState string = ""
	var velocity string = "1"
	var percent string = "0"
	for {
		if speed == 0 {
			velocity = "1"
		} else {
			velocity = fmt.Sprintf("%d", speed)
		}
		percent = fmt.Sprintf("%f", float32(batteryLevel) / float32(chargeLimitSoc) * 100)
		switch true {
		case geoFence == "Home" && pluggedIn == true:
			body = "{\"animation\": \"bargraph\", \"rgbw\": \"0,255,0,0\", \"percent\": "+percent+", \"velocity\" : "+velocity+"}"
			break
		case geoFence == "Home" && pluggedIn == false:
			body = "{\"animation\": \"bargraph\", \"rgbw\": \"0,128,128,0\", \"percent\": "+percent+", \"velocity\" : "+velocity+"}"
			break
		case geoFence != "Home":
			body = "{\"animation\": \"rainbow\", \"rgbw\": \"0,0,0,0\", \"percent\": "+percent+", \"velocity\" : "+velocity+"}"
			break
		}
		if body != lastBody || state != lastState { // Todo: or longer than 90 seconds from last change
			fmt.Printf("GeoFence: %s, Speed: %d, State: %s, Plugged In: %t, Charge Limit: %d, Charge Level: %d, Percent: %s \n%s\n", geoFence, speed, state, pluggedIn, chargeLimitSoc, batteryLevel, percent, body)
			lastBody = body
			lastState = state
			httpClient := &http.Client{}
			req, err := http.NewRequest(http.MethodPut, "http://192.168.1.127:9000/lumen", strings.NewReader(body))
			if debug == true {
				fmt.Print(err)
			}
			resp, err := httpClient.Do(req)
			if debug == true {
				fmt.Print(resp)
				fmt.Print(err)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}

	//c.Disconnect(250)
}
