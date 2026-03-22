build "web" {
  base    = "node"
  command = "npm run build"
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
  }
}
