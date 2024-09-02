package builtin

var Basicbuiltins = []Builtin{
	{
		Name:        "node",
		Description: "Nodejs builtin",
		Details: `Requires package.json, package-lock.json and an app in server.js. 
Runs a production npm install and copies all files across. 
When run will call npm start to start the application.
Uses and exposes port 8080 internally.`,
		Template: `FROM node:current-slim
WORKDIR /app			
COPY package.json .
RUN npm install --production
COPY . .
RUN npm run build --if-present
ENV PORT=8080
EXPOSE 8080
CMD [ "npm","start" ]
	`,
	},
	{
		Name:        "ruby",
		Description: "Ruby builtin",
		Details: `Builtin for a Ruby application with a Gemfile. Runs bundle install to build. 
At runtime, it uses rackup to run config.ru and start the application as configured.
Uses and exposes port 8080 internally.`,
		Template: `FROM ruby:2.7
WORKDIR /usr/src/app
COPY Gemfile ./
RUN bundle install
COPY . .
ENV PORT=8080
EXPOSE 8080
CMD ["bundle", "exec", "rackup", "--host", "0.0.0.0", "-p", "8080"]
`},
	{Name: "deno",
		Description: "Deno builtin",
		Details: `Uses Debian image from https://github.com/hayd/deno-docker.
runs main.ts with --allow-net set and requires deps.ts for dependencies.
Uses and exposes port 8080 internally.`,
		Template: `FROM hayd/debian-deno:1.4.0
ENV PORT=8080
EXPOSE 8080
WORKDIR /app
USER deno
COPY main.ts deps.* ./
RUN /bin/bash -c "deno cache deps.ts || true"
ADD . .
RUN deno cache main.ts
CMD ["run", {{range .perms}}"{{.}}",{{end}} "main.ts"]
`,
		Settings: []Setting{{"perms", []string{`--allow-net`}, "Array of command line settings to grant permissions, e.g. [\"--allow-net\",\"--allow-read\"] "}},
	},
	{Name: "go",
		Description: "Go Builtin",
		Details: `Builds main.go from the directory, the app should use go modules.
Uses and exposes port 8080 internally.
`,
		Template: `FROM golang:latest				
		# Set the Current Working Directory inside the container
		WORKDIR /app		
		# Copy go mod and sum files
		COPY go.mod go.sum ./		
		# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
		RUN go mod download		
		# Copy the source from the current directory to the Working Directory inside the container
		COPY . .		
		# Build the Go app
		RUN go build -o main .		
		# Expose port 8080 to the outside world
		EXPOSE 8080		
		# Command to run the executable
		CMD ["./main"]
`},

	{Name: "static react",
		Description: "Web server builtin",
		Details:     `All files are copied to the image and served, It will work with ReactJS and AngularJS`,
		Template: `FROM node:14-alpine
		WORKDIR /app
		COPY package*.json ./
		RUN npm install
		COPY . .
		RUN npm run build
		EXPOSE 3000
		CMD [ "npm", "start" ]
`, Settings: []Setting{{"httpsonly", false, "Enable http to https promotion"}, {"log", false, "Enable basic logging"}}},

	{Name: "static angular",
		Description: "Web server builtin",
		Details:     `All files are copied to the image and served, It will work with AngularJS`,
		Template: `FROM node:16 as build
WORKDIR /usr/local/app
COPY ./ /usr/local/app/
RUN export NODE_OPTIONS=--openssl-legacy-provider
RUN npm install
RUN npm run build
FROM nginx:latest
COPY --from=build  /usr/local/app/dist/app /usr/share/nginx/html
EXPOSE 80
`, Settings: []Setting{{"httpsonly", false, "Enable http to https promotion"}, {"log", false, "Enable basic logging"}}},
	{Name: "typescript",
		Description: "typescript builtin",
		Details: `runs index.ts with --allow-net set and requires deps.ts for dependencies.
Uses and exposes port 8080 internally.`,
		Template: `FROM node
WORKDIR /app
COPY . .
RUN npm install
EXPOSE 4000
CMD ["npm", "run", "dev"]
`,
		Settings: []Setting{{"perms", []string{`--allow-net`}, "Array of command line settings to grant permissions, e.g. [\"--allow-net\",\"--allow-read\"] "}},
	},
	{Name: "hugo-static",
		Description: "Hugo static build with web server builtin",
		Details:     `Hugo static build, then all public files are copied to the image and served, except files with executable permission set. Uses and exposes port 8080 internally.`,
		Template: `FROM klakegg/hugo:0.74.0-onbuild AS hugo
FROM pierrezemb/gostatic
COPY --from=hugo /target /srv/http/
CMD ["-port","8080"{{if .httpsonly}},"-https-promote"{{ end }}{{if .log}},"-enable-logging"{{end}}]
`, Settings: []Setting{{"httpsonly", false, "Enable http to https promotion"}, {"log", false, "Enable basic logging"}}},

	{Name: "python",
		Description: "Python builtin",
		Details:     `Python/Procfile based builder. Requires requirements.txt and Procfile. Uses and exposes port 8080 internally.`,
		Template: `FROM python:3.8-slim-buster
ENV PORT 8080
RUN mkdir /app
RUN set -ex && \
	apt-get update && \
	apt-get install -y --no-install-recommends wget && \
	wget -O /usr/bin/hivemind.gz https://github.com/DarthSim/hivemind/releases/download/v1.0.6/hivemind-v1.0.6-linux-amd64.gz && \
	gzip -d /usr/bin/hivemind.gz && \
	chmod +x /usr/bin/hivemind
COPY . /app
WORKDIR /app
RUN pip install -r requirements.txt
CMD ["/usr/bin/hivemind", "/app/Procfile"]
`, Settings: []Setting{{"hiveversion", "1.0.6", "Version of Hivemind"}, {"pythonbase", "3.8-slim-buster", "Tag for base Python image"}}},

	{Name: "elixir",
		Description: "Elixir builtin",
		Details:     `All files are copied to the image and served, It will work with Elixir`,
		Template: `FROM elixir:latest
RUN mkdir /app
COPY . /app
WORKDIR /app
RUN mix local.hex --force
RUN mix do deps.get
RUN mix do compile
CMD ["mix", "phx.server"]
EXPOSE 4000
`, Settings: []Setting{{"httpsonly", false, "Enable http to https promotion"}, {"log", false, "Enable basic logging"}}},

	{Name: "remix",
		Description: "Go Builtin",
		Details:     `All files are copied to the image and served`,
		Template: `FROM node:latest
WORKDIR /usr/src/app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 3000
CMD ["npm", "start"]
`, Settings: []Setting{{"httpsonly", false, "Enable http to https promotion"}, {"log", false, "Enable basic logging"}}},

	{Name: "java server page",
		Description: "Web server builtin",
		Details:     `All files are copied to the image and served`,
		Template: `FROM tomcat:8.0-alpine
ADD ./Abbott /usr/local/tomcat/webapps/abbott		
EXPOSE 8080		
CMD ["catalina.sh", "run"]
`, Settings: []Setting{{"httpsonly", false, "Enable http to https promotion"}, {"log", false, "Enable basic logging"}}},

	{Name: "mysql",
		Description: "MySql",
		Details:     ``,
		Template: `

`, Settings: []Setting{{"httpsonly", false, "Enable http to https promotion"}, {"log", false, "Enable basic logging"}}},
	{Name: "postgres",
		Description: "Postgres",
		Details:     ``,
		Template: `

`, Settings: []Setting{{"httpsonly", false, "Enable http to https promotion"}, {"log", false, "Enable basic logging"}}},

	{Name: "unity",
		Description: "unity builtin",
		Details: `runs index.ts with --allow-net set and requires deps.ts for dependencies.
Uses and exposes port 8080 internally.`,
		Template: `FROM nginx:alpine

WORKDIR /etc/nginx/conf.d
COPY webgl.conf default.conf

WORKDIR /webgl
COPY webgl/ . 
EXPOSE 8080
`,
		Settings: []Setting{{"perms", []string{`--allow-net`}, "Array of command line settings to grant permissions, e.g. [\"--allow-net\",\"--allow-read\"] "}},
	},
}
