// Copyright 2013 Rainer Volz. All rights reserved.
// See the LICENSE file for license information.

package gomosquittogo

import (
	core "github.com/rvolz/gomosquittogo/core"
	"log"
	"time"
)

// MQTT client definition
type Client struct {
	broker string // hostname of the broker
	port   int    // port number of the broker
	name   string // ID for client <-> broker communication, max length 23
	// if nil a random ID will be generated, CleanSession will then be false
	messages chan<- *core.MosquittoMessage // channel for incoming messages,
	// if nil incoming messages will be discarded
	cleanSession    bool // if true the broker will clean the session on client disconnect
	mosquitto       *core.MosquittoClient
	clientCreated   bool
	clientConnected bool
	subscribed      bool
	subscriptions   map[string]uint
	data            *core.MosquittoCallbackData
	will            *core.MosquittoMessage
	user            string // user name when authentication is required by the broker
	password        string // password when authentication is required by the broker
}

// Create a new Mosquitto client. The parameter "broker" is the net address
// of the MQTT broker you want to connect to. Pass an output channel via "messages"
// to receive incoming messages. Use nil if you only want to send messages.
//
// Besides these the client uses the default settings, which can
// changed by using the appropriate Setters before connecting:
// 	- name: will use a generated name to connect to the broker if none is set
// 	- port: MQTT standard port 1833
//
func NewClient(broker string, messages chan<- *core.MosquittoMessage) *Client {
	client := new(Client)
	client.name = ""
	client.messages = messages
	client.broker = broker
	client.port = core.MqttStandardPort
	client.cleanSession = false
	client.clientCreated = false
	client.clientConnected = false
	client.subscriptions = make(map[string]uint)
	return client
}

// Close must always be called to terminate connections and free resources.
// Will unsubscribe and clear wills if necessary.
func (client *Client) Close() {
	log.Println("Client Closing")
	if client.subscribed && !client.cleanSession {
		// Unsubscribe all topics
		for topic, _ := range client.subscriptions {
			client.UnsubscribeTopic(topic)
		}
		log.Println("Client unsubscribed")
	}
	if client.clientConnected {
		client.mosquitto.StopMessageCallback()
		dstatus := client.mosquitto.Disconnect()
		log.Println("Client disconnected: ", dstatus)
		// Clear the will, if there is one, to free the memory
		if client.will != nil {
			client.will = nil
			client.mosquitto.ClearWill()
		}
	}
	if client.clientCreated {
		statusStop := client.mosquitto.StopLoop(false)
		log.Println("Client loop stopped ", statusStop)
		time.Sleep(100 * time.Millisecond)
		log.Println("Destroying Client")
		client.mosquitto.DestroyInstance()
		client.clientCreated = false
		log.Println("Client destroyed")
	}
	core.Cleanup()
	log.Println("Client Closed")
}

// Set the name of the client that will be used for the connection to the broker.
// The MQTT max. length for names is 23.
func (client *Client) Name(name string) {
	client.name = name
	client.cleanSession = true
}

// Set to true if you want to instruct the broker to clear all subscriptions
// and messages when the client disconnects. Default is false. Can only be
// set to true if a name was set.
func (client *Client) CleanSession(cs bool) {
	if client.name != "" {
		client.cleanSession = cs
	}
}

// Use a different network port for communication with the broker. Default is
// the MQTT standard port 1883.
func (client *Client) Port(port int) {
	client.port = port
}

// Set the user name for authentication if required by the broker.
func (client *Client) User(user string) {
	client.user = user
}

// Set the password for authentication if required by the broker.
func (client *Client) Password(password string) {
	client.password = password
}

// Start the connection to the MQTT broker.
// Connect wil create the Mosquitto client, start the internal message loop
// and connect to the broker. If the client as an output channel it will
// also start the message callback to receive incoming messages.
// Returns false if an error occured.
func (client *Client) Connect() bool {
	// Create the client
	if client.clientCreated == false {
		client.data = (*core.MosquittoCallbackData)(nil)
		if client.messages != nil {
			client.data = new(core.MosquittoCallbackData)
			client.data.MessageChannel(client.messages)
		}
		if client.name != "" {
			client.mosquitto = core.NewNamedInstance(client.name, client.cleanSession, client.data)
		} else {
			client.mosquitto = core.NewInstance(client.data)
		}
		if client.mosquitto != nil {
			client.clientCreated = true
			client.mosquitto.StartLoop()
			client.mosquitto.SetLoginData(client.user, client.password)
			if client.will != nil {
				client.mosquitto.SetWillMessage(client.will.Topic, client.will.Payload, client.will.QoS, client.will.Retained)
				client.mosquitto.SetWill()
			}
		} else {
			return false
		}
	}
	client.mosquitto.StartLogCallback()
	client.mosquitto.StartConnectCallback()
	if client.clientCreated == true && client.clientConnected == false {
		status := client.mosquitto.Connect(client.broker, client.port, 60)
		if status == core.Success {
			client.clientConnected = true
			if client.messages != nil {
				client.mosquitto.StartMessageCallback()
				client.mosquitto.StartDisconnectCallback()
				client.mosquitto.StartSubscribeCallback()
				client.mosquitto.StartUnsubscribeCallback()
				client.mosquitto.StartPublishCallback()
			}
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// Subscribe to a broker topic. Parameters are the topic name or pattern
// and the QoS level desired. See http://mosquitto.org/man/mqtt-7.html for
// more info about subscription patterns.
func (client *Client) SubscribeTopic(topic string, qos uint) error {
	if client.clientConnected {
		status := client.mosquitto.Subscribe(topic, qos)
		if status == core.Success {
			client.subscribed = true
			client.subscriptions[topic] = qos
		}
		return status
	} else {
		return core.ErrNoCon
	}
}

// Unsubscribe from a subscribed topic.
func (client *Client) UnsubscribeTopic(topic string) error {
	if client.clientConnected {
		status := client.mosquitto.Unsubscribe(topic)
		if status == core.Success {
			delete(client.subscriptions, topic)
			if len(client.subscriptions) == 0 {
				client.subscribed = false
			}
		}
		return status
	} else {
		return core.ErrNoCon
	}
}

// Send a byte buffer to a broker/topic:
// 	- topic: name of the broker topic
// 	- payload: the bytes to send
// 	- qos: QoS level required for sending the message
// 	- retain: if true the broker will retain (keep) the message
func (client *Client) SendBytes(topic string, payload []byte, qos int, retain bool) error {
	if client.clientConnected {
		status := client.mosquitto.Publish(0, topic, payload, qos, retain)
		return status
	} else {
		return core.ErrNoCon
	}
}

// Convenience function to send a string to a broker/topic:
// 	- topic: name of the broker topic
// 	- payload: the string to send
// 	- qos: QoS level required for sending the message
// 	- retain: if true the broker will retain (keep) the message
func (client *Client) SendString(topic string, payload string, qos int, retain bool) error {
	if client.clientConnected {
		status := client.mosquitto.PublishString(0, topic, payload, qos, retain)
		return status
	} else {
		return core.ErrNoCon
	}
}

// Set a byte buffer as a will for a topic. Will messages will be sent to the topic
// subscribers by the broker if the connection to the client dies unexpectedly.
// Will messages must be set before connecting. Parameters:
// 	- topic: name of the broker topic
// 	- payload: the bytes to send
// 	- qos: QoS level required for sending the message
// 	- retain: if true the broker will retain (keep) the message
func (client *Client) WillBytes(topic string, payload []byte, qos int, retain bool) error {
	if !client.clientConnected {
		client.will = new(core.MosquittoMessage)
		client.will.Topic = topic
		client.will.Payload = payload
		client.will.PayloadLen = uint(len(payload))
		client.will.QoS = qos
		client.will.Retained = retain
		return core.Success
	} else {
		return core.ErrConnPending
	}
}

// Convenience method to set a string as a will for a topic. Will messages will be sent to the topic
// subscribers by the broker if the connection to the client dies unexpectedly.
// Will messages must be set before connecting. Parameters:
// 	- topic: name of the broker topic
// 	- payload: the string to send
// 	- qos: QoS level required for sending the message
// 	- retain: if true the broker will retain (keep) the message
func (client *Client) WillString(topic string, payload string, qos int, retain bool) error {
	return client.WillBytes(topic, ([]byte)(payload), qos, retain)
}
