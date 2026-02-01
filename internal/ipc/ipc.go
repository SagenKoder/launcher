package ipc

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// Command types for IPC communication
const (
	CmdShow = "show"
	CmdHide = "hide"
	CmdQuit = "quit"
)

// Server handles incoming IPC commands from clients
type Server struct {
	listener net.Listener
	showFn   func()
	hideFn   func()
	quitFn   func()
}

// GetSocketPath returns the Unix socket path for the launcher daemon
func GetSocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("launcher-%d.sock", os.Getuid()))
}

// NewServer creates a new IPC server
func NewServer(showFn, hideFn, quitFn func()) *Server {
	return &Server{
		showFn: showFn,
		hideFn: hideFn,
		quitFn: quitFn,
	}
}

// Start begins listening for IPC commands
func (s *Server) Start() error {
	socketPath := GetSocketPath()

	// Remove existing socket if present
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	s.listener = listener

	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Listener closed
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	cmd := strings.TrimSpace(line)
	switch cmd {
	case CmdShow:
		if s.showFn != nil {
			s.showFn()
		}
		conn.Write([]byte("ok\n"))
	case CmdHide:
		if s.hideFn != nil {
			s.hideFn()
		}
		conn.Write([]byte("ok\n"))
	case CmdQuit:
		conn.Write([]byte("ok\n"))
		if s.quitFn != nil {
			s.quitFn()
		}
	default:
		conn.Write([]byte("unknown command\n"))
	}
}

// Close shuts down the IPC server
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// Cleanup removes the socket file
func Cleanup() {
	os.Remove(GetSocketPath())
}

// TryShow attempts to connect to an existing daemon and send a show command.
// Returns true if successful, false if no daemon is running.
func TryShow() bool {
	return sendCommand(CmdShow)
}

// TryQuit attempts to connect to an existing daemon and send a quit command.
// Returns true if successful, false if no daemon is running.
func TryQuit() bool {
	return sendCommand(CmdQuit)
}

// TryPing checks if a daemon is running by attempting to connect to the socket.
// Returns true if a daemon is running, false otherwise.
func TryPing() bool {
	socketPath := GetSocketPath()
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func sendCommand(cmd string) bool {
	socketPath := GetSocketPath()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	defer conn.Close()

	_, err = conn.Write([]byte(cmd + "\n"))
	if err != nil {
		return false
	}

	// Wait for acknowledgment
	reader := bufio.NewReader(conn)
	_, err = reader.ReadString('\n')
	return err == nil
}
