# Stage 1: Build the export API and export worker.
FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git tzdata

ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/export-service ./services/export-service/cmd

# Stage 2: Runtime dependencies owned by export-service.
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    chromium \
    pandoc \
    font-noto-cjk \
    freetype \
    harfbuzz \
    nss \
    ttf-freefont \
    tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

RUN mkdir /usr/share/fonts/inkwords-reviewed && \
    ln -s /usr/share/fonts/freefont/FreeMono.otf /usr/share/fonts/inkwords-reviewed/FreeMono.otf && \
    ln -s /usr/share/fonts/freefont/FreeSans.otf /usr/share/fonts/inkwords-reviewed/FreeSans.otf && \
    ln -s /usr/share/fonts/noto/NotoSansCJK-Regular.ttc /usr/share/fonts/inkwords-reviewed/NotoSansCJK-Regular.ttc && \
    ln -s /usr/share/fonts/noto/NotoSansCJK-Bold.ttc /usr/share/fonts/inkwords-reviewed/NotoSansCJK-Bold.ttc

RUN addgroup -S -g 10001 inkwords && \
    adduser -S -D -H -u 10001 -G inkwords inkwords && \
    mkdir -p /app/export-artifacts /app/assets/publishing && \
    chown -R inkwords:inkwords /app

COPY --from=builder --chown=inkwords:inkwords /out/export-service ./export-service
COPY --from=builder --chown=inkwords:inkwords /src/services/export-service/assets/publishing/inkwords-reference.docx ./assets/publishing/inkwords-reference.docx
COPY --from=builder --chown=inkwords:inkwords /src/services/export-service/assets/licenses ./assets/licenses

USER inkwords:inkwords

EXPOSE 8080

CMD ["./export-service"]
