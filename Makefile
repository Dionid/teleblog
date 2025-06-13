-include .env
export $(shell sed 's/=.*//' .env)

PROJECT_NAME=teleblog
BINARY_NAME=${PROJECT_NAME}

# Setup

setup:
	go install github.com/a-h/templ/cmd/templ@latest
	go mod tidy

# Generate

templ:
	cd cmd/teleblog \
	&& go generate ./...

# Dev & Run

serve:
	npx tailwindcss build -i tailwind.css -o cmd/teleblog/httpapi/public/style.css --minify
	cd cmd/teleblog \
	&& go generate ./... \
	&& go run . serve

# Scripts

upload-history:
	cd cmd/teleblog \
	&& go generate ./... \
	&& go run . upload-history

extract-tags:
	cd cmd/teleblog \
	&& go generate ./... \
	&& go run . extract-tags

# Build

build-teleblog-mac:
	npx tailwindcss build -i tailwind.css -o cmd/teleblog/httpapi/public/style.css
	make templ
	GOARCH=amd64 GOOS=darwin go build -o ./cmd/teleblog/${BINARY_NAME}-darwin ./cmd/teleblog

clean-mac:
	go clean
	rm ${BINARY_NAME}-darwin

build-teleblog-linux:
	npx tailwindcss build -i tailwind.css -o cmd/teleblog/httpapi/public/style.css --minify
	make templ
	GOARCH=amd64 GOOS=linux go build -o ./cmd/teleblog/${BINARY_NAME}-linux ./cmd/teleblog

clean:
	go clean
	rm ${BINARY_NAME}-cli-darwin
	rm ${BINARY_NAME}-cli-linux
	rm ${BINARY_NAME}-teleblog-darwin
	rm ${BINARY_NAME}-teleblog-linux

# Deploy

increment-production-app-version:
	ssh root@${SERVER_IP} '\
		cd /root/teleblog && \
		current=$$(grep "^APP_VERSION=" app.env | cut -d= -f2) && \
		IFS=. read -r major minor patch <<< "$$current" && \
		new_minor=$$((minor + 1)) && \
		new_version="$$major.$$new_minor.$$patch" && \
		sed -i "s/^APP_VERSION=.*/APP_VERSION=$$new_version/" app.env && \
		echo "Version updated to $$new_version"'

backup-production-db:
	ssh root@${SERVER_IP} '\
		cd /root/teleblog/pb_data/backups && \
		backup_dir=$$(date +%Y-%m-%d-%H-%M-%S) && \
		mkdir $$backup_dir && \
		cp ../data.db $$backup_dir/data.db && \
		cp ../logs.db $$backup_dir/logs.db && \
		zip -r $$backup_dir.zip $$backup_dir && \
		rm -rf $$backup_dir && \
		echo "Backup created: $$backup_dir.zip"'

deploy:
	make build-teleblog-linux
	make increment-production-app-version
	ssh root@${SERVER_IP} "systemctl stop teleblog"
	make backup-production-db
	scp ./cmd/${PROJECT_NAME}/${BINARY_NAME}-linux root@${SERVER_IP}:/root/${PROJECT_NAME}/${BINARY_NAME}-linux
	ssh root@${SERVER_IP} "systemctl start teleblog"
