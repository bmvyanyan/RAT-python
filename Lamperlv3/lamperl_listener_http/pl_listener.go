package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// HandlerListenerValid validates listener configuration before creation.
// Called by Adaptix when user submits the listener creation form.
// Checks that all required fields are present and valid.
// Returns error if validation fails, nil if configuration is valid.
func (m *ModuleExtender) HandlerListenerValid(data string) error {

	/// START CODE HERE

	var conf HTTPConfig
	err := json.Unmarshal([]byte(data), &conf)
	if err != nil {
		return err
	}

	if conf.HostBind == "" {
		return errors.New("host_bind is required")
	}

	if conf.PortBind < 1 || conf.PortBind > 65535 {
		return errors.New("port_bind must be in range 1-65535")
	}

	if conf.CallbackAddress == "" {
		return errors.New("callback_address is required")
	}

	if conf.ApiPath == "" {
		return errors.New("api_path is required")
	}

	/// END CODE

	return nil
}

// HandlerCreateListenerDataAndStart creates and starts a new listener instance.
// This is the main initialization function called when a listener is created.
// Parameters:
//   - name: Unique identifier for this listener instance
//   - configData: JSON-encoded configuration from the UI
//   - listenerCustomData: Optional custom data from previous session (unused)
//
// Returns:
//   - ListenerData: Metadata for Adaptix UI (bind address, port, status)
//   - customData: Serialized config to persist across restarts
//   - listenerObject: The actual HTTP server instance
//   - error: If initialization or startup fails
func (m *ModuleExtender) HandlerCreateListenerDataAndStart(name string, configData string, listenerCustomData []byte) (adaptix.ListenerData, []byte, any, error) {
	var (
		listenerData adaptix.ListenerData
		customdData  []byte
	)

	/// START CODE HERE

	var (
		listener *HTTP
		conf     HTTPConfig
		err      error
	)

	err = json.Unmarshal([]byte(configData), &conf)
	if err != nil {
		return listenerData, customdData, nil, err
	}

	listener = &HTTP{
		Config: conf,
		Name:   name,
		Active: false,
	}

	err = listener.Start(ModuleObject.ts)
	if err != nil {
		return listenerData, customdData, nil, err
	}

	listenerData = adaptix.ListenerData{
		BindHost:  conf.HostBind,
		BindPort:  fmt.Sprintf("%d", conf.PortBind),
		AgentAddr: conf.CallbackAddress,
		Status:    "Listen",
	}

	// Save config to customData
	var buffer bytes.Buffer
	err = json.NewEncoder(&buffer).Encode(conf)
	if err != nil {
		return listenerData, customdData, nil, err
	}
	customdData = buffer.Bytes()

	/// END CODE

	return listenerData, customdData, listener, nil
}

// HandlerEditListenerData updates an existing listener's configuration.
// Currently unimplemented - listener must be stopped and recreated to change config.
func (m *ModuleExtender) HandlerEditListenerData(name string, listenerObject any, configData string) (adaptix.ListenerData, []byte, bool) {
	var (
		listenerData adaptix.ListenerData
		customdData  []byte
		ok           bool = false
	)

	/// START CODE HERE

	/// END CODE

	return listenerData, customdData, ok
}

// HandlerListenerStop gracefully shuts down a running listener.
// Called when user stops a listener from the Adaptix UI.
// Parameters:
//   - name: Listener identifier (unused in this implementation)
//   - listenerObject: The HTTP server instance to stop
//
// Returns: true if stopped successfully, false and error otherwise
func (m *ModuleExtender) HandlerListenerStop(name string, listenerObject any) (bool, error) {
	var (
		err error = nil
		ok  bool  = false
	)

	/// START CODE HERE

	listener, valid := listenerObject.(*HTTP)
	if !valid {
		return false, errors.New("invalid listener object")
	}

	err = listener.Stop()
	if err != nil {
		return false, err
	}

	ok = true

	/// END CODE

	return ok, err
}

// HandlerListenerGetProfile returns the listener's current configuration.
// Called when displaying listener details or populating agent generation dropdowns.
// Parameters:
//   - name: Listener identifier to retrieve config for
//   - listenerObject: The HTTP server instance
//
// Returns: JSON-encoded configuration and true if successful
func (m *ModuleExtender) HandlerListenerGetProfile(name string, listenerObject any) ([]byte, bool) {
	var (
		object bytes.Buffer
		ok     bool = false
	)

	/// START CODE HERE

	listener, valid := listenerObject.(*HTTP)
	if !valid || listener.Name != name {
		return object.Bytes(), false
	}

	_ = json.NewEncoder(&object).Encode(listener.Config)
	ok = true

	/// END CODE

	return object.Bytes(), ok
}
