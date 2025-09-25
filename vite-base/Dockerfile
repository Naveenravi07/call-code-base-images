FROM node:20-alpine AS development

RUN addgroup -S coder && adduser -S coder -G coder -h /home/coder

RUN mkdir -p /home/coder/workspace && chown -R coder:coder /home/coder

WORKDIR /home/coder/workspace

COPY --chown=coder:coder package.json package-lock.json ./

USER coder

RUN npm ci

COPY --chown=coder:coder . .

EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host"]
