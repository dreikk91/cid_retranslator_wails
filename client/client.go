package client

import (
	"cid_retranslator/config"
	"cid_retranslator/queue"
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Client struct {
	host             string
	port             string
	conn             *net.TCPConn
	queue            *queue.Queue
	reconnectInitial time.Duration
	reconnectMax     time.Duration
	cancel           context.CancelFunc
	stopOnce         sync.Once
}

func New(cfg *config.ClientConfig, q *queue.Queue) *Client {
	return &Client{
		host:             cfg.Host,
		port:             cfg.Port,
		queue:            q,
		reconnectInitial: cfg.ReconnectInitial,
		reconnectMax:     cfg.ReconnectMax,
	}
}

// GetQueueStats повертає статистику з черги
func (client *Client) GetQueueStats() (int, int, int, time.Duration) {
    return client.queue.Stats()
}

func (client *Client) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	client.cancel = cancel

	tcpAddr, err := net.ResolveTCPAddr("tcp", client.host+":"+client.port)
	if err != nil {
		slog.Error("Failed to resolve TCP address", "addr", client.host+":"+client.port, "error", err)
		return
	}

	go func() {
		delay := client.reconnectInitial
		reconnectAttempts := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				reconnectAttempts++
				client.queue.IncrementReconnects()
				logMessage := fmt.Sprintf("Dial failed (attempt %d), retrying in %s", reconnectAttempts, delay)
				if reconnectAttempts > 10 { // After 10 attempts, log as a warning
					slog.Warn(logMessage, "target", tcpAddr, "error", err)
				} else {
					slog.Error(logMessage, "target", tcpAddr, "error", err)
				}

				time.Sleep(delay)
				delay *= 2
				if delay > client.reconnectMax {
					delay = client.reconnectMax
				}
				continue
			}

			slog.Info("Connected to target", "target", tcpAddr)
			reconnectAttempts = 0 // Reset on successful connection
			client.conn = conn

			// handleConnection blocks until connection is lost or shutdown
			client.handleConnection(ctx, conn)

			conn.Close()
			client.conn = nil
			delay = client.reconnectInitial
			slog.Info("Connection closed, reconnecting...")
		}
	}()

	// Wait for stop signal
	<-ctx.Done()
	slog.Info("Client stopping...")
}

func (client *Client) Stop() {
	client.stopOnce.Do(func() {
		if client.cancel != nil {
			slog.Info("Stopping client...")
			client.cancel()
			if client.conn != nil {
				client.conn.Close()
			}
		}
	})
}

func (client *Client) handleConnection(ctx context.Context, conn *net.TCPConn) {
	for {
		select {
		case data, ok := <-client.queue.DataChannel:
			if !ok {
				slog.Info("DataChannel closed, stopping connection handler.")
				return
			}

			_, err := conn.Write(data.Payload)
			if err != nil {
				slog.Error("Write to server failed", "error", err)
				// Don't close the reply channel, server will timeout
				return // Exit to reconnect
			}
			slog.Debug("Wrote to server", "data", string(data.Payload))

			reply := make([]byte, 1024)
			n, err := conn.Read(reply)
			if err != nil {
				slog.Error("Read from server failed", "error", err)
				// Don't close the reply channel, server will timeout
				return // Exit to reconnect
			}

			slog.Debug("Reply from server", "reply", string(reply[:n]))
			if n == 1 && reply[0] == 0x06 {
				slog.Info("Received ACK")
				data.ReplyCh <- queue.DeliveryData{Status: true}
				client.queue.IncrementAccepted()
			} else {
				slog.Warn("Received NACK or other non-ACK response")
				data.ReplyCh <- queue.DeliveryData{Status: false}
				client.queue.IncrementRejected()
			}
			close(data.ReplyCh)

		case <-ctx.Done():
			slog.Info("Stopping connection handler due to shutdown signal.")
			return
		}
	}
}