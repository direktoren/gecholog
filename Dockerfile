# Download and unpack nats & root certificates
FROM alpine AS nats-downloader

# To upgrade packages (less vulnerablilites)
RUN apk update && apk upgrade && \
    rm -rf /var/cache/apk/*

# Get system pool of certificates
RUN apk --update add ca-certificates

WORKDIR /nats

ADD https://github.com/nats-io/nats-server/releases/download/v2.10.18/nats-server-v2.10.18-linux-amd64.zip /nats/

RUN unzip /nats/nats-server-v2.10.18-linux-amd64.zip && mv /nats/nats-server-v2.10.18-linux-amd64/nats-server /nats-server && mv /nats/nats-server-v2.10.18-linux-amd64/LICENSE /LICENSE

# Building the Gecholog binaries
FROM golang:latest AS builder

# Set the working directory inside the builder stage
WORKDIR /build

ARG VERSION

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY cmd/tokencounter/*.go ./cmd/tokencounter/
COPY cmd/nats2log/*.go ./cmd/nats2log/
COPY cmd/gl/*.go ./cmd/gl/
COPY cmd/ginit/*.go ./cmd/ginit/
COPY cmd/gui/*.go ./cmd/gui/
COPY cmd/healthcheck/*.go ./cmd/healthcheck/
COPY cmd/entrypoint/*.go ./cmd/entrypoint/
COPY internal/ ./internal/

# Compile the binary, disable cgo, and statically link all libraries.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=$VERSION'" -o /tokencounter ./cmd/tokencounter
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=$VERSION'" -o /nats2log ./cmd/nats2log
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=$VERSION'" -o /gl ./cmd/gl
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=$VERSION'" -o /ginit ./cmd/ginit
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=$VERSION'" -o /gui ./cmd/gui
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=$VERSION'" -o /healthcheck ./cmd/healthcheck
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.version=$VERSION'" -o /entrypoint ./cmd/entrypoint

# Remove the source code
RUN rm -rf /build

# Create a .version file
RUN echo $VERSION > /.version

FROM golang:latest AS directories

# Create necessary libraries
RUN mkdir -p /usr/share/doc && mkdir -p /usr/share/doc/nats-server 
RUN mkdir -p /app && mkdir -p /app/conf && mkdir -p /app/default-conf && mkdir -p /app/log && mkdir -p /app/checksum && mkdir -p /app/working && mkdir -p /app/archive
RUN mkdir -p /config && mkdir -p /config/certs

RUN mkdir -p /templates && mkdir -p /static

# Final image
FROM scratch

# ----------------------- Folders -----------------------
COPY --from=directories --chmod=777 /app /app
COPY --from=directories --chmod=777 /config /config
COPY --from=directories --chmod=777 /templates /templates
COPY --from=directories --chmod=777 /static /static

# ----------------------- Files -----------------------
# Readonly

# licenses
COPY --from=nats-downloader --chmod=444 /LICENSE /usr/share/doc/nats-server/LICENSE
COPY --chmod=444 LICENSE /usr/share/doc/gecholog/LICENSE

# .version file
COPY --from=builder --chmod=444 /.version /.version

# root certificates
COPY --from=nats-downloader --chmod=444 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# default config files
COPY --chmod=444 config/ginit_config.json /app/default-conf/ginit_config.json
COPY --chmod=444 config/ginit_nogui_config.json /app/default-conf/ginit_nogui_config.json
COPY --chmod=444 config/gl_config.json /app/default-conf/gl_config.json
COPY --chmod=444 config/gui_config.json /app/default-conf/gui_config.json
COPY --chmod=444 config/tokencounter_config.json /app/default-conf/tokencounter_config.json
COPY --chmod=444 config/nats2log_config.json /app/default-conf/nats2log_config.json
COPY --chmod=444 config/nats2file_config.json /app/default-conf/nats2file_config.json
COPY --chmod=444 config/nats-server.conf /app/default-conf/nats-server.conf

# gui assets
COPY --chmod=444 cmd/gui/static/logo_black_t.png /static/logo_black_t.png
COPY --chmod=444 cmd/gui/static/copy.png /static/copy.png
COPY --chmod=444 cmd/gui/static/styles.css /static/styles.css
COPY --chmod=444 cmd/gui/static/favicon.ico /static/favicon.ico
COPY --chmod=444 cmd/gui/static/highlight.min.js /static/highlight.min.js
COPY --chmod=444 cmd/gui/templates/header.html /templates/header.html
COPY --chmod=444 cmd/gui/templates/tutorials.html /templates/tutorials.html
COPY --chmod=444 cmd/gui/templates/archive-query.html /templates/archive-query.html
COPY --chmod=444 cmd/gui/templates/form.html /templates/form.html
COPY --chmod=444 cmd/gui/templates/login.html /templates/login.html
COPY --chmod=444 cmd/gui/templates/menu.html /templates/menu.html
COPY --chmod=444 cmd/gui/templates/mainmenu.html /templates/mainmenu.html
COPY --chmod=444 cmd/gui/templates/open.html /templates/open.html
COPY --chmod=444 cmd/gui/templates/publish.html /templates/publish.html
COPY --chmod=444 cmd/gui/templates/processors.html /templates/processors.html
COPY --chmod=444 cmd/gui/templates/routers.html /templates/routers.html
COPY --chmod=444 cmd/gui/templates/python.html /templates/python.html
COPY --chmod=444 cmd/gui/templates/logs.html /templates/logs.html

# ----------------------- Binaries -----------------------
# Executable but not writable
COPY --from=builder --chmod=111 /ginit /ginit
COPY --from=builder --chmod=111 /gui /gui
COPY --from=builder --chmod=111 /healthcheck /healthcheck
COPY --from=builder --chmod=111 /gl /gl
COPY --from=builder --chmod=111 /tokencounter /tokencounter
COPY --from=builder --chmod=111 /nats2log /nats2log
COPY --from=builder --chmod=111 /entrypoint /entrypoint

# nats server
COPY --from=nats-downloader --chmod=111 /nats-server /nats-server

HEALTHCHECK CMD ["/healthcheck"]

# 5380 gl 4222 8222 6222 nats 8080 gui
EXPOSE 5380 4222 8222 6222 8080

# Gecholog user id
USER 10001 

CMD ["./entrypoint", "-c", "./ginit:-o:/app/conf/ginit_config.json","-t","/app/conf/","-s","/app/default-conf/","-f","ginit_config.json:gl_config.json:tokencounter_config.json:nats2log_config.json:nats2file_config.json:nats-server.conf:gui_config.json","-e","NATS_TOKEN:GUI_SECRET"]