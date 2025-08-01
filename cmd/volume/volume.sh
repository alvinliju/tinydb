#!/bin/bash

PORT=${1:-8080}
DATADIR=${2:-/tmp/nginx_data}
CONF_PATH="/tmp/nginx-volume-$PORT.conf"

mkdir -p "$DATADIR"
chmod 777 $DATADIR
mkdir -p "$DATADIR/body_temp"

cat > "$CONF_PATH" <<EOF
worker_processes auto;
worker_rlimit_nofile 100000;
pid $DATADIR/nginx.pid;
events {
    # determines how much clients will be served per worker
    # max clients = worker_connections * worker_processes
    # max clients is also limited by the number of socket connections available on the system (~64k)
    worker_connections 4000;

    # optimized to serve many clients with each thread, essential for linux -- for testing environment
    use epoll;

    # accept as many connections as possible, may flood worker connections if set too low -- for testing environment
    multi_accept on;
}

http {
    default_type  application/octet-stream;
 access_log off;
 server_tokens off;
    sendfile        on;
    sendfile_max_chunk 1024k;
    disable_symlinks off;

    tcp_nopush     on;
    tcp_nodelay    on;
    keepalive_timeout  65;

    server {
        listen       $PORT;

        location / {
            root   $DATADIR;
            index  index.html index.htm;

            client_body_temp_path $DATADIR/body_temp;

            dav_methods PUT DELETE;
            dav_access group:rw all:r;
            create_full_put_path on;

            client_max_body_size 0;

            autoindex on;
        }
    }
}
EOF

echo "Starting nginx on port $PORT with data dir $DATADIR"

# Optional: test config before starting
nginx -t -c "$CONF_PATH" || { echo "Config test failed"; exit 1; }

# Start nginx - no -p flag to avoid prefix path doubling issues
nginx -c "$CONF_PATH" -p "$DATADIR"
