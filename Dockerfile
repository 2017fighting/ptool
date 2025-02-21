FROM alpine:latest
RUN apk add go
WORKDIR /ptool
COPY . .
RUN go build

FROM alpine:latest
COPY --from=0 /ptool/ptool /usr/bin/ptool
WORKDIR /root
VOLUME /root/.config/ptool
ENTRYPOINT ["/usr/bin/ptool"]
