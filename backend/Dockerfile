# ============================================
# Go 1.26 开发容器
# UID/GID 由 build args 传入（macOS 501/20，Linux 1000/1000）
# ============================================

FROM golang:1.26-bookworm

ARG UID=1000
ARG GID=1000

# 创建应用用户组和用户，UID/GID 与宿主机一致
RUN ( getent group ${GID} || groupadd -g ${GID} app ) && \
    ( id -u app 2>/dev/null || useradd -u ${UID} -g ${GID} -m -s /bin/bash app )

# 预创建工作目录和 Go 缓存目录
RUN mkdir -p /app && \
    mkdir -p /home/app/go/pkg/mod && \
    mkdir -p /home/app/.cache/go-build && \
    chown -R ${UID}:${GID} /app /home/app

# 切换到非 root 用户（通过 user: 运行时指定，Dockerfile 这里保持灵活）
USER ${UID}:${GID}
WORKDIR /app

ENV GOPATH=/home/app/go
ENV GOCACHE=/home/app/.cache/go-build
ENV GOMODCACHE=/home/app/go/pkg/mod
ENV PATH=$GOPATH/bin:$PATH
ENV TZ=Asia/Shanghai
ENV APP_PORT=8080

# sqlc：由 internal/core/*.sql 生成 sqlc/*.go（`sqlc generate`，工作目录 /app）
# 写明版本而非 @latest：生成器版本决定产物，浮动版本会让不同人 generate 出不同结果
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

EXPOSE 8080

CMD ["tail", "-f", "/dev/null"]
