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

# an existing .env holds the encryption key; replacing it makes stored secrets unreadable
if [[ -e .env ]]; then
  echo "Error: .env already exists. Remove it first if you really want a new setup." >&2
  exit 1
fi

# generate secrets
meili_key=$(openssl rand -hex 32)
pocket_key=$(openssl rand -hex 16)
proxy_secret=$(openssl rand -hex 32)

# Download docker-compose.yml and .env.example using curl or wget
base_url=https://raw.githubusercontent.com/open-wanderer/wanderer/refs/heads/main
download() {
  if command -v wget >/dev/null 2>&1; then
    wget -O "$2" "${base_url}/$1"
  elif command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$2" "${base_url}/$1"
  else
    echo "Error: neither wget nor curl is installed." >&2
    exit 1
  fi
}
download docker-compose.yml docker-compose.yml
download .env.example .env

# write secrets and configuration to .env
SED_INPLACE "s/^MEILI_MASTER_KEY=.*/MEILI_MASTER_KEY=${meili_key}/" .env
SED_INPLACE "s/^POCKETBASE_ENCRYPTION_KEY=.*/POCKETBASE_ENCRYPTION_KEY=${pocket_key}/" .env
SED_INPLACE "s/^POCKETBASE_PROXY_SECRET=.*/POCKETBASE_PROXY_SECRET=${proxy_secret}/" .env
SED_INPLACE "s|^ORIGIN=.*|ORIGIN=${origin}|" .env
SED_INPLACE "s/^PUBLIC_DISABLE_SIGNUP=.*/PUBLIC_DISABLE_SIGNUP=${public_disable_signup}/" .env

echo "✅ Setup complete. Run 'docker compose up -d' to start the services."
