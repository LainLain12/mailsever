package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/emersion/go-imap/server"
	"github.com/emersion/go-smtp"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type EmailServer struct {
	db         *sql.DB
	imapServer *server.Server
	smtpServer *smtp.Server
	domains    []string // Available domains for email creation
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Created  string `json:"created"`
}

type Email struct {
	ID      int    `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Date    string `json:"date"`
	Read    bool   `json:"read"`
}

func main() {
	server := &EmailServer{}
	if err := server.Initialize(); err != nil {
		log.Fatal("Failed to initialize server:", err)
	}

	// Start SMTP server
	go func() {
		log.Println("Starting SMTP server on :2525")
		if err := server.smtpServer.ListenAndServe(); err != nil {
			log.Printf("SMTP server error: %v", err)
		}
	}()

	// Start IMAP server
	go func() {
		log.Println("Starting IMAP server on :1143")
		if err := server.imapServer.ListenAndServe(); err != nil {
			log.Printf("IMAP server error: %v", err)
		}
	}()

	// Start web server
	server.StartWebServer()
}

func (s *EmailServer) Initialize() error {
	var err error

	// Initialize available domains
	s.domains = []string{"localhost.com", "emailserver.local", "testmail.dev", "myemail.local"}

	// Initialize database
	s.db, err = sql.Open("sqlite3", "email_server.db")
	if err != nil {
		return err
	}

	if err := s.createTables(); err != nil {
		return err
	}

	// Initialize IMAP server
	imapBackend := NewIMAPBackend(s.db)
	s.imapServer = server.New(imapBackend)
	s.imapServer.Addr = ":1143"
	s.imapServer.AllowInsecureAuth = true

	// Initialize SMTP server
	smtpBackend := NewSMTPBackend(s.db)
	s.smtpServer = smtp.NewServer(smtpBackend)
	s.smtpServer.Addr = ":2525"
	s.smtpServer.Domain = "localhost"
	s.smtpServer.AllowInsecureAuth = true

	return nil
}

func (s *EmailServer) createTables() error {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	emailTable := `
	CREATE TABLE IF NOT EXISTS emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_email TEXT NOT NULL,
		to_email TEXT NOT NULL,
		subject TEXT NOT NULL,
		body TEXT NOT NULL,
		date DATETIME DEFAULT CURRENT_TIMESTAMP,
		read BOOLEAN DEFAULT FALSE
	);`

	if _, err := s.db.Exec(userTable); err != nil {
		return err
	}

	if _, err := s.db.Exec(emailTable); err != nil {
		return err
	}

	return nil
}

func (s *EmailServer) StartWebServer() {
	r := mux.NewRouter()

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Web routes
	r.HandleFunc("/", s.homeHandler).Methods("GET")
	r.HandleFunc("/login", s.loginPageHandler).Methods("GET")
	r.HandleFunc("/login", s.loginHandler).Methods("POST")
	r.HandleFunc("/register", s.registerPageHandler).Methods("GET")
	r.HandleFunc("/register", s.registerHandler).Methods("POST")
	r.HandleFunc("/dashboard", s.dashboardHandler).Methods("GET")
	r.HandleFunc("/emails", s.emailsHandler).Methods("GET")
	r.HandleFunc("/email/{id}", s.emailDetailHandler).Methods("GET")
	r.HandleFunc("/compose", s.composePageHandler).Methods("GET")
	r.HandleFunc("/compose", s.sendEmailHandler).Methods("POST")
	r.HandleFunc("/logout", s.logoutHandler).Methods("POST")
	r.HandleFunc("/api/domains", s.getDomainsHandler).Methods("GET")

	log.Println("Starting web server on :8585")
	log.Fatal(http.ListenAndServe(":8585", r))
}

func (s *EmailServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	domains := s.getAvailableDomains()

	tmpl := template.Must(template.New("home").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Email Server - Modern Email Solution</title>
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
    <link rel="stylesheet" href="/static/style.css">
    <link href="https://fonts.googleapis.com/css2?family=Google+Sans:wght@400;500;600&display=swap" rel="stylesheet">
    <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
</head>
<body>
    <!-- Header -->
    <div class="header">
        <div class="header-content">
            <a href="/" class="logo">
                <span class="material-icons">email</span>
                Email Server
            </a>
            <div class="nav-actions">
                <a href="/login" class="btn btn-secondary">
                    <span class="material-icons">login</span>
                    <span class="hidden-mobile">Sign In</span>
                </a>
                <a href="/register" class="btn btn-primary">
                    <span class="material-icons">person_add</span>
                    <span class="hidden-mobile">Create Account</span>
                </a>
            </div>
        </div>
    </div>

    <!-- Main Content -->
    <div class="container">
        <div style="max-width: 800px; margin: 40px auto;">
            <!-- Hero Section -->
            <div class="card">
                <div class="card-body" style="text-align: center; padding: 40px 20px;">
                    <div style="font-size: 48px; color: var(--primary-color); margin-bottom: 16px;">
                        <span class="material-icons" style="font-size: inherit;">email</span>
                    </div>
                    <h1 style="font-size: 32px; font-weight: 500; margin-bottom: 16px; color: var(--text-primary);">
                        Modern Email Server
                    </h1>
                    <p style="font-size: 18px; color: var(--text-secondary); margin-bottom: 32px; max-width: 600px; margin-left: auto; margin-right: auto;">
                        A clean, modern email solution with Gmail-like interface. Create your account, send emails, and manage your inbox with ease.
                    </p>
                    
                    <div style="display: flex; gap: 16px; justify-content: center; flex-wrap: wrap;">
                        <a href="/register" class="btn btn-primary" style="padding: 12px 24px; font-size: 16px;">
                            <span class="material-icons">person_add</span>
                            Get Started
                        </a>
                        <a href="/login" class="btn btn-secondary" style="padding: 12px 24px; font-size: 16px;">
                            <span class="material-icons">login</span>
                            Sign In
                        </a>
                    </div>
                </div>
            </div>

            <!-- Features Grid -->
            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin: 40px 0;">
                <div class="card">
                    <div class="card-body" style="text-align: center;">
                        <div style="color: var(--primary-color); font-size: 32px; margin-bottom: 16px;">
                            <span class="material-icons" style="font-size: inherit;">smartphone</span>
                        </div>
                        <h3 style="font-weight: 500; margin-bottom: 12px;">Mobile Friendly</h3>
                        <p style="color: var(--text-secondary); font-size: 14px;">
                            Responsive design that works perfectly on all devices, from phones to desktops.
                        </p>
                    </div>
                </div>

                <div class="card">
                    <div class="card-body" style="text-align: center;">
                        <div style="color: var(--success-color); font-size: 32px; margin-bottom: 16px;">
                            <span class="material-icons" style="font-size: inherit;">security</span>
                        </div>
                        <h3 style="font-weight: 500; margin-bottom: 12px;">Secure</h3>
                        <p style="color: var(--text-secondary); font-size: 14px;">
                            Built-in security with encrypted passwords and secure email protocols.
                        </p>
                    </div>
                </div>

                <div class="card">
                    <div class="card-body" style="text-align: center;">
                        <div style="color: var(--warning-color); font-size: 32px; margin-bottom: 16px;">
                            <span class="material-icons" style="font-size: inherit;">speed</span>
                        </div>
                        <h3 style="font-weight: 500; margin-bottom: 12px;">Fast</h3>
                        <p style="color: var(--text-secondary); font-size: 14px;">
                            Lightning-fast email delivery and real-time updates with modern technologies.
                        </p>
                    </div>
                </div>
            </div>

            <!-- Domain Information -->
            <div class="card">
                <div class="card-header">
                    <span class="material-icons" style="vertical-align: middle; margin-right: 8px;">domain</span>
                    Available Email Domains
                </div>
                <div class="card-body">
                    <p style="color: var(--text-secondary); margin-bottom: 16px;">
                        Choose from our available domains when creating your email account:
                    </p>
                    <div style="display: flex; flex-wrap: wrap; gap: 8px;">
                        {{range .}}
                        <span style="background: var(--background-light); padding: 6px 12px; border-radius: 16px; font-size: 14px; color: var(--primary-color); border: 1px solid var(--border-color);">
                            @{{.}}
                        </span>
                        {{end}}
                    </div>
                    <div class="alert alert-info" style="margin-top: 16px;">
                        <strong>Pro Tip:</strong> All domains work with standard email clients like Outlook, Thunderbird, and Apple Mail!
                    </div>
                </div>
            </div>

            <!-- Server Information -->
            <div class="card">
                <div class="card-header">
                    <span class="material-icons" style="vertical-align: middle; margin-right: 8px;">settings</span>
                    Server Configuration
                </div>
                <div class="card-body">
                    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px;">
                        <div>
                            <h4 style="font-weight: 500; margin-bottom: 8px; color: var(--text-primary);">SMTP Server</h4>
                            <p style="font-size: 14px; color: var(--text-secondary); margin: 4px 0;">Host: localhost</p>
                            <p style="font-size: 14px; color: var(--text-secondary); margin: 4px 0;">Port: 2525</p>
                        </div>
                        <div>
                            <h4 style="font-weight: 500; margin-bottom: 8px; color: var(--text-primary);">IMAP Server</h4>
                            <p style="font-size: 14px; color: var(--text-secondary); margin: 4px 0;">Host: localhost</p>
                            <p style="font-size: 14px; color: var(--text-secondary); margin: 4px 0;">Port: 1143</p>
                        </div>
                        <div>
                            <h4 style="font-weight: 500; margin-bottom: 8px; color: var(--text-primary);">Web Interface</h4>
                            <p style="font-size: 14px; color: var(--text-secondary); margin: 4px 0;">Port: 8080</p>
                            <p style="font-size: 14px; color: var(--text-secondary); margin: 4px 0;">Protocol: HTTP</p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Footer -->
    <footer style="background: var(--background-white); border-top: 1px solid var(--border-color); margin-top: 60px; padding: 40px 0;">
        <div class="container" style="text-align: center;">
            <p style="color: var(--text-secondary); font-size: 14px;">
                © 2025 Email Server. Built with Go, HTMX, and modern web technologies.
            </p>
        </div>
    </footer>
</body>
</html>`))

	tmpl.Execute(w, domains)
}

func (s *EmailServer) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sign In - Email Server</title>
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
    <link rel="stylesheet" href="/static/style.css">
    <link href="https://fonts.googleapis.com/css2?family=Google+Sans:wght@400;500;600&display=swap" rel="stylesheet">
    <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
</head>
<body>
    <!-- Header -->
    <div class="header">
        <div class="header-content">
            <a href="/" class="logo">
                <span class="material-icons">email</span>
                Email Server
            </a>
        </div>
    </div>

    <!-- Main Content -->
    <div class="container">
        <div style="max-width: 400px; margin: 40px auto;">
            <div class="card">
                <div class="card-header" style="text-align: center;">
                    <div style="color: var(--primary-color); font-size: 40px; margin-bottom: 8px;">
                        <span class="material-icons" style="font-size: inherit;">account_circle</span>
                    </div>
                    <h1 style="font-size: 24px; font-weight: 500; margin: 0;">Sign in to your account</h1>
                </div>
                <div class="card-body">
                    <form hx-post="/login" hx-target="#message" hx-indicator="#loading">
                        <div class="form-group">
                            <label for="email" class="form-label">
                                <span class="material-icons" style="vertical-align: middle; margin-right: 4px; font-size: 18px;">email</span>
                                Email address
                            </label>
                            <input type="email" id="email" name="email" required class="form-input" 
                                   placeholder="Enter your email address"
                                   autocomplete="email">
                        </div>
                        
                        <div class="form-group">
                            <label for="password" class="form-label">
                                <span class="material-icons" style="vertical-align: middle; margin-right: 4px; font-size: 18px;">lock</span>
                                Password
                            </label>
                            <input type="password" id="password" name="password" required class="form-input" 
                                   placeholder="Enter your password"
                                   autocomplete="current-password">
                        </div>
                        
                        <button type="submit" class="btn btn-primary" style="width: 100%; margin-bottom: 16px;">
                            <span id="loading" class="spinner" style="display: none;"></span>
                            <span class="material-icons">login</span>
                            Sign In
                        </button>
                    </form>
                    
                    <div id="message" class="fade-in"></div>
                    
                    <div style="text-align: center; margin-top: 24px; padding-top: 24px; border-top: 1px solid var(--border-color);">
                        <p style="color: var(--text-secondary); font-size: 14px; margin-bottom: 16px;">
                            Don't have an account?
                        </p>
                        <a href="/register" class="btn btn-secondary" style="width: 100%;">
                            <span class="material-icons">person_add</span>
                            Create Account
                        </a>
                    </div>
                    
                    <div style="text-align: center; margin-top: 16px;">
                        <a href="/" style="color: var(--primary-color); text-decoration: none; font-size: 14px;">
                            <span class="material-icons" style="vertical-align: middle; margin-right: 4px; font-size: 16px;">arrow_back</span>
                            Back to Home
                        </a>
                    </div>
                </div>
            </div>
            
            <!-- Additional Info -->
            <div class="card" style="margin-top: 20px;">
                <div class="card-body" style="text-align: center;">
                    <div class="alert alert-info">
                        <span class="material-icons" style="vertical-align: middle; margin-right: 8px;">info</span>
                        <strong>Demo Server:</strong> Use any email address and password you created during registration.
                    </div>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, tmpl)
}

func (s *EmailServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	var user User
	var hashedPassword string
	err := s.db.QueryRow("SELECT id, username, email, password FROM users WHERE email = ?", email).
		Scan(&user.ID, &user.Username, &user.Email, &hashedPassword)

	if err != nil || bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="alert alert-error">
			<span class="material-icons" style="vertical-align: middle; margin-right: 8px;">error</span>
			Invalid email or password. Please try again.
		</div>`)
		return
	}

	// Set session cookie (simple implementation)
	http.SetCookie(w, &http.Cookie{
		Name:  "user_id",
		Value: strconv.Itoa(user.ID),
		Path:  "/",
	})

	w.Header().Set("HX-Redirect", "/dashboard")
}

func (s *EmailServer) registerPageHandler(w http.ResponseWriter, r *http.Request) {
	domains := s.getAvailableDomains()

	tmpl := template.Must(template.New("register").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Register - Email Server</title>
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <div class="max-w-md mx-auto bg-white rounded-lg shadow-md p-6">
            <h1 class="text-2xl font-bold text-center text-gray-800 mb-6">Create Account</h1>
            <form hx-post="/register" hx-target="#message" class="space-y-4">
                <div>
                    <label for="username" class="block text-sm font-medium text-gray-700">Username</label>
                    <input type="text" id="username" name="username" required
                           class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                           oninput="updateEmailPreview()">
                    <p class="text-xs text-gray-500 mt-1">Choose a username for your email address</p>
                </div>
                <div>
                    <label for="domain" class="block text-sm font-medium text-gray-700">Domain</label>
                    <select id="domain" name="domain" required
                            class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                            onchange="updateEmailPreview()">
                        {{range .}}
                        <option value="{{.}}">{{.}}</option>
                        {{end}}
                    </select>
                    <p class="text-xs text-gray-500 mt-1">Select your email domain</p>
                </div>
                <div>
                    <label class="block text-sm font-medium text-gray-700">Your Email Address</label>
                    <div id="email-preview" class="mt-1 px-3 py-2 bg-gray-50 border border-gray-300 rounded-md text-gray-600">
                        username@domain.com
                    </div>
                    <p class="text-xs text-gray-500 mt-1">This will be your complete email address</p>
                </div>
                <div>
                    <label for="password" class="block text-sm font-medium text-gray-700">Password</label>
                    <input type="password" id="password" name="password" required
                           class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500">
                </div>
                <button type="submit"
                        class="w-full bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded transition duration-200">
                    Create Account
                </button>
            </form>
            <div id="message" class="mt-4"></div>
            <div class="mt-4 text-center">
                <a href="/" class="text-blue-500 hover:text-blue-600">Back to Home</a> |
                <a href="/login" class="text-blue-500 hover:text-blue-600">Login</a>
            </div>
        </div>
    </div>
    
    <script>
        function updateEmailPreview() {
            const username = document.getElementById('username').value.toLowerCase().trim();
            const domain = document.getElementById('domain').value;
            const preview = document.getElementById('email-preview');
            
            if (username && domain) {
                preview.textContent = username + '@' + domain;
                preview.classList.remove('text-gray-600');
                preview.classList.add('text-blue-600', 'font-medium');
            } else {
                preview.textContent = 'username@domain.com';
                preview.classList.remove('text-blue-600', 'font-medium');
                preview.classList.add('text-gray-600');
            }
        }
        
        // Initialize preview
        document.addEventListener('DOMContentLoaded', function() {
            updateEmailPreview();
        });
    </script>
</body>
</html>`))

	tmpl.Execute(w, domains)
}

func (s *EmailServer) registerHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(strings.ToLower(r.FormValue("username")))
	domain := r.FormValue("domain")
	password := r.FormValue("password")

	// Validate inputs
	if username == "" || domain == "" || password == "" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">All fields are required</div>`)
		return
	}

	// Validate domain is in allowed list
	validDomain := false
	for _, allowedDomain := range s.domains {
		if domain == allowedDomain {
			validDomain = true
			break
		}
	}
	if !validDomain {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">Invalid domain selected</div>`)
		return
	}

	// Create full email address
	email := s.createEmailAddress(username, domain)

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">Error creating account</div>`)
		return
	}

	// Insert user
	_, err = s.db.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, string(hashedPassword))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			fmt.Fprint(w, `<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">Email address already exists. Please choose a different username.</div>`)
		} else {
			fmt.Fprint(w, `<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">Error creating account</div>`)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded">
		Account created successfully!<br>
		<strong>Email:</strong> %s<br>
		<a href="/login" class="underline">Login here</a>
	</div>`, email)
}

func (s *EmailServer) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var user User
	err := s.db.QueryRow("SELECT id, username, email FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tmpl := template.Must(template.New("dashboard").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Username}} - Email Server</title>
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
    <link rel="stylesheet" href="/static/style.css">
    <link href="https://fonts.googleapis.com/css2?family=Google+Sans:wght@400;500;600&display=swap" rel="stylesheet">
    <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
</head>
<body>
    <!-- Header -->
    <div class="header">
        <div class="header-content">
            <div style="display: flex; align-items: center; gap: 16px;">
                <button class="mobile-menu-toggle" onclick="toggleSidebar()">
                    <span class="material-icons">menu</span>
                </button>
                <a href="/dashboard" class="logo">
                    <span class="material-icons">email</span>
                    Email
                </a>
                
                <!-- Search Bar -->
                <div class="search-container hidden-mobile">
                    <span class="material-icons search-icon">search</span>
                    <input type="text" class="search-input" placeholder="Search mail" 
                           hx-get="/search" hx-target="#content" hx-trigger="keyup changed delay:300ms">
                </div>
            </div>
            
            <div class="user-info">
                <span class="hidden-mobile">{{.Email}}</span>
                <form hx-post="/logout" class="inline">
                    <button type="submit" class="btn btn-danger">
                        <span class="material-icons">logout</span>
                        <span class="hidden-mobile">Logout</span>
                    </button>
                </form>
            </div>
        </div>
    </div>

    <!-- Main Layout -->
    <div class="dashboard-layout container">
        <!-- Sidebar -->
        <div class="sidebar" id="sidebar">
            <a href="/compose" class="btn btn-primary" style="width: 100%; margin-bottom: 20px;">
                <span class="material-icons">edit</span>
                Compose
            </a>
            
            <nav>
                <a href="#" class="sidebar-item active" hx-get="/emails" hx-target="#content">
                    <span class="material-icons">inbox</span>
                    Inbox
                </a>
                <a href="#" class="sidebar-item" hx-get="/emails?type=starred" hx-target="#content">
                    <span class="material-icons">star</span>
                    Starred
                </a>
                <a href="#" class="sidebar-item" hx-get="/emails?type=sent" hx-target="#content">
                    <span class="material-icons">send</span>
                    Sent
                </a>
                <a href="#" class="sidebar-item" hx-get="/emails?type=drafts" hx-target="#content">
                    <span class="material-icons">drafts</span>
                    Drafts
                </a>
                <a href="#" class="sidebar-item" hx-get="/emails?type=spam" hx-target="#content">
                    <span class="material-icons">report</span>
                    Spam
                </a>
                <a href="#" class="sidebar-item" hx-get="/emails?type=trash" hx-target="#content">
                    <span class="material-icons">delete</span>
                    Trash
                </a>
            </nav>
            
            <!-- Account Info -->
            <div style="margin-top: 20px; padding-top: 20px; border-top: 1px solid var(--border-color);">
                <div class="alert alert-info">
                    <h4 style="font-weight: 500; margin-bottom: 8px;">
                        <span class="material-icons" style="vertical-align: middle; margin-right: 4px;">settings</span>
                        Email Client Setup
                    </h4>
                    <p style="font-size: 12px; margin-bottom: 4px;"><strong>SMTP:</strong> localhost:2525</p>
                    <p style="font-size: 12px; margin-bottom: 4px;"><strong>IMAP:</strong> localhost:1143</p>
                    <p style="font-size: 12px;"><strong>User:</strong> {{.Email}}</p>
                </div>
            </div>
        </div>

        <!-- Main Content -->
        <div class="main-content">
            <div id="content" class="fade-in" hx-get="/emails" hx-trigger="load">
                <!-- Welcome Content -->
                <div class="card">
                    <div class="card-body" style="text-align: center; padding: 60px 20px;">
                        <div style="color: var(--primary-color); font-size: 64px; margin-bottom: 20px;">
                            <span class="material-icons" style="font-size: inherit;">inbox</span>
                        </div>
                        <h2 style="font-size: 24px; font-weight: 500; margin-bottom: 12px;">
                            Welcome to your inbox, {{.Username}}!
                        </h2>
                        <p style="color: var(--text-secondary); margin-bottom: 24px;">
                            Your emails will appear here. Click "View Emails" to see your messages.
                        </p>
                        <button hx-get="/emails" hx-target="#content" class="btn btn-primary">
                            <span class="material-icons">refresh</span>
                            Load Inbox
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Floating Action Button (Mobile) -->
    <a href="/compose" class="fab">
        <span class="material-icons">edit</span>
    </a>

    <script>
        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            sidebar.style.display = sidebar.style.display === 'none' ? 'block' : 'none';
        }
        
        // Close sidebar on mobile when clicking outside
        document.addEventListener('click', function(event) {
            const sidebar = document.getElementById('sidebar');
            const toggle = document.querySelector('.mobile-menu-toggle');
            
            if (window.innerWidth < 768 && !sidebar.contains(event.target) && !toggle.contains(event.target)) {
                sidebar.style.display = 'none';
            }
        });
        
        // Auto-refresh emails every 30 seconds
        setInterval(function() {
            if (document.querySelector('.sidebar-item.active')?.textContent.includes('Inbox')) {
                htmx.trigger('#content', 'refresh');
            }
        }, 30000);
    </script>
</body>
</html>`))

	tmpl.Execute(w, user)
}

func (s *EmailServer) emailsHandler(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	if userID == 0 {
		return
	}

	var user User
	s.db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&user.Email)

	rows, err := s.db.Query("SELECT id, from_email, to_email, subject, body, date, read FROM emails WHERE to_email = ? ORDER BY date DESC", user.Email)
	if err != nil {
		fmt.Fprint(w, "Error loading emails")
		return
	}
	defer rows.Close()

	var emails []Email
	for rows.Next() {
		var email Email
		rows.Scan(&email.ID, &email.From, &email.To, &email.Subject, &email.Body, &email.Date, &email.Read)
		emails = append(emails, email)
	}

	tmpl := template.Must(template.New("emails").Parse(`
<div class="email-list">
    <div style="display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border-color); background: var(--background-light);">
        <h2 style="font-size: 20px; font-weight: 500; margin: 0; display: flex; align-items: center; gap: 8px;">
            <span class="material-icons">inbox</span>
            Inbox
        </h2>
        <div style="display: flex; gap: 8px;">
            <button class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px;" 
                    hx-get="/emails" hx-target="#content">
                <span class="material-icons" style="font-size: 16px;">refresh</span>
                Refresh
            </button>
        </div>
    </div>
    
    {{range .}}
    <div class="email-item {{if not .Read}}unread{{end}}" 
         hx-get="/email/{{.ID}}" hx-target="#content">
        
        <input type="checkbox" class="email-checkbox">
        
        <span class="email-star material-icons">star_border</span>
        
        <div class="email-content">
            <div class="email-header">
                <div class="email-from">{{.From}}</div>
                <div class="email-date">{{.Date}}</div>
                {{if not .Read}}
                <span class="unread-badge">New</span>
                {{end}}
            </div>
            <div class="email-subject">{{if .Subject}}{{.Subject}}{{else}}(No Subject){{end}}</div>
            <div class="email-preview">{{printf "%.100s" .Body}}{{if gt (len .Body) 100}}...{{end}}</div>
        </div>
    </div>
    {{else}}
    <div style="text-align: center; padding: 60px 20px; color: var(--text-secondary);">
        <div style="font-size: 48px; margin-bottom: 16px; color: var(--border-color);">
            <span class="material-icons" style="font-size: inherit;">inbox</span>
        </div>
        <h3 style="font-weight: 500; margin-bottom: 8px;">Your inbox is empty</h3>
        <p style="font-size: 14px;">No emails found. Send yourself a test email to get started!</p>
        <a href="/compose" class="btn btn-primary" style="margin-top: 16px;">
            <span class="material-icons">edit</span>
            Compose Email
        </a>
    </div>
    {{end}}
</div>

<script>
    // Add click handlers for email actions
    document.querySelectorAll('.email-star').forEach(star => {
        star.addEventListener('click', function(e) {
            e.stopPropagation();
            this.textContent = this.textContent === 'star_border' ? 'star' : 'star_border';
            this.classList.toggle('starred');
        });
    });
    
    document.querySelectorAll('.email-checkbox').forEach(checkbox => {
        checkbox.addEventListener('click', function(e) {
            e.stopPropagation();
        });
    });
</script>`))

	tmpl.Execute(w, emails)
}

func (s *EmailServer) emailDetailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["id"]

	userID := s.getUserID(r)
	if userID == 0 {
		return
	}

	var user User
	s.db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&user.Email)

	var email Email
	err := s.db.QueryRow("SELECT id, from_email, to_email, subject, body, date FROM emails WHERE id = ? AND to_email = ?",
		emailID, user.Email).Scan(&email.ID, &email.From, &email.To, &email.Subject, &email.Body, &email.Date)
	if err != nil {
		fmt.Fprint(w, "Email not found")
		return
	}

	// Mark as read
	s.db.Exec("UPDATE emails SET read = TRUE WHERE id = ?", emailID)

	tmpl := template.Must(template.New("email").Parse(`
<div class="border-b pb-4 mb-4">
    <button hx-get="/emails" hx-target="#content" class="text-blue-500 hover:text-blue-600 mb-2">
        ← Back to Emails
    </button>
    <h2 class="text-xl font-bold">{{.Subject}}</h2>
    <div class="text-sm text-gray-600 mt-2">
        <strong>From:</strong> {{.From}}<br>
        <strong>To:</strong> {{.To}}<br>
        <strong>Date:</strong> {{.Date}}
    </div>
</div>
<div class="prose max-w-none">
    <div class="whitespace-pre-wrap">{{.Body}}</div>
</div>`))

	tmpl.Execute(w, email)
}

func (s *EmailServer) composePageHandler(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var user User
	s.db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&user.Email)

	tmpl := template.Must(template.New("compose").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Compose - Email Server</title>
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
    <link rel="stylesheet" href="/static/style.css">
    <link href="https://fonts.googleapis.com/css2?family=Google+Sans:wght@400;500;600&display=swap" rel="stylesheet">
    <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
</head>
<body>
    <!-- Header -->
    <div class="header">
        <div class="header-content">
            <a href="/dashboard" class="logo">
                <span class="material-icons">email</span>
                Email
            </a>
            <div class="user-info">
                <span>{{.Email}}</span>
                <a href="/dashboard" class="btn btn-secondary">
                    <span class="material-icons">arrow_back</span>
                    <span class="hidden-mobile">Back to Inbox</span>
                </a>
            </div>
        </div>
    </div>

    <!-- Main Content -->
    <div class="container compose-container">
        <div class="compose-form" style="margin-top: 20px;">
            <div class="compose-header">
                <div style="display: flex; align-items: center; justify-content: space-between;">
                    <h1 style="margin: 0; display: flex; align-items: center; gap: 8px;">
                        <span class="material-icons">edit</span>
                        New Message
                    </h1>
                    <button onclick="window.location.href='/dashboard'" class="btn btn-secondary" style="padding: 6px 12px;">
                        <span class="material-icons">close</span>
                    </button>
                </div>
            </div>
            
            <div class="compose-body">
                <form hx-post="/compose" hx-target="#message" hx-indicator="#sending" class="space-y-4">
                    <div class="form-group">
                        <div style="display: flex; align-items: center; border-bottom: 1px solid var(--border-color); padding-bottom: 12px;">
                            <label style="min-width: 60px; color: var(--text-secondary); font-size: 14px;">From:</label>
                            <span style="color: var(--text-primary);">{{.Email}}</span>
                        </div>
                    </div>
                    
                    <div class="form-group">
                        <div style="display: flex; align-items: center; border-bottom: 1px solid var(--border-color); padding-bottom: 12px;">
                            <label for="to" style="min-width: 60px; color: var(--text-secondary); font-size: 14px;">To:</label>
                            <input type="email" id="to" name="to" required 
                                   style="flex: 1; border: none; outline: none; font-size: 14px; padding: 4px 0;"
                                   placeholder="Recipients">
                        </div>
                    </div>
                    
                    <div class="form-group">
                        <div style="display: flex; align-items: center; border-bottom: 1px solid var(--border-color); padding-bottom: 12px;">
                            <label for="subject" style="min-width: 60px; color: var(--text-secondary); font-size: 14px;">Subject:</label>
                            <input type="text" id="subject" name="subject" required 
                                   style="flex: 1; border: none; outline: none; font-size: 14px; padding: 4px 0;"
                                   placeholder="Subject">
                        </div>
                    </div>
                    
                    <div class="form-group" style="margin-top: 20px;">
                        <textarea id="body" name="body" required class="compose-textarea"
                                  placeholder="Compose your message..."></textarea>
                    </div>
                    
                    <div style="display: flex; align-items: center; justify-content: space-between; padding-top: 16px; border-top: 1px solid var(--border-color);">
                        <button type="submit" class="btn btn-primary">
                            <span id="sending" class="spinner" style="display: none;"></span>
                            <span class="material-icons">send</span>
                            Send
                        </button>
                        
                        <div style="display: flex; gap: 8px;">
                            <button type="button" class="btn btn-secondary" onclick="formatText('bold')">
                                <span class="material-icons">format_bold</span>
                            </button>
                            <button type="button" class="btn btn-secondary" onclick="formatText('italic')">
                                <span class="material-icons">format_italic</span>
                            </button>
                            <button type="button" class="btn btn-secondary" onclick="attachFile()">
                                <span class="material-icons">attach_file</span>
                            </button>
                            <button type="button" class="btn btn-secondary" onclick="insertEmoji()">
                                <span class="material-icons">emoji_emotions</span>
                            </button>
                        </div>
                    </div>
                </form>
                
                <div id="message" class="fade-in" style="margin-top: 16px;"></div>
            </div>
        </div>
        
        <!-- Quick Tips -->
        <div class="card" style="margin-top: 20px;">
            <div class="card-header">
                <span class="material-icons" style="vertical-align: middle; margin-right: 8px;">lightbulb</span>
                Quick Tips
            </div>
            <div class="card-body">
                <ul style="margin: 0; padding-left: 20px; color: var(--text-secondary); font-size: 14px;">
                    <li>Use Tab to quickly move between fields</li>
                    <li>Press Ctrl+Enter to send your message</li>
                    <li>Your emails are stored locally in the server database</li>
                    <li>Recipients must have accounts on this server to receive emails</li>
                </ul>
            </div>
        </div>
    </div>

    <script>
        // Keyboard shortcuts
        document.addEventListener('keydown', function(e) {
            if (e.ctrlKey && e.key === 'Enter') {
                document.querySelector('form').dispatchEvent(new Event('submit'));
            }
        });
        
        // Auto-save draft (placeholder)
        let draftTimeout;
        document.getElementById('body').addEventListener('input', function() {
            clearTimeout(draftTimeout);
            draftTimeout = setTimeout(function() {
                console.log('Auto-saving draft...');
            }, 2000);
        });
        
        // Format text functions (placeholder)
        function formatText(format) {
            console.log('Format:', format);
        }
        
        function attachFile() {
            console.log('Attach file');
        }
        
        function insertEmoji() {
            const textarea = document.getElementById('body');
            const emojis = ['😊', '👍', '❤️', '😂', '🎉', '✨', '🚀', '💡'];
            const emoji = emojis[Math.floor(Math.random() * emojis.length)];
            textarea.value += emoji;
            textarea.focus();
        }
    </script>
</body>
</html>`))

	tmpl.Execute(w, struct{ Email string }{Email: user.Email})
}

func (s *EmailServer) sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserID(r)
	if userID == 0 {
		return
	}

	var user User
	s.db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&user.Email)

	to := r.FormValue("to")
	subject := r.FormValue("subject")
	body := r.FormValue("body")

	// Store email in database
	_, err := s.db.Exec("INSERT INTO emails (from_email, to_email, subject, body) VALUES (?, ?, ?, ?)",
		user.Email, to, subject, body)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="alert alert-error">
			<span class="material-icons" style="vertical-align: middle; margin-right: 8px;">error</span>
			Error sending email. Please try again.
		</div>`)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<div class="alert alert-success">
		<span class="material-icons" style="vertical-align: middle; margin-right: 8px;">check_circle</span>
		Email sent successfully! 
		<a href="/dashboard" style="color: var(--success-color); text-decoration: underline; margin-left: 8px;">Back to Dashboard</a>
	</div>`)
}

func (s *EmailServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "user_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.Header().Set("HX-Redirect", "/")
}

func (s *EmailServer) getUserID(r *http.Request) int {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return 0
	}
	userID, err := strconv.Atoi(cookie.Value)
	if err != nil {
		return 0
	}
	return userID
}

// Helper function to get available domains
func (s *EmailServer) getAvailableDomains() []string {
	return s.domains
}

// Helper function to create full email from username and domain
func (s *EmailServer) createEmailAddress(username, domain string) string {
	return strings.ToLower(username) + "@" + domain
}

// API handler to get available domains
func (s *EmailServer) getDomainsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	domains := s.getAvailableDomains()

	result := `{"domains":["` + strings.Join(domains, `","`) + `"]}`
	fmt.Fprint(w, result)
}
