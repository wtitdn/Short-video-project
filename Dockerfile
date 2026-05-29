FROM alpine:3.20

RUN apk add --no-cache tzdata ca-certificates
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime

WORKDIR /app

COPY deploy/api/dist/app /app/app
COPY deploy/api/dist/worker /app/worker

RUN chmod 0755 /app/app /app/worker

EXPOSE 7878

USER nobody

ENTRYPOINT ["sh", "-c", "/app/worker & /app/app"]