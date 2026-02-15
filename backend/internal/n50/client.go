package n50

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"mpd-client-modern/internal/config"
)

// Command constants for N50 component
const (
	CmdGetId             = "?NGD"
	CmdGetPowerStatus    = "?P"
	CmdPowerUp           = "PO"
	CmdStandBy           = "PF"
	CmdGetCurrentFunction = "?F"
	CmdSetIPod           = "10FN"
	CmdSetInternetRadio  = "13FN"
	CmdSetMusicServer    = "14FN"
	CmdSetUSB            = "12FN"
	CmdSetBTAudio        = "11FN"
	CmdSetAirJam         = "20FN"
	CmdSetDigitalInUSB   = "17FN"
	CmdSetDigitalIn1     = "15FN"
	CmdSetDigitalIn2     = "16FN"
)

// Response mappings
var (
	// Power status responses
	PowerStatusUp      = "PWR0"
	PowerStatusStandBy = "PWR2"

	// Function/Input responses
	FunctionIPod          = "FN10"
	FunctionBTAudio       = "FN11"
	FunctionUSB           = "FN12"
	FunctionInternetRadio = "FN13"
	FunctionMusicServer   = "FN14"
	FunctionDigitalIn1    = "FN15"
	FunctionDigitalIn2    = "FN16"
	FunctionDigitalInUSB  = "FN17"
	FunctionAirJam        = "FN20"

	// Input name to command mapping
	InputCommands = map[string]string{
		"iPod":           CmdSetIPod,
		"InternetRadio":  CmdSetInternetRadio,
		"MusicServer":    CmdSetMusicServer,
		"USB":            CmdSetUSB,
		"BTAudio":        CmdSetBTAudio,
		"AirJam":         CmdSetAirJam,
		"DigitalInUSB":   CmdSetDigitalInUSB,
		"DigitalIn1":     CmdSetDigitalIn1,
		"DigitalIn2":     CmdSetDigitalIn2,
	}

	// Input name to response mapping
	InputResponses = map[string]string{
		"iPod":           FunctionIPod,
		"InternetRadio":  FunctionInternetRadio,
		"MusicServer":    FunctionMusicServer,
		"USB":            FunctionUSB,
		"BTAudio":        FunctionBTAudio,
		"AirJam":         FunctionAirJam,
		"DigitalInUSB":   FunctionDigitalInUSB,
		"DigitalIn1":     FunctionDigitalIn1,
		"DigitalIn2":     FunctionDigitalIn2,
	}

	// Response to human-readable mapping
	ResponseDescriptions = map[string]string{
		"R":        "Error",
		PowerStatusUp:      "Powered up",
		PowerStatusStandBy: "Stand-by",
		FunctionIPod:          "iPod",
		FunctionBTAudio:       "BT Audio",
		FunctionUSB:           "USB",
		FunctionInternetRadio: "Internet Radio",
		FunctionMusicServer:   "Music Server",
		FunctionDigitalIn1:    "Digital in 1",
		FunctionDigitalIn2:    "Digital in 2",
		FunctionDigitalInUSB:  "Digital in USB",
		FunctionAirJam:        "Air Jam",
	}
)

// Client represents an N50 HIFI component client
type Client struct {
	mu          sync.Mutex
	conn        net.Conn
	reader      *bufio.Reader
	writer      *bufio.Writer
	lastUsed    time.Time
	isConnected bool
}

// Status represents the current status of the N50 component
type Status struct {
	IsConnected   bool   `json:"isConnected"`
	PowerStatus   string `json:"powerStatus"`
	CurrentInput  string `json:"currentInput"`
	PowerRaw      string `json:"powerRaw,omitempty"`
	InputRaw      string `json:"inputRaw,omitempty"`
	Error         string `json:"error,omitempty"`
}

// NewClient creates a new N50 client
func NewClient() *Client {
	return &Client{}
}

// getAddress returns the configured N50 address
func getAddress() string {
	cfg := config.Get()
	if cfg.N50Host == "" {
		return ""
	}
	port := cfg.N50Port
	if port <= 0 {
		port = 8102
	}
	return net.JoinHostPort(cfg.N50Host, fmt.Sprintf("%d", port))
}

// isEnabled returns whether N50 is enabled in config
func isEnabled() bool {
	cfg := config.Get()
	return cfg.N50Enabled && cfg.N50Host != ""
}

// EnsureConnection ensures the TCP connection is established
func (c *Client) EnsureConnection() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !isEnabled() {
		return fmt.Errorf("N50 is not enabled or host is not configured")
	}

	if c.conn != nil {
		// Check if connection is still alive
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		one := make([]byte, 1)
		if _, err := c.conn.Read(one); err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Still alive
				c.conn.SetReadDeadline(time.Time{})
				return nil
			}
			// Connection dead
			c.conn.Close()
			c.conn = nil
			c.isConnected = false
		}
	}

	if c.conn == nil {
		addr := getAddress()
		if addr == "" {
			return fmt.Errorf("N50 address not configured")
		}

		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			c.isConnected = false
			return fmt.Errorf("failed to connect to N50: %w", err)
		}

		c.conn = conn
		c.reader = bufio.NewReader(conn)
		c.writer = bufio.NewWriter(conn)
		c.lastUsed = time.Now()
		c.isConnected = true

		log.Printf("[N50] Connected to %s", addr)
	}

	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return err
	}
	return nil
}

// sendCommand sends a command to the N50 and returns the response
func (c *Client) sendCommand(command string) (string, error) {
	if err := c.EnsureConnection(); err != nil {
		return "", err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Set write deadline
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.writer.WriteString(command + "\r\n"); err != nil {
		c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return "", fmt.Errorf("write error: %w", err)
	}
	if err := c.writer.Flush(); err != nil {
		c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return "", fmt.Errorf("flush error: %w", err)
	}

	// Set read deadline
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	response, err := c.reader.ReadString('\n')
	if err != nil {
		c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return "", fmt.Errorf("read error: %w", err)
	}

	c.lastUsed = time.Now()
	return strings.TrimSpace(response), nil
}

// GetPowerStatus gets the current power status of the N50
func (c *Client) GetPowerStatus() (string, error) {
	resp, err := c.sendCommand(CmdGetPowerStatus)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// GetCurrentFunction gets the current input function of the N50
func (c *Client) GetCurrentFunction() (string, error) {
	resp, err := c.sendCommand(CmdGetCurrentFunction)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// PowerUp powers on the N50
func (c *Client) PowerUp() error {
	_, err := c.sendCommand(CmdPowerUp)
	return err
}

// StandBy puts the N50 in standby mode
func (c *Client) StandBy() error {
	_, err := c.sendCommand(CmdStandBy)
	return err
}

// SetInput sets the input source
func (c *Client) SetInput(inputName string) error {
	cmd, ok := InputCommands[inputName]
	if !ok {
		return fmt.Errorf("unknown input: %s", inputName)
	}
	_, err := c.sendCommand(cmd)
	return err
}

// GetStatus gets the full status of the N50
func (c *Client) GetStatus() (*Status, error) {
	status := &Status{
		IsConnected: c.isConnected,
	}

	// Get power status
	powerResp, err := c.GetPowerStatus()
	if err != nil {
		status.Error = err.Error()
		return status, err
	}
	status.PowerRaw = powerResp
	status.PowerStatus = ResponseDescriptions[powerResp]
	if status.PowerStatus == "" {
		status.PowerStatus = powerResp
	}

	// Get current input
	inputResp, err := c.GetCurrentFunction()
	if err != nil {
		status.Error = err.Error()
		return status, err
	}
	status.InputRaw = inputResp
	status.CurrentInput = ResponseDescriptions[inputResp]
	if status.CurrentInput == "" {
		status.CurrentInput = inputResp
	}

	return status, nil
}

// IsPoweredUp checks if the N50 is powered up
func (c *Client) IsPoweredUp() (bool, error) {
	powerStatus, err := c.GetPowerStatus()
	if err != nil {
		return false, err
	}
	return powerStatus == PowerStatusUp, nil
}

// IsCorrectInput checks if the N50 is set to the configured input
func (c *Client) IsCorrectInput() (bool, error) {
	cfg := config.Get()
	currentInput, err := c.GetCurrentFunction()
	if err != nil {
		return false, err
	}
	expectedInput := InputResponses[cfg.N50Input]
	return currentInput == expectedInput, nil
}

// EnsureReady ensures the N50 is powered up and set to the correct input
// Returns true if the N50 was already ready, false if it had to be changed
func (c *Client) EnsureReady() (bool, error) {
	cfg := config.Get()

	// Check if powered up
	isPowered, err := c.IsPoweredUp()
	if err != nil {
		return false, fmt.Errorf("failed to check power status: %w", err)
	}

	wasReady := true

	if !isPowered {
		log.Printf("[N50] Powering up N50...")
		if err := c.PowerUp(); err != nil {
			return false, fmt.Errorf("failed to power up N50: %w", err)
		}
		// Wait a moment for the power up to take effect
		time.Sleep(500 * time.Millisecond)
		wasReady = false
	}

	// Check if correct input
	isCorrectInput, err := c.IsCorrectInput()
	if err != nil {
		return false, fmt.Errorf("failed to check input status: %w", err)
	}

	if !isCorrectInput {
		log.Printf("[N50] Setting input to %s...", cfg.N50Input)
		if err := c.SetInput(cfg.N50Input); err != nil {
			return false, fmt.Errorf("failed to set input: %w", err)
		}
		// Wait a moment for the input change to take effect
		time.Sleep(300 * time.Millisecond)
		wasReady = false
	}

	// Verify the status
	status, err := c.GetStatus()
	if err != nil {
		return false, fmt.Errorf("failed to verify N50 status: %w", err)
	}

	if status.PowerRaw != PowerStatusUp {
		return false, fmt.Errorf("N50 failed to power up")
	}

	expectedInput := InputResponses[cfg.N50Input]
	if status.InputRaw != expectedInput {
		return false, fmt.Errorf("N50 failed to switch to correct input")
	}

	return wasReady, nil
}

// Singleton instance
var (
	instance *Client
	once     sync.Once
)

// GetClient returns the singleton N50 client
func GetClient() *Client {
	once.Do(func() {
		instance = NewClient()
	})
	return instance
}

// ResetClient resets the N50 client connection
func ResetClient() {
	if instance != nil {
		instance.Close()
		instance = NewClient()
	}
}

// IsEnabled returns whether N50 control is enabled
func IsEnabled() bool {
	return isEnabled()
}

// ShouldAutoControl returns whether auto control is enabled
func ShouldAutoControl() bool {
	cfg := config.Get()
	return cfg.N50Enabled && cfg.N50AutoControl && cfg.N50Host != ""
}

// ShouldIgnoreOnStart returns whether to ignore N50 for starting playback
func ShouldIgnoreOnStart() bool {
	cfg := config.Get()
	return cfg.N50IgnoreOnStart
}

// GetConfiguredInput returns the configured input name
func GetConfiguredInput() string {
	cfg := config.Get()
	return cfg.N50Input
}

// GetAvailableInputs returns the list of available input names
func GetAvailableInputs() []string {
	inputs := make([]string, 0, len(InputCommands))
	for input := range InputCommands {
		inputs = append(inputs, input)
	}
	return inputs
}
