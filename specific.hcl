build "web" {
  base    = "node"
  command = "npm run build"

  env = {
    VITE_API_URL = "https://${service.api.public_url}"
  }
}

service "web" {
  build   = build.web
  command = "npx serve dist -l $PORT"

  endpoint {
    public = true
  }

  env = {
    PORT = port
  }

  dev {
    command = "npx vite --port $PORT"
    env = {
      VITE_API_URL = "http://${service.api.public_url}"
    }
  }
}

build "api" {
  base    = "go"
  root    = "api"
  command = "go build -o api"
}

service "api" {
  build   = build.api
  command = "./api"

  endpoint {
    public = true
  }

  env = {
    PORT = port
  }

  dev {
    command = "go run ."
  }
}
