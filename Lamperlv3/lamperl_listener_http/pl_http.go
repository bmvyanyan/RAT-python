package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// HTTPConfig holds configuration for the HTTP listener.
// These values come from the UI form defined in ax_config.axs.
type HTTPConfig struct {
	HostBind        string `json:"host_bind"`
	PortBind        int    `json:"port_bind"`
	CallbackAddress string `json:"callback_address"`
	ApiPath         string `json:"api_path"`
}

// HTTP represents an HTTP listener instance.
// Manages the Gin web server and handles agent communication.
type HTTP struct {
	GinEngine *gin.Engine
	Server    *http.Server
	Config    HTTPConfig
	Name      string
	Active    bool
}

// AgentRequest represents the JSON structure sent by agents.
// Beat: 18-char string (8-char watermark + 10-char agent ID)
// Init: System information sent only on first check-in
// Results: Array of task execution results from previous beacon
type AgentRequest struct {
	Beat    string                   `json:"beat"`
	Init    map[string]interface{}   `json:"init,omitempty"`
	Results []map[string]interface{} `json:"results,omitempty"`
}

// Start initializes and launches the HTTP server.
// Creates a Gin router, registers the API endpoint, and starts listening.
// The server runs in a goroutine to avoid blocking.
func (handler *HTTP) Start(ts Teamserver) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Register the API endpoint
	router.POST(handler.Config.ApiPath, func(c *gin.Context) {
		handler.processRequest(c, ts)
	})

	handler.Active = true
	handler.Server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", handler.Config.HostBind, handler.Config.PortBind),
		Handler: router,
	}

	fmt.Printf("   Started listener: http://%s:%d%s\n",
		handler.Config.HostBind, handler.Config.PortBind, handler.Config.ApiPath)

	go func() {
		err := handler.Server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Error starting HTTP server: %v\n", err)
			return
		}
	}()

	time.Sleep(500 * time.Millisecond)
	return nil
}

// Stop gracefully shuts down the HTTP server.
// Waits up to 3 seconds for existing connections to complete.
func (handler *HTTP) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return handler.Server.Shutdown(ctx)
}

// processRequest handles incoming agent beacons.
// This is the core request handler that:
//  1. Parses the JSON request to extract beat, init data, and results
//  2. Creates new agents on first check-in
//  3. Processes task results from the agent
//  4. Returns pending tasks for the agent to execute
func (handler *HTTP) processRequest(ctx *gin.Context, ts Teamserver) {
	var (
		externalIP   string
		agentType    string
		agentId      string
		beat         []byte
		bodyData     []byte
		responseData []byte
		err          error
	)

	fmt.Printf("[LISTENER] Received request from %s to %s\n", ctx.Request.RemoteAddr, ctx.Request.URL.Path)
	fmt.Printf("[LISTENER] Method: %s, Content-Type: %s\n", ctx.Request.Method, ctx.Request.Header.Get("Content-Type"))

	// Get agent's IP
	externalIP = strings.Split(ctx.Request.RemoteAddr, ":")[0]

	// Parse the request
	agentType, agentId, beat, bodyData, err = handler.parseRequest(ctx)
	if err != nil {
		fmt.Printf("[LISTENER ERROR] Failed to parse request: %v\n", err)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		return
	}

	fmt.Printf("[LISTENER] Parsed - AgentType: %s, AgentID: %s, Beat len: %d, Body len: %d\n",
		agentType, agentId, len(beat), len(bodyData))

	// Create agent if doesn't exist
	if !ModuleObject.ts.TsAgentIsExists(agentId) {
		fmt.Printf("[LISTENER] Creating new agent: %s\n", agentId)
		_, err = ModuleObject.ts.TsAgentCreate(agentType, agentId, beat, handler.Name, externalIP, true)
		if err != nil {
			fmt.Printf("[LISTENER ERROR] Failed to create agent: %v\n", err)
			ctx.Writer.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Printf("[LISTENER] Agent created successfully\n")
	} else {
		fmt.Printf("[LISTENER] Agent %s already exists\n", agentId)
	}

	// Update agent's last check-in time
	_ = ModuleObject.ts.TsAgentSetTick(agentId)

	// Process agent data (task results)
	fmt.Printf("[LISTENER] Processing agent data...\n")
	_ = ModuleObject.ts.TsAgentProcessData(agentId, bodyData)

	// Get tasks for agent
	fmt.Printf("[LISTENER] Getting tasks for agent...\n")
	responseData, err = ModuleObject.ts.TsAgentGetHostedAll(agentId, 0x1900000) // 25 MB
	if err != nil {
		fmt.Printf("[LISTENER ERROR] Failed to get tasks: %v\n", err)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		return
	}

	fmt.Printf("[LISTENER] Sending response: %d bytes\n", len(responseData))

	// Send response
	ctx.Writer.Header().Set("Content-Type", "application/json")
	_, err = ctx.Writer.Write(responseData)
	if err != nil {
		fmt.Printf("[LISTENER ERROR] Failed to write response: %v\n", err)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		return
	}

	ctx.AbortWithStatus(http.StatusOK)
	fmt.Printf("[LISTENER] Request completed successfully\n")
}

// parseRequest extracts and validates data from an agent's HTTP request.
// Parses the JSON body and separates the beat into watermark and agent ID.
// Returns:
//   - watermark: 8-character hex identifier for agent type
//   - agentId: 10-character hex unique agent instance ID
//   - beat: Initial check-in data (JSON) or empty for regular beacons
//   - bodyData: Task results (JSON) or empty array
//   - error: If parsing fails or format is invalid
func (handler *HTTP) parseRequest(ctx *gin.Context) (string, string, []byte, []byte, error) {
	// Read POST body
	bodyData, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to read request body: %v", err)
	}

	fmt.Printf("[PARSE] Raw body (%d bytes): %s\n", len(bodyData), string(bodyData))

	// Parse JSON
	var req AgentRequest
	err = json.Unmarshal(bodyData, &req)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	fmt.Printf("[PARSE] Beat from JSON: %s (len=%d)\n", req.Beat, len(req.Beat))

	// Parse beat: watermark (8 hex chars) + agent_id (10 hex chars) = 18 chars total
	if len(req.Beat) != 18 {
		return "", "", nil, nil, fmt.Errorf("invalid beat format: expected 18 chars, got %d", len(req.Beat))
	}

	watermark := req.Beat[:8]
	agentIdHex := req.Beat[8:]

	// The "beat" parameter is what gets passed to CreateAgent - it should be the init data for first checkin
	var beat []byte
	var agentData []byte

	if req.Init != nil {
		// Initial check-in - encode init data as JSON for both beat and agentData
		beat, err = json.Marshal(req.Init)
		if err != nil {
			return "", "", nil, nil, errors.New("failed to encode init data")
		}
		agentData = beat // Same data for initial checkin
	} else if req.Results != nil {
		// Regular beacon - encode results as JSON
		beat = []byte{} // Empty beat for regular checkins
		agentData, err = json.Marshal(req.Results)
		if err != nil {
			return "", "", nil, nil, errors.New("failed to encode results")
		}
	} else {
		// Empty beacon
		beat = []byte{}
		agentData = []byte("[]")
	}

	return watermark, agentIdHex, beat, agentData, nil
}
