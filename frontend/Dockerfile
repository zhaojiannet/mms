# ============================================
# Nuxt 4 + pnpm 开发容器（Node 24 = Active LTS）
# UID/GID 由 build args 传入（macOS 501/20，Linux 1000/1000）
# ============================================

FROM node:24-bookworm-slim

ARG UID=1000
ARG GID=1000

# 启用 Corepack
RUN corepack enable && corepack prepare pnpm@11.21.0 --activate

# 【关键】调整 node 用户 UID/GID 与宿主机一致
USER root
RUN ( getent group ${GID} || groupmod -g ${GID} node ) && \
    usermod -u ${UID} -g ${GID} node 2>/dev/null || true

# 预创建所有挂载点（named volume 首次挂载时继承此权限）
RUN mkdir -p /app /app/node_modules /app/.nuxt /app/.output && \
    mkdir -p /home/node/.local/share/pnpm/store && \
    chown -R ${UID}:${GID} /app && \
    chown -R ${UID}:${GID} /home/node

# 切换到 node 用户
USER node
WORKDIR /app

# 配置 pnpm store 目录
RUN pnpm config set store-dir /home/node/.local/share/pnpm/store

ENV NUXT_HOST=0.0.0.0
ENV NUXT_PORT=3000
ENV TZ=Asia/Shanghai

EXPOSE 3000

# 容器保持运行，启动命令由 docker-compose 的 command 提供
CMD ["tail", "-f", "/dev/null"]
