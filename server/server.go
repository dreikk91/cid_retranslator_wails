package server

import (
	"bufio"
	"cid_retranslator/cidParser"
	"cid_retranslator/config"
	"cid_retranslator/queue"
	"context"
	"io"
	"log"
	"log/slog"
	"net"
	"slices"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	host               string
	port               string
	queue              *queue.Queue
	rules              *config.CIDRules
	cancel             context.CancelFunc
	stopOnce           sync.Once
	listener           net.Listener
	isRunning          bool
	devices            []Device
	deviceMu           sync.RWMutex
	globalEvents       []GlobalEvent
	globalMu           sync.RWMutex
}

// Event represents an event for a device
type Event struct {
	Time string `json:"time"`
	Data string `json:"data"`
}

// Device represents a device with its events
type Device struct {
	ID           int      `json:"id"`
	LastEventTime string  `json:"lastEventTime"`
	LastEvent    string   `json:"lastEvent"`
	Events       []Event  `json:"events"`
}

// GlobalEvent represents a global event across all devices
type GlobalEvent struct {
	Time     string `json:"time"`
	DeviceID int    `json:"deviceID"`
	Data     string `json:"data"`
}

// connection represents a client connection to the server.
type connection struct {
	conn   net.Conn
	queue  *queue.Queue
	rules  *config.CIDRules
	server *Server // Reference to server for access to devices
}

func New(cfg *config.ServerConfig, q *queue.Queue, rules *config.CIDRules) *Server {
	return &Server{
		host:        cfg.Host,
		port:        cfg.Port,
		queue:       q,
		rules:       rules,
		devices:     make([]Device, 0),
		globalEvents: make([]GlobalEvent, 0),
	}
}

func (server *Server) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	server.cancel = cancel
	server.queue.UpdateStartTime()

	listener, err := net.Listen("tcp", server.host+":"+server.port)
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
	server.listener = listener
	server.isRunning = true

	slog.Info("Server started", "host", server.host, "port", server.port)

	go func() {
		defer server.listener.Close()
		for {
			conn, err := server.listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					slog.Info("Server listener stopped.")
					return
				default:
					slog.Error("Accept error", "error", err)
				}
				continue
			}
			slog.Info("Accepted connection", "from", conn.RemoteAddr())
			connHandler := &connection{conn: conn, queue: server.queue, rules: server.rules, server: server}
			go connHandler.handleRequest(ctx)
		}
	}()

	<-ctx.Done()
	slog.Info("Server stopping...")
	server.isRunning = false
}

func (server *Server) Stop() {
	server.stopOnce.Do(func() {
		if server.cancel != nil {
			slog.Info("Stopping server...")
			server.cancel()
			if server.listener != nil {
				server.listener.Close()
			}
		}
	})
}

// UpdateDevice updates or adds an event for the device
func (server *Server) UpdateDevice(id int, event string) {
	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")

	server.deviceMu.Lock()
	defer server.deviceMu.Unlock()

	found := false
	for i := range server.devices {
		if server.devices[i].ID == id {
			server.devices[i].LastEventTime = nowStr
			server.devices[i].LastEvent = event
			server.devices[i].Events = append(server.devices[i].Events, Event{Time: nowStr, Data: event})
			if len(server.devices[i].Events) > 100 {
				server.devices[i].Events = server.devices[i].Events[len(server.devices[i].Events)-100:]
			}
			found = true
			break
		}
	}

	if !found {
		newDevice := Device{
			ID:           id,
			LastEventTime: nowStr,
			LastEvent:    event,
			Events:       []Event{{Time: nowStr, Data: event}},
		}
		server.devices = append(server.devices, newDevice)
	}

	// Add to global events
	server.globalMu.Lock()
	server.globalEvents = append(server.globalEvents, GlobalEvent{Time: nowStr, DeviceID: id, Data: event})
	if len(server.globalEvents) > 500 {
		server.globalEvents = server.globalEvents[len(server.globalEvents)-500:]
	}
	server.globalMu.Unlock()
}

// GetDevices returns a list of devices (without full events history for efficiency)
func (server *Server) GetDevices() []Device {
	server.deviceMu.RLock()
	defer server.deviceMu.RUnlock()

	devs := make([]Device, len(server.devices))
	for i, d := range server.devices {
		devs[i] = Device{
			ID:           d.ID,
			LastEventTime: d.LastEventTime,
			LastEvent:    d.LastEvent,
			// Events omitted for summary
		}
	}
	return devs
}

// GetGlobalEvents returns the global list of events
func (server *Server) GetGlobalEvents() []GlobalEvent {
	server.globalMu.RLock()
	defer server.globalMu.RUnlock()
	events := append([]GlobalEvent{}, server.globalEvents...)
	slices.Reverse(events)
	return events
}

// GetDeviceEvents returns the events for a specific device
func (server *Server) GetDeviceEvents(id int) []Event {
	server.deviceMu.RLock()
	defer server.deviceMu.RUnlock()

	for _, d := range server.devices {
		if d.ID == id {
			return append([]Event{}, d.Events...)
		}
	}
	return []Event{}
}

func (c *connection) handleRequest(ctx context.Context) {
	remoteAddr := c.conn.RemoteAddr()
	slog.Debug("Handling request", "from", remoteAddr)
	defer c.conn.Close()

	reader := bufio.NewReader(c.conn)
	for {
		select {
		case <-ctx.Done():
			slog.Info("Closing connection due to server shutdown.", "client", remoteAddr)
			return
		default:
		}

		messageBytes, err := reader.ReadBytes(0x14)
		if err != nil {
			if err != io.EOF {
				slog.Error("Read error", "from", remoteAddr, "error", err)
			} else {
				slog.Info("Connection closed by client", "client", remoteAddr)
			}
			return
		}

		if len(messageBytes) == 0 || messageBytes[len(messageBytes)-1] != 0x14 {
			slog.Warn("Malformed message", "from", remoteAddr, "data", string(messageBytes))
			if _, err := c.conn.Write([]byte{0x15}); err != nil {
				slog.Error("Error sending NACK for malformed message", "error", err)
			}
			return
		}
		slog.Debug("Received message", "from", remoteAddr, "data", string(messageBytes))

		messageWithoutDelimiter := string(messageBytes)
		if !cidparser.IsMessageValid(messageWithoutDelimiter, c.rules) {
			slog.Warn("Invalid message format", "from", remoteAddr, "data", string(messageBytes))
			if _, err := c.conn.Write([]byte{0x15}); err != nil {
				slog.Error("Error sending NACK for invalid format", "error", err)
			}
			continue
		}

		newMessage, err := cidparser.ChangeAccountNumber(messageBytes, c.rules)
		if err != nil {
			slog.Error("Error processing message", "from", remoteAddr, "error", err)
			if _, err := c.conn.Write([]byte{0x15}); err != nil {
				slog.Error("Error sending NACK for processing error", "error", err)
			}
			continue
		}

		replyCh := make(chan queue.DeliveryData, 1)
		sharedData := queue.SharedData{
			Payload: newMessage,
			ReplyCh: replyCh,
		}

		select {
		case c.queue.DataChannel <- sharedData:
			log.Println(c.queue.DataChannel)
			// Add event for device
			deviceID := extractDeviceID(newMessage)
			c.server.UpdateDevice(deviceID, string(newMessage))

			select {
			case clientReply, ok := <-replyCh:
				if !ok {
					slog.Warn("Reply channel closed unexpectedly", "from", remoteAddr)
					return
				}

				response, responseType := []byte{0x15}, "NACK"
				if clientReply.Status {
					response, responseType = []byte{0x06}, "ACK"
				}

				if _, err := c.conn.Write(response); err != nil {
					slog.Error("Error sending response", "type", responseType, "error", err)
					return
				}
				slog.Info("Message relayed", "from", remoteAddr, "status", responseType, "data", string(messageBytes))

			case <-time.After(10 * time.Second):
				slog.Error("Timeout waiting for client reply", "from", remoteAddr)
				if _, err := c.conn.Write([]byte{0x15}); err != nil {
					slog.Error("Error sending NACK after timeout", "error", err)
				}
			}
		default:
			slog.Warn("Queue buffer full, rejecting message", "from", remoteAddr)
			if _, err := c.conn.Write([]byte{0x15}); err != nil {
				slog.Error("Error sending NACK for buffer full", "error", err)
			}
		}
	}
}

func extractDeviceID(message []byte) int {
	accountNumber, err := strconv.Atoi(string(message[7:11]))
	if err != nil {
		slog.Error("Failed to extract device ID", "error", err)
		return 0
	}
	return accountNumber
}