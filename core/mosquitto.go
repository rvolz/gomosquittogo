// Copyright 2013 Rainer Volz. All rights reserved.
// See the LICENSE file for license information.

// Package core provides low-level wrappers for the libmosquitto library functions.

package core

/*
#cgo LDFLAGS: -lmosquitto
#include <mosquitto.h>
#include <stdlib.h>

extern void setConnectCallback(struct mosquitto *mosq, bool);
extern void setDisconnectCallback(struct mosquitto *mosq, bool);
extern void setLogCallback(struct mosquitto *mosq, bool);
extern void setMessageCallback(struct mosquitto *mosq, bool);
extern void setPublishCallback(struct mosquitto *mosq, bool);
extern void setSubscribeCallback(struct mosquitto *mosq, bool);
extern void setUnsubscribeCallback(struct mosquitto *mosq, bool);
extern struct mosquitto *mosquitto_new_anonymous(void* data);
extern struct mosquitto *mosquitto_new_anonymous_no_data();

*/
import "C"

import (
	"fmt"
	"log"
	"time"
	"unsafe"
)

const (
	// The max length for client names
	MqttMaxClientName int = 23
	// The MQTT standard port
	MqttStandardPort int = 1883
)

// MQTT QoS levels
const (
	QosFireAndForget       = iota // QOS 0
	QosAcknowledgeDelivery        // QOS 1
	QosAssuredDelivery            // QOS 2
)

// TLS versions supported by OpenSSL
const (
	TlsV1  string = "tlsv1"
	TlsV11 string = "tlsv1.1"
	TlsV12 string = "tlsv1.2"
)

// Certificate verification requirments
const (
	SslVerifyNone = 0 // SSL_VERIFY_NONE, no server verification
	SslVerifyPeer = 1 // SSL_VERFY_PEER, check server certificate and abort if the check fails
)

// Mosquitto client instance data
type MosquittoClient struct {
	instance  unsafe.Pointer
	connected bool                 // true if connected
	host      string               // current hostname if connected
	port      int                  // current port on host if connected
	control   chan *ControlMessage // input channel for control messages from callbacks
	will      *willMessage         // a will message for the topic
	user      *C.char              // user name for authentication
	password  *C.char              // password name for authentication
}

// Content for a MQTT will message. These are defined per topic and
// will be sent to the broker during Connect().
type willMessage struct {
	topic      string  // topic for the optional will message
	ctopic     *C.char // converted topic
	payload    []byte  // content for the optional will message sent during Connect()
	qosLevel   int     // QoS level for the message sending
	retainFlag bool    // should this message be retained?
}

// MQTT message content
type MosquittoMessage struct {
	MessageId  uint   // message id
	Topic      string // message topic
	Payload    []byte // message content
	PayloadLen uint   // length of message content
	QoS        int    // QoS level
	Retained   bool   // true if the message was retained by the broker
}

const (
	Timeout = -1
	ConnAck = iota // Connection acknowledged by broker
	ConnNak
	DisconnAck
	DisconnNak
	PubAck
	SubAck // Subscribe acknowledged by broker
	SubNak
	UnsubAck // Unsubscribe acknowledged by broker
	UnsubNak
)

// Internal messages between callbacks and the main functions
type ControlMessage struct {
	kind      int       // see ControlMessageKind
	timestamp time.Time // timestamp of message creation
}

// Data that will be passed to Mosquitto callback functions
type MosquittoCallbackData struct {
	out     chan<- *MosquittoMessage // output channel for messages
	control chan<- *ControlMessage   // output channel for internal state messaging
}

// Set the channel for incoming broker data messages
func (m *MosquittoCallbackData) MessageChannel(messages chan<- *MosquittoMessage) {
	m.out = messages
}

type Errno int

func (e Errno) Error() string {
	s := C.GoString(C.mosquitto_strerror(C.int(e)))
	if s == "" {
		return fmt.Sprintf("errno %d", int(e))
	}
	return s
}

// Mosquitto return values
var (
	ErrConnPending  error = Errno(C.MOSQ_ERR_CONN_PENDING)
	Success         error = Errno(C.MOSQ_ERR_SUCCESS)      // Success value
	ErrNoMem        error = Errno(C.MOSQ_ERR_NOMEM)        // No memory
	ErrProtocol     error = Errno(C.MOSQ_ERR_PROTOCOL)     // Protocol error
	ErrInVal        error = Errno(C.MOSQ_ERR_INVAL)        // Bad input values
	ErrNoCon        error = Errno(C.MOSQ_ERR_NO_CONN)      // Not connected
	ErrConnRefused  error = Errno(C.MOSQ_ERR_CONN_REFUSED) // Connection refused
	ErrNotFound     error = Errno(C.MOSQ_ERR_NOT_FOUND)    // Host/Port not found
	ErrConnLost     error = Errno(C.MOSQ_ERR_CONN_LOST)    // Connection lost
	ErrTls          error = Errno(C.MOSQ_ERR_TLS)          // TLS Error
	ErrPayloadSize  error = Errno(C.MOSQ_ERR_PAYLOAD_SIZE) // Payload is too large
	ErrNotSupported error = Errno(C.MOSQ_ERR_NOT_SUPPORTED)
	ErrAuth         error = Errno(C.MOSQ_ERR_AUTH)
	ErrAclDenied    error = Errno(C.MOSQ_ERR_ACL_DENIED)
	ErrUnknown      error = Errno(C.MOSQ_ERR_UNKNOWN)
	ErrErrNo        error = Errno(C.MOSQ_ERR_ERRNO) // OS error
	ErrErrEai       error = Errno(C.MOSQ_ERR_EAI)
)

// Initialize the library
func init() {
	C.mosquitto_lib_init()
}

// Cleanup the library. Must be called at the end to free resources.
func Cleanup() {
	C.mosquitto_lib_cleanup()
}

// Return the client library version numbers: <major>,<minor>,<revision>
func Version() (major uint, minor uint, revision uint) {
	var (
		vmajor    C.int = 0
		vminor    C.int = 0
		vrevision C.int = 0
	)
	C.mosquitto_lib_version(&vmajor, &vminor, &vrevision)
	major = (uint)(vmajor)
	minor = (uint)(vminor)
	revision = (uint)(vrevision)
	return major, minor, revision
}

// Create the internal control channel for communication between
// callbacks and the main program.
func createControlChannel() chan *ControlMessage {
	cc := make(chan *ControlMessage, 10)
	return cc
}

// Called by functions in the main program to receive answers
// from the server via asynchronous callback routines. Looks only for
// certain incoming control messages (kinds), all others are discarded.
// Waits for a "duration" number of seconds and returns a timeout if
// no suitable message was received.
func waitForControlChannel(control chan *ControlMessage, duration int, kinds ...int) int {
	for {
		select {
		case cmsg := <-control:
			if containsInt(cmsg.kind, kinds) {
				return cmsg.kind
			}
		case <-time.After(time.Second * time.Duration(duration)):
			return Timeout
		}
	}
}

/*
Create a new Mosquitto client instance with a random ID.
The data structure will be passed to callback functions and may contain
data useful for them.

Returns nil if the instance couldn't be created, otherwise the
client instance.

Note: The client instance must be destroyed explicitely! See DestroyInstance()
*/
func NewInstance(data *MosquittoCallbackData) *MosquittoClient {
	var m = (*C.struct_mosquitto)(nil)
	if data == nil {
		data = new(MosquittoCallbackData)
	}
	control := createControlChannel()
	data.control = control
	m = C.mosquitto_new_anonymous(unsafe.Pointer(data))

	if m == nil {
		close(control)
		return nil
	} else {
		client := new(MosquittoClient)
		client.instance = unsafe.Pointer(m)
		client.control = control
		return client
	}
}

/*
Create a new, named Mosquitto client instance. The id string identifies
the client in message exchanges, the data structure will be passed
to callback functions and may contain data useful for them. If the parameter
cleanSession is true, the broker will clean messages ond subscription on
disconnect.

Returns nil if the instance couldn't be created, otherwise the
client instance.

Note: The client instance must be destroyed explicitely! See DestroyInstance()
*/
func NewNamedInstance(id string, cleanSession bool, data *MosquittoCallbackData) *MosquittoClient {
	// ID max length = 23
	cs := C.CString(id)
	defer C.free(unsafe.Pointer(cs))

	if data == nil {
		data = new(MosquittoCallbackData)
	}
	control := createControlChannel()
	data.control = control
	m := C.mosquitto_new(cs, C.bool(cleanSession), unsafe.Pointer(data))

	if m == nil {
		close(control)
		return nil
	} else {
		client := new(MosquittoClient)
		client.instance = unsafe.Pointer(m)
		client.control = control
		return client
	}
}

// Destroy a client instance. Clears instance memory used in the C library.
func (client *MosquittoClient) DestroyInstance() {
	if client.instance != nil {
		m := (*C.struct_mosquitto)(client.instance)
		if client.control != nil {
			close(client.control)
		}
		C.mosquitto_destroy(m)
	}
}

// Configure the will message. Must be called before Connect().
// This function only fills the data structure.
func (client *MosquittoClient) SetWillMessage(topic string, payload []byte, qos int, retain bool) {
	if client.will != nil {
		C.free(unsafe.Pointer(client.will.ctopic))
		client.will = nil
	}
	client.will = new(willMessage)
	client.will.topic = topic
	client.will.ctopic = C.CString(topic)
	client.will.payload = payload
	client.will.qosLevel = qos
	client.will.retainFlag = retain
}

// Configure the will message. Must be called before Connect().
// This function sets the configured message.
func (client *MosquittoClient) SetWill() error {
	if client.will != nil {
		m := (*C.struct_mosquitto)(client.instance)
		status := C.mosquitto_will_set(m, client.will.ctopic, C.int(len(client.will.payload)),
			unsafe.Pointer(&client.will.payload[0]), C.int(client.will.qosLevel),
			C.bool(client.will.retainFlag))
		if Errno(status) != Success {
			C.free(unsafe.Pointer(client.will.ctopic))
			client.will = nil
		}
		return Errno(status)
	} else {
		return Success
	}
}

// Delete the configured will message for a topic. Must be called to free the memory.
func (client *MosquittoClient) ClearWill() error {
	m := (*C.struct_mosquitto)(client.instance)
	if client.will != nil {
		status := C.mosquitto_will_clear(m)
		C.free(unsafe.Pointer(client.will.ctopic))
		client.will = nil
		return Errno(status)
	} else {
		return Success
	}
}

// Set the authentication data for the broker. Must be called before Connect().
// Use empty strings to delete the current data. An empty user name disables authentication.
func (client *MosquittoClient) SetLoginData(user string, password string) error {
	m := (*C.struct_mosquitto)(client.instance)
	C.free(unsafe.Pointer(client.password))
	C.free(unsafe.Pointer(client.user))
	if user != "" {
		client.user = C.CString(user)
	}
	if password != "" {
		client.password = C.CString(password)
	}
	if user != "" {
		status := C.mosquitto_username_pw_set(m, client.user, client.password)
		return Errno(status)
	} else {
		return Success
	}
}

/*
Use SSL/TLS with a pre-shared key when connecting to the broker. Encryption is optional,
but must be configured before CONNECT when used. Make sure to use the right TLS version,
otherwise you will get a MQTT 'protocol error' or an SSL error like 'no shared cipher'.
	id - a PSK ID configured in the broker's configuration
	psk - the actual pre-shared key consisting of hexadecimal values only
	tlsVersion - the TLS version used, must be identical to the one configured in the broker
	ciphers - list of OpenSSL ciphers to use, if empty the defaults will be used
*/
func (client *MosquittoClient) UseSslPsk(id string, psk string, tlsVersion string, ciphers string) error {
	cid := C.CString(id)
	defer C.free(unsafe.Pointer(cid))
	cpsk := C.CString(psk)
	defer C.free(unsafe.Pointer(cpsk))
	ctlsVersion := C.CString(tlsVersion)
	defer C.free(unsafe.Pointer(ctlsVersion))
	var cciphers = unsafe.Pointer(nil)
	if ciphers != "" {
		cciphers = unsafe.Pointer(C.CString(ciphers))
		defer C.free(cciphers)
	}
	m := (*C.struct_mosquitto)(client.instance)
	rc := C.mosquitto_tls_opts_set(m, SslVerifyNone, ctlsVersion, (*_Ctype_char)(cciphers))
	if Errno(rc) != Success {
		log.Println("Unable to set SSL/TLS options ", rc)
		return Errno(rc)
	}
	rc = C.mosquitto_tls_psk_set(m, cpsk, cid, (*_Ctype_char)(cciphers))
	if Errno(rc) != Success {
		log.Println("Unable to set PSK data for SSL/TLS ", rc)
		return Errno(rc)
	} else {
		return Success
	}

}

/*
Connect to broker.
Connection parameters are:
	host - hostname to connect to
	port - port to connect to, if 0 the MQTT standrad 1883 will be used
	keepAlive - time interval (seconds) for ping messages to keep the connection up
Returns status Success or an error code
*/
func (client *MosquittoClient) Connect(host string, port int, keepAlive int) error {
	if port == 0 { // use the standard port if the port number is 0
		port = MqttStandardPort
	}
	chost := C.CString(host)
	defer C.free(unsafe.Pointer(chost))
	m := (*C.struct_mosquitto)(client.instance)
	status := C.mosquitto_connect(m, chost, C.int(port),
		C.int(keepAlive))
	if status == 0 {
		answer := waitForControlChannel(client.control, 3, ConnAck, ConnNak)
		if answer == ConnAck {
			client.host = host
			client.port = port
			client.connected = true
			return Success
		} else {
			log.Println("Bad connect answer: ", answer)
			client.host = ""
			client.port = 0
			client.connected = false
			return ErrNoCon
		}
	} else {
		client.host = ""
		client.port = 0
		client.connected = false
		return Errno(status)
	}
}

/*
Disconnect from a broker.
Returns status Success or an error code
*/
func (client *MosquittoClient) Disconnect() error {
	if client.instance != nil {
		m := (*C.struct_mosquitto)(client.instance)
		status := C.mosquitto_disconnect(m)
		if Errno(status) == Success {
			answer := waitForControlChannel(client.control, 3, DisconnAck, DisconnNak)
			if answer == DisconnAck {
				return Success
			} else {
				return ErrNoCon
			}
		} else {
			return Errno(status)
		}
	} else {
		return Success
	}
}

// Publish a string message
func (client *MosquittoClient) PublishString(messageId uint, topic string, payload string, qos int, retain bool) error {
	m := (*C.struct_mosquitto)(client.instance)
	cmsgId := C.int(messageId)
	ctopic := C.CString(topic)
	cpayload := C.CString(payload)
	status := C.mosquitto_publish(m, &cmsgId, ctopic, C.int(len(payload)), unsafe.Pointer(cpayload),
		C.int(qos), C.bool(retain))
	return Errno(status)
}

// Publish a byte array message
func (client *MosquittoClient) Publish(messageId uint, topic string, payload []byte, qos int, retain bool) error {
	m := (*C.struct_mosquitto)(client.instance)
	cmsgId := C.int(messageId)
	ctopic := C.CString(topic)
	status := C.mosquitto_publish(m, &cmsgId, ctopic,
		C.int(len(payload)), unsafe.Pointer(&payload[0]), C.int(qos), C.bool(retain))
	return Errno(status)
}

/*
	Subscribe to a broker topic
	Parameters:
		subscription: name or pattern for subscription topic(s)
		qos: Quality of Service setting for this subscription
	Returns:
		status Success or an error code
*/
func (client *MosquittoClient) Subscribe(subscription string, qos uint) error {
	m := (*C.struct_mosquitto)(client.instance)
	status := C.mosquitto_subscribe(m, nil, C.CString(subscription), C.int(qos))
	return Errno(status)
}

/*
	Unsubscribe from a broker topic.
	Parameters:
		subscription: name or pattern of subscription topic(s) to be unsubscribed
	Returns:
		status Success or an error code
*/
func (client *MosquittoClient) Unsubscribe(subscription string) error {
	m := (*C.struct_mosquitto)(client.instance)
	status := C.mosquitto_unsubscribe(m, nil, C.CString(subscription))
	return Errno(status)
}

// Start the asynchronous processing loop for message data.
// The underlying C code starts an external thread that must be stopped
// later, see StopLoop
func (client *MosquittoClient) StartLoop() error {
	m := (*C.struct_mosquitto)(client.instance)
	status := C.mosquitto_loop_start(m)
	return Errno(status)
}

// Stops the asynchronous processing loop for message data.
// This function terminates an external thread that was started
// with StartLoop.
// Note that this function should normally called with "force = false",
// after the client has been disconnected.
// "force = true" will generate a runtime panic, due to a signal used
// in the C code.
func (client *MosquittoClient) StopLoop(force bool) error {
	m := (*C.struct_mosquitto)(client.instance)
	status := C.mosquitto_loop_stop(m, C.bool(force))
	return Errno(status)
}

// Synchronous processing loop for message data that must be called repeatedly.
// Causes sometimes problems (DestroyInstance hangs) if used in goroutines.
func (client *MosquittoClient) Loop(timeout int) error {
	m := (*C.struct_mosquitto)(client.instance)
	status := C.mosquitto_loop(m, C.int(timeout), C.int(1))
	return Errno(status)
}

// Synchronous processing loop for message data that never stops. Causes
// sometimes problems (DestroyInstance hangs) if used in goroutines.
func (client *MosquittoClient) LoopForever(timeout int) error {
	m := (*C.struct_mosquitto)(client.instance)
	status := C.mosquitto_loop_forever(m, C.int(timeout), C.int(1))
	return Errno(status)
}

/*
// Synchronous processing loop for message data that is handled in a goroutine.
func (client *MosquittoClient) LoopMySelf(timeout int) error {
	m := (*C.struct_mosquitto)(client.instance)
	socket := C.mosquitto_socket(m)
	mosq_conn, err := net.FileConn(os.NewFile(uintptr(socket), "mosquitto"))
	return Success
}
*/

// Go part of the message callback. Creates a Go data structure from the
// C data and sends that to the output channel. If no channel is defined
// nothing is done. While the output channel is full new messages will be discarded.
//
//export onMessage
func onMessage(mosq unsafe.Pointer, data unsafe.Pointer, message unsafe.Pointer) {
	callbackData := (*MosquittoCallbackData)(data)
	if callbackData.out != nil {
		// Check if the channel can hold the new message
		if len(callbackData.out) < cap(callbackData.out) {
			// Copy and send the message
			goMessage := new(MosquittoMessage)
			msg := (*C.struct_mosquitto_message)(message)
			goMessage.MessageId = (uint)(msg.mid)
			goMessage.Topic = C.GoString(msg.topic)
			goMessage.Payload = C.GoBytes(msg.payload, msg.payloadlen)
			goMessage.PayloadLen = (uint)(msg.payloadlen)
			goMessage.QoS = (int)(msg.qos)
			goMessage.Retained = (bool)(msg.retain)
			callbackData.out <- goMessage
		}
	}
}

// Start receiving incoming messages
func (client *MosquittoClient) StartMessageCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setMessageCallback(m, true)
}

// Stop receiving incoming messages
func (client *MosquittoClient) StopMessageCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setMessageCallback(m, false)
}

// Go part of the log callback. Will just log the client library message
//
//export onLog
func onLog(mosq unsafe.Pointer, data unsafe.Pointer, level int, message *C.char) {
	logMessage := C.GoString(message)
	log.Printf("mosquitto log: %d, %s", level, logMessage)
}

// Start the client library logging
func (client *MosquittoClient) StartLogCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setLogCallback(m, true)
}

// Stop the client library logging
func (client *MosquittoClient) StopLogCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setLogCallback(m, false)
}

// Go part of the connect callback. Will just log the code for the connect:
//     	0 - success
// 		1 - connection refused (unacceptable protocol version)
//		2 - connection refused (identifier rejected)
//		3 - connection refused (broker unavailable)
//		... reserved
//
//export onConnect
func onConnect(mosq unsafe.Pointer, data unsafe.Pointer, rc int) {
	log.Printf("mosquitto connected: %d", rc)
	callbackData := (*MosquittoCallbackData)(data)
	m := new(ControlMessage)
	m.timestamp = time.Now()
	if rc == 0 {
		m.kind = ConnAck
	} else {
		m.kind = ConnNak
	}
	callbackData.control <- m
}

// Start the connect callback
func (client *MosquittoClient) StartConnectCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setConnectCallback(m, true)
}

// Stop the connect callback
func (client *MosquittoClient) StopConnectCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setConnectCallback(m, false)
}

// Go part of the disconnect callback. Will just log the code for the connect:
//     	0 - client called disconnect
//		... unexpected disconnect
//
//export onDisconnect
func onDisconnect(mosq unsafe.Pointer, data unsafe.Pointer, rc int) {
	log.Printf("mosquitto disconnected: %d", rc)

	callbackData := (*MosquittoCallbackData)(data)
	m := new(ControlMessage)
	m.timestamp = time.Now()
	if rc == 0 {
		m.kind = DisconnAck
	} else {
		m.kind = DisconnNak
	}
	callbackData.control <- m

}

// Start the disconnect callback
func (client *MosquittoClient) StartDisconnectCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setDisconnectCallback(m, true)
}

// Stop the disconnect callback
func (client *MosquittoClient) StopDisconnectCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setDisconnectCallback(m, false)
}

// Go part of the publish callback. Will just log the code for the connect:
// 	mid - message id of the successfully sent message
//
//export onPublish
func onPublish(mosq unsafe.Pointer, data unsafe.Pointer, mid int) {
	log.Printf("mosquitto published: %d", mid)
	/*
		callbackData := (*MosquittoCallbackData)(data)
		m := new(ControlMessage)
		m.timestamp = time.Now()
		m.kind = PubAck
		callbackData.control <- m
	*/

}

// Start the publish callback
func (client *MosquittoClient) StartPublishCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setPublishCallback(m, true)
}

// Stop the publish callback
func (client *MosquittoClient) StopPublishCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setPublishCallback(m, false)
}

// Go part of the subscribe callback. Will just log the code for the connect:
// 	mid - message id of the subscribe message
// 	qosCount - the number of granted subscriptions
//	grantedQos - the list of QoS levels granted for each subscription (array of ints)
//
//export onSubscribe
func onSubscribe(mosq unsafe.Pointer, data unsafe.Pointer, mid int, qosCount int, grantedQos unsafe.Pointer) {
	grantedLevels := cIntArray2Go(grantedQos, qosCount)
	log.Printf("mosquitto subscribed: %d subscriptions with QoS levels: %v", qosCount, grantedLevels)
	/*
		callbackData := (*MosquittoCallbackData)(data)
		m := new(ControlMessage)
		m.timestamp = time.Now()
		m.kind = SubAck
		// TODO: check if the subscribe worked, relate mid to the subscription request
		callbackData.control <- m
	*/
}

// Start the subscribe callback
func (client *MosquittoClient) StartSubscribeCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setSubscribeCallback(m, true)
}

// Stop the subscribe callback
func (client *MosquittoClient) StopSubscribeCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setSubscribeCallback(m, false)
}

// Go part of the unsubscribe callback. Will just log the code for the connect:
// 	mid - message id of the unsubscribe message
//
//export onUnsubscribe
func onUnsubscribe(mosq unsafe.Pointer, data unsafe.Pointer, mid int) {
	log.Printf("mosquitto unsubscribed: %d", mid)
	/*
		callbackData := (*MosquittoCallbackData)(data)
		m := new(ControlMessage)
		m.timestamp = time.Now()
		m.kind = UnsubAck
		// TODO: check if the unsubscribe worked, relate mid to the subscription request
		callbackData.control <- m
	*/
}

// Start the unsubscribe callback
func (client *MosquittoClient) StartUnsubscribeCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setUnsubscribeCallback(m, true)
}

// Stop the unsubscribe callback
func (client *MosquittoClient) StopUnsubscribeCallback() {
	m := (*C.struct_mosquitto)(client.instance)
	C.setUnsubscribeCallback(m, false)
}
