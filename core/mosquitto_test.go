// Copyright 2013 Rainer Volz.
// See the LICENSE file for license information.

/*
Tests for the Mosquitto MQTT Go wrapper.
Needs a local Mosquitto instance, address 127.0.0.1 is assumed.
*/
package core

import (
	"testing"
	"time"
)

func TestVersion(t *testing.T) {
	const (
		major = 1
		minor = 2
	)

	defer Cleanup()
	if vmajor, vminor, _ := Version(); vmajor != major || vminor != minor {
		t.Errorf("Version() = %v.%v, want %v.%v", vmajor, vminor, major, minor)
	}
}

func TestNewInstance(t *testing.T) {
	defer Cleanup()
	mosq := NewInstance(nil)
	if mosq == nil {
		t.Errorf("NewInstance() shouldn't be nil")
	} else {
		mosq.DestroyInstance()
	}
}

func TestNewNamedInstance(t *testing.T) {
	defer Cleanup()
	mosq := NewNamedInstance("test", true, nil)
	if mosq == nil {
		t.Errorf("NewInstance() shouldn't be nil")
	} else {
		mosq.DestroyInstance()
	}
}

func TestConnect(t *testing.T) {
	defer Cleanup()
	mosq := NewInstance(nil)
	if mosq == nil {
		t.Errorf("NewInstance() shouldn't be nil")
	}
	host := "127.0.0.1"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	status := mosq.Connect(host, 0, 10)
	if status != Success {
		t.Errorf("Connect(%v) shouldn't fail with %v", host, status)
	}
	mosq.Disconnect()
	mosq.StopLoop(false)
	mosq.DestroyInstance()
}

func TestConnect_WithWill(t *testing.T) {
	defer Cleanup()
	mosq := NewInstance(nil)
	if mosq == nil {
		t.Errorf("NewInstance() shouldn't be nil")
	}
	host := "127.0.0.1"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	mosq.SetWillMessage("test-will-bytes", ([]byte)("If you read this my connection died ..."), QosAssuredDelivery, false)
	status := mosq.SetWill()
	if status != Success {
		t.Errorf("Unable to set will: %v", status)
	}
	statusc := mosq.Connect(host, 0, 10)
	if statusc != Success {
		t.Errorf("Connect(%v) shouldn't fail with %v", host, statusc)
	}
	mosq.Disconnect()
	status2 := mosq.ClearWill()
	if status2 != Success {
		t.Errorf("Unable to clear will: %v", status2)
	}
	mosq.StopLoop(false)
	mosq.DestroyInstance()
}

func TestDisconnect(t *testing.T) {
	defer Cleanup()
	mosq := NewInstance(nil)
	host := "127.0.0.1"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	mosq.StartDisconnectCallback()
	mosq.Connect(host, 0, 10)
	status2 := mosq.Disconnect()
	if status2 != Success {
		t.Errorf("Disonnect(%v) should not fail with %v", host, status2)
	}
	mosq.StopLoop(false)
	mosq.DestroyInstance()
}

func TestPublishString(t *testing.T) {
	defer Cleanup()
	mosq := NewInstance(nil)
	host := "127.0.0.1"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	mosq.Connect(host, 0, 10)
	status := mosq.PublishString(1, "test", "Hello Go World!", 0, false)
	if status != Success {
		t.Errorf("PublishString(%v) should not fail with %v", host, status)
	}
	mosq.Disconnect()
	mosq.StopLoop(false)
	mosq.DestroyInstance()
}

func TestPublish(t *testing.T) {
	defer Cleanup()
	mosq := NewInstance(nil)
	host := "127.0.0.1"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	mosq.Connect(host, 0, 10)
	inBytes := []byte("Hello Go World (in bytes)!")
	status := mosq.Publish(1, "test", inBytes, 0, false)
	if status != Success {
		t.Errorf("Publish(%v) should not fail with %v", host, status)
	}
	mosq.Disconnect()
	mosq.StopLoop(false)
	mosq.DestroyInstance()
}

func TestSubscribe(t *testing.T) {
	defer Cleanup()
	mosq := NewInstance(nil)
	host := "127.0.0.1"
	topic := "testTopic"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	mosq.Connect(host, 0, 10)
	status := mosq.Subscribe(topic, QosFireAndForget)
	if status != Success {
		t.Errorf("Subscribe(%v) should not fail with %v", topic, status)
	}
	mosq.Disconnect()
	mosq.StopLoop(false)
	mosq.DestroyInstance()
}

func TestUnsubscribe(t *testing.T) {
	defer Cleanup()
	mosq := NewInstance(nil)
	host := "127.0.0.1"
	topic := "testTopic"
	topic2 := "testTopic2"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	mosq.Connect(host, 0, 10)
	mosq.Subscribe(topic, QosFireAndForget)
	status := mosq.Unsubscribe(topic)
	if status != Success {
		t.Errorf("Unsubscribe(%v) should not fail with %v", topic, status)
	}
	status2 := mosq.Unsubscribe(topic2)
	if status2 != Success {
		t.Errorf("Unsubscribe(%v) of not susscribed topics should succeed too, error %v", topic2, status2)
	}
	mosq.Disconnect()
	mosq.StopLoop(false)
	mosq.DestroyInstance()
}

func TestStartMessageCallback(t *testing.T) {
	defer Cleanup()
	messages := make(chan *MosquittoMessage, 4)
	data := new(MosquittoCallbackData)
	data.out = messages
	mosq := NewInstance(data)
	host := "127.0.0.1"
	topic := "test"
	content := "Hello Go World Callback!"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	mosq.Connect(host, 0, 10)
	mosq.StartMessageCallback()
	mosq.Subscribe(topic, QosFireAndForget)
	mosq.PublishString(1, topic, content, QosAcknowledgeDelivery, false)
	time.Sleep(time.Second * 2)
	x := <-messages
	if x.Topic != topic {
		t.Errorf("Bad message topic received %v, expected %v", x.Topic, topic)
	}
	if x.PayloadLen != (uint)(len(content)) {
		t.Errorf("Bad message length received %v, expected %v", x.PayloadLen, len(content))
	}
	mosq.Unsubscribe(topic)
	mosq.Disconnect()
	mosq.StopLoop(false)
	close(messages)
	mosq.DestroyInstance()
}

func TestLoop(t *testing.T) {
	defer Cleanup()
	messages := make(chan *MosquittoMessage, 4)
	data := new(MosquittoCallbackData)
	data.out = messages
	mosq := NewInstance(data)
	host := "127.0.0.1"
	topic := "test"
	content := "Hello Go World Callback!"
	mosq.StartLoop()
	mosq.StartLogCallback()
	mosq.StartConnectCallback()
	mosq.Connect(host, 0, 10)
	mosq.StartMessageCallback()
	mosq.Subscribe(topic, QosFireAndForget)
	mosq.PublishString(1, topic, content, QosAcknowledgeDelivery, false)
	time.Sleep(1 * time.Second)
	x := <-messages
	if x.Topic != topic {
		t.Errorf("Bad message topic received %v, expected %v", x.Topic, topic)
	}
	if x.PayloadLen != (uint)(len(content)) {
		t.Errorf("Bad message length received %v, expected %v", x.PayloadLen, len(content))
	}
	mosq.Unsubscribe(topic)
	mosq.Disconnect()
	mosq.StopLoop(false)
	close(messages)
	mosq.DestroyInstance()
}
