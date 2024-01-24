package messages

import (
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)


var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

func Subscribe1() {
	opts := mqtt.NewClientOptions().AddBroker("tcp://edgex-mqtt-broker:1883")
	opts.SetDefaultPublishHandler(messageHandler)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	token = client.Subscribe("topic", 0, nil)
	token.Wait()
	fmt.Printf("Successfully subscribed to topic: %s\n", "topic")

	select {} // block forever
}
