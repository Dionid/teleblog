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
