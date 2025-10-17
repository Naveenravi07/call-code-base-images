FROM oven/bun:1-alpine AS development

RUN addgroup -S coder && adduser -S coder -G coder -h /home/coder

RUN mkdir -p /home/coder/workspace && chown -R coder:coder /home/coder

WORKDIR /home/coder/workspace

COPY --chown=coder:coder package.json ./

USER coder

COPY --chown=coder:coder . .

EXPOSE 3000
CMD ["tail", "-f", "/dev/null"]