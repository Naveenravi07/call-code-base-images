FROM node:20-alpine AS development

# Create restricted user
RUN addgroup -S coder && adduser -S coder -G coder -h /home/coder

# Prepare workspace and give coder ownership
RUN mkdir -p /home/coder/workspace && chown -R coder:coder /home/coder

WORKDIR /home/coder/workspace

# Copy package files with proper ownership
COPY --chown=coder:coder package.json package-lock.json ./

# Switch to coder
USER coder

# Install dependencies
RUN npm ci

# Copy the rest of the source
COPY --chown=coder:coder . .

EXPOSE 3000
CMD ["npm", "run", "dev"]
