package main

import (
	"log"
	"net/http"
	"os"

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
	api.HandleFunc("/system/update", RequireAuth(RequireAdmin(SystemUpdateHandler))).Methods("POST")
	api.HandleFunc("/system/control", RequireAuth(RequireAdmin(SystemControlHandler))).Methods("POST")

	// Terminal routes
	api.HandleFunc("/terminal/execute", RequireAuth(RequireAdmin(ExecuteTerminalHandler))).Methods("POST")

	// Logs routes
	api.HandleFunc("/logs", RequireAuth(GetLogsHandler)).Methods("GET")
	api.HandleFunc("/logs/activity", RequireAuth(GetActivityLogsHandler)).Methods("GET")

	// Frontend routes (serve index.html for SPA routing)
	// In development, frontend is served separately by Vite
	// In production, use nginx or another web server to serve frontend/dist
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve frontend files if dist exists, otherwise redirect to login
		if _, err := os.Stat("./frontend/dist/index.html"); err == nil {
			http.ServeFile(w, r, "./frontend/dist/index.html")
		} else {
			// If frontend not built, serve API info
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"TSO API","version":"2.0","status":"running","frontend":"not built - run 'cd frontend && npm run build'"}`))
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

