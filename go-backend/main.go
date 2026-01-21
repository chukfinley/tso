package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	workingDir, _ := os.Getwd()
	loadEnvConfig(workingDir)

	// Initialize session store
	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	if len(os.Getenv("SESSION_SECRET")) == 0 {
		store = sessions.NewCookieStore([]byte("default-secret-change-in-production"))
	}

	// Initialize database
	db, err := NewDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize router
	r := mux.NewRouter()

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Auth routes
	api.HandleFunc("/login", LoginHandler).Methods("POST")
	api.HandleFunc("/logout", LogoutHandler).Methods("POST")
	api.HandleFunc("/auth/check", AuthCheckHandler).Methods("GET")

	// User routes
	api.HandleFunc("/users", RequireAuth(RequireAdmin(ListUsersHandler))).Methods("GET")
	api.HandleFunc("/users", RequireAuth(RequireAdmin(CreateUserHandler))).Methods("POST")
	api.HandleFunc("/users/{id}", RequireAuth(RequireAdmin(GetUserHandler))).Methods("GET")
	api.HandleFunc("/users/{id}", RequireAuth(RequireAdmin(UpdateUserHandler))).Methods("PUT")
	api.HandleFunc("/users/{id}", RequireAuth(RequireAdmin(DeleteUserHandler))).Methods("DELETE")
	api.HandleFunc("/users/{id}/password", RequireAuth(UpdatePasswordHandler)).Methods("PUT")
	api.HandleFunc("/profile", RequireAuth(GetProfileHandler)).Methods("GET")
	api.HandleFunc("/profile", RequireAuth(UpdateProfileHandler)).Methods("PUT")

	// Share routes
	api.HandleFunc("/shares", RequireAuth(ListSharesHandler)).Methods("GET")
	api.HandleFunc("/shares", RequireAuth(CreateShareHandler)).Methods("POST")
	api.HandleFunc("/shares/{id}", RequireAuth(GetShareHandler)).Methods("GET")
	api.HandleFunc("/shares/{id}", RequireAuth(UpdateShareHandler)).Methods("PUT")
	api.HandleFunc("/shares/{id}", RequireAuth(DeleteShareHandler)).Methods("DELETE")
	api.HandleFunc("/shares/{id}/toggle", RequireAuth(ToggleShareHandler)).Methods("POST")
	api.HandleFunc("/shares/{id}/permissions", RequireAuth(GetSharePermissionsHandler)).Methods("GET")
	api.HandleFunc("/shares/{id}/permissions", RequireAuth(SetSharePermissionHandler)).Methods("POST")
	api.HandleFunc("/shares/{id}/permissions/{userId}", RequireAuth(RemoveSharePermissionHandler)).Methods("DELETE")
	api.HandleFunc("/share-users", RequireAuth(ListShareUsersHandler)).Methods("GET")
	api.HandleFunc("/share-users", RequireAuth(CreateShareUserHandler)).Methods("POST")
	api.HandleFunc("/share-users/{id}", RequireAuth(GetShareUserHandler)).Methods("GET")
	api.HandleFunc("/share-users/{id}/password", RequireAuth(UpdateShareUserPasswordHandler)).Methods("PUT")
	api.HandleFunc("/share-users/{id}", RequireAuth(DeleteShareUserHandler)).Methods("DELETE")
	api.HandleFunc("/share-users/{id}/toggle", RequireAuth(ToggleShareUserHandler)).Methods("POST")
	api.HandleFunc("/shares/config/test", RequireAuth(TestSambaConfigHandler)).Methods("GET")
	api.HandleFunc("/shares/status", RequireAuth(GetSambaStatusHandler)).Methods("GET")
	api.HandleFunc("/shares/clients", RequireAuth(GetConnectedClientsHandler)).Methods("GET")
	api.HandleFunc("/shares/logs", RequireAuth(GetShareLogsHandler)).Methods("GET")

	// VM routes
	api.HandleFunc("/vms", RequireAuth(ListVMsHandler)).Methods("GET")
	api.HandleFunc("/vms", RequireAuth(CreateVMHandler)).Methods("POST")
	api.HandleFunc("/vms/{id}", RequireAuth(GetVMHandler)).Methods("GET")
	api.HandleFunc("/vms/{id}", RequireAuth(UpdateVMHandler)).Methods("PUT")
	api.HandleFunc("/vms/{id}", RequireAuth(DeleteVMHandler)).Methods("DELETE")
	api.HandleFunc("/vms/{id}/start", RequireAuth(StartVMHandler)).Methods("POST")
	api.HandleFunc("/vms/{id}/stop", RequireAuth(StopVMHandler)).Methods("POST")
	api.HandleFunc("/vms/{id}/restart", RequireAuth(RestartVMHandler)).Methods("POST")
	api.HandleFunc("/vms/{id}/status", RequireAuth(GetVMStatusHandler)).Methods("GET")
	api.HandleFunc("/vms/{id}/logs", RequireAuth(GetVMLogsHandler)).Methods("GET")
	api.HandleFunc("/vms/{id}/spice", RequireAuth(GetVMSpiceHandler)).Methods("GET")
	api.HandleFunc("/vms/isos", RequireAuth(ListISOsHandler)).Methods("GET")
	api.HandleFunc("/vms/isos", RequireAuth(UploadISOHandler)).Methods("POST")
	api.HandleFunc("/vms/disks", RequireAuth(ListPhysicalDisksHandler)).Methods("GET")
	api.HandleFunc("/vms/bridges", RequireAuth(ListNetworkBridgesHandler)).Methods("GET")
	api.HandleFunc("/vms/{id}/backups", RequireAuth(ListVMBackupsHandler)).Methods("GET")
	api.HandleFunc("/vms/{id}/backups", RequireAuth(CreateVMBackupHandler)).Methods("POST")
	api.HandleFunc("/vms/backups/{backupId}/status", RequireAuth(CheckBackupStatusHandler)).Methods("GET")
	api.HandleFunc("/vms/backups/{backupId}/restore", RequireAuth(RestoreBackupHandler)).Methods("POST")
	api.HandleFunc("/vms/backups/{backupId}", RequireAuth(DeleteBackupHandler)).Methods("DELETE")

	// System routes
	api.HandleFunc("/system/stats", RequireAuth(GetSystemStatsHandler)).Methods("GET")
	api.HandleFunc("/system/info", GetServerInfoHandler).Methods("GET")
	api.HandleFunc("/system/update", RequireAuth(RequireAdmin(SystemUpdateHandler))).Methods("POST")
	api.HandleFunc("/system/control", RequireAuth(RequireAdmin(SystemControlHandler))).Methods("POST")

	// Terminal routes
	api.HandleFunc("/terminal/execute", RequireAuth(RequireAdmin(ExecuteTerminalHandler))).Methods("POST")
	api.HandleFunc("/terminal/ws", RequireAuth(RequireAdmin(TerminalWebSocketHandler))).Methods("GET")

	// Logs routes
	api.HandleFunc("/logs", RequireAuth(GetLogsHandler)).Methods("GET")
	api.HandleFunc("/logs/activity", RequireAuth(GetActivityLogsHandler)).Methods("GET")

	// Network routes
	api.HandleFunc("/network/interfaces", RequireAuth(ListNetworkInterfacesHandler)).Methods("GET")
	api.HandleFunc("/network/interfaces/{name}", RequireAuth(GetNetworkInterfaceHandler)).Methods("GET")
	api.HandleFunc("/network/interfaces/{name}/toggle", RequireAuth(RequireAdmin(ToggleNetworkInterfaceHandler))).Methods("POST")
	api.HandleFunc("/network/bandwidth", RequireAuth(GetNetworkBandwidthHandler)).Methods("GET")
	api.HandleFunc("/network/history", RequireAuth(GetNetworkHistoryHandler)).Methods("GET")
	api.HandleFunc("/network/events", RequireAuth(GetNetworkEventsHandler)).Methods("GET")
	api.HandleFunc("/network/processes", RequireAuth(GetNetworkProcessesHandler)).Methods("GET")
	api.HandleFunc("/network/connections", RequireAuth(GetNetworkConnectionsHandler)).Methods("GET")
	api.HandleFunc("/network/session/reset", RequireAuth(ResetSessionStatsHandler)).Methods("POST")

	// Network throttling routes
	api.HandleFunc("/network/throttle/support", RequireAuth(CheckThrottleSupportHandler)).Methods("GET")
	api.HandleFunc("/network/throttle", RequireAuth(RequireAdmin(GetProcessThrottlesHandler))).Methods("GET")
	api.HandleFunc("/network/throttle", RequireAuth(RequireAdmin(SetProcessThrottleHandler))).Methods("POST")
	api.HandleFunc("/network/throttle/{pid}", RequireAuth(RequireAdmin(RemoveProcessThrottleHandler))).Methods("DELETE")

	// Frontend routes (serve static files and SPA routing)
	// Find frontend dist directory
	installDir := os.Getenv("INSTALL_DIR")
	if installDir == "" {
		installDir = "/opt/serveros"
	}

	// Get current working directory (where backend is running from)
	if wd, err := os.Getwd(); err == nil && wd != "" {
		workingDir = wd
	}
	if workingDir == "" {
		workingDir = installDir + "/go-backend"
	}

	// Build list of possible frontend directories
	frontendDirs := []string{
		installDir + "/frontend/dist",
		"/opt/serveros/frontend/dist",
	}

	// Add working directory relative paths
	if workingDir != "" {
		frontendDirs = append(frontendDirs,
			filepath.Join(workingDir, "..", "frontend", "dist"),
			filepath.Join(workingDir, "..", "..", "frontend", "dist"),
			filepath.Join(filepath.Dir(workingDir), "frontend", "dist"),
		)
	}

	// Add more fallback paths
	frontendDirs = append(frontendDirs,
		"/opt/serveros/go-backend/../frontend/dist",
		"./frontend/dist",
		"../frontend/dist",
	)

	var frontendDir string
	for _, dir := range frontendDirs {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			continue
		}

		dirInfo, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		if !dirInfo.IsDir() {
			continue
		}

		indexPath := filepath.Join(absPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			frontendDir = absPath
			log.Printf("✓ Frontend found at: %s", frontendDir)
			break
		}
	}

	frontendPort := os.Getenv("FRONTEND_PORT")
	if frontendPort == "" {
		frontendPort = "80"
	}

	if frontendDir == "" {
		log.Printf("⚠ WARNING: Frontend not found!")
		log.Printf("  Install dir: %s", installDir)
		log.Printf("  Working dir: %s", workingDir)
		log.Printf("  Searched in: %v", frontendDirs)
	}

	// API port configuration
	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	// Get server info for startup message and CORS configuration
	serverIP := getServerIP()
	hostname := getHostname()

	allowedOrigins := buildAllowedOrigins(serverIP, hostname, frontendPort)
	apiHandler := withCORS(r, allowedOrigins)
	log.Printf("CORS allowed origins: %v", allowedOrigins)

	// Serve frontend static files on dedicated port if available
	if frontendDir != "" {
		log.Printf("✓ Serving frontend from: %s", frontendDir)

		frontendFS := http.FileServer(http.Dir(frontendDir))
		frontendHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api") {
				http.NotFound(w, req)
				return
			}

			if req.URL.Path == "/" || req.URL.Path == "" {
				http.ServeFile(w, req, filepath.Join(frontendDir, "index.html"))
				return
			}

			requestedPath := filepath.Join(frontendDir, req.URL.Path)
			if info, err := os.Stat(requestedPath); err == nil && !info.IsDir() {
				frontendFS.ServeHTTP(w, req)
				return
			}

			http.ServeFile(w, req, filepath.Join(frontendDir, "index.html"))
		})

		go func() {
			log.Printf("Frontend server starting on port %s", frontendPort)
			if err := http.ListenAndServe(":"+frontendPort, frontendHandler); err != nil && err != http.ErrServerClosed {
				if errors.Is(err, syscall.EACCES) {
					log.Printf("⚠ Unable to bind frontend server to port %s (permission denied). Adjust FRONTEND_PORT or run with elevated privileges.", frontendPort)
					return
				}
				log.Fatalf("Frontend server failed: %v", err)
			}
		}()
	} else {
		r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api") {
				http.NotFound(w, req)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"TSO API","version":"2.0","status":"running","frontend":"not built - run 'cd frontend && npm run build'"}`))
		})
	}

	// Check if another web server might conflict with the frontend port
	existingWebServer := false
	if cmd := exec.Command("systemctl", "is-active", "--quiet", "nginx"); cmd.Run() == nil {
		existingWebServer = true
	} else if cmd := exec.Command("systemctl", "is-active", "--quiet", "apache2"); cmd.Run() == nil {
		existingWebServer = true
	}
	if existingWebServer {
		log.Printf("⚠ Detected existing web server (nginx/apache2). Ensure it does not conflict with port %s.", frontendPort)
	}

	// installDir was already defined above, just use it for messaging
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "servermanager"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "tso"
	}

	// Print startup message with server info (similar to old script)
	fmt.Println("")
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  TSO Server Started!                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("ServerOS Installation Credentials")
	fmt.Println("==================================")
	fmt.Printf("Generated: %s\n", time.Now().Format("Mon Jan 2 15:04:05 MST 2006"))
	fmt.Println("")
	fmt.Println("Web Access:")
	fmt.Println("-----------")
	if frontendDir != "" {
		fmt.Printf("URL: http://%s\n", formatHostWithPort(serverIP, frontendPort))
		fmt.Printf("Hostname: http://%s\n", formatHostWithPort(hostname, frontendPort))
	} else {
		fmt.Printf("URL: http://%s:%s\n", serverIP, apiPort)
		fmt.Printf("Hostname: http://%s:%s\n", hostname, apiPort)
	}
	fmt.Println("")
	fmt.Println("API:")
	fmt.Println("----")
	fmt.Printf("Base URL: http://%s:%s/api\n", serverIP, apiPort)
	fmt.Printf("Hostname: http://%s:%s/api\n", hostname, apiPort)
	fmt.Println("")
	fmt.Println("Default Login:")
	fmt.Println("--------------")
	fmt.Println("Username: admin")
	fmt.Println("Password: admin123")
	fmt.Println("")
	fmt.Println("Database:")
	fmt.Println("---------")
	fmt.Printf("Database: %s\n", dbName)
	fmt.Printf("User: %s\n", dbUser)
	if dbPass := os.Getenv("DB_PASS"); dbPass != "" {
		fmt.Printf("Password: %s\n", dbPass)
	}
	fmt.Println("")
	fmt.Println("Installation Path:")
	fmt.Println("------------------")
	fmt.Printf("%s\n", installDir)
	fmt.Println("")
	fmt.Println("Apache Config:")
	fmt.Println("--------------")
	fmt.Println("/etc/apache2/sites-available/serveros.conf")
	fmt.Println("")
	fmt.Println("IMPORTANT: Change default admin password immediately!")
	fmt.Println("")
	log.Printf("API server starting on port %s", apiPort)
	log.Fatal(http.ListenAndServe(":"+apiPort, apiHandler))
}

func loadEnvConfig(initialWorkingDir string) {
	seen := map[string]struct{}{}
	addCandidate := func(path string) {
		if path == "" {
			return
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		if _, exists := seen[absPath]; exists {
			return
		}
		seen[absPath] = struct{}{}

		if err := loadEnvFile(absPath); err == nil {
			log.Printf("Loaded environment configuration from %s", absPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			log.Printf("⚠ Unable to load environment file %s: %v", absPath, err)
		}
	}

	if initialWorkingDir != "" {
		addCandidate(filepath.Join(initialWorkingDir, ".env"))
	}

	if installDir := os.Getenv("INSTALL_DIR"); installDir != "" {
		addCandidate(filepath.Join(installDir, "go-backend", ".env"))
	}

	addCandidate("/opt/serveros/go-backend/.env")
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export"))
		}

		sep := strings.Index(line, "=")
		if sep == -1 {
			continue
		}

		key := strings.TrimSpace(line[:sep])
		value := strings.TrimSpace(line[sep+1:])

		if key == "" {
			continue
		}

		value = strings.Trim(value, "\"'")

		if envVal := os.Getenv(key); envVal != "" {
			continue
		}

		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func formatHostWithPort(host, port string) string {
	if host == "" {
		return host
	}
	if port == "" || port == "80" {
		return host
	}
	return fmt.Sprintf("%s:%s", host, port)
}

func withCORS(handler http.Handler, allowedOrigins []string) http.Handler {
	allowAny := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAny = true
			break
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "null" {
			if allowAny || isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

func buildAllowedOrigins(serverIP, hostname, frontendPort string) []string {
	if env := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")); env != "" {
		parts := strings.Split(env, ",")
		origins := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				appendIfMissing(&origins, trimmed)
			}
		}
		return origins
	}

	origins := []string{}
	defaultOrigins := []string{
		"http://localhost",
		"http://localhost:3000",
		"http://127.0.0.1",
		"http://127.0.0.1:3000",
	}

	for _, origin := range defaultOrigins {
		appendIfMissing(&origins, origin)
	}

	addOriginForHost(&origins, hostname, frontendPort)
	addOriginForHost(&origins, serverIP, frontendPort)

	return origins
}

func addOriginForHost(origins *[]string, host, port string) {
	trimmedHost := strings.TrimSpace(host)
	if trimmedHost == "" || trimmedHost == "localhost" {
		return
	}

	appendIfMissing(origins, fmt.Sprintf("http://%s", trimmedHost))
	if port != "" {
		appendIfMissing(origins, fmt.Sprintf("http://%s:%s", trimmedHost, port))
	}
}

func appendIfMissing(origins *[]string, value string) {
	if value == "" {
		return
	}
	for _, existing := range *origins {
		if strings.EqualFold(existing, value) {
			return
		}
	}
	*origins = append(*origins, value)
}

