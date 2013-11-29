gomosquittogo
=============

gomosquittogo is a Go (cgo) wrapper for the client library of the Mosquitto MQTT broker. 

## Building

This project was built and tested with Go 1.1 and libmosquitto 1.2. cgo support and an installed libmosquitto is required to build it:

    go get github.com/rvolz/gomosquittogo

## Testing

Use the usual go command to test:

	go test github.com/rvolz/gomosquittogo ...

Most tests assume that a local (127.0.0.1) Mosquitto instance is available.

## Using

The library consists of two packages. The main package `gomosquittogo` provides a high-level client for MQTT communication. The `core` package contains low-level wrappers for many libmosquitto API functions. If the functionality of the high-level client is not adequate for your needs, just use the core package to create your own.

The library is in an early development stage. It provides functionality to send and receive MQTT messages. Other functions like authentication, SSL ... will follow.

API Documentation: [![GoDoc](http://godoc.org/github.com/rvolz/gomosquittogo?status.png)](http://godoc.org/github.com/rvolz/gomosquittogo)


### Examples


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
	client.Subscribe("test")
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