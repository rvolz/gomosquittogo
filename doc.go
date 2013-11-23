// Copyright 2013 Rainer Volz. All rights reserved.
// See the LICENSE file for license information.

/*
Package gomosquittogo provides, as the name might imply, a Go language wrapper
(cgo) for the client library of the Mosquitto MQTT broker. See
http://mosquitto.org/man/mqtt-7.html for more information about the terminology
used here.

Create an anonymous client for a broker and send a string:

	client := NewClient("127.0.0.1", nil)
	defer client.Close()
	client.Connect()
	client.SendString("my topic","this is my message")

Create a named client and send some bytes:

	client := NewClient("127.0.0.1", nil)
	defer client.Close()
	client.Name("test client")
	client.Port(1884) // uses a non-standard port
	client.Connect()
	client.Send("my topic",([]byte)("this is my message"))

Receive messages:

	// create a buffered output channel, size depends on the traffic expected
	messages := make(<-chan *MosquittoMessage, 100)
	client := NewClient("127.0.0.1", messages)
	defer client.Close()
	client.Name("test client")
	client.Connect()
	go func(incoming <-chan *MosquittoMessage, control <-chan bool) {
		for {
			select {
			case message := <-incoming:
				// process incoming message here
			case y := <-control:
				// cheap solution to stop this goroutine
				return
			}
		}
	}(messages, control)
	...


*/
package gomosquittogo
