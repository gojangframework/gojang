# Distributed Deployment & Load Balancing Guide

## Overview

**Yes, Gojang supports distributed deployment and load balancing!** 🚀

Gojang applications can be deployed across multiple servers with load balancing to achieve:
- **High availability** - No single point of failure
- **Horizontal scaling** - Handle more traffic by adding servers
- **Better performance** - Distribute load across multiple instances
- **Zero-downtime deployments** - Update servers one at a time

**Key Requirements:**
1. **PostgreSQL database** (not SQLite) - Shared database for all instances
2. **Persistent session storage** (Redis or database-backed) - Share sessions across instances
3. **Load balancer** (Nginx, HAProxy, or cloud load balancer) - Distribute traffic

---

## Why It Works

Gojang is stateless by design (except for sessions), which makes it perfect for distributed deployments:

✅ **Shared Database** - All instances connect to the same PostgreSQL database
✅ **Session Storage** - Sessions can be stored in Redis or PostgreSQL (not in-memory)
✅ **No Local File Dependencies** - Static files can be served via CDN or shared storage
✅ **Stateless Handlers** - Request handlers don't maintain local state

---

## Architecture Patterns

### Pattern 1: Simple Load Balancing (Single Server, Multiple Instances)

Run multiple instances on the same server on different ports:

```
┌─────────────────────────────────────┐
│         Load Balancer (Nginx)       │
│            Port 80/443              │
└─────────────────────────────────────┘
              │
    ┌─────────┼─────────┐
    │         │         │
    ▼         ▼         ▼
┌────────┐ ┌────────┐ ┌────────┐
│ App:1  │ │ App:2  │ │ App:3  │
│ :8080  │ │ :8081  │ │ :8082  │
└────────┘ └────────┘ └────────┘
              │
              ▼
    ┌──────────────────┐
    │   PostgreSQL     │
    │   Redis (opt)    │
    └──────────────────┘
```

**Benefits:**
- Easy to set up
- No network latency between instances and database
- Good for vertical scaling

**Limitations:**
- Single server is still a point of failure
- Limited by single server's resources

---

### Pattern 2: Multi-Server Deployment (Distributed)

Deploy instances across multiple servers:

```
┌────────────────────────────────────┐
│     Cloud Load Balancer (AWS)      │
│        or HAProxy/Nginx            │
└────────────────────────────────────┘
              │
    ┌─────────┼──────────┐
    │         │          │
    ▼         ▼          ▼
┌────────┐ ┌────────┐ ┌────────┐
│Server 1│ │Server 2│ │Server 3│
│ App:80 │ │ App:80 │ │ App:80 │
└────────┘ └────────┘ └────────┘
              │
    ┌─────────┴──────────┐
    │                    │
    ▼                    ▼
┌──────────────┐  ┌──────────────┐
│  PostgreSQL  │  │    Redis     │
│  (Primary)   │  │  (Sessions)  │
└──────────────┘  └──────────────┘
```

**Benefits:**
- True high availability
- Geographic distribution possible
- Independent server failures

**Limitations:**
- More complex setup
- Network latency considerations
- Higher costs

---

### Pattern 3: Container Orchestration (Docker/Kubernetes)

Deploy as containers with auto-scaling:

```
┌────────────────────────────────────┐
│         Cloud Load Balancer         │
└────────────────────────────────────┘
              │
┌─────────────────────────────────────┐
│     Kubernetes / Docker Swarm       │
│  ┌──────┐ ┌──────┐ ┌──────┐        │
│  │ Pod1 │ │ Pod2 │ │ Pod3 │ ...    │
│  │ App  │ │ App  │ │ App  │        │
│  └──────┘ └──────┘ └──────┘        │
└─────────────────────────────────────┘
              │
    ┌─────────┴──────────┐
    │                    │
    ▼                    ▼
┌──────────────┐  ┌──────────────┐
│  PostgreSQL  │  │    Redis     │
│   (RDS)      │  │ (ElastiCache)│
└──────────────┘  └──────────────┘
```

**Benefits:**
- Auto-scaling based on load
- Easy deployment and rollback
- Self-healing (restart failed containers)

---

## Step-by-Step Setup

### Step 1: Configure PostgreSQL (Shared Database)

All instances must connect to the same PostgreSQL database.

**Environment variable:**
```bash
DATABASE_URL=postgres://user:password@db-server:5432/gojang?sslmode=require
```

**In your application code:**
```go
// Database URL is read from environment variable
// All instances will connect to the same PostgreSQL server
dbURL := os.Getenv("DATABASE_URL")
```

**Using Docker Compose:**
```yaml
services:
  db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=gojang
      - POSTGRES_USER=gojang
      - POSTGRES_PASSWORD=securepassword
    volumes:
      - postgres-data:/var/lib/postgresql/data
```

---

### Step 2: Configure Persistent Session Storage

**Default behavior** - Sessions are stored in memory (NOT suitable for distributed deployment):
```go
// ❌ In-memory sessions - lost on restart, not shared between instances
sessionManager := scs.New()
// Default store is in-memory
```

**For distributed deployment** - Use Redis or database-backed sessions:

#### Option A: Redis Sessions (Recommended)

**1. Add Redis dependency:**
```bash
go get github.com/alexedwards/scs/redisstore
go get github.com/gomodule/redigo/redis
```

**2. Update session manager code:**

Edit `gojang/http/middleware/session.go`:

```go
package middleware

import (
	"time"
	"os"

	"github.com/gojangframework/gojang/gojang/config"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/redisstore"
	"github.com/gomodule/redigo/redis"
)

func NewSessionManager(cfg *config.Config) *scs.SessionManager {
	sessionManager := scs.New()
	sessionManager.Lifetime = cfg.SessionLifetime
	sessionManager.Cookie.Name = "session_id"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Secure = !cfg.Debug
	sessionManager.Cookie.SameSite = 2
	sessionManager.Cookie.Path = "/"
	sessionManager.IdleTimeout = 30 * time.Minute

	// Use Redis for distributed sessions
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		pool := &redis.Pool{
			MaxIdle:     10,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.DialURL(redisURL)
			},
		}
		sessionManager.Store = redisstore.New(pool)
	}

	return sessionManager
}
```

**3. Set Redis URL environment variable:**
```bash
REDIS_URL=redis://localhost:6379
```

**4. Docker Compose with Redis:**
```yaml
services:
  app:
    environment:
      - REDIS_URL=redis://redis:6379
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    volumes:
      - redis-data:/data
```

#### Option B: PostgreSQL Sessions

**1. Add PostgreSQL session store:**
```bash
go get github.com/alexedwards/scs/postgresstore
```

**2. Update session manager:**
```go
import (
	"github.com/alexedwards/scs/postgresstore"
)

func NewSessionManager(cfg *config.Config, db *sql.DB) *scs.SessionManager {
	sessionManager := scs.New()
	// ... other configuration ...
	
	// Use PostgreSQL for sessions
	sessionManager.Store = postgresstore.New(db)
	
	return sessionManager
}
```

**3. Create sessions table:**
```sql
CREATE TABLE sessions (
	token TEXT PRIMARY KEY,
	data BYTEA NOT NULL,
	expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);
```

---

### Step 3: Configure Load Balancer

#### Option A: Nginx Load Balancer

**1. Create `/etc/nginx/conf.d/gojang-lb.conf`:**

```nginx
upstream gojang_backend {
    # Load balancing method
    least_conn;  # Route to server with fewest connections
    
    # Your application instances
    server 127.0.0.1:8080 max_fails=3 fail_timeout=30s;
    server 127.0.0.1:8081 max_fails=3 fail_timeout=30s;
    server 127.0.0.1:8082 max_fails=3 fail_timeout=30s;
    
    # Health checks (requires nginx-plus or use proxy_next_upstream)
    # For graceful handling of down servers
}

server {
    listen 80;
    server_name yourdomain.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;

    location / {
        proxy_pass http://gojang_backend;
        
        # Pass original headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support (if needed)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # Retry failed requests on next server
        proxy_next_upstream error timeout invalid_header http_500 http_502 http_503;
    }
}
```

**2. Test and reload Nginx:**
```bash
sudo nginx -t
sudo systemctl reload nginx
```

#### Option B: HAProxy Load Balancer

**Create `/etc/haproxy/haproxy.cfg`:**

```haproxy
global
    log /dev/log local0
    log /dev/log local1 notice
    maxconn 4096

defaults
    log global
    mode http
    option httplog
    option dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000

frontend http_front
    bind *:80
    redirect scheme https code 301 if !{ ssl_fc }

frontend https_front
    bind *:443 ssl crt /etc/haproxy/certs/cert.pem
    default_backend gojang_backend
    
    # Sticky sessions (if not using Redis)
    # cookie SERVERID insert indirect nocache

backend gojang_backend
    balance leastconn  # Load balancing algorithm
    option httpchk GET /health  # Health check endpoint
    
    server app1 127.0.0.1:8080 check
    server app2 127.0.0.1:8081 check
    server app3 127.0.0.1:8082 check
```

#### Option C: Cloud Load Balancers

**AWS Application Load Balancer (ALB):**
- Automatically distributes traffic across multiple targets
- Health checks included
- Auto-scaling group integration
- SSL/TLS termination

**Google Cloud Load Balancer:**
- Global or regional load balancing
- Automatic failover
- CDN integration

**Cloudflare Load Balancing:**
- Geographic routing
- DDoS protection
- Health monitoring

---

### Step 4: Run Multiple Instances

#### Using systemd (single server)

**1. Create service files for each instance:**

`/etc/systemd/system/gojang@.service`:
```ini
[Unit]
Description=Gojang Application Instance %i
After=network.target postgresql.service

[Service]
Type=simple
User=gojang
WorkingDirectory=/opt/gojang
Environment="PORT=808%i"
Environment="DEBUG=false"
Environment="DATABASE_URL=postgres://user:pass@localhost:5432/gojang"
Environment="REDIS_URL=redis://localhost:6379"
ExecStart=/opt/gojang/gojang
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

**2. Enable and start instances:**
```bash
sudo systemctl enable gojang@0
sudo systemctl enable gojang@1
sudo systemctl enable gojang@2

sudo systemctl start gojang@0  # Runs on port 8080
sudo systemctl start gojang@1  # Runs on port 8081
sudo systemctl start gojang@2  # Runs on port 8082
```

#### Using Docker Compose

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  app1:
    build: .
    restart: unless-stopped
    environment:
      - PORT=8080
      - DEBUG=false
      - DATABASE_URL=postgres://gojang:password@db:5432/gojang?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - SESSION_KEY=${SESSION_KEY}
    depends_on:
      - db
      - redis
    networks:
      - gojang-network

  app2:
    build: .
    restart: unless-stopped
    environment:
      - PORT=8080
      - DEBUG=false
      - DATABASE_URL=postgres://gojang:password@db:5432/gojang?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - SESSION_KEY=${SESSION_KEY}
    depends_on:
      - db
      - redis
    networks:
      - gojang-network

  app3:
    build: .
    restart: unless-stopped
    environment:
      - PORT=8080
      - DEBUG=false
      - DATABASE_URL=postgres://gojang:password@db:5432/gojang?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - SESSION_KEY=${SESSION_KEY}
    depends_on:
      - db
      - redis
    networks:
      - gojang-network

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - app1
      - app2
      - app3
    networks:
      - gojang-network

  db:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      - POSTGRES_DB=gojang
      - POSTGRES_USER=gojang
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - gojang-network

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    volumes:
      - redis-data:/data
    networks:
      - gojang-network

networks:
  gojang-network:
    driver: bridge

volumes:
  postgres-data:
  redis-data:
```

**Start all services:**
```bash
docker-compose up -d --scale app=3
```

---

## Health Checks

Add a health check endpoint to your application for load balancer monitoring.

**Create `gojang/http/handlers/health.go`:**

```go
package handlers

import (
	"net/http"
)

// HealthCheck returns 200 OK if the application is healthy
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
```

**Add to router:**
```go
r.Get("/health", handlers.HealthCheck)
```

**Test:**
```bash
curl http://localhost:8080/health
# Should return: OK
```

---

## Testing Your Distributed Setup

### 1. Verify All Instances Are Running

```bash
# Check all instances
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
```

### 2. Test Session Persistence

**Login on one instance:**
```bash
curl -c cookies.txt -X POST http://localhost:8080/login \
  -d "email=user@example.com" \
  -d "password=password123"
```

**Verify session works on another instance:**
```bash
curl -b cookies.txt http://localhost:8081/dashboard
# Should still be logged in
```

### 3. Test Load Distribution

```bash
# Send multiple requests through load balancer
for i in {1..10}; do
  curl http://localhost/
  sleep 1
done

# Check nginx access logs to see distribution
sudo tail -f /var/log/nginx/access.log
```

### 4. Test Failover

**Stop one instance:**
```bash
sudo systemctl stop gojang@1
```

**Verify requests still work:**
```bash
curl http://localhost/
# Should work - load balancer routes to other instances
```

---

## Deployment Strategies

### Zero-Downtime Deployment

Deploy updates without taking the entire site offline:

**1. Update one instance at a time:**
```bash
# Stop first instance
sudo systemctl stop gojang@0

# Update application binary
sudo cp new-gojang /opt/gojang/gojang

# Start first instance
sudo systemctl start gojang@0

# Verify it works
curl http://localhost:8080/health

# Repeat for other instances
```

**2. Using Docker:**
```bash
# Build new image
docker-compose build

# Update one service at a time
docker-compose up -d --no-deps --build app1
sleep 10
docker-compose up -d --no-deps --build app2
sleep 10
docker-compose up -d --no-deps --build app3
```

### Blue-Green Deployment

Run two complete environments and switch traffic:

```
Production (Blue) → Users
Staging (Green)   → Testing

After validation:
Production (Green) → Users
Staging (Blue)     → Idle
```

---

## Monitoring & Troubleshooting

### Monitor Instance Health

**Check instance status:**
```bash
# systemd
sudo systemctl status gojang@0
sudo systemctl status gojang@1
sudo systemctl status gojang@2

# Docker
docker-compose ps
```

**Monitor logs:**
```bash
# All instances
sudo journalctl -u 'gojang@*' -f

# Specific instance
sudo journalctl -u gojang@0 -f
```

### Monitor Load Balancer

**Nginx statistics:**
```bash
# Check which backends are up
sudo tail -f /var/log/nginx/access.log

# Backend status (requires nginx-plus or stub_status module)
curl http://localhost/nginx_status
```

### Monitor Redis

```bash
# Connect to Redis
redis-cli

# Check session count
DBSIZE

# Monitor commands
MONITOR
```

### Common Issues

**Issue: Sessions lost after deployment**
- Ensure Redis is running and accessible
- Verify `REDIS_URL` is set correctly on all instances
- Check Redis connection in logs

**Issue: Uneven load distribution**
- Check load balancing algorithm (try `least_conn` instead of `round_robin`)
- Verify all instances are responding to health checks
- Check server weights in load balancer config

**Issue: Database connection errors**
- Increase PostgreSQL `max_connections`
- Configure connection pooling properly
- Check network connectivity from all instances

---

## Best Practices

### ✅ DO

1. **Use PostgreSQL** instead of SQLite for distributed deployments
2. **Use Redis** for session storage (or database-backed sessions)
3. **Implement health checks** for load balancer monitoring
4. **Use connection pooling** for database connections
5. **Set proper timeouts** on load balancer and application
6. **Monitor all instances** individually and collectively
7. **Use secrets management** for sensitive configuration
8. **Test failover scenarios** before going to production
9. **Implement graceful shutdown** to finish ongoing requests
10. **Use same SECRET_KEY** across all instances

### ❌ DON'T

1. **Don't use in-memory sessions** for distributed setups
2. **Don't use SQLite** - it doesn't support concurrent access from multiple servers
3. **Don't store files locally** - use S3 or shared storage
4. **Don't skip health checks** - load balancer needs them
5. **Don't use different SECRET_KEY values** - sessions won't work
6. **Don't deploy to all instances simultaneously** - use rolling deployment
7. **Don't ignore database connection limits** - scale accordingly

---

## Scaling Checklist

Before going distributed:

- [ ] **Switch to PostgreSQL** from SQLite
- [ ] **Implement Redis** session storage (or database-backed)
- [ ] **Add health check endpoint** (`/health`)
- [ ] **Configure load balancer** (Nginx, HAProxy, or cloud LB)
- [ ] **Use environment variables** for all configuration
- [ ] **Set same SESSION_KEY** on all instances
- [ ] **Set same CSRF_SECRET** on all instances
- [ ] **Test session persistence** across instances
- [ ] **Test database connectivity** from all instances
- [ ] **Configure monitoring** and alerting
- [ ] **Plan deployment strategy** (rolling, blue-green)
- [ ] **Document runbooks** for common operations
- [ ] **Test failover scenarios**

---

## Example: Complete AWS Deployment

**Architecture:**
- Application Load Balancer (ALB)
- EC2 Auto Scaling Group (3 instances)
- RDS PostgreSQL (primary + read replica)
- ElastiCache Redis (for sessions)

**Benefits:**
- Auto-scaling based on CPU/memory
- Multi-AZ deployment for high availability
- Managed database and cache
- SSL termination at load balancer
- CloudWatch monitoring included

**Estimated cost:** ~$150-300/month for small setup

---

## Conclusion

**Gojang is fully capable of distributed deployment** with the following considerations:

✅ **Use PostgreSQL** - Required for shared state
✅ **Use Redis** - Required for shared sessions  
✅ **Use Load Balancer** - Required for traffic distribution
✅ **Stateless design** - Makes scaling easy

With proper configuration, Gojang applications can scale horizontally to handle millions of requests with high availability and zero-downtime deployments.

**Next Steps:**
1. Review the [Deployment Guide](./deployment-guide.md) for basic deployment
2. Set up PostgreSQL and Redis
3. Configure persistent session storage
4. Set up load balancer
5. Test distributed deployment in staging
6. Monitor and optimize

Happy scaling! 🚀

