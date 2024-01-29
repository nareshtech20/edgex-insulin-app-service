package messages

import (
	"fmt"
	"os"
    "net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"strconv"

	log "github.com/sirupsen/logrus"
	mqtt "github.com/eclipse/paho.mqtt.golang"

)

type AlertData struct {
	AssetId int `json:"assetId"`
	EventCode  string    `json:"eventCode"`
	DeviceName  string    `json:"deviceName"`
	Value int `json:"value"`
	Message string    `json:"message"`
	SensorName string `json:"sensorName"`
}

func makeMessageHandler() mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {

		fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		intVar, err := strconv.Atoi(string(msg.Payload()))
		alertData := &AlertData{
			AssetId: 34,
			EventCode: "INSULIN_ACTUATOR",
			DeviceName:  "Patient_Monitor_19524",
			Value: intVar,
			Message: "Anomaly observed for Patient_Monitor_19524 - "+string(msg.Payload()) ,
			SensorName: "glucose",
		}

		jsonData, err := json.Marshal(alertData)
		if err != nil {
				log.Error("Json Marshal...%+v", alertData)
		}

		res, err := postData("", "", "POST", jsonData)
		if err != nil {
				log.Error("Json Marshal...")
		}
		log.Info("postData.."+res)

		log.Info("Sending Insulin actuate command...")

		device := "insulin-injector"
		command := "WriteBoolValue"
		settings := make(map[string]string)
		settings["Bool"] = "true"
		settings["EnableRandomization_Bool"] = "false"

		jsonData, err = json.Marshal(settings)
		if err != nil {
			log.Error("Json Marshal...")
		}
		res, err = sendCommand(device, command, "post", jsonData)
		if err != nil {
			log.Error("sendCommand error...%v", err)
		}
		log.Debug("sendCommand.."+res)
		//go stopInsulin()
	}
}

func sendCommand(deviceName string, commandName string, method string, jsonData []byte) (string, error) {
	url := "http://edgex-device-virtual:59900/api/v3/device/name/" + deviceName + "/command/" + commandName

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(respBody), nil
}

func postData(deviceName string, commandName string, method string, jsonData []byte) (string, error) {

	log.Info("Sending live data...")
	url := "http://10.239.80.228:8085/api/alerts/createAppAlert"

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error occurred during http request: %s", err.Error())
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %s", err.Error())
	}

	return string(body), nil
}

/*
func stopInsulin() {

	log.Info("Scheduling Insulin stop command...")
	time.Sleep(time.Minute)
	log.Info("Sending Insulin stop command...")

	//device := "Random-Boolean-Device"
	device := "insulin-injector"
	command := "WriteBoolValue"
	settings := make(map[string]string)
	settings["Bool"] = "false"
	settings["EnableRandomization_Bool"] = "false"

	commandClient := command.NewCommandClient()
	commandClient.IssueSetCommandByName(context.Background(), device, command, settings)
}
*/
func Subscribe() {

	opts := mqtt.NewClientOptions().AddBroker("tcp://edgex-mqtt-broker:1883")
	//opts.SetDefaultPublishHandler(messageHandler)
	opts.SetDefaultPublishHandler(makeMessageHandler())
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	token = client.Subscribe("high-glucose", 0, nil)
	token.Wait()
	fmt.Printf("Successfully subscribed to topic: %s\n", "high-glucose")

	select {} // block forever
    	// Start a goroutine to keep the application running until interrupted.
    	//go func() {
        //	select {}
    	//}()	
}
