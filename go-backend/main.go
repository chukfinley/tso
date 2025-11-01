package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
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

	// Logs routes
	api.HandleFunc("/logs", RequireAuth(GetLogsHandler)).Methods("GET")
	api.HandleFunc("/logs/activity", RequireAuth(GetActivityLogsHandler)).Methods("GET")

	// Frontend routes (serve static files and SPA routing)
	// Find frontend dist directory
	installDir := os.Getenv("INSTALL_DIR")
	if installDir == "" {
		installDir = "/opt/serveros"
	}
	
	// Get current working directory (where backend is running from)
	workingDir, _ := os.Getwd()
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
		// Clean the path
		absPath, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		
		// Check if directory exists and has index.html
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			indexPath := absPath + "/index.html"
			if _, err := os.Stat(indexPath); err == nil {
				frontendDir = absPath
				log.Printf("Frontend found at: %s", frontendDir)
				break
			}
		}
	}
	
	if frontendDir == "" {
		log.Printf("WARNING: Frontend not found. Searched in: %v", frontendDirs)
		log.Printf("Install dir: %s, Working dir: %s", installDir, workingDir)
	}
	
	// Serve frontend static files if available
	if frontendDir != "" {
		// Use FileServer for better static file serving with proper MIME types
		frontendFS := http.FileServer(http.Dir(frontendDir))
		
		// Handler function that serves static files or index.html for SPA routing
		frontendHandler := func(w http.ResponseWriter, req *http.Request) {
			// Skip API routes - let API router handle them
			if strings.HasPrefix(req.URL.Path, "/api") {
				http.NotFound(w, req)
				return
			}
			
			// Try to serve the requested file first
			requestedPath := frontendDir + req.URL.Path
			
			// Serve index.html for root path
			if req.URL.Path == "/" {
				http.ServeFile(w, req, frontendDir+"/index.html")
				return
			}
			
			// Check if file exists
			if info, err := os.Stat(requestedPath); err == nil && !info.IsDir() {
				// File exists - serve it using FileServer for proper MIME types
				frontendFS.ServeHTTP(w, req)
				return
			}
			
			// File doesn't exist - serve index.html for SPA routing
			http.ServeFile(w, req, frontendDir+"/index.html")
		}
		
		// Register handler for all non-API routes
		// This must come after API routes but will match everything else
		r.PathPrefix("/").HandlerFunc(frontendHandler)
	} else {
		// If frontend not built, serve API info
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Get server info for startup message
	serverIP := getServerIP()
	hostname := getHostname()
	installDir := os.Getenv("INSTALL_DIR")
	if installDir == "" {
		installDir = "/opt/serveros"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "servermanager"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "tso"
	}

	// Check if web server is running
	hasWebServer := false
	if cmd := exec.Command("systemctl", "is-active", "--quiet", "nginx"); cmd.Run() == nil {
		hasWebServer = true
	} else if cmd := exec.Command("systemctl", "is-active", "--quiet", "apache2"); cmd.Run() == nil {
		hasWebServer = true
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
	if hasWebServer {
		fmt.Printf("URL: http://%s\n", serverIP)
		fmt.Printf("Hostname: http://%s\n", hostname)
	} else {
		fmt.Printf("URL: http://%s:%s\n", serverIP, port)
		fmt.Printf("Hostname: http://%s:%s\n", hostname, port)
	}
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
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

