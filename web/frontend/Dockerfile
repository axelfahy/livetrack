# Use an official Node.js runtime as the base image
FROM arm64v8/node:20-alpine3.18 AS builder

# Set the working directory in the container
WORKDIR /app

# Copy package.json and package-lock.json to the container
COPY package*.json ./

# Install app dependencies
RUN npm install

# Copy the rest of the app source code to the container
COPY . .

# Build the Svelte app
RUN npm run build

# Use a lightweight web server to serve the built app
FROM arm64v8/node:20-alpine3.18

# Copy the built Svelte app to the NGINX web server directory
COPY --from=builder /app/build build/
COPY --from=builder /app/node_modules node_modules/
COPY package.json .

EXPOSE 3000

ENV NODE_ENV=production
CMD ["node", "build"]
