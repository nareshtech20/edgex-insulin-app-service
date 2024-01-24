//
// Copyright (c) 2021 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package functions

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
        "github.com/edgexfoundry/go-mod-core-contracts/v3/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/http"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/dtos/common"
	"github.com/google/uuid"
)

type ActionRequest struct {
	Action       string `json:"action"`
	DeviceName   string `json:"deviceName"`
	CommandName  string `json:"commandName"`
	ResourceName string `json:"resourceName"`
	Value        string `json:"value"`
}

type SendCommand struct {
}

func NewSendCommand() SendCommand {
	return SendCommand{}
}

func(s *SendCommand) CheckAndSendCommand(funcCtx interfaces.AppFunctionContext, data interface{})  (bool, interface{}) {

	lc := funcCtx.LoggingClient()

	lc.Debug("in CheckAndSendCommand")

	if event, ok := data.(dtos.Event); ok {
		for _, reading := range event.Readings {
			intVar, err := strconv.Atoi(reading.Value)
			if err != nil {
				fmt.Errorf("CheckAndSendCommand int conversion error: %s", err.Error())
				return false, data
			}
			if reading.ResourceName == "Uint16" && intVar > 120 {
				lc.Info("Sending Insulin actuate command...")

				device := "insulin-injector"
				command := "WriteBoolValue"
				settings := make(map[string]string)
				settings["Bool"] = "true"
				settings["EnableRandomization_Bool"] = "false"
				funcCtx.CommandClient().IssueSetCommandByName(context.Background(), device, command, settings)

				go stopInsulin(funcCtx)

				//Sending notifications
				notify(funcCtx, intVar)

				lc.Info("Sending glucose set command...")

				//device = "Random-UnsignedInteger-Device"
				device = "blood-glucose-monitor"
				command = "WriteUint16Value"
				settings = make(map[string]string)
				settings["Uint16"] = "91"
				settings["EnableRandomization_Uint16"] = "false"
				funcCtx.CommandClient().IssueSetCommandByName(context.Background(), device, command, settings)

			}
		}
	}

	return true, data
}

func stopInsulin(funcCtx interfaces.AppFunctionContext) {

	lc := funcCtx.LoggingClient()
	lc.Info("Scheduling Insulin stop command...")
	time.Sleep(time.Minute)
	lc.Info("Sending Insulin stop command...")

	//device := "Random-Boolean-Device"
	device := "insulin-injector"
	command := "WriteBoolValue"
	settings := make(map[string]string)
	settings["Bool"] = "false"
	settings["EnableRandomization_Bool"] = "false"
	funcCtx.CommandClient().IssueSetCommandByName(context.Background(), device, command, settings)
}

func notify(funcCtx interfaces.AppFunctionContext, reading int) {
	lc := funcCtx.LoggingClient()
 	// Create a new notification client edgex-support-notifications 10.43.117.99
	client := http.NewNotificationClient("http://edgex-support-notifications:59860", nil, false)

	// Create a new notification
	notification := requests.AddNotificationRequest{
		BaseRequest: common.BaseRequest{
			RequestId: uuid.New().String(),  // Generate a new UUID
			Versionable: common.Versionable{
				ApiVersion: "v3",  // Replace with the API version you're using
			},
		},
		Notification: dtos.Notification{
			Sender:   "Glucose-Monitor-Device",
			Category: "ALERT",
			Severity: "CRITICAL",
			Content:  "Glucose level - "+strconv.Itoa(reading),
			Labels:   []string{"glucose", "alert"},
			Status:   "NEW",
			ContentType: "json",
			Description: "High Glucose Level Alert",
		},
	}

	// Send the notification
	_, err := client.SendNotification(context.Background(), []requests.AddNotificationRequest{notification})
	if err != nil {
		// Handle error
		fmt.Printf("Error sending notification: %v", err)
	}
	lc.Info("Notification sent successfully")
}

func (s *SendCommand) SendCommand(funcCtx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	lc := funcCtx.LoggingClient()

	lc.Info("Sending Command")

	if data == nil {
		return false, errors.New("SendCommand: No data received")
	}

	if funcCtx.CommandClient() == nil {
		return false, errors.New("SendCommand: Command client is not available")
	}

	/*actionRequest, ok := data.(ActionRequest)
	if !ok {
		return false, errors.New("SendCommand: Data received is not the expected 'ActionRequest' type")
	}
	action := actionRequest.Action
	device := actionRequest.DeviceName
	command := actionRequest.CommandName*/

	event, ok := data.(dtos.Event)
	if !ok {
		return false, fmt.Errorf("function LogEventDetails in pipeline '%s', type received is not an Event", funcCtx.PipelineId())
	}


	action := "set"
	device := event.DeviceName
	command := "WriteUint16Value"

	var response interface{}
	var err error

	switch action {
	case "set":
		lc.Infof("executing %s action", action)
		lc.Infof("Sending command '%s' for device '%s'", command, device)

		settings := make(map[string]string)
		settings["Uint16"] = "88"
		response, err = funcCtx.CommandClient().IssueSetCommandByName(context.Background(), device, command, settings)
		//response, err = funcCtx.CommandClient().IssueSetCommand("Random-Integer-Device", "Uint16", "100") 
		if err != nil {
			return false, fmt.Errorf("failed to send '%s' set command to '%s' device: %s", command, device, err.Error())
		}

	case "get":
		lc.Infof("executing %s action", action)
		lc.Infof("Sending command '%s' for device '%s'", command, device)
		response, err = funcCtx.CommandClient().IssueGetCommandByName(context.Background(), device, command, false, true)
		if err != nil {
			return false, fmt.Errorf("failed to send '%s' get command to '%s' device: %s", command, device, err.Error())
		}

	default:
		lc.Errorf("Invalid action requested: %s", action)
		return false, nil
	}

	return true, response
}
