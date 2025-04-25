package sse

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/lib/pq"
)

const (
	maxReconnectInterval = 30 * time.Second
	minReconnectInterval = 10 * time.Second
)

type Metrics interface {
	AddClient()
	DelClient()
	MsgsSent()
	NotificationReceived()
}

type Client struct {
	channel chan string
}

type Server struct {
	clients  map[*Client]struct{}
	addChan  chan *Client
	delChan  chan *Client
	msgChan  chan string
	listener *pq.Listener
	logger   *slog.Logger
	metrics  Metrics
}

func NewServer(databaseURL string, logger *slog.Logger, metrics Metrics) *Server {
	return &Server{
		clients:  make(map[*Client]struct{}),
		addChan:  make(chan *Client),
		delChan:  make(chan *Client),
		msgChan:  make(chan string),
		listener: pq.NewListener(databaseURL, minReconnectInterval, maxReconnectInterval, nil),
		logger:   logger,
		metrics:  metrics,
	}
}

func (s *Server) Close() error {
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("closing listener: %w", err)
	}

	return nil
}

// Run manages the clients and broadcasting of messages.
func (s *Server) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Context is done")

			return nil
		case client := <-s.addChan:
			s.logger.Debug("Add client")
			s.clients[client] = struct{}{}
			s.metrics.AddClient()
		case client := <-s.delChan:
			s.logger.Debug("Remove client")
			delete(s.clients, client)
			close(client.channel)
			s.metrics.DelClient()
		case msg := <-s.msgChan:
			s.logger.Debug("Send message", "message", msg)

			for client := range s.clients {
				select {
				case client.channel <- msg:
					s.metrics.MsgsSent()
				default:
					// If the client channel is full, remove it (prevents deadlock)
					delete(s.clients, client)
					close(client.channel)
					s.metrics.DelClient()
				}
			}
		}
	}
}

// ServeHTTP handles incoming SSE connections.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)

		return
	}

	client := &Client{channel: make(chan string)}
	s.addChan <- client

	defer func() {
		s.delChan <- client
	}()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			s.logger.Debug("Context is done, disconnecting")

			return
		case msg, ok := <-client.channel:
			if !ok {
				s.logger.Debug("Channel is closed, disconnecting")

				return
			}

			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			// Send heartbeat (comment prevents browsers from parsing it)
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// ListenForNotifications listens to postgresql notifications.
//
// New points are broadcasted to clients.
func (s *Server) ListenForNotifications(ctx context.Context) error {
	if err := s.listener.Listen("new_track_data"); err != nil {
		return fmt.Errorf("listening to new data: %w", err)
	}

	for {
		select {
		case notification := <-s.listener.Notify:
			s.logger.Info("Notification received", "notification", fmt.Sprintf("%+v", notification))

			if notification == nil {
				continue
			}

			s.metrics.NotificationReceived()

			// Broadcast new track point + pilot data to SSE clients
			s.Broadcast(notification.Extra)
		case <-ctx.Done():
			s.logger.WarnContext(ctx, "Listening for notifications, context is done.")

			return nil
		}
	}
}

// Broadcast sends the new point to all connected SSE clients.
func (s *Server) Broadcast(message string) {
	s.logger.Info("Broadcasting", "message", message)
	s.msgChan <- message
}
