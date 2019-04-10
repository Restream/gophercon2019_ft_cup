FROM ubuntu:18.04

RUN apt update -y && apt install -y curl git

# Install golang
RUN curl -L https://redirector.gvt1.com/edgedl/go/go1.10.4.linux-amd64.tar.gz | tar xzv -C /usr/local
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

# Download & build fasthttp
RUN go get github.com/buaazp/fasthttprouter && \
    go get github.com/valyala/fasthttp

# Download & build reindexer
RUN curl -L https://github.com/restream/reindexer/raw/master/dependencies.sh | bash -s
RUN go get -a github.com/restream/reindexer
RUN cd /go/src/github.com/restream/reindexer/ && \
    mkdir -p build && \
    cd build && \
    cmake -DCMAKE_BUILD_TYPE=Release .. && \
    make reindexer -j4 && \
    rm -rf cpp_src/CMakeFiles

# Build & install our go application server
ADD app /app
WORKDIR /app
RUN go build ./cmd/server && \
    mv /app/server /bin/ftcup_server && \
    rm -rf /app
WORKDIR /

# Expose 8080 port to host
EXPOSE 8080:8080

# Start application server on container startup
CMD ["/bin/ftcup_server"]
