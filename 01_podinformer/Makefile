BINPATH = cmd/podinformer
BIN = podinformer
REPO = embano1/podinformer
VERSION = 1.0

default: build

build:
	GOOS=linux CGO_ENABLED=0 go build -installsuffix "static" -o ${BIN} ${BINPATH}/*.go

image: 
	docker build -t ${REPO}:${VERSION} .

push: image
	docker push ${REPO}:${VERSION}

clean:
	rm ${BIN}

all: build image push clean