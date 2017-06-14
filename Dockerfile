FROM alpine:3.6
ENV GOPATH /go
ENV PACKAGE github.com/tomologic/kube2clouddns/
ENV PROJECT_HOME $GOPATH/src/$PACKAGE
RUN mkdir -p $PROJECT_HOME
WORKDIR $PROJECT_HOME
COPY glide.yaml $PROJECT_HOME
COPY glide.lock $PROJECT_HOME
COPY *.go $PROJECT_HOME

RUN apk --no-cache add -t deps --update gcc git openssl ca-certificates musl-dev \
    && export GOPATH=/go \
    && apk --no-cache add -t deps --update --repository http://dl-cdn.alpinelinux.org/alpine/edge/community go glide  \
    && glide install \
    #&& go build -v \
    && go install \
    && cp /go/bin/* /usr/bin/ \
    && rm -rf /go $HOME/.glide \
    && apk del --purge deps; rm -rf /tmp/* /var/cache/apk/*

ENTRYPOINT ["kube2clouddns"]
