# Nginx Configuration for AgentAPI

This directory contains optional Nginx reverse proxy configuration for production deployments.

## Overview

The Nginx reverse proxy provides:
- SSL/TLS termination
- Load balancing
- Request routing
- Rate limiting
- Security headers
- WebSocket support
- Compression

## Files

- `nginx.conf` - Main Nginx configuration
- `conf.d/agentapi.conf` - Virtual host configuration
- `ssl/` - SSL certificates (you must provide these)

## Setup

### 1. SSL Certificates

Place your SSL certificates in the `ssl/` directory:

```bash
mkdir -p nginx/ssl
cp your-cert.crt nginx/ssl/cert.crt
cp your-cert.key nginx/ssl/cert.key
```

**For testing with self-signed certificates:**

```bash
mkdir -p nginx/ssl
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout nginx/ssl/cert.key \
  -out nginx/ssl/cert.crt \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
```

### 2. Enable Nginx Service

Uncomment the nginx service in `docker-compose.multitenant.yml`:

```yaml
services:
  # ... other services ...

  nginx:
    image: nginx:alpine
    container_name: agentapi-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - nginx_logs:/var/log/nginx
    # ... rest of configuration ...
```

Also uncomment the `nginx_logs` volume at the bottom of the file.

### 3. Start Services

```bash
docker-compose -f docker-compose.multitenant.yml up -d
```

## Configuration

### Rate Limiting

Default rate limits:
- API endpoints: 10 requests/second (burst: 20)
- MCP endpoints: 20 requests/second (burst: 40)

Adjust in `nginx.conf`:
```nginx
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=mcp_limit:10m rate=20r/s;
```

### Timeouts

- AgentAPI: 300s read timeout
- FastMCP: 600s read timeout
- WebSocket: 7 days

Adjust in `conf.d/agentapi.conf` under each location block.

### Client Body Size

Maximum upload size: 100MB

Adjust in `nginx.conf`:
```nginx
client_max_body_size 100M;
```

### Compression

Gzip compression is enabled for:
- Text files (HTML, CSS, JavaScript)
- JSON and XML
- Fonts
- SVG images

### Security Headers

The following security headers are automatically added:
- `Strict-Transport-Security` - Forces HTTPS
- `X-Frame-Options` - Prevents clickjacking
- `X-Content-Type-Options` - Prevents MIME sniffing
- `X-XSS-Protection` - XSS protection

## URL Routes

When using Nginx:

| Route | Backend | Purpose |
|-------|---------|---------|
| `/` | AgentAPI (3284) | Main API endpoints |
| `/mcp/` | FastMCP (8000) | MCP service endpoints |
| `/ws` | AgentAPI (3284) | WebSocket connections |
| `/health` | Nginx | Health check (no backend) |

## Access URLs

- **HTTP**: http://localhost (redirects to HTTPS)
- **HTTPS**: https://localhost
- **Health**: http://localhost/health or https://localhost/health

## Monitoring

### View Logs

```bash
# All nginx logs
docker-compose -f docker-compose.multitenant.yml logs -f nginx

# Access log (JSON format)
docker-compose -f docker-compose.multitenant.yml exec nginx tail -f /var/log/nginx/access.log

# Error log
docker-compose -f docker-compose.multitenant.yml exec nginx tail -f /var/log/nginx/error.log
```

### Test Configuration

```bash
# Test nginx config syntax
docker-compose -f docker-compose.multitenant.yml exec nginx nginx -t

# Reload nginx
docker-compose -f docker-compose.multitenant.yml exec nginx nginx -s reload
```

## Load Balancing

To add more AgentAPI instances for load balancing:

1. Scale the agentapi service:
```bash
docker-compose -f docker-compose.multitenant.yml up -d --scale agentapi=3
```

2. Nginx will automatically distribute requests using `least_conn` algorithm.

## Customization

### Add Custom Headers

Edit `conf.d/agentapi.conf`:
```nginx
location / {
    add_header X-Custom-Header "value" always;
    # ... rest of config ...
}
```

### Add IP Whitelist

```nginx
location /admin {
    allow 192.168.1.0/24;
    deny all;
    # ... rest of config ...
}
```

### Add Basic Auth

```bash
# Create password file
docker-compose -f docker-compose.multitenant.yml exec nginx sh -c \
  "echo 'username:$(openssl passwd -apr1 password)' > /etc/nginx/.htpasswd"

# Add to location block in conf.d/agentapi.conf
location /admin {
    auth_basic "Restricted";
    auth_basic_user_file /etc/nginx/.htpasswd;
    # ... rest of config ...
}
```

## Troubleshooting

### SSL Certificate Errors

```bash
# Verify certificate files exist
ls -l nginx/ssl/

# Check certificate details
openssl x509 -in nginx/ssl/cert.crt -text -noout

# Verify private key matches certificate
openssl x509 -noout -modulus -in nginx/ssl/cert.crt | openssl md5
openssl rsa -noout -modulus -in nginx/ssl/cert.key | openssl md5
```

### Configuration Errors

```bash
# Test configuration
docker-compose -f docker-compose.multitenant.yml exec nginx nginx -t

# View error log
docker-compose -f docker-compose.multitenant.yml logs nginx
```

### Connection Issues

```bash
# Check nginx is listening
docker-compose -f docker-compose.multitenant.yml exec nginx netstat -tlnp

# Test backend connectivity
docker-compose -f docker-compose.multitenant.yml exec nginx wget http://agentapi:3284/status
```

### Rate Limiting Issues

If legitimate traffic is being rate limited:

1. Increase rate limit in `nginx.conf`
2. Increase burst size in `conf.d/agentapi.conf`
3. Or disable for specific IPs:

```nginx
geo $limit {
    default 1;
    192.168.1.0/24 0;  # Whitelist
}

map $limit $limit_key {
    0 "";
    1 $binary_remote_addr;
}

limit_req_zone $limit_key zone=api_limit:10m rate=10r/s;
```

## Production Considerations

1. **SSL Certificates**: Use proper certificates from Let's Encrypt or a CA
2. **Security**: Keep Nginx and OpenSSL updated
3. **Monitoring**: Set up log aggregation and monitoring
4. **Backup**: Backup SSL certificates securely
5. **Testing**: Test configuration before deploying
6. **Performance**: Tune worker processes and connections for your workload

## Alternative: Let's Encrypt

To use Let's Encrypt for free SSL certificates:

1. Install certbot:
```bash
docker run -it --rm --name certbot \
  -v ./nginx/ssl:/etc/letsencrypt \
  -v ./nginx/webroot:/var/www/certbot \
  certbot/certbot certonly --webroot \
  -w /var/www/certbot \
  -d your-domain.com
```

2. Update certificate paths in `conf.d/agentapi.conf`:
```nginx
ssl_certificate /etc/nginx/ssl/live/your-domain.com/fullchain.pem;
ssl_certificate_key /etc/nginx/ssl/live/your-domain.com/privkey.pem;
```

3. Set up auto-renewal with cron.

## Disable Nginx

To remove Nginx and access services directly:

1. Comment out nginx service in `docker-compose.multitenant.yml`
2. Access services directly:
   - AgentAPI: http://localhost:3284
   - FastMCP: http://localhost:8000

## Support

- Nginx Documentation: https://nginx.org/en/docs/
- Docker Nginx Image: https://hub.docker.com/_/nginx
- AgentAPI Documentation: ../DOCKER_COMPOSE_README.md
