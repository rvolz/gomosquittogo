/*
	C utility functions for gomosquittogo
*/

#include <sys/types.h>
#include <stdlib.h>
#include <mosquitto.h>	
#include "_cgo_export.h"

void connectCallback(struct mosquitto *mosq, void *data, int);
void disconnectCallback(struct mosquitto *mosq, void *data, int);
void logCallback(struct mosquitto *mosq, void *data, int level, const char *message);
void messageCallback(struct mosquitto *mosq, void *data, const struct mosquitto_message *message);
void publishCallback(struct mosquitto *mosq, void *data, int);
void subscribeCallback(struct mosquitto *mosq, void *data, int, int, const int*);
void unsubscribeCallback(struct mosquitto *mosq, void *data, int);


// Start/stop the handler for log callbacks, called from Go
void setConnectCallback(struct mosquitto *mosq, bool toggle) {
	if (toggle)
		mosquitto_connect_callback_set(mosq, &connectCallback);
	else
		mosquitto_connect_callback_set(mosq, NULL);
}

/* 
Callback handler for Mosquitto connect events. Passes all data on
to the exported Go message handler callback: onConnect

Parameters:
	mosq 	- Mosquitto client
	data 	- user defined data
	rc		- broker response code
*/
void connectCallback(struct mosquitto *mosq, void *data, int rc) {
	onConnect(mosq, data, rc);
}

// Start/stop the handler for disconnect callbacks, called from Go
void setDisconnectCallback(struct mosquitto *mosq, bool toggle) {
	if (toggle)
		mosquitto_disconnect_callback_set(mosq, &disconnectCallback);
	else
		mosquitto_disconnect_callback_set(mosq, NULL);
}

/* 
Callback handler for Mosquitto connect events. Passes all data on
to the exported Go message handler callback: onConnect

Parameters:
	mosq 	- Mosquitto client
	data 	- user defined data
	rc		- broker response code
*/
void disconnectCallback(struct mosquitto *mosq, void *data, int rc) {
	onDisconnect(mosq, data, rc);
}



// Start/stop the handler for log callbacks, called from Go
void setLogCallback(struct mosquitto *mosq, bool toggle) {
	if (toggle)
		mosquitto_log_callback_set(mosq, &logCallback);
	else
		mosquitto_log_callback_set(mosq, NULL);
}

/* 
Callback handler for Mosquitto log events. Passes all data on
to the exported Go message handler callback: onLog

Parameters:
	mosq 	- Mosquitto client
	data 	- user defined data
	level	- log level
	message - log message
*/
void logCallback(struct mosquitto *mosq, void *data, int level, const char *message) {
	onLog(mosq, data, level, (char*)message);
}

// Start/stop the handler for log callbacks, called from Go
void setMessageCallback(struct mosquitto *mosq, bool toggle) {
	if (toggle)
		mosquitto_message_callback_set(mosq, &messageCallback);
	else
		mosquitto_message_callback_set(mosq, NULL);	
}

/* 
Callback handler for Mosquitto message events. Passes all data on
to the exported Go message handler callback: onMessage

Parameters:
	mosq 	- Mosquitto client
	data 	- user defined data
	message - incoming message data
*/
void messageCallback(struct mosquitto *mosq, void *data, const struct mosquitto_message *message) {
	onMessage(mosq, data, (void*)message);
}

// Start/stop the handler for publish callbacks, called from Go
void setPublishCallback(struct mosquitto *mosq, bool toggle) {
	if (toggle)
		mosquitto_publish_callback_set(mosq, &publishCallback);
	else
		mosquitto_publish_callback_set(mosq, NULL);
}

/* 
Callback handler for Mosquitto publish events. Passes all data on
to the exported Go message handler callback: onPublish

Parameters:
	mosq 	- Mosquitto client
	data 	- user defined data
	mid		- id of the sent message
*/
void publishCallback(struct mosquitto *mosq, void *data, int mid) {
	onPublish(mosq, data, mid);
}

// Start/stop the handler for subscribe callbacks, called from Go
void setSubscribeCallback(struct mosquitto *mosq, bool toggle) {
	if (toggle)
		mosquitto_subscribe_callback_set(mosq, &subscribeCallback);
	else
		mosquitto_subscribe_callback_set(mosq, NULL);
}

/* 
Callback handler for Mosquitto susbscribe events. Passes all data on
to the exported Go message handler callback: onSubscribe

Parameters:
	mosq 		- Mosquitto client
	data 		- user defined data
	mid			- id of the subscribe message
	qos_count	- number of granted subscriptions
	granted_qos - the list of QoS levels granted for each subscription (array of ints)
*/
void subscribeCallback(struct mosquitto *mosq, void *data, int mid, int qos_count, const int *granted_qos) {
	onSubscribe(mosq, data, mid, qos_count, (void*)granted_qos);
}

// Start/stop the handler for unsubscribe callbacks, called from Go
void setUnsubscribeCallback(struct mosquitto *mosq, bool toggle) {
	if (toggle)
		mosquitto_unsubscribe_callback_set(mosq, &unsubscribeCallback);
	else
		mosquitto_unsubscribe_callback_set(mosq, NULL);
}

/* 
Callback handler for Mosquitto unsubscribe events. Passes all data on
to the exported Go message handler callback: onUnsubscribe

Parameters:
	mosq 		- Mosquitto client
	data 		- user defined data
	mid			- id of the unsubscribe message
*/
void unsubscribeCallback(struct mosquitto *mosq, void *data, int mid) {
	onUnsubscribe(mosq, data, mid);
}


// Helper function for creating an anonymous Mosquitto client with user data
struct mosquitto *mosquitto_new_anonymous(void* data) {
	return mosquitto_new(NULL, true, data);
}
// Helper function for creating an anonymous Mosquitto client without user data
struct mosquitto *mosquitto_new_anonymous_no_data() {
	return mosquitto_new(NULL, true, NULL);
}

