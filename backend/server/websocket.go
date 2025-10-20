package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	opencode "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/tsukinoko-kun/orca/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type DirectoryInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type DirectoryListData struct {
	Directories []DirectoryInfo `json:"directories"`
}

type SelectDirectoryData struct {
	Path string `json:"path"`
}

type ServerInfo struct {
	URL          string   `json:"url"`
	Directory    string   `json:"directory"`
	ShareURL     string   `json:"shareUrl"`
	SessionID    string   `json:"sessionId"`
	CurrentModel string   `json:"currentModel"`
	CurrentAgent string   `json:"currentAgent"`
	Models       []string `json:"models"`
	Agents       []Agent  `json:"agents"`
}

type Agent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	BuiltIn     bool   `json:"builtIn"`
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	if err := sendDirectoryList(conn); err != nil {
		log.Printf("error sending directory list: %v", err)
		return
	}

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			break
		}

		switch msg.Type {
		case "selectDirectory":
			var data SelectDirectoryData
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				log.Printf("error unmarshaling selectDirectory data: %v", err)
				continue
			}
			s.handleOpenCodeSession(conn, data.Path)
			return
		}
	}
}

func sendDirectoryList(conn *websocket.Conn) error {
	var directories []DirectoryInfo

	err := filepath.WalkDir(config.Home, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if d.Name() == "node_modules" {
			return filepath.SkipDir
		}

		if isCodeProject(path) {
			relPath, _ := filepath.Rel(config.Home, path)
			if relPath == "." {
				relPath = filepath.Base(path)
			}
			directories = append(directories, DirectoryInfo{
				Name: relPath,
				Path: path,
			})
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory tree: %w", err)
	}

	data, err := json.Marshal(DirectoryListData{Directories: directories})
	if err != nil {
		return fmt.Errorf("failed to marshal directory list: %w", err)
	}

	msg := Message{
		Type: "directoryList",
		Data: data,
	}

	return conn.WriteJSON(msg)
}

func isCodeProject(path string) bool {
	gitPath := filepath.Join(path, ".git")
	if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
		return true
	}

	hgPath := filepath.Join(path, ".hg")
	if info, err := os.Stat(hgPath); err == nil && info.IsDir() {
		return true
	}

	pogoPath := filepath.Join(path, ".pogo.yaml")
	if _, err := os.Stat(pogoPath); err == nil {
		return true
	}

	return false
}

func (s *Server) handleOpenCodeSession(conn *websocket.Conn, dir string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "opencode", "serve", "--port", "0", "--hostname", "0.0.0.0")
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("error creating stdout pipe: %v", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("error creating stderr pipe: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("error starting opencode server: %v", err)
		return
	}
	defer func() {
		cancel()
		cmd.Wait()
	}()

	serverURL, err := waitForServerURL(stdout, stderr, 30*time.Second)
	if err != nil {
		log.Printf("error getting server URL: %v", err)
		return
	}

	models, err := getModelsList(dir)
	if err != nil {
		log.Printf("error getting models: %v", err)
	}

	currentModel, err := getCurrentModelFromAPI(serverURL)
	if err != nil {
		log.Printf("error getting current model from API: %v", err)
		if len(models) > 0 {
			currentModel = models[0]
		}
	}

	agents, currentAgent, err := getAgentsFromAPI(serverURL)
	if err != nil {
		log.Printf("error getting agents from API: %v", err)
	}

	client := opencode.NewClient(option.WithBaseURL(serverURL))

	sessionResp, err := client.Session.New(context.Background(), opencode.SessionNewParams{})
	if err != nil {
		log.Printf("error creating session: %v", err)
		return
	}

	shareResp, err := client.Session.Share(context.Background(), sessionResp.ID, opencode.SessionShareParams{})
	if err != nil {
		log.Printf("error sharing session: %v", err)
		return
	}

	time.Sleep(2 * time.Second)

	maxRetries := 5
	shareURL := shareResp.Share.URL
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(shareURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if i < maxRetries-1 {
			time.Sleep(1 * time.Second)
		}
	}

	s.opencodeServersMu.Lock()
	s.opencodeServers[sessionResp.ID] = serverURL
	s.opencodeServersMu.Unlock()

	defer func() {
		s.opencodeServersMu.Lock()
		delete(s.opencodeServers, sessionResp.ID)
		s.opencodeServersMu.Unlock()
	}()

	serverInfo := ServerInfo{
		URL:          serverURL,
		Directory:    dir,
		ShareURL:     shareURL,
		SessionID:    sessionResp.ID,
		CurrentModel: currentModel,
		CurrentAgent: currentAgent,
		Models:       models,
		Agents:       agents,
	}

	data, err := json.Marshal(serverInfo)
	if err != nil {
		log.Printf("error marshaling server info: %v", err)
		return
	}

	if err := conn.WriteJSON(Message{Type: "serverReady", Data: data}); err != nil {
		log.Printf("error sending server info: %v", err)
		return
	}

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
	}
}

func waitForServerURL(stdout, stderr io.Reader, timeout time.Duration) (string, error) {
	type result struct {
		url string
		err error
	}
	ch := make(chan result, 1)

	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("opencode: %s", line)
			if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
						ch <- result{url: strings.TrimSpace(part)}
						return
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- result{err: err}
		}
	}()

	select {
	case res := <-ch:
		return res.url, res.err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for server URL")
	}
}

func getModelsList(dir string) ([]string, error) {
	cmd := exec.Command("opencode", "models")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var models []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "█") {
			models = append(models, line)
		}
	}

	return models, nil
}

func getCurrentModelFromAPI(serverURL string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(serverURL + "/config")
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var configResponse struct {
		Model string `json:"model"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&configResponse); err != nil {
		return "", fmt.Errorf("failed to decode config: %w", err)
	}

	return configResponse.Model, nil
}

func getAgentsFromAPI(serverURL string) ([]Agent, string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(serverURL + "/agent")
	if err != nil {
		return nil, "", fmt.Errorf("failed to get agents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var agentsResponse []Agent
	if err := json.NewDecoder(resp.Body).Decode(&agentsResponse); err != nil {
		return nil, "", fmt.Errorf("failed to decode agents: %w", err)
	}

	configResp, err := client.Get(serverURL + "/config")
	if err != nil {
		return agentsResponse, "", fmt.Errorf("failed to get config: %w", err)
	}
	defer configResp.Body.Close()

	var configResponse struct {
		Agent struct {
			General struct {
				Disable bool `json:"disable"`
			} `json:"general"`
			Plan struct {
				Disable bool `json:"disable"`
			} `json:"plan"`
			Build struct {
				Disable bool `json:"disable"`
			} `json:"build"`
		} `json:"agent"`
	}

	if err := json.NewDecoder(configResp.Body).Decode(&configResponse); err == nil {
		if !configResponse.Agent.General.Disable {
			return agentsResponse, "general", nil
		}
		if !configResponse.Agent.Plan.Disable {
			return agentsResponse, "plan", nil
		}
		if !configResponse.Agent.Build.Disable {
			return agentsResponse, "build", nil
		}
	}

	if len(agentsResponse) > 0 {
		return agentsResponse, agentsResponse[0].Name, nil
	}

	return agentsResponse, "", nil
}
