#!/usr/bin/env bash
set -euo pipefail

# Detect OS for sed compatibility
if [[ "$OSTYPE" == "darwin"* ]]; then
  SED_INPLACE() { sed -i '' "$@"; }
else
  SED_INPLACE() { sed -i "$@"; }
fi

# prompt for environment type
read -r -p "Is this a local development setup? (y/n): " is_dev

if [[ "${is_dev}" =~ ^[Yy]$ ]]; then
  origin="http://localhost:3000"
  public_disable_signup=false
else
  read -r -p "Enter the domain (e.g., example.com): " domain
  origin="https://${domain}"
  read -r -p "Allow public signups? (y/n): " allow_signups
  if [[ "${allow_signups}" =~ ^[Yy]$ ]]; then
    public_disable_signup=false
  else
    public_disable_signup=true
  fi
fi

# generate secrets
meili_key=$(openssl rand -hex 32)
pocket_key=$(openssl rand -hex 16)

# Download docker-compose.yml using curl or wget
if command -v wget >/dev/null 2>&1; then
  wget -O docker-compose.yml https://raw.githubusercontent.com/open-wanderer/wanderer/refs/heads/main/docker-compose.yml
elif command -v curl >/dev/null 2>&1; then
  curl -fsSL -o docker-compose.yml https://raw.githubusercontent.com/open-wanderer/wanderer/refs/heads/main/docker-compose.yml
else
  echo "Error: neither wget nor curl is installed." >&2
  exit 1
fi

# update docker-compose.yml with secrets and configuration
SED_INPLACE "s/MEILI_MASTER_KEY:.*/MEILI_MASTER_KEY: ${meili_key}/" docker-compose.yml
SED_INPLACE "s/POCKETBASE_ENCRYPTION_KEY:.*/POCKETBASE_ENCRYPTION_KEY: ${pocket_key}/" docker-compose.yml
SED_INPLACE "s|ORIGIN:.*|ORIGIN: ${origin}|" docker-compose.yml
SED_INPLACE "s/PUBLIC_DISABLE_SIGNUP: .*/PUBLIC_DISABLE_SIGNUP: \"${public_disable_signup}\"/" docker-compose.yml

echo "✅ Setup complete. Run 'docker compose up -d' to start the services."
