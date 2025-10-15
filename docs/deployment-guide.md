# Deployment Guide

This comprehensive guide covers deploying your Gojang application to production environments including Docker, VPS servers, and cloud platforms.

> 💡 **Looking for distributed deployment and load balancing?** See the [Distributed Deployment Guide](./distributed-deployment.md) for horizontal scaling, load balancing, and high-availability setups.

## Overview

Gojang applications can be deployed to:
- 🐳 **Docker** - Containerized deployments
- 🖥️ **VPS** - Traditional server deployments (DigitalOcean, Linode, etc.)
- ☁️ **Cloud Platforms** - AWS, Google Cloud, Azure, Fly.io
- 🚀 **PaaS** - Heroku, Railway, Render

**What you'll learn:**
- Building production-ready binaries
- Docker containerization
- Environment configuration
- Database setup (SQLite vs PostgreSQL)
- Reverse proxy configuration (Nginx)
- SSL/TLS setup
- Process management
- Monitoring and logging

---

## Pre-Deployment Checklist

Before deploying, ensure:

- [ ] All tests pass: `go test ./...`
- [ ] Code is formatted: `go fmt ./...`
- [ ] No linting errors: `go vet ./...`
- [ ] Environment variables are configured
- [ ] Secrets are not committed to git
- [ ] Database migrations are ready
- [ ] Static assets are compiled
- [ ] HTTPS/SSL certificates are obtained
- [ ] Backup strategy is planned
- [ ] Monitoring is set up

---

## Building for Production

### Build Optimized Binary

```bash
# Build for Linux (most common server OS)
GOOS=linux GOARCH=amd64 go build -o gojang-app \
    -ldflags="-s -w" \
    ./gojang/cmd/web

# Explanation:
# -ldflags="-s -w" removes debug info (smaller binary)
# GOOS=linux for Linux servers
# GOARCH=amd64 for 64-bit processors
```

**Build flags:**
- `-s` - Strip debug information
- `-w` - Omit DWARF symbol table
- Results in ~30-50% smaller binary

### Cross-Platform Builds

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o gojang-linux ./gojang/cmd/web

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o gojang-macos ./gojang/cmd/web

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o gojang-windows.exe ./gojang/cmd/web

# Build for ARM (Raspberry Pi, etc.)
GOOS=linux GOARCH=arm64 go build -o gojang-arm ./gojang/cmd/web
```

### Using Task for Builds

```bash
# Use the provided Taskfile
task build

# This runs:
# go build -o app -ldflags="-s -w" ./gojang/cmd/web
```

---

## Environment Configuration

### Production Environment Variables

Create `.env` file (never commit to git!):

```bash
# Server
PORT=8080
DEBUG=false
ALLOWED_HOSTS=yourdomain.com,www.yourdomain.com

# Database - PostgreSQL recommended for production
DATABASE_URL=postgres://username:password@localhost:5432/gojang?sslmode=require

# Session secret - generate with: openssl rand -base64 32
SESSION_KEY=your-random-32-byte-string-here

# CSRF secret - generate with: openssl rand -base64 32
CSRF_SECRET=your-random-csrf-secret-here

# SMTP for emails
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASS=your-sendgrid-api-key
SMTP_FROM=noreply@yourdomain.com

# Optional: Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Generate Secure Secrets

```bash
# Generate SESSION_KEY
openssl rand -base64 32

# Generate CSRF_SECRET
openssl rand -base64 32

# Or using Go
go run -c 'import "crypto/rand"; import "encoding/base64"; b := make([]byte, 32); rand.Read(b); print(base64.StdEncoding.EncodeToString(b))'
```

### Environment-Specific Config

```go
// gojang/config/config.go
func Load() (*Config, error) {
    cfg := &Config{
        Port:  getEnv("PORT", "8080"),
        Debug: getEnvBool("DEBUG", false),
        
        // Production defaults
        DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/gojang"),
        SessionKey:  getEnv("SESSION_KEY", ""),
        
        // Security
        AllowedHosts: strings.Split(getEnv("ALLOWED_HOSTS", ""), ","),
    }
    
    // Validate required fields in production
    if !cfg.Debug {
        if cfg.SessionKey == "" {
            return nil, errors.New("SESSION_KEY required in production")
        }
        if cfg.DatabaseURL == "" {
            return nil, errors.New("DATABASE_URL required")
        }
    }
    
    return cfg, nil
}
```

---

## Docker Deployment

### Dockerfile

Create `Dockerfile` in project root:

```dockerfile
# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gojang-app \
    -ldflags="-s -w" \
    ./gojang/cmd/web

# Stage 2: Run
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S gojang && \
    adduser -u 1001 -S gojang -G gojang

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gojang-app .

# Copy templates and static files
COPY --from=builder /app/gojang/views ./gojang/views

# Change ownership
RUN chown -R gojang:gojang /app

# Switch to non-root user
USER gojang

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run
CMD ["./gojang-app"]
```

**Dockerfile best practices:**
- ✅ Multi-stage build (smaller image)
- ✅ Run as non-root user (security)
- ✅ Health checks (monitoring)
- ✅ Alpine base (minimal size)
- ✅ Static binary (no CGO dependencies)

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  app:
    build: .
    container_name: gojang-app
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DEBUG=false
      - DATABASE_URL=postgres://gojang:password@db:5432/gojang?sslmode=disable
      - SESSION_KEY=${SESSION_KEY}
      - CSRF_SECRET=${CSRF_SECRET}
    depends_on:
      db:
        condition: service_healthy
    networks:
      - gojang-network
    volumes:
      - ./data:/app/data

  db:
    image: postgres:15-alpine
    container_name: gojang-db
    restart: unless-stopped
    environment:
      - POSTGRES_DB=gojang
      - POSTGRES_USER=gojang
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - gojang-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U gojang"]
      interval: 10s
      timeout: 5s
      retries: 5

  nginx:
    image: nginx:alpine
    container_name: gojang-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - app
    networks:
      - gojang-network

networks:
  gojang-network:
    driver: bridge

volumes:
  postgres-data:
```

### Build and Run with Docker

```bash
# Build image
docker build -t gojang-app .

# Run container
docker run -d \
  --name gojang \
  -p 8080:8080 \
  -e DATABASE_URL="sqlite:///app/data/app.db" \
  -e SESSION_KEY="your-secret-key" \
  -v $(pwd)/data:/app/data \
  gojang-app

# Or use docker-compose
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop
docker-compose down

# Rebuild and restart
docker-compose up -d --build
```

### .dockerignore

Create `.dockerignore`:

```
# Git
.git
.gitignore

# Development
.env
.env.local
*.log
tmp/
dev.db

# Build artifacts
*.exe
*.dll
*.so
*.dylib
app
web

# Dependencies
vendor/

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Documentation
README.md
docs/

# Tests
*_test.go
```

---

## VPS Deployment

### Server Setup (Ubuntu/Debian)

#### 1. Initial Server Configuration

```bash
# SSH into server
ssh root@your-server-ip

# Update system
apt update && apt upgrade -y

# Create application user
adduser gojang
usermod -aG sudo gojang

# Switch to new user
su - gojang
```

#### 2. Install Dependencies

```bash
# Install required packages
sudo apt install -y nginx postgresql postgresql-contrib certbot python3-certbot-nginx

# Install Go (if building on server)
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify Go installation
go version
```

#### 3. Setup PostgreSQL

```bash
# Switch to postgres user
sudo -i -u postgres

# Create database and user
createdb gojang
createuser gojang
psql -c "ALTER USER gojang WITH PASSWORD 'your-secure-password';"
psql -c "GRANT ALL PRIVILEGES ON DATABASE gojang TO gojang;"

# Exit postgres user
exit
```

#### 4. Deploy Application

```bash
# Create application directory
sudo mkdir -p /opt/gojang
sudo chown gojang:gojang /opt/gojang
cd /opt/gojang

# Upload binary (from local machine)
# scp gojang-app gojang@your-server-ip:/opt/gojang/

# Or clone and build on server
git clone https://github.com/yourusername/your-gojang-app.git .
go build -o gojang-app ./gojang/cmd/web

# Create .env file
nano .env
# Add production environment variables
# Save and exit (Ctrl+X, Y, Enter)

# Make binary executable
chmod +x gojang-app

# Test run
./gojang-app
# Press Ctrl+C to stop
```

#### 5. Setup Systemd Service

Create `/etc/systemd/system/gojang.service`:

```bash
sudo nano /etc/systemd/system/gojang.service
```

```ini
[Unit]
Description=Gojang Web Application
After=network.target postgresql.service

[Service]
Type=simple
User=gojang
Group=gojang
WorkingDirectory=/opt/gojang
ExecStart=/opt/gojang/gojang-app
Restart=always
RestartSec=5
StandardOutput=append:/var/log/gojang/app.log
StandardError=append:/var/log/gojang/error.log

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/gojang/data
ProtectHome=true

# Environment
Environment="PORT=8080"
EnvironmentFile=/opt/gojang/.env

[Install]
WantedBy=multi-user.target
```

Start the service:

```bash
# Create log directory
sudo mkdir -p /var/log/gojang
sudo chown gojang:gojang /var/log/gojang

# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable gojang

# Start service
sudo systemctl start gojang

# Check status
sudo systemctl status gojang

# View logs
sudo journalctl -u gojang -f
```

#### 6. Configure Nginx as Reverse Proxy

Create `/etc/nginx/sites-available/gojang`:

```bash
sudo nano /etc/nginx/sites-available/gojang
```

```nginx
# Rate limiting
limit_req_zone $binary_remote_addr zone=app_limit:10m rate=10r/s;

# Upstream
upstream gojang_backend {
    server 127.0.0.1:8080;
}

# HTTP - redirect to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name yourdomain.com www.yourdomain.com;
    
    # Let's Encrypt challenge
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    
    # Redirect to HTTPS
    location / {
        return 301 https://$server_name$request_uri;
    }
}

# HTTPS
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name yourdomain.com www.yourdomain.com;
    
    # SSL certificates (will be set by certbot)
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    ssl_trusted_certificate /etc/letsencrypt/live/yourdomain.com/chain.pem;
    
    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # Logging
    access_log /var/log/nginx/gojang_access.log;
    error_log /var/log/nginx/gojang_error.log;
    
    # Max upload size
    client_max_body_size 10M;
    
    # Proxy settings
    location / {
        # Rate limiting
        limit_req zone=app_limit burst=20 nodelay;
        
        proxy_pass http://gojang_backend;
        proxy_http_version 1.1;
        
        # Headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # Buffering
        proxy_buffering off;
    }
    
    # Static files (optional - if serving directly)
    location /static/ {
        alias /opt/gojang/gojang/views/static/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

Enable site:

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/gojang /etc/nginx/sites-enabled/

# Remove default site
sudo rm /etc/nginx/sites-enabled/default

# Test configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx
```

#### 7. Setup SSL with Let's Encrypt

```bash
# Obtain certificate
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Follow prompts:
# - Enter email
# - Agree to terms
# - Choose redirect option (2)

# Test renewal
sudo certbot renew --dry-run

# Auto-renewal is set up via cron/systemd timer
```

#### 8. Firewall Setup

```bash
# Enable UFW
sudo ufw enable

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Check status
sudo ufw status
```

---

## Cloud Platform Deployments

### Fly.io

**1. Install flyctl:**
```bash
curl -L https://fly.io/install.sh | sh
```

**2. Login:**
```bash
fly auth login
```

**3. Create `fly.toml`:**
```toml
app = "your-app-name"
primary_region = "sjc"

[build]
  dockerfile = "Dockerfile"

[env]
  PORT = "8080"
  DEBUG = "false"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 1

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 256
```

**4. Deploy:**
```bash
# Create app
fly launch

# Set secrets
fly secrets set SESSION_KEY="your-secret"
fly secrets set CSRF_SECRET="your-secret"
fly secrets set DATABASE_URL="your-postgres-url"

# Deploy
fly deploy

# Open in browser
fly open
```

### Railway

**1. Install Railway CLI:**
```bash
npm install -g @railway/cli
```

**2. Login:**
```bash
railway login
```

**3. Initialize:**
```bash
railway init
```

**4. Set environment variables:**
```bash
railway variables set SESSION_KEY="your-secret"
railway variables set DATABASE_URL="your-postgres-url"
```

**5. Deploy:**
```bash
railway up
```

### Heroku

**1. Install Heroku CLI:**
```bash
curl https://cli-assets.heroku.com/install.sh | sh
```

**2. Login:**
```bash
heroku login
```

**3. Create app:**
```bash
heroku create your-app-name
```

**4. Add PostgreSQL:**
```bash
heroku addons:create heroku-postgresql:mini
```

**5. Set environment variables:**
```bash
heroku config:set SESSION_KEY="your-secret"
heroku config:set CSRF_SECRET="your-secret"
```

**6. Create `Procfile`:**
```
web: ./gojang-app
```

**7. Deploy:**
```bash
git push heroku main
```

### DigitalOcean App Platform

**1. Create `app.yaml`:**
```yaml
name: gojang-app
services:
- name: web
  github:
    repo: your-username/your-repo
    branch: main
    deploy_on_push: true
  build_command: go build -o app ./gojang/cmd/web
  run_command: ./app
  environment_slug: go
  instance_count: 1
  instance_size_slug: basic-xxs
  http_port: 8080
  envs:
  - key: PORT
    value: "8080"
  - key: DEBUG
    value: "false"
  - key: SESSION_KEY
    type: SECRET
    value: your-secret
  - key: DATABASE_URL
    type: SECRET
    value: ${db.DATABASE_URL}

databases:
- name: db
  engine: PG
  production: true
  version: "15"
```

**2. Deploy via UI or CLI:**
```bash
doctl apps create --spec app.yaml
```

---

## Database Migration Strategies

### Development to Production

**Option 1: Auto-migrate on startup (simple)**

```go
// In main.go
if err := client.Schema.Create(ctx); err != nil {
    log.Fatal(err)
}
```

**Option 2: Manual migration (recommended for production)**

```bash
# Generate migration files
cd gojang/models
go run -mod=mod entgo.io/ent/cmd/ent generate ./schema

# Create migration SQL
go run main.go migrate --dry-run > migration.sql

# Review migration.sql
# Apply manually to production database
psql -U gojang -d gojang -f migration.sql
```

**Option 3: Use Atlas (Ent's migration tool)**

```bash
# Install Atlas
curl -sSf https://atlasgo.sh | sh

# Generate migration
atlas migrate diff initial \
  --to "ent://gojang/models/schema" \
  --dev-url "sqlite://dev.db"

# Apply migration
atlas migrate apply \
  --url "postgres://user:pass@localhost:5432/gojang"
```

### Backup Before Migration

```bash
# PostgreSQL backup
pg_dump -U gojang gojang > backup.sql

# Restore if needed
psql -U gojang gojang < backup.sql

# SQLite backup
sqlite3 app.db ".backup backup.db"
```

---

## Monitoring & Logging

### Application Logging

Add structured logging:

```go
import "log/slog"

// Configure logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Log events
logger.Info("server started", "port", cfg.Port)
logger.Error("database error", "error", err)
```

### System Monitoring

**Check application status:**
```bash
# Service status
sudo systemctl status gojang

# View logs
sudo journalctl -u gojang -f

# View last 100 lines
sudo journalctl -u gojang -n 100
```

**Monitor resources:**
```bash
# CPU and memory usage
htop

# Disk usage
df -h

# Network connections
netstat -tulpn | grep :8080
```

### External Monitoring

**Uptime monitoring:**
- UptimeRobot - https://uptimerobot.com/
- Pingdom - https://www.pingdom.com/
- StatusCake - https://www.statuscake.com/

**Application monitoring:**
- Sentry - Error tracking
- New Relic - Performance monitoring
- DataDog - Full-stack monitoring

---

## Performance Optimization

### Go Application

```go
// Enable HTTP/2
srv := &http.Server{
    Addr:    ":8080",
    Handler: router,
    // Timeouts
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
    // HTTP/2
    TLSConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,
    },
}
```

### Database Optimization

```go
// Connection pooling
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)

// Use indexes
field.String("email").Unique().Index()

// Eager loading (avoid N+1)
posts, err := client.Post.Query().
    WithAuthor().
    All(ctx)
```

### Nginx Caching

```nginx
# Enable gzip compression
gzip on;
gzip_vary on;
gzip_types text/plain text/css text/xml text/javascript application/javascript application/json;

# Browser caching for static assets
location /static/ {
    expires 1y;
    add_header Cache-Control "public, immutable";
}
```

---

## Security Checklist

Production security essentials:

- [ ] **HTTPS enabled** with valid SSL certificate
- [ ] **Secrets** stored in environment variables (not code)
- [ ] **DEBUG mode** disabled in production
- [ ] **CSRF protection** enabled
- [ ] **Rate limiting** configured (Nginx or middleware)
- [ ] **SQL injection** prevention (use ORM, parameterized queries)
- [ ] **XSS protection** (sanitize input, escape output)
- [ ] **Secure cookies** (HttpOnly, Secure, SameSite)
- [ ] **Strong passwords** enforced (bcrypt with cost 10+)
- [ ] **Firewall** configured (allow only 22, 80, 443)
- [ ] **Regular backups** scheduled
- [ ] **Updates** automated (unattended-upgrades)
- [ ] **Monitoring** and alerting set up
- [ ] **Error pages** don't expose sensitive info
- [ ] **Dependencies** regularly updated

---

## Backup Strategy

### Database Backups

**Automated PostgreSQL backups:**

Create `/opt/scripts/backup-db.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/opt/backups"
DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="gojang_backup_$DATE.sql"

# Create backup directory
mkdir -p $BACKUP_DIR

# Dump database
pg_dump -U gojang gojang > $BACKUP_DIR/$FILENAME

# Compress
gzip $BACKUP_DIR/$FILENAME

# Keep only last 7 days
find $BACKUP_DIR -name "gojang_backup_*.sql.gz" -mtime +7 -delete

echo "Backup completed: $FILENAME.gz"
```

**Schedule with cron:**

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /opt/scripts/backup-db.sh >> /var/log/backup.log 2>&1
```

**Backup to S3:**

```bash
# Install AWS CLI
sudo apt install awscli

# Configure credentials
aws configure

# Upload to S3
aws s3 cp $BACKUP_DIR/$FILENAME.gz s3://your-bucket/backups/
```

---

## Troubleshooting

### Application Won't Start

```bash
# Check service status
sudo systemctl status gojang

# Check logs
sudo journalctl -u gojang -n 50

# Common issues:
# - Missing .env file
# - Wrong permissions
# - Port already in use
# - Database connection failed
```

### Database Connection Issues

```bash
# Test PostgreSQL connection
psql -U gojang -d gojang -h localhost

# Check PostgreSQL status
sudo systemctl status postgresql

# Check connection string
echo $DATABASE_URL
```

### Nginx Issues

```bash
# Test configuration
sudo nginx -t

# Check Nginx logs
sudo tail -f /var/log/nginx/error.log

# Restart Nginx
sudo systemctl restart nginx
```

### SSL Certificate Issues

```bash
# Test certificate
sudo certbot certificates

# Renew certificate
sudo certbot renew

# Check certificate expiry
echo | openssl s_client -servername yourdomain.com -connect yourdomain.com:443 2>/dev/null | openssl x509 -noout -dates
```

---

## Updating Application

### Zero-Downtime Deployment

**Option 1: Systemd reload**

```bash
# Upload new binary
scp gojang-app gojang@server:/opt/gojang/gojang-app-new

# On server, atomic swap
cd /opt/gojang
mv gojang-app gojang-app-old
mv gojang-app-new gojang-app
chmod +x gojang-app

# Restart service
sudo systemctl restart gojang

# Verify
sudo systemctl status gojang

# If OK, remove old
rm gojang-app-old
```

**Option 2: Blue-Green Deployment**

Run two instances behind load balancer:
1. Deploy to "green" instance
2. Test green instance
3. Switch traffic to green
4. Update "blue" instance

**Option 3: Rolling Update (Docker)**

```bash
# Update docker-compose.yml with new image tag
docker-compose up -d --no-deps --build app

# Docker will gracefully stop old container
# and start new one
```

---

## Scaling

### Vertical Scaling

Upgrade server resources:
- More CPU cores
- More RAM
- Faster disk (SSD)

**Update systemd service limits:**

```ini
[Service]
LimitNOFILE=65536
LimitNPROC=4096
```

### Horizontal Scaling

Run multiple instances behind load balancer:

**Nginx load balancing:**

```nginx
upstream gojang_backend {
    least_conn;  # Load balancing method
    
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}
```

**Session management:**

Use persistent session store (Redis):

```go
import "github.com/alexedwards/scs/redisstore"

// Connect to Redis
pool := &redis.Pool{
    Addr: "localhost:6379",
}

// Use Redis for sessions
sessionManager.Store = redisstore.New(pool)
```

> 📚 **For comprehensive distributed deployment guide**, including multi-server architectures, load balancing configurations, health checks, and deployment strategies, see the [Distributed Deployment & Load Balancing Guide](./distributed-deployment.md).

---

## Next Steps

- ✅ **Deploy:** Follow VPS or Docker guide for your setup
- ✅ **Secure:** Complete security checklist
- ✅ **Monitor:** Set up logging and monitoring
- ✅ **Backup:** Schedule automated backups
- ✅ **Test:** Verify all functionality in production
- ✅ **Read:** [Testing Best Practices](./testing-best-practices.md)

---

## Quick Reference

### Useful Commands

```bash
# Build
go build -o app -ldflags="-s -w" ./gojang/cmd/web

# Docker
docker build -t gojang-app .
docker run -d -p 8080:8080 gojang-app

# Systemd
sudo systemctl start|stop|restart|status gojang
sudo journalctl -u gojang -f

# Nginx
sudo nginx -t
sudo systemctl reload nginx

# SSL
sudo certbot renew
sudo certbot certificates

# Database
pg_dump -U gojang gojang > backup.sql
psql -U gojang gojang < backup.sql
```

---

Happy deploying! 🚀
