-include .env
export $(shell sed 's/=.*//' .env)

PROJECT_NAME=teleblog
BINARY_NAME=${PROJECT_NAME}

# Run

parse:
	cd cmd/cli && go run .

serve-teleblog:
	npx tailwindcss build -i tailwind.css -o cmd/teleblog/httpapi/public/style.css
	cd cmd/teleblog \
	&& go generate ./... \
	&& go run . serve

upload-history:
	cd cmd/teleblog \
	&& go generate ./... \
	&& go run . upload-history

# Cmds

extract-tags:
	cd cmd/teleblog \
	&& go generate ./... \
	&& go run . extract-tags

# Generate

templ:
	cd cmd/teleblog \
	&& go generate ./...

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

# Setup

setup:
	npm i
	go install github.com/a-h/templ/cmd/templ@latest
	go mod tidy

setup-droplet:
	scp ./infra/teleblog.service root@${SERVER_IP}:/lib/systemd/system/teleblog.service
	ssh root@${SERVER_IP} "apt update \
	&& apt-get install -y zip \
	&& mkdir -p /root/teleblog \
	&& systemctl enable teleblog \
	&& systemctl daemon-reload"
	scp ./cmd/teleblog/app.env.example root@${SERVER_IP}:/root/teleblog/app.env
	scp ./infra/davidshekunts.ru root@${SERVER_IP}:/etc/nginx/sites-available/davidshekunts.ru
	scp ./infra/davidshekunts.ru root@${SERVER_IP}:/etc/nginx/sites-enabled/davidshekunts.ru

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

# connect to server
# add folder to root/teleblog/pb_data/backups with current date
# copy root/teleblog/pb_data/data.db and root/teleblog/pb_data/logs.db to this folder
# zip this folder
# delete folder
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
	scp ./cmd/teleblog/${BINARY_NAME}-linux root@${SERVER_IP}:/root/teleblog/${BINARY_NAME}-linux
	ssh root@${SERVER_IP} "systemctl start teleblog"