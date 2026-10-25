# ============================================
# Nuxt 4 + pnpm 开发容器
# UID/GID 由 build args 传入（macOS 501/20，Linux 1000/1000）
# 标签跟官方浮动：node 的长期支持版换代时自动跟上
# ============================================

FROM node:lts-trixie-slim

ARG UID=1000
ARG GID=1000

# 这里装的 pnpm 只负责启动，实际用 package.json 里 packageManager 写的版本，pnpm 会自己下载切换。
# 不用 corepack：Node 25 起它不随 Node 发行。--before 只取发布满 3 天的版本。
# libatomic1：版本不同时 pnpm 下载的是独立可执行版 @pnpm/exe，它要这个库，slim 镜像里没有
RUN apt-get update \
    && apt-get install -y --no-install-recommends libatomic1 \
    && rm -rf /var/lib/apt/lists/* \
    && npm install -g pnpm --before="$(date -u -d '3 days ago' +%Y-%m-%dT%H:%M:%SZ)" \
    && npm cache clean --force

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

# store 指到 compose 挂的卷上。pnpm 12 只认 pnpm_config_* 环境变量，pnpm config set 写的配置它不读。
# confirm_modules_purge：容器里没有终端，pnpm 要重建 node_modules 时没法问，会直接报错退出
ENV pnpm_config_store_dir=/home/node/.local/share/pnpm/store \
    pnpm_config_confirm_modules_purge=false

ENV NUXT_HOST=0.0.0.0
ENV NUXT_PORT=3000
ENV TZ=Asia/Shanghai

EXPOSE 3000

# 容器保持运行，启动命令由 docker-compose 的 command 提供
CMD ["tail", "-f", "/dev/null"]
