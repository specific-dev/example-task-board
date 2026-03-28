secret "google_client_id" {}
secret "google_client_secret" {}

secret "jwt_secret" {
  generated = true
}

build "web" {
  base    = "node"
  command = "npm run build"

  env = {
    VITE_API_URL = "https://${service.api.public_url}"
  }
}

service "web" {
  build   = build.web
  command = "npx serve dist -s -l $PORT"

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
    PORT                 = port
    DATABASE_URL         = postgres.main.url
    DATABASE_SYNC_URL    = postgres.main.sync.url
    DATABASE_SYNC_SECRET = postgres.main.sync.secret
    GOOGLE_CLIENT_ID     = secret.google_client_id
    GOOGLE_CLIENT_SECRET = secret.google_client_secret
    JWT_SECRET           = secret.jwt_secret
    WEB_URL              = "https://${service.web.public_url}"
    API_URL              = "https://${service.api.public_url}"
    S3_ENDPOINT          = storage.attachments.endpoint
    S3_ACCESS_KEY        = storage.attachments.access_key
    S3_SECRET_KEY        = storage.attachments.secret_key
    S3_BUCKET            = storage.attachments.bucket
  }

  dev {
    command = "go run ."
    env = {
      WEB_URL = "http://${service.web.public_url}"
      API_URL = "http://${service.api.public_url}"
    }
  }
}

postgres "main" {
  reshape {
    enabled = true
  }
}

storage "attachments" {}
