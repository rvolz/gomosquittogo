// Copyright 2013 Rainer Volz.
// See the LICENSE file for license information.

/*
Tests for the Mosquitto MQTT Go wrapper.
Needs a local Mosquitto instance.
*/
package gomosquittogo

import (
	"bytes"
	"fmt"
	core "github.com/rvolz/gomosquittogo/core"
	"testing"
	"time"
)

var defaultBroker = "127.0.0.1"

func TestClientNewClient(t *testing.T) {
	client1 := NewClient(defaultBroker, nil)
	defer client1.Close()
	if client1 == nil {
		t.Error("New client without channel not created")
	}
	msgs := make(chan *core.MosquittoMessage, 1)
	client2 := NewClient(defaultBroker, msgs)
	defer client2.Close()
	if client2 == nil {
		t.Error("New client with channel not created")
	}
}

func TestClientClose(t *testing.T) {
	client1 := NewClient(defaultBroker, nil)
	client1.Close()
	msgs := make(chan *core.MosquittoMessage, 1)
	client2 := NewClient(defaultBroker, msgs)
	client2.Close()
}

func TestClientName(t *testing.T) {
	client1 := NewClient(defaultBroker, nil)
	defer client1.Close()
	client1.Name("testClient")
	if client1.name != "testClient" {
		t.Error("Client name not set")
	}
	if client1.cleanSession != true {
		t.Error("Named clients should have clean sessions set")
	}
}

func TestClientConnect(t *testing.T) {
	client1 := NewClient(defaultBroker, nil)
	status := client1.Connect()
	if !status {
		t.Error("Client not connected")
	}
	client1.Close()
}

func TestClientSubscribe(t *testing.T) {
	msgs := make(chan *core.MosquittoMessage, 1)
	client1 := NewClient(defaultBroker, msgs)
	defer client1.Close()
	client1.Name("testClient")
	client1.Connect()
	status := client1.SubscribeTopic("test", core.QosFireAndForget)
	if status != core.Success {
		t.Error("Client not subscribed due to %v", status)
	}
}

func TestClientUnsubscribe(t *testing.T) {
	msgs := make(chan *core.MosquittoMessage, 1)
	client1 := NewClient(defaultBroker, msgs)
	defer client1.Close()
	client1.Name("testClient")
	client1.Connect()
	client1.SubscribeTopic("test", core.QosFireAndForget)
	status := client1.UnsubscribeTopic("test")
	if status != core.Success {
		t.Error("Client not unsubscribed due to %v", status)
	}
}

func TestClientLoopAsync(t *testing.T) {
	msgs := make(chan *core.MosquittoMessage, 2)
	controlR := make(chan bool)
	client1 := NewClient(defaultBroker, msgs)
	client1.Name("testClient")
	client1.Connect()
	client1.SubscribeTopic("test2", core.QosFireAndForget)
	go func(ch <-chan *core.MosquittoMessage, control <-chan bool) {
		for {
			select {
			case x := <-ch:
				if x != nil {
					if x.Topic != "test2" {
						t.Error("Bad topic ", x.Topic)
					}
					if (string)(x.Payload) != "Hello World" {
						t.Error("Bad payload ", x.Payload)
					}
				} else {
					fmt.Printf("nil received\n")
				}
				break
			case y := <-control:
				if y == true {
					fmt.Println("Reader: Stopped")
					return
				}
				break
			}
		}
	}(msgs, controlR)
	time.Sleep(1 * time.Second)
	client1.SendString("test2", "Hello World", core.QosFireAndForget, false)
	fmt.Printf("sent\n")
	time.Sleep(10 * time.Second)
	controlR <- true
	client1.Close()
	close(controlR)
	close(msgs)
}

func TestClientSendBytes(t *testing.T) {
	var content []byte = ([]byte)("Hello World")
	msgs := make(chan *core.MosquittoMessage, 2)
	controlR := make(chan bool)
	client1 := NewClient(defaultBroker, msgs)
	client1.Name("testClient")
	client1.Connect()
	client1.SubscribeTopic("test2", core.QosFireAndForget)
	go func(ch <-chan *core.MosquittoMessage, control <-chan bool) {
		for {
			select {
			case x := <-ch:
				if x.Topic != "test2" {
					t.Error("Bad topic ", x.Topic)
				}
				if bytes.Compare(x.Payload, content) != 0 {
					t.Error("Bad payload ", x.Payload)
				}
			case y := <-control:
				if y == true {
					fmt.Println("Reader: Stopped")
					return
				}
			}
		}
	}(msgs, controlR)
	time.Sleep(1 * time.Second)
	client1.SendBytes("test2", content, core.QosFireAndForget, false)
	fmt.Printf("sent\n")
	time.Sleep(10 * time.Second)
	controlR <- true
	client1.Close()
	close(controlR)
	close(msgs)
}
